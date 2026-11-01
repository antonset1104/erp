package mrp_advanced

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/mrp"
)

type Service interface {
	// Work Centers
	CreateWorkCenter(req *CreateWorkCenterRequest, companyID, userID uint) (*WorkCenter, error)
	ListWorkCenters(companyID uint) ([]WorkCenter, error)
	GetWorkCenter(id uint) (*WorkCenter, error)

	// Routings
	CreateRouting(req *CreateRoutingRequest, companyID, userID uint) (*Routing, error)
	ListRoutings(companyID uint) ([]Routing, error)
	GetRouting(id uint) (*Routing, error)

	// Work Orders Execution
	GenerateWorkOrdersForMO(req *GenerateWorkOrdersRequest, companyID, userID uint) ([]WorkOrder, error)
	StartWorkOrder(id uint, companyID, userID uint) (*WorkOrder, error)
	FinishWorkOrder(id uint, req *CompleteWorkOrderRequest, companyID, userID uint) (*WorkOrder, error)
	ListWorkOrders(companyID uint, state string) ([]WorkOrder, error)
	GetWorkOrder(id uint) (*WorkOrder, error)
	ListWorkOrdersByMO(moID uint) ([]WorkOrder, error)

	// Gantt Capacity Planning & Analytics
	GetGanttCapacitySchedule(companyID uint, days int) (*GanttCapacityResponse, error)
	GetStats(companyID uint) (*MRPAdvancedStats, error)
	AnalyzeMRPCapacityWithAI(companyID uint) (*AIMRPCapacityReport, error)
}

