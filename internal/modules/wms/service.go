package wms

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"
)

type Service interface {
	// Zones
	CreateZone(req CreateZoneRequest) (*WMSZone, error)
	ListZones(warehouseID uint) ([]WMSZone, error)

	// Bins
	CreateBin(req CreateBinRequest) (*WMSBinLocation, error)
	ListBins(warehouseID uint, zoneID uint, status string) ([]WMSBinLocation, error)
	GetBin(id uint) (*WMSBinLocation, error)

	// Putaway
	CreatePutawayRule(req CreatePutawayRuleRequest) (*WMSPutawayRule, error)
	ListPutawayRules(warehouseID uint) ([]WMSPutawayRule, error)
	SuggestPutawayLocation(req SuggestPutawayRequest) (*PutawaySuggestion, error)
	AssignInventoryToBin(binID uint, productID uint, qty float64, weightKg float64, volumeCbm float64) error

	// Wave Picking
	CreateWavePicking(req CreateWavePickingRequest) (*WMSWavePicking, error)
	GetWavePicking(id uint) (*WMSWavePicking, error)
	ListWavePickings(warehouseID uint, status string) ([]WMSWavePicking, error)
	StartWavePicking(waveID uint) (*WMSWavePicking, error)
	ConfirmPickLine(waveID uint, lineID uint, qtyPicked float64) (*WMSWavePickingLine, error)
	CompleteWavePicking(waveID uint) (*WMSWavePicking, error)

	// Stats & AI
	GetStats(warehouseID uint) (*WMSStats, error)
	AnalyzeWarehouseSlottingWithAI(warehouseID uint) (*AIWarehouseSlottingReport, error)
}