type service struct {
	repo       Repository
	db         *gorm.DB
	mrpSvc     *mrp.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(repo Repository, db *gorm.DB, mrpSvc *mrp.Service, chatterSvc *chatter.Service, aiClient *ai.Client) Service {
	return &service{
		repo:       repo,
		db:         db,
		mrpSvc:     mrpSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// ----------------------------------------------------------------------------
// WORK CENTERS
// ----------------------------------------------------------------------------

func (s *service) CreateWorkCenter(req *CreateWorkCenterRequest, companyID, userID uint) (*WorkCenter, error) {
	code, err := s.repo.GetNextWorkCenterCode(s.db)
	if err != nil {
		return nil, err
	}

	wcType := req.WorkCenterType
	if wcType == "" {
		wcType = "machining"
	}
	capPerHour := req.CapacityPerHour
	if capPerHour <= 0 {
		capPerHour = 10.0
	}
	workHours := req.WorkingHoursPerDay
	if workHours <= 0 {
		workHours = 8.0
	}
	oee := req.TotalOEEPercentage
	if oee <= 0 {
		oee = 85.0
	}
	costPerHour := req.CostPerHour
	if costPerHour <= 0 {
		costPerHour = 150000.0
	}
	powerKW := req.MachinePowerKW
	if powerKW <= 0 {
		powerKW = 15.0
	}

	wc := &WorkCenter{
		Code:               code,
		Name:               req.Name,
		WorkCenterType:     wcType,
		CapacityPerHour:    capPerHour,
		WorkingHoursPerDay: workHours,
		TotalOEEPercentage: oee,
		CostPerHour:        costPerHour,
		MachinePowerKW:     powerKW,
		Status:             "available",
		Notes:              req.Notes,
	}
	wc.CompanyID = companyID
	wc.CreateUID = userID
	wc.Active = true

	if err := s.repo.CreateWorkCenter(wc); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			"mrp.workcenter", wc.ID, userID, "MRP Admin",
			fmt.Sprintf("🏭 Pusat Kerja Produksi '%s' (%s) berhasil didaftarkan. Kapasitas: %.1f unit/jam, Biaya: Rp %.0f/jam.", wc.Name, wc.Code, wc.CapacityPerHour, wc.CostPerHour),
			"", companyID,
		)
	}

	return wc, nil
}

func (s *service) ListWorkCenters(companyID uint) ([]WorkCenter, error) {
	return s.repo.ListWorkCenters(companyID)
}

func (s *service) GetWorkCenter(id uint) (*WorkCenter, error) {
	return s.repo.GetWorkCenterByID(id)
}

// ----------------------------------------------------------------------------
// ROUTINGS & OPERATIONS
// ----------------------------------------------------------------------------

func (s *service) CreateRouting(req *CreateRoutingRequest, companyID, userID uint) (*Routing, error) {
	code, err := s.repo.GetNextRoutingCode(s.db)
	if err != nil {
		return nil, err
	}

	routing := &Routing{
		RoutingCode: code,
		Name:        req.Name,
		ProductID:   req.ProductID,
		BomID:       req.BomID,
		Notes:       req.Notes,
	}
	routing.CompanyID = companyID
	routing.CreateUID = userID
	routing.Active = true

	// Build operations
	for i, opDto := range req.Operations {
		seq := opDto.Sequence
		if seq <= 0 {
			seq = (i + 1) * 10
		}
		setup := opDto.SetupTimeMinutes
		if setup < 0 {
			setup = 15.0
		}
		cycle := opDto.CycleTimePerUnitMinutes
		if cycle <= 0 {
			cycle = 5.0
		}
		teardown := opDto.TeardownTimeMinutes
		if teardown < 0 {
			teardown = 10.0
		}
		batch := opDto.BatchSize
		if batch <= 0 {
			batch = 50.0
		}

		op := RoutingOperation{
			Sequence:                seq,
			Name:                    opDto.Name,
			WorkCenterID:            opDto.WorkCenterID,
			SetupTimeMinutes:        setup,
			CycleTimePerUnitMinutes: cycle,
			TeardownTimeMinutes:     teardown,
			BatchSize:               batch,
			Description:             opDto.Description,
		}
		op.CompanyID = companyID
		op.CreateUID = userID
		op.Active = true
		routing.Operations = append(routing.Operations, op)
	}

	if err := s.repo.CreateRouting(routing); err != nil {
		return nil, err
	}

	return routing, nil
}

func (s *service) ListRoutings(companyID uint) ([]Routing, error) {
	return s.repo.ListRoutings(companyID)
}

func (s *service) GetRouting(id uint) (*Routing, error) {
	return s.repo.GetRoutingByID(id)
}

// ----------------------------------------------------------------------------
// WORK ORDERS EXECUTION
// ----------------------------------------------------------------------------

func (s *service) GenerateWorkOrdersForMO(req *GenerateWorkOrdersRequest, companyID, userID uint) ([]WorkOrder, error) {
	// 1. Fetch MO from mrp.production
	var mo mrp.MrpProduction
	if err := s.db.First(&mo, req.ProductionOrderID).Error; err != nil {
		return nil, fmt.Errorf("perintah produksi (MO) ID #%d tidak ditemukan: %w", req.ProductionOrderID, err)
	}

	// 2. Fetch Routing with Operations
	routing, err := s.repo.GetRoutingByID(req.RoutingID)
	if err != nil {
		return nil, fmt.Errorf("rute operasi (Routing) ID #%d tidak ditemukan: %w", req.RoutingID, err)
	}
	if len(routing.Operations) == 0 {
		return nil, fmt.Errorf("rute '%s' tidak memiliki langkah operasi", routing.Name)
	}

	// 3. Sequential Scheduling Timeline
	now := time.Now()
	currentTime := now
	var createdWorkOrders []WorkOrder

	for i, op := range routing.Operations {
		woCode, err := s.repo.GetNextWorkOrderCode(s.db)
		if err != nil {
			return nil, err
		}

		// Calculate Planned Duration in Minutes: Setup + (Cycle * Qty) + Teardown
		plannedMinutes := op.SetupTimeMinutes + (op.CycleTimePerUnitMinutes * mo.ProductQty) + op.TeardownTimeMinutes
		plannedStart := currentTime
		plannedEnd := plannedStart.Add(time.Duration(plannedMinutes) * time.Minute)

		initialState := "pending"
		if i == 0 {
			initialState = "ready" // First operation is ready to execute immediately
		}

		wo := WorkOrder{
			WorkOrderCode:          woCode,
			ProductionOrderID:      mo.ID,
			ProductionOrderName:    mo.Name,
			RoutingOperationID:     op.ID,
			WorkCenterID:           op.WorkCenterID,
			Sequence:               op.Sequence,
			OperationName:          op.Name,
			PlannedStartTime:       &plannedStart,
			PlannedEndTime:         &plannedEnd,
			PlannedDurationMinutes: plannedMinutes,
			QtyPlanned:             mo.ProductQty,
			QtyProduced:            0,
			QtyScrap:               0,
			State:                  initialState,
			Notes:                  fmt.Sprintf("Operasi urutan %d: %s untuk MO %s", op.Sequence, op.Name, mo.Name),
		}
		wo.CompanyID = companyID
		wo.CreateUID = userID
		wo.Active = true

		if err := s.repo.CreateWorkOrder(&wo); err != nil {
			return nil, err
		}

		// Preload WorkCenter for response
		wc, _ := s.repo.GetWorkCenterByID(op.WorkCenterID)
		if wc != nil {
			wo.WorkCenter = *wc
		}
		createdWorkOrders = append(createdWorkOrders, wo)

		// Chain next operation start time after current operation finishes
		currentTime = plannedEnd
	}

	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			"mrp.production", mo.ID, userID, "MRP Admin",
			fmt.Sprintf("⚙️ %d Perintah Kerja (Work Orders) berhasil dijadwalkan secara berantai dari Rute '%s' (%s). Total estimasi durasi: %.1f menit.",
				len(createdWorkOrders), routing.Name, routing.RoutingCode, currentTime.Sub(now).Minutes()),
			"", companyID,
		)
	}

	return createdWorkOrders, nil
}

func (s *service) StartWorkOrder(id uint, companyID, userID uint) (*WorkOrder, error) {
	wo, err := s.repo.GetWorkOrderByID(id)
	if err != nil {
		return nil, err
	}
	if wo.State == "in_progress" {
		return wo, nil
	}
	if wo.State == "done" || wo.State == "cancelled" {
		return nil, fmt.Errorf("perintah kerja '%s' sudah berstatus %s", wo.WorkOrderCode, wo.State)
	}

	now := time.Now()
	wo.State = "in_progress"
	wo.ActualStartTime = &now

	if err := s.repo.UpdateWorkOrder(wo); err != nil {
		return nil, err
	}

	// Update Work Center status to busy
	wc, err := s.repo.GetWorkCenterByID(wo.WorkCenterID)
	if err == nil && wc != nil {
		wc.Status = "busy"
		s.repo.UpdateWorkCenter(wc)
	}

	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			"mrp.workorder", wo.ID, userID, "MRP Admin",
			fmt.Sprintf("▶️ Perintah Kerja '%s' (%s) dimulai pada Pusat Kerja '%s'.", wo.WorkOrderCode, wo.OperationName, wo.WorkCenter.Name),
			"", companyID,
		)
	}

	return wo, nil
}

func (s *service) FinishWorkOrder(id uint, req *CompleteWorkOrderRequest, companyID, userID uint) (*WorkOrder, error) {
	wo, err := s.repo.GetWorkOrderByID(id)
	if err != nil {
		return nil, err
	}
	if wo.State == "done" {
		return wo, nil
	}

	now := time.Now()
	wo.ActualEndTime = &now
	wo.State = "done"

	// Duration calculation
	actualDuration := req.ActualDurationMinutes
	if actualDuration <= 0 {
		if wo.ActualStartTime != nil {
			actualDuration = now.Sub(*wo.ActualStartTime).Minutes()
		} else {
			actualDuration = wo.PlannedDurationMinutes
		}
	}
	if actualDuration <= 0 {
		actualDuration = 10.0
	}
	wo.ActualDurationMinutes = actualDuration

	// Machine Hours & Costs
	machineHours := actualDuration / 60.0
	wo.MachineHours = machineHours

	wc, _ := s.repo.GetWorkCenterByID(wo.WorkCenterID)
	if wc != nil {
		wo.TotalMachineCost = machineHours * wc.CostPerHour
		wc.Status = "available"
		s.repo.UpdateWorkCenter(wc)
	}

	qtyProduced := req.QtyProduced
	if qtyProduced <= 0 {
		qtyProduced = wo.QtyPlanned
	}
	wo.QtyProduced = qtyProduced
	wo.QtyScrap = req.QtyScrap
	if req.Notes != "" {
		wo.Notes = req.Notes
	}

	if err := s.repo.UpdateWorkOrder(wo); err != nil {
		return nil, err
	}

	// Find the next Work Order in sequence for this MO and transition to READY!
	var nextWOs []WorkOrder
	s.db.Where("production_order_id = ? AND sequence > ? AND state = 'pending'", wo.ProductionOrderID, wo.Sequence).
		Order("sequence asc").
		Find(&nextWOs)

	if len(nextWOs) > 0 {
		nextWO := &nextWOs[0]
		nextWO.State = "ready"
		s.repo.UpdateWorkOrder(nextWO)

		if s.chatterSvc != nil {
			s.chatterSvc.LogTracking(
				"mrp.workorder", nextWO.ID, userID, "MRP Admin",
				fmt.Sprintf("⏩ Operasi berikutnya '%s' (%s) kini Siap (READY) setelah operasi sebelumnya selesai.", nextWO.WorkOrderCode, nextWO.OperationName),
				"", companyID,
			)
		}
	}

	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			"mrp.workorder", wo.ID, userID, "MRP Admin",
			fmt.Sprintf("✅ Perintah Kerja '%s' Selesai. Hasil: %.0f unit (Scrap: %.0f), Jam Mesin: %.2f jam, Biaya Mesin: Rp %.0f.",
				wo.WorkOrderCode, wo.QtyProduced, wo.QtyScrap, wo.MachineHours, wo.TotalMachineCost),
			"", companyID,
		)
	}

	return wo, nil
}