type service struct {
	repo       Repository
	db         *gorm.DB
	stockSvc   *stock.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(repo Repository, db *gorm.DB, stockSvc *stock.Service, chatterSvc *chatter.Service, aiClient *ai.Client) Service {
	return &service{
		repo:       repo,
		db:         db,
		stockSvc:   stockSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// ---------------- Zones ----------------

func (s *service) CreateZone(req CreateZoneRequest) (*WMSZone, error) {
	zoneType := req.ZoneType
	if zoneType == "" {
		zoneType = "dry"
	}
	maxWeight := req.MaxWeightCapacity
	if maxWeight <= 0 {
		maxWeight = 20000 // default 20 ton
	}

	zone := &WMSZone{
		WarehouseID:       req.WarehouseID,
		Code:              strings.ToUpper(strings.TrimSpace(req.Code)),
		Name:              strings.TrimSpace(req.Name),
		ZoneType:          zoneType,
		TemperatureMin:    req.TemperatureMin,
		TemperatureMax:    req.TemperatureMax,
		MaxWeightCapacity: maxWeight,
		Active:            true,
		Description:       req.Description,
	}

	if err := s.repo.CreateZone(zone); err != nil {
		return nil, fmt.Errorf("gagal membuat zona gudang: %w", err)
	}
	return zone, nil
}

func (s *service) ListZones(warehouseID uint) ([]WMSZone, error) {
	return s.repo.ListZones(warehouseID)
}

// ---------------- Bins ----------------

func (s *service) CreateBin(req CreateBinRequest) (*WMSBinLocation, error) {
	zone, err := s.repo.GetZoneByID(req.ZoneID)
	if err != nil {
		return nil, fmt.Errorf("zona gudang dengan ID %d tidak ditemukan", req.ZoneID)
	}

	aisle := strings.ToUpper(strings.TrimSpace(req.Aisle))
	rack := strings.ToUpper(strings.TrimSpace(req.Rack))
	shelf := strings.ToUpper(strings.TrimSpace(req.Shelf))
	binNum := strings.ToUpper(strings.TrimSpace(req.Bin))

	// Format coordinate: WH{wh_id}-{ZONE_CODE}-{AISLE}-{RACK}-{SHELF}-{BIN}
	code := fmt.Sprintf("WH%d-%s-%s-%s-%s-%s", req.WarehouseID, zone.Code, aisle, rack, shelf, binNum)

	maxWeight := req.MaxWeightKg
	if maxWeight <= 0 {
		maxWeight = 500
	}
	maxVol := req.MaxVolumeCbm
	if maxVol <= 0 {
		maxVol = 1.5
	}
	velocity := req.VelocityCategory
	if velocity == "" {
		velocity = "B"
	}

	bin := &WMSBinLocation{
		WarehouseID:      req.WarehouseID,
		ZoneID:           req.ZoneID,
		Code:             code,
		Aisle:            aisle,
		Rack:             rack,
		Shelf:            shelf,
		Bin:              binNum,
		MaxWeightKg:      maxWeight,
		MaxVolumeCbm:     maxVol,
		Status:           "available",
		VelocityCategory: velocity,
		PickingSequence:  req.PickingSequence,
	}

	if err := s.repo.CreateBin(bin); err != nil {
		return nil, fmt.Errorf("gagal membuat bin location: %w", err)
	}
	return bin, nil
}

func (s *service) ListBins(warehouseID uint, zoneID uint, status string) ([]WMSBinLocation, error) {
	return s.repo.ListBins(warehouseID, zoneID, status)
}

func (s *service) GetBin(id uint) (*WMSBinLocation, error) {
	return s.repo.GetBinByID(id)
}

// ---------------- Putaway ----------------

func (s *service) CreatePutawayRule(req CreatePutawayRuleRequest) (*WMSPutawayRule, error) {
	strategy := req.Strategy
	if strategy == "" {
		strategy = "fastest_access"
	}
	priority := req.Priority
	if priority <= 0 {
		priority = 10
	}

	rule := &WMSPutawayRule{
		WarehouseID:  req.WarehouseID,
		ProductID:    req.ProductID,
		CategoryID:   req.CategoryID,
		TargetZoneID: req.TargetZoneID,
		Strategy:     strategy,
		Priority:     priority,
		Active:       true,
		Description:  req.Description,
	}

	if err := s.repo.CreatePutawayRule(rule); err != nil {
		return nil, fmt.Errorf("gagal membuat aturan putaway: %w", err)
	}
	return rule, nil
}

func (s *service) ListPutawayRules(warehouseID uint) ([]WMSPutawayRule, error) {
	return s.repo.ListPutawayRules(warehouseID)
}

func (s *service) SuggestPutawayLocation(req SuggestPutawayRequest) (*PutawaySuggestion, error) {
	var prod product.Product
	var categoryID uint
	if err := s.db.First(&prod, req.ProductID).Error; err == nil {
		categoryID = prod.CategoryID
	}

	// Calculate estimated weight and volume if not specified
	weight := req.WeightKg
	volume := req.VolumeCbm
	if weight <= 0 {
		weight = req.Quantity * 1.0 // default 1 kg per unit
	}
	if volume <= 0 {
		volume = req.Quantity * 0.005 // default 0.005 cbm per unit
	}

	rule, err := s.repo.FindMatchingPutawayRule(req.WarehouseID, req.ProductID, categoryID)
	if err != nil {
		// Fallback: pick any active zone in warehouse
		zones, zErr := s.repo.ListZones(req.WarehouseID)
		if zErr != nil || len(zones) == 0 {
			return nil, fmt.Errorf("tidak ada zona gudang aktif pada gudang #%d", req.WarehouseID)
		}
		rule = &WMSPutawayRule{
			TargetZoneID: zones[0].ID,
			Strategy:     "fastest_access",
		}
	}

	bin, err := s.repo.FindAvailableBinInZone(rule.TargetZoneID, weight, volume)
	if err != nil {
		return nil, fmt.Errorf("ruang penyimpanan tidak cukup: %w", err)
	}

	zoneName := "Unknown Zone"
	zoneType := "dry"
	if bin.Zone != nil {
		zoneName = bin.Zone.Name
		zoneType = bin.Zone.ZoneType
	}

	remWeight := bin.MaxWeightKg - bin.CurrentWeightKg - weight
	remVol := bin.MaxVolumeCbm - bin.CurrentVolumeCbm - volume

	return &PutawaySuggestion{
		BinLocationID:   bin.ID,
		BinLocationCode: bin.Code,
		ZoneName:        zoneName,
		ZoneType:        zoneType,
		RemainingWeight: remWeight,
		RemainingVolume: remVol,
		StrategyApplied: rule.Strategy,
		ConfidenceScore: 94,
	}, nil
}

func (s *service) AssignInventoryToBin(binID uint, productID uint, qty float64, weightKg float64, volumeCbm float64) error {
	bin, err := s.repo.GetBinByID(binID)
	if err != nil {
		return fmt.Errorf("bin location #%d tidak ditemukan: %w", binID, err)
	}

	slot, err := s.repo.GetSlot(binID, productID)
	now := time.Now()
	if err != nil {
		// New slot
		slot = &WMSInventorySlot{
			BinLocationID:    binID,
			ProductID:        productID,
			Quantity:         qty,
			WeightKg:         weightKg,
			VolumeCbm:        volumeCbm,
			LastStockCountAt: &now,
		}
	} else {
		slot.Quantity += qty
		slot.WeightKg += weightKg
		slot.VolumeCbm += volumeCbm
		slot.LastStockCountAt = &now
	}

	if err := s.repo.SaveSlot(slot); err != nil {
		return fmt.Errorf("gagal memperbarui slot inventaris: %w", err)
	}

	// Update bin metrics
	bin.CurrentWeightKg += weightKg
	bin.CurrentVolumeCbm += volumeCbm
	bin.IsOccupied = true
	bin.Status = "occupied"
	return s.repo.UpdateBin(bin)
}

// ---------------- Wave Picking ----------------

func (s *service) CreateWavePicking(req CreateWavePickingRequest) (*WMSWavePicking, error) {
	if len(req.PickingIDs) == 0 {
		return nil, fmt.Errorf("minimal satu picking / pengiriman harus disertakan dalam wave")
	}

	waveNum, err := s.repo.NextWaveNumber()
	if err != nil {
		return nil, err
	}

	wave := &WMSWavePicking{
		WaveNumber:         waveNum,
		Name:               req.Name,
		WarehouseID:        req.WarehouseID,
		Status:             "draft",
		AssignedPickerID:   req.AssignedPickerID,
		AssignedPickerName: req.AssignedPickerName,
		TotalPickings:      len(req.PickingIDs),
		Notes:              req.Notes,
	}

	// Gather StockMoves from all pickings
	var pickings []stock.StockPicking
	if err := s.db.Preload("MoveLines.Product").Where("id IN ?", req.PickingIDs).Find(&pickings).Error; err != nil {
		return nil, fmt.Errorf("gagal memuat data picking: %w", err)
	}

	// Preload all available bins for default routing mapping
	allBins, _ := s.repo.ListBins(req.WarehouseID, 0, "")
	binByProduct := make(map[uint]WMSBinLocation)

	// Map products to slots if any
	var slots []WMSInventorySlot
	s.db.Preload("BinLocation").Find(&slots)
	for _, sl := range slots {
		if sl.BinLocation != nil && sl.Quantity > 0 {
			binByProduct[sl.ProductID] = *sl.BinLocation
		}
	}

	var waveLines []WMSWavePickingLine
	for _, p := range pickings {
		for _, m := range p.MoveLines {
			targetBin, exists := binByProduct[m.ProductID]
			if !exists {
				// Assign from first available bin or placeholder
				if len(allBins) > 0 {
					targetBin = allBins[0]
				} else {
					targetBin = WMSBinLocation{
						Code:            "WH1-DRY-A01-R01-S01-B01",
						PickingSequence: 10,
					}
				}
			}

			sku := ""
			if m.Product.DefaultCode != "" {
				sku = m.Product.DefaultCode
			}

			line := WMSWavePickingLine{
				PickingID:         p.ID,
				PickingName:       p.Name,
				StockMoveID:       m.ID,
				ProductID:         m.ProductID,
				ProductName:       m.ProductName,
				ProductSku:        sku,
				BinLocationID:     targetBin.ID,
				BinLocationCode:   targetBin.Code,
				PickingSequence:   targetBin.PickingSequence,
				QuantityRequested: m.ProductUomQty,
				QuantityPicked:    0,
				Status:            "pending",
			}
			waveLines = append(waveLines, line)
		}
	}

	wave.TotalLines = len(waveLines)
	wave.PickedLines = 0
	wave.Lines = waveLines

	if err := s.repo.CreateWave(wave); err != nil {
		return nil, fmt.Errorf("gagal menyimpan wave picking: %w", err)
	}

	// Log Chatter on each stock.picking
	if s.chatterSvc != nil {
		for _, p := range pickings {
			s.chatterSvc.LogTracking(
				"stock.picking",
				p.ID,
				1,
				"WMS System",
				fmt.Sprintf("Dokumen dikelompokkan ke dalam Wave Picking %s (%s)", wave.WaveNumber, wave.Name),
				"in_progress",
				1,
			)
		}
	}

	return wave, nil
}

func (s *service) GetWavePicking(id uint) (*WMSWavePicking, error) {
	return s.repo.GetWaveByID(id)
}

func (s *service) ListWavePickings(warehouseID uint, status string) ([]WMSWavePicking, error) {
	return s.repo.ListWaves(warehouseID, status)
}

func (s *service) StartWavePicking(waveID uint) (*WMSWavePicking, error) {
	wave, err := s.repo.GetWaveByID(waveID)
	if err != nil {
		return nil, err
	}
	if wave.Status == "completed" {
		return nil, fmt.Errorf("wave picking %s sudah selesai", wave.WaveNumber)
	}

	now := time.Now()
	wave.Status = "in_progress"
	wave.StartedAt = &now

	if err := s.repo.UpdateWave(wave); err != nil {
		return nil, err
	}
	return wave, nil
}

func (s *service) ConfirmPickLine(waveID uint, lineID uint, qtyPicked float64) (*WMSWavePickingLine, error) {
	wave, err := s.repo.GetWaveByID(waveID)
	if err != nil {
		return nil, err
	}

	line, err := s.repo.GetWaveLineByID(lineID)
	if err != nil {
		return nil, fmt.Errorf("line #%d tidak ditemukan: %w", lineID, err)
	}
	if line.WaveID != waveID {
		return nil, fmt.Errorf("line #%d bukan bagian dari wave #%d", lineID, waveID)
	}

	now := time.Now()
	line.QuantityPicked = qtyPicked
	line.PickedAt = &now
	if qtyPicked >= line.QuantityRequested {
		line.Status = "picked"
	} else {
		line.Status = "shortage"
	}

	if err := s.repo.UpdateWaveLine(line); err != nil {
		return nil, err
	}

	// Deduct from slot if available
	if line.BinLocationID > 0 {
		if slot, err := s.repo.GetSlot(line.BinLocationID, line.ProductID); err == nil {
			if slot.Quantity >= qtyPicked {
				slot.Quantity -= qtyPicked
			} else {
				slot.Quantity = 0
			}
			_ = s.repo.SaveSlot(slot)
		}
	}

	// Recheck wave completion
	pickedCount := 0
	for _, l := range wave.Lines {
		if l.ID == lineID {
			if line.Status == "picked" {
				pickedCount++
			}
		} else if l.Status == "picked" {
			pickedCount++
		}
	}

	wave.PickedLines = pickedCount
	if pickedCount == len(wave.Lines) && len(wave.Lines) > 0 {
		wave.Status = "completed"
		wave.CompletedAt = &now
	} else if wave.Status == "draft" {
		wave.Status = "in_progress"
		wave.StartedAt = &now
	}
	_ = s.repo.UpdateWave(wave)

	return line, nil
}

func (s *service) CompleteWavePicking(waveID uint) (*WMSWavePicking, error) {
	wave, err := s.repo.GetWaveByID(waveID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	wave.Status = "completed"
	wave.CompletedAt = &now
	wave.PickedLines = len(wave.Lines)

	for i := range wave.Lines {
		if wave.Lines[i].Status != "picked" {
			wave.Lines[i].Status = "picked"
			wave.Lines[i].QuantityPicked = wave.Lines[i].QuantityRequested
			wave.Lines[i].PickedAt = &now
			_ = s.repo.UpdateWaveLine(&wave.Lines[i])
		}
	}

	if err := s.repo.UpdateWave(wave); err != nil {
		return nil, err
	}

	// Update corresponding StockPickings state to 'assigned' / 'done'
	pickingIDsMap := make(map[uint]bool)
	for _, l := range wave.Lines {
		pickingIDsMap[l.PickingID] = true
	}

	for pID := range pickingIDsMap {
		var p stock.StockPicking
		if err := s.db.First(&p, pID).Error; err == nil {
			p.State = "assigned"
			_ = s.db.Save(&p)

			if s.chatterSvc != nil {
				s.chatterSvc.LogTracking(
					"stock.picking",
					p.ID,
					1,
					"WMS System",
					fmt.Sprintf("Barang selesai diambil melalui Wave Picking %s. Siap dikemas dan dikirim.", wave.WaveNumber),
					"done",
					1,
				)
			}
		}
	}

	return wave, nil
}

// ---------------- Stats & AI ----------------

func (s *service) GetStats(warehouseID uint) (*WMSStats, error) {
	return s.repo.GetStats(warehouseID)
}

func (s *service) AnalyzeWarehouseSlottingWithAI(warehouseID uint) (*AIWarehouseSlottingReport, error) {
	stats, err := s.repo.GetStats(warehouseID)
	if err != nil {
		return nil, err
	}

	bins, _ := s.repo.ListBins(warehouseID, 0, "")
	zones, _ := s.repo.ListZones(warehouseID)

	fastInSlow := 0
	for _, b := range bins {
		if b.VelocityCategory == "A" && (b.Shelf == "S03" || b.Shelf == "S04") {
			fastInSlow++
		}
	}

	// Default Deterministic Heuristic Engine
	efficiencyScore := 88
	status := "OPTIMIZED"
	if fastInSlow > 2 {
		efficiencyScore = 72
		status = "SUB_OPTIMAL"
	}
	if stats.BinUtilizationPct > 85.0 {
		efficiencyScore = 65
		status = "HIGH_CONGESTION"
	}

	avgDistance := 42.5
	if status == "SUB_OPTIMAL" {
		avgDistance = 68.0
	}

	report := &AIWarehouseSlottingReport{
		OverallStatus:          status,
		EfficiencyScore:        efficiencyScore,
		AverageTravelDistanceM: avgDistance,
		FastMovingInColdBins:   fastInSlow,
		RecommendedRelocations: []string{
			"Pindahkan barang SKU Fast-Moving (Kategori A) dari rak tingkat S03 ke tingkat S01 untuk mempercepat picking 35%",
			"Relokasi bahan baku curah di lorong A01 ke zona Bulk B03 guna mengurangi kepadatan forklift di jam sibuk",
			"Tetapkan area buffer di depan Staging Dock 1 untuk menampung batch wave picking sebelum muat ke truk",
		},
		ExecutiveSummary: fmt.Sprintf(
			"Analisis Slotting Gudang AI: Total %d zona dan %d bin locations dievaluasi (Utilisasi Ruang: %.1f%%). Status tata letak gudang berstatus %s dengan estimasi jarak tempuh rata-rata picker %.1f meter per pesanan.",
			len(zones), len(bins), stats.BinUtilizationPct, status, avgDistance,
		),
		ActionableAdvice: []string{
			"Terapkan aturan Wave Picking otomatis untuk pesanan sebelum pukul 14:00 guna memangkas waktu tempuh staf",
			"Gunakan strategi putaway 'Fastest Access' untuk produk promosi atau yang memiliki turnover mingguan tinggi",
			"Lakukan cycle counting mingguan pada bin dengan utilisasi di atas 80% guna memastikan integritas stok fisik",
			"Pertimbangkan pemasangan barcode verifikasi 2D di setiap bin lokasi untuk meminimalisir kesalahan picking (human-error)",
		},
		AIProvider: "Local Heuristic Engine (Lean WMS & 3D Slotting Baseline)",
	}

	// Try Cloudflare Workers AI if configured
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		prompt := fmt.Sprintf(`Sebagai AI Warehouse Logistics Specialist, analisislah tata letak gudang dan berikan rekomendasi slotting:
Total Zona: %d, Total Bin: %d, Bins Terpakai: %d (%.1f%%), Barang Kategori A di Rak Atas: %d.
Berikan format JSON valid:
{
  "overall_status": "OPTIMIZED",
  "efficiency_score": 85,
  "average_travel_distance_m": 45.0,
  "recommended_relocations": ["saran 1", "saran 2"],
  "executive_summary": "ringkasan eksekutif",
  "actionable_advice": ["langkah 1", "langkah 2"]
}`, len(zones), len(bins), stats.OccupiedBins, stats.BinUtilizationPct, fastInSlow)

		systemPrompt := "Anda adalah Asisten AI Logistik Gudang & WMS 3D Slotting. Berikan hanya JSON valid."
		resp, err := s.aiClient.CallRaw(prompt, systemPrompt)
		if err == nil && resp != "" {
			var aiReport AIWarehouseSlottingReport
			cleanJSON := strings.TrimSpace(resp)
			cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
			cleanJSON = strings.TrimPrefix(cleanJSON, "```")
			cleanJSON = strings.TrimSuffix(cleanJSON, "```")
			cleanJSON = strings.TrimSpace(cleanJSON)

			if jErr := json.Unmarshal([]byte(cleanJSON), &aiReport); jErr == nil && aiReport.ExecutiveSummary != "" {
				aiReport.AIProvider = "Cloudflare Workers AI (@cf/meta/llama-3.1-8b-instruct)"
				return &aiReport, nil
			}
		} else {
			log.Printf("[WMS AI] Cloudflare AI fallback triggered: %v", err)
		}
	}

	return report, nil
}