func (s *service) ListWorkOrders(companyID uint, state string) ([]WorkOrder, error) {
	return s.repo.ListWorkOrders(companyID, state)
}

func (s *service) GetWorkOrder(id uint) (*WorkOrder, error) {
	return s.repo.GetWorkOrderByID(id)
}

func (s *service) ListWorkOrdersByMO(moID uint) ([]WorkOrder, error) {
	return s.repo.ListWorkOrdersByMO(moID)
}

// ----------------------------------------------------------------------------
// GANTT CAPACITY PLANNING & ANALYTICS
// ----------------------------------------------------------------------------

func (s *service) GetGanttCapacitySchedule(companyID uint, days int) (*GanttCapacityResponse, error) {
	if days <= 0 {
		days = 14
	}

	start := time.Now().AddDate(0, 0, -2) // 2 days ago
	end := start.AddDate(0, 0, days+2)

	workOrders, err := s.repo.ListWorkOrdersForGantt(companyID, start, end)
	if err != nil {
		return nil, err
	}

	workCenters, err := s.repo.ListWorkCenters(companyID)
	if err != nil {
		return nil, err
	}

	var slots []GanttCapacitySlot
	wcScheduledMap := make(map[uint]float64)
	wcActiveWOMap := make(map[uint]int)

	for _, wo := range workOrders {
		startTime := time.Now()
		if wo.PlannedStartTime != nil {
			startTime = *wo.PlannedStartTime
		}
		if wo.ActualStartTime != nil {
			startTime = *wo.ActualStartTime
		}

		endTime := startTime.Add(time.Duration(wo.PlannedDurationMinutes) * time.Minute)
		if wo.PlannedEndTime != nil {
			endTime = *wo.PlannedEndTime
		}
		if wo.ActualEndTime != nil {
			endTime = *wo.ActualEndTime
		}

		durationMins := endTime.Sub(startTime).Minutes()
		if durationMins <= 0 {
			durationMins = wo.PlannedDurationMinutes
		}

		color := "#38bdf8" // info blue
		switch wo.State {
		case "ready":
			color = "#fbbf24" // yellow
		case "in_progress":
			color = "#a855f7" // purple
		case "done":
			color = "#34d399" // green
		case "paused":
			color = "#f97316" // orange
		case "cancelled":
			color = "#ef4444" // red
		}

		slot := GanttCapacitySlot{
			WorkCenterID:    wo.WorkCenterID,
			WorkCenterName:  wo.WorkCenter.Name,
			WorkCenterCode:  wo.WorkCenter.Code,
			WorkOrderID:     wo.ID,
			WorkOrderCode:   wo.WorkOrderCode,
			OperationName:   wo.OperationName,
			MOName:          wo.ProductionOrderName,
			StartTime:       startTime,
			EndTime:         endTime,
			DurationMinutes: durationMins,
			State:           wo.State,
			ColorStatus:     color,
		}
		slots = append(slots, slot)

		wcScheduledMap[wo.WorkCenterID] += durationMins / 60.0
		if wo.State == "ready" || wo.State == "in_progress" {
			wcActiveWOMap[wo.WorkCenterID]++
		}
	}

	var summaries []WorkCenterCapacitySummary
	for _, wc := range workCenters {
		totalCapacityHours := wc.WorkingHoursPerDay * float64(days)
		scheduled := wcScheduledMap[wc.ID]

		utilization := 0.0
		if totalCapacityHours > 0 {
			utilization = (scheduled / totalCapacityHours) * 100.0
		}
		isBottleneck := utilization > 85.0

		summaries = append(summaries, WorkCenterCapacitySummary{
			WorkCenterID:          wc.ID,
			WorkCenterName:        wc.Name,
			CapacityHoursPerDay:   wc.WorkingHoursPerDay,
			ScheduledHoursTotal:   scheduled,
			UtilizationPercentage: utilization,
			IsBottleneck:          isBottleneck,
			ActiveWorkOrdersCount: wcActiveWOMap[wc.ID],
		})
	}

	return &GanttCapacityResponse{
		Slots:             slots,
		CapacitySummaries: summaries,
		HorizonDays:       days,
		StartDate:         start.Format("2006-01-02"),
		EndDate:           end.Format("2006-01-02"),
	}, nil
}

func (s *service) GetStats(companyID uint) (*MRPAdvancedStats, error) {
	stats, err := s.repo.GetStats(companyID)
	if err != nil {
		return nil, err
	}

	// Count critical bottlenecks from Gantt schedule (14 days)
	sched, err := s.GetGanttCapacitySchedule(companyID, 14)
	if err == nil && sched != nil {
		bottlenecks := 0
		for _, cs := range sched.CapacitySummaries {
			if cs.IsBottleneck {
				bottlenecks++
			}
		}
		stats.CriticalBottlenecksNum = bottlenecks
	}

	return stats, nil
}

// ----------------------------------------------------------------------------
// CLOUDFLARE WORKERS AI CAPACITY ADVISOR
// ----------------------------------------------------------------------------

func (s *service) AnalyzeMRPCapacityWithAI(companyID uint) (*AIMRPCapacityReport, error) {
	sched, err := s.GetGanttCapacitySchedule(companyID, 14)
	if err != nil {
		return nil, err
	}

	stats, _ := s.GetStats(companyID)

	// Identify worst bottleneck
	worstWC := "Semua Pusat Kerja Seimbang"
	worstUtil := 0.0
	var criticalOps []string

	for _, cs := range sched.CapacitySummaries {
		if cs.UtilizationPercentage > worstUtil {
			worstUtil = cs.UtilizationPercentage
			worstWC = cs.WorkCenterName
		}
	}

	for _, slot := range sched.Slots {
		if slot.State == "in_progress" || slot.State == "ready" {
			criticalOps = append(criticalOps, fmt.Sprintf("%s (%s di %s)", slot.OperationName, slot.WorkOrderCode, slot.WorkCenterName))
			if len(criticalOps) >= 4 {
				break
			}
		}
	}

	// Cloudflare Workers AI Call
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "Anda adalah Lead Industrial Engineer & Master Production Scheduler (MRP II) berbasis standar APICS / CPIM. " +
			"Analisis jadwal kapasitas produksi, deteksi bottleneck mesin, dan berikan rekomendasi line balancing serta optimalisasi Overall Equipment Effectiveness (OEE)."

		userPrompt := fmt.Sprintf(
			"Data Kapasitas Produksi Pabrik 14 Hari:\n"+
				"- Total Pusat Kerja: %d\n"+
				"- Rata-rata OEE Mesin: %.1f%%\n"+
				"- Pusat Kerja Beban Tertinggi: %s (Utilisasi: %.1f%%)\n"+
				"- Total Perintah Kerja (WO): %d (Aktif/In-Progress: %d, Selesai: %d)\n"+
				"- Operasi Jalur Kritis: %s\n\n"+
				"Berikan analisis JSON dengan format:\n"+
				"{\n"+
				"  \"schedule_efficiency_score\": <skor 0-100>,\n"+
				"  \"overall_status\": \"BALANCED\" | \"BOTTLENECK_DETECTED\" | \"OVERLOADED\",\n"+
				"  \"bottleneck_work_center\": \"%s\",\n"+
				"  \"bottleneck_utilization\": %.1f,\n"+
				"  \"recommended_lead_time_days\": <estimasi hari>,\n"+
				"  \"executive_summary\": \"<ringkasan singkat industrial engineering>\",\n"+
				"  \"line_balancing_advice\": [\"<saran 1>\", \"<saran 2>\", \"<saran 3>\"],\n"+
				"  \"critical_path_operations\": [\"<operasi 1>\", \"<operasi 2>\"]\n"+
				"}",
			stats.TotalWorkCenters, stats.OverallAvgOEE, worstWC, worstUtil,
			stats.TotalWorkOrders, stats.InProgressWorkOrders, stats.CompletedWorkOrders,
			strings.Join(criticalOps, ", "),
			worstWC, worstUtil,
		)

		aiResp, aiErr := s.aiClient.CallRaw(userPrompt, systemPrompt)
		if aiErr == nil && aiResp != "" {
			var report AIMRPCapacityReport
			if jsonErr := json.Unmarshal([]byte(aiResp), &report); jsonErr == nil {
				report.AIProvider = "Cloudflare Workers AI (@cf/meta/llama-3.1-8b-instruct)"
				return &report, nil
			}
		}
	}

	// Fallback to Smart Heuristic Industrial Engineering Engine
	score := 88
	status := "BALANCED"
	if worstUtil > 85.0 {
		score = 65
		status = "BOTTLENECK_DETECTED"
	} else if worstUtil > 100.0 {
		score = 45
		status = "OVERLOADED"
	}

	var advice []string
	if worstUtil > 80.0 {
		advice = append(advice, fmt.Sprintf("Tambahkan shift kerja lembur atau mesin paralel pada stasiun kerja '%s' untuk mengurai antrean.", worstWC))
		advice = append(advice, "Lakukan batch size reduction (pemotongan ukuran lot) untuk mempercepat transit komponen ke tahap berikutnya.")
	} else {
		advice = append(advice, "Kapasitas lini perakitan dan permesinan saat ini dalam kondisi optimal dengan variasi beban kerja yang terkendali.")
		advice = append(advice, "Pertahankan Total Productive Maintenance (TPM) berkala guna menjaga angka OEE mesin di atas 85%.")
	}
	advice = append(advice, "Gunakan rute alternatif (routing bypass) untuk pesanan dengan prioritas tinggi guna meminimalkan lead time.")
	advice = append(advice, "Sinkronkan rilis Purchase Order bahan baku dengan jadwal mulai Work Order pertama untuk mencegah penumpukan WIP.")

	summary := fmt.Sprintf(
		"Lini produksi beroperasi dengan status %s (Efisiensi Penjadwalan: %d/100). Pusat kerja dengan utilisasi tertinggi adalah '%s' pada %.1f%% kapasitas. Rata-rata OEE mesin pabrik tercatat %.1f%%.",
		status, score, worstWC, worstUtil, stats.OverallAvgOEE,
	)

	return &AIMRPCapacityReport{
		ScheduleEfficiencyScore: score,
		OverallStatus:            status,
		BottleneckWorkCenter:     worstWC,
		BottleneckUtilization:    worstUtil,
		RecommendedLeadTimeDays:  5,
		ExecutiveSummary:         summary,
		LineBalancingAdvice:      advice,
		CriticalPathOperations:   criticalOps,
		AIProvider:               "Local Heuristic Engine (APICS / CPIM MRP II Baseline)",
	}, nil
}
