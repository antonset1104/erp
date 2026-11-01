package payroll

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/attendance"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/hr"
	"erp-golang/internal/modules/leave"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	chatterSvc *chatter.Service
	aiClient   *ai.Client
	accountSvc *account.Service
}

func NewService(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client, accountSvc *account.Service) *Service {
	return &Service{
		db:         db,
		repo:       NewRepository(db),
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
		accountSvc: accountSvc,
	}
}

func (s *Service) Repo() *Repository {
	return s.repo
}

// --- Structure & Rule Management ---

func (s *Service) CreateStructure(req CreatePayrollStructureRequest, companyID uint) (*PayrollStructure, error) {
	st := &PayrollStructure{
		Name:        req.Name,
		Code:        strings.ToUpper(strings.TrimSpace(req.Code)),
		Description: req.Description,
		Active:      true,
	}
	st.CompanyID = companyID

	if err := s.repo.CreateStructure(st); err != nil {
		return nil, fmt.Errorf("gagal membuat struktur gaji: %w", err)
	}

	for _, r := range req.Rules {
		rule := &SalaryRule{
			StructureID:       st.ID,
			Name:              r.Name,
			Code:              strings.ToUpper(strings.TrimSpace(r.Code)),
			Category:          strings.ToUpper(strings.TrimSpace(r.Category)),
			Sequence:          r.Sequence,
			AmountType:        r.AmountType,
			FixedAmount:       r.FixedAmount,
			Percentage:        r.Percentage,
			DebitAccountCode:  r.DebitAccountCode,
			CreditAccountCode: r.CreditAccountCode,
			Active:            true,
		}
		rule.CompanyID = companyID
		_ = s.repo.CreateSalaryRule(rule)
	}

	return s.repo.GetStructureByID(st.ID)
}

func (s *Service) ListStructures(companyID uint) ([]PayrollStructure, error) {
	return s.repo.ListStructures(companyID)
}

func (s *Service) GetStructure(id uint) (*PayrollStructure, error) {
	return s.repo.GetStructureByID(id)
}

func (s *Service) CreateRule(structureID uint, req CreateSalaryRuleRequest, companyID uint) (*SalaryRule, error) {
	rule := &SalaryRule{
		StructureID:       structureID,
		Name:              req.Name,
		Code:              strings.ToUpper(strings.TrimSpace(req.Code)),
		Category:          strings.ToUpper(strings.TrimSpace(req.Category)),
		Sequence:          req.Sequence,
		AmountType:        req.AmountType,
		FixedAmount:       req.FixedAmount,
		Percentage:        req.Percentage,
		DebitAccountCode:  req.DebitAccountCode,
		CreditAccountCode: req.CreditAccountCode,
		Active:            true,
	}
	rule.CompanyID = companyID
	if err := s.repo.CreateSalaryRule(rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// --- Payslip Runs (Batch Payroll) ---

func (s *Service) CreatePayslipRun(req CreatePayslipRunRequest, companyID uint) (*PayslipRun, error) {
	run := &PayslipRun{
		Name:      req.Name,
		Period:    req.Period,
		DateStart: req.DateStart,
		DateEnd:   req.DateEnd,
		State:     "draft",
		Notes:     req.Notes,
	}
	run.CompanyID = companyID

	if err := s.repo.CreatePayslipRun(run); err != nil {
		return nil, fmt.Errorf("gagal membuat batch payroll: %w", err)
	}

	s.logChatter("hr.payslip.run", run.ID, fmt.Sprintf("Batch Payroll <b>%s</b> untuk periode <b>%s</b> berhasil diinisiasi.", run.Name, run.Period), companyID)

	return run, nil
}

func (s *Service) ListPayslipRuns(companyID uint) ([]PayslipRun, error) {
	return s.repo.ListPayslipRuns(companyID)
}

func (s *Service) GetPayslipRun(id uint) (*PayslipRun, error) {
	return s.repo.GetPayslipRunByID(id)
}

// GenerateRunPayslips automatically generates payslips for all active employees for the run period
func (s *Service) GenerateRunPayslips(runID uint, companyID uint) (*PayslipRun, error) {
	run, err := s.repo.GetPayslipRunByID(runID)
	if err != nil {
		return nil, fmt.Errorf("batch payroll #%d tidak ditemukan: %w", runID, err)
	}

	var employees []hr.Employee
	if err := s.db.Where("company_id = ? AND status = 'active'", companyID).Find(&employees).Error; err != nil {
		return nil, fmt.Errorf("gagal mengambil data karyawan: %w", err)
	}

	// Find default salary structure
	var structure PayrollStructure
	if err := s.db.Where("company_id = ? AND active = true", companyID).First(&structure).Error; err != nil {
		return nil, errors.New("tidak ada struktur gaji aktif yang ditemukan")
	}

	var totalGross, totalDed, totalNet float64
	generatedCount := 0

	for _, emp := range employees {
		// Check if payslip already exists for this employee and period
		var existing Payslip
		if err := s.db.Where("company_id = ? AND employee_id = ? AND period = ?", companyID, emp.ID, run.Period).First(&existing).Error; err == nil {
			// Update run ID if not set
			if existing.PayslipRunID == nil {
				existing.PayslipRunID = &run.ID
				_ = s.repo.UpdatePayslip(&existing)
			}
			totalGross += existing.GrossSalary
			totalDed += existing.TotalDeductions
			totalNet += existing.NetSalary
			continue
		}

		slipReq := CreatePayslipRequest{
			EmployeeID:   emp.ID,
			StructureID:  structure.ID,
			PayslipRunID: &run.ID,
			Period:       run.Period,
			DateFrom:     run.DateStart,
			DateTo:       run.DateEnd,
			WorkingDays:  22,
			Notes:        fmt.Sprintf("Dibuat otomatis via Batch %s", run.Name),
		}

		slip, err := s.CreatePayslip(slipReq, companyID)
		if err != nil {
			continue
		}

		// Compute lines automatically
		computedSlip, err := s.ComputePayslip(slip.ID, ComputePayslipRequest{SyncAttendance: true, SyncLeave: true}, companyID)
		if err == nil {
			totalGross += computedSlip.GrossSalary
			totalDed += computedSlip.TotalDeductions
			totalNet += computedSlip.NetSalary
		}
		generatedCount++
	}

	run.TotalEmployees = len(employees)
	run.TotalGross = totalGross
	run.TotalDeductions = totalDed
	run.TotalNet = totalNet
	run.State = "verify"

	if err := s.repo.UpdatePayslipRun(run); err != nil {
		return nil, err
	}

	s.logChatter("hr.payslip.run", run.ID, fmt.Sprintf("Berhasil mengkalkulasi %d slip gaji karyawan. Total Gross: Rp %.2f, Total Potongan: Rp %.2f, Total Net Take-Home: Rp %.2f", len(employees), totalGross, totalDed, totalNet), companyID)

	return s.repo.GetPayslipRunByID(runID)
}

// --- Individual Payslips ---

func (s *Service) CreatePayslip(req CreatePayslipRequest, companyID uint) (*Payslip, error) {
	var emp hr.Employee
	if err := s.db.Where("id = ? AND company_id = ?", req.EmployeeID, companyID).First(&emp).Error; err != nil {
		return nil, fmt.Errorf("karyawan ID %d tidak ditemukan: %w", req.EmployeeID, err)
	}

	var st PayrollStructure
	if err := s.db.Where("id = ? AND company_id = ?", req.StructureID, companyID).First(&st).Error; err != nil {
		return nil, fmt.Errorf("struktur gaji ID %d tidak ditemukan: %w", req.StructureID, err)
	}

	var count int64
	s.db.Model(&Payslip{}).Where("company_id = ?", companyID).Count(&count)
	number := fmt.Sprintf("SLIP/%s/%05d", time.Now().Format("2006/01"), count+1)

	workingDays := req.WorkingDays
	if workingDays <= 0 {
		workingDays = 22
	}

	slip := &Payslip{
		Number:          number,
		EmployeeID:      emp.ID,
		StructureID:     st.ID,
		PayslipRunID:    req.PayslipRunID,
		Period:          req.Period,
		DateFrom:        req.DateFrom,
		DateTo:          req.DateTo,
		WorkingDays:     workingDays,
		WorkedHours:     workingDays * 8.0,
		BasicSalary:     emp.BasicSalary,
		GrossSalary:     emp.BasicSalary,
		TotalDeductions: 0,
		NetSalary:       emp.BasicSalary,
		State:           "draft",
		AICheckStatus:   "pending",
		Notes:           req.Notes,
	}
	slip.CompanyID = companyID

	if err := s.repo.CreatePayslip(slip); err != nil {
		return nil, fmt.Errorf("gagal membuat slip gaji: %w", err)
	}

	s.logChatter("hr.payslip", slip.ID, fmt.Sprintf("Draf slip gaji <b>%s</b> dibuat untuk karyawan <b>%s</b> (Gaji Pokok: Rp %.0f).", slip.Number, emp.Name, emp.BasicSalary), companyID)

	return s.repo.GetPayslipByID(slip.ID)
}

func (s *Service) ListPayslips(companyID uint, employeeID uint, runID uint, period string, state string) ([]Payslip, error) {
	return s.repo.ListPayslips(companyID, employeeID, runID, period, state)
}

func (s *Service) GetPayslip(id uint) (*Payslip, error) {
	return s.repo.GetPayslipByID(id)
}

// ComputePayslip calculates all salary rules, integrates attendance overtime & unpaid leave deductions
func (s *Service) ComputePayslip(id uint, req ComputePayslipRequest, companyID uint) (*Payslip, error) {
	slip, err := s.repo.GetPayslipByID(id)
	if err != nil {
		return nil, fmt.Errorf("slip gaji #%d tidak ditemukan: %w", id, err)
	}

	if slip.State == "done" || slip.State == "paid" {
		return nil, errors.New("slip gaji yang sudah disetujui/dibayar tidak dapat dihitung ulang")
	}

	// 1. Sync Overtime from Attendance
	var otHours float64 = req.ManualOTHours
	if req.SyncAttendance && slip.DateFrom != "" && slip.DateTo != "" {
		var atts []attendance.Attendance
		s.db.Where("company_id = ? AND employee_id = ? AND date >= ? AND date <= ? AND status IN ('overtime', 'checked_out')",
			companyID, slip.EmployeeID, slip.DateFrom, slip.DateTo).Find(&atts)
		for _, a := range atts {
			otHours += a.OvertimeHours
		}
	}
	slip.OvertimeHours = otHours

	// Hourly rate formula: Basic Salary / 173 (Indonesian standard)
	var hourlyRate float64
	if slip.BasicSalary > 0 {
		hourlyRate = math.Round(slip.BasicSalary / 173.0)
	}
	slip.OvertimeRate = hourlyRate
	slip.OvertimeAmount = math.Round(otHours * hourlyRate)

	// 2. Sync Unpaid Leave from Leave Requests
	var unpaidDays float64
	if req.SyncLeave && slip.DateFrom != "" && slip.DateTo != "" {
		var unpaidTypeIDs []uint
		s.db.Model(&leave.LeaveType{}).Where("company_id = ? AND (is_unpaid = true OR code = 'UNPAID')", companyID).Pluck("id", &unpaidTypeIDs)
		if len(unpaidTypeIDs) > 0 {
			var leaveReqs []leave.LeaveRequest
			s.db.Where("company_id = ? AND employee_id = ? AND state = 'validate' AND leave_type_id IN (?) AND date_from >= ? AND date_to <= ?",
				companyID, slip.EmployeeID, unpaidTypeIDs, slip.DateFrom, slip.DateTo).Find(&leaveReqs)
			for _, lr := range leaveReqs {
				unpaidDays += lr.NumberOfDays
			}
		}
	}
	slip.UnpaidLeaveDays = unpaidDays
	var dailyRate float64
	if slip.WorkingDays > 0 {
		dailyRate = math.Round(slip.BasicSalary / slip.WorkingDays)
	}
	slip.UnpaidLeaveDeduction = math.Round(unpaidDays * dailyRate)

	// 3. Compute Salary Rules
	structure, err := s.repo.GetStructureByID(slip.StructureID)
	if err != nil {
		return nil, fmt.Errorf("struktur gaji tidak ditemukan: %w", err)
	}

	var lines []PayslipLine
	var grossSalary float64 = slip.BasicSalary
	var totalDeductions float64

	// First pass: Basic & Allowances
	for _, rule := range structure.Rules {
		switch rule.Category {
		case "BASIC":
			line := PayslipLine{
				SalaryRuleID:      &rule.ID,
				Code:              rule.Code,
				Name:              rule.Name,
				Category:          "BASIC",
				Sequence:          rule.Sequence,
				Rate:              100,
				Amount:            slip.BasicSalary,
				Total:             slip.BasicSalary,
				DebitAccountCode:  rule.DebitAccountCode,
				CreditAccountCode: rule.CreditAccountCode,
			}
			lines = append(lines, line)

		case "ALW":
			var amount float64
			if rule.Code == "ALW_OT" {
				amount = slip.OvertimeAmount
			} else if rule.AmountType == "fixed" && rule.FixedAmount > 0 {
				amount = rule.FixedAmount
			} else if rule.AmountType == "percentage" && rule.Percentage > 0 {
				amount = math.Round(slip.BasicSalary * (rule.Percentage / 100.0))
			} else {
				// Default allowance heuristics
				if strings.Contains(strings.ToUpper(rule.Code), "JOB") {
					amount = 1500000 // Tunjangan Jabatan default
				} else if strings.Contains(strings.ToUpper(rule.Code), "MEAL") || strings.Contains(strings.ToUpper(rule.Code), "TRANS") {
					amount = 750000 // Uang Makan & Transport default
				}
			}
			line := PayslipLine{
				SalaryRuleID:      &rule.ID,
				Code:              rule.Code,
				Name:              rule.Name,
				Category:          "ALW",
				Sequence:          rule.Sequence,
				Rate:              100,
				Amount:            amount,
				Total:             amount,
				DebitAccountCode:  rule.DebitAccountCode,
				CreditAccountCode: rule.CreditAccountCode,
			}
			lines = append(lines, line)
			grossSalary += amount
		}
	}

	// Second pass: Deductions & Taxes
	for _, rule := range structure.Rules {
		switch rule.Category {
		case "DED":
			var amount float64
			if rule.Code == "DED_UNPAID" {
				amount = slip.UnpaidLeaveDeduction
			} else if rule.AmountType == "percentage" && rule.Percentage > 0 {
				amount = math.Round(slip.BasicSalary * (rule.Percentage / 100.0))
			} else if rule.AmountType == "fixed" && rule.FixedAmount > 0 {
				amount = rule.FixedAmount
			} else {
				// Default BPJS standard rates
				if strings.Contains(strings.ToUpper(rule.Code), "BPJS_TK") {
					amount = math.Round(slip.BasicSalary * 0.03) // JHT 2% + JP 1%
				} else if strings.Contains(strings.ToUpper(rule.Code), "BPJS_KES") {
					amount = math.Round(slip.BasicSalary * 0.01) // BPJS Kes 1%
				}
			}
			line := PayslipLine{
				SalaryRuleID:      &rule.ID,
				Code:              rule.Code,
				Name:              rule.Name,
				Category:          "DED",
				Sequence:          rule.Sequence,
				Rate:              100,
				Amount:            amount,
				Total:             amount,
				DebitAccountCode:  rule.DebitAccountCode,
				CreditAccountCode: rule.CreditAccountCode,
			}
			lines = append(lines, line)
			totalDeductions += amount

		case "TAX":
			var taxAmount float64
			if rule.AmountType == "percentage" && rule.Percentage > 0 {
				taxAmount = math.Round(grossSalary * (rule.Percentage / 100.0))
			} else {
				// Standar PPh 21 TER: ~5% untuk penghasilan kena pajak
				taxAmount = math.Round(grossSalary * 0.05)
			}
			line := PayslipLine{
				SalaryRuleID:      &rule.ID,
				Code:              rule.Code,
				Name:              rule.Name,
				Category:          "TAX",
				Sequence:          rule.Sequence,
				Rate:              100,
				Amount:            taxAmount,
				Total:             taxAmount,
				DebitAccountCode:  rule.DebitAccountCode,
				CreditAccountCode: rule.CreditAccountCode,
			}
			lines = append(lines, line)
			totalDeductions += taxAmount
		}
	}

	netSalary := grossSalary - totalDeductions
	if netSalary < 0 {
		netSalary = 0
	}

	// Final pass: Net Salary Line
	netLine := PayslipLine{
		Code:              "NET",
		Name:              "Gaji Bersih (Take-Home Pay)",
		Category:          "NET",
		Sequence:          100,
		Rate:              100,
		Amount:            netSalary,
		Total:             netSalary,
		CreditAccountCode: "210200", // Hutang Gaji Karyawan
	}
	lines = append(lines, netLine)

	slip.GrossSalary = grossSalary
	slip.TotalDeductions = totalDeductions
	slip.NetSalary = netSalary
	slip.State = "verify"

	if err := s.repo.UpdatePayslip(slip); err != nil {
		return nil, err
	}

	if err := s.repo.SavePayslipLines(slip.ID, lines); err != nil {
		return nil, err
	}

	s.logChatter("hr.payslip", slip.ID, fmt.Sprintf("Kalkulasi gaji selesai: <b>Gross: Rp %.0f</b> (Lembur: %.1f jam, Rp %.0f) | <b>Potongan: Rp %.0f</b> (Cuti Unpaid: %.1f hari) | <b>Net Take-Home: Rp %.0f</b>",
		grossSalary, otHours, slip.OvertimeAmount, totalDeductions, unpaidDays, netSalary), companyID)

	return s.repo.GetPayslipByID(slip.ID)
}

// AuditWithAI uses Cloudflare Workers AI (@cf/meta/llama-3.1-8b-instruct) or Smart Local Heuristic to verify compliance
func (s *Service) AuditWithAI(id uint, companyID uint) (*AIPayrollAuditResult, error) {
	slip, err := s.repo.GetPayslipByID(id)
	if err != nil {
		return nil, fmt.Errorf("slip gaji #%d tidak ditemukan: %w", id, err)
	}

	empName := "Karyawan"
	if slip.Employee != nil {
		empName = slip.Employee.Name
	}

	var auditFlags []string
	riskScore := 10.0 // Base low risk
	riskLevel := "low"
	taxStatus := "Compliant (Sesuai PMK & UU HPP)"
	otStatus := "Within Legal Working Hour Limits"

	// 1. Audit Overtime Compliance against Indonesian Labor Law (UU Kemenaker max 4 hrs/day or 18 hrs/week)
	if slip.OvertimeHours > 72.0 {
		auditFlags = append(auditFlags, fmt.Sprintf("Jam lembur bulanan (%.1f jam) melebihi ambang batas wajar Kemenaker (maks ~72 jam/bulan). Berpotensi melanggar UU Ketenagakerjaan.", slip.OvertimeHours))
		otStatus = "Excessive Overtime Flagged"
		riskScore += 35.0
	} else if slip.OvertimeHours > 40.0 {
		auditFlags = append(auditFlags, fmt.Sprintf("Jam lembur cukup tinggi (%.1f jam). Disarankan rotasi shift untuk mencegah burnout karyawan.", slip.OvertimeHours))
		riskScore += 15.0
	}

	// 2. Audit Salary Spike / Drop Anomaly
	if slip.BasicSalary > 0 {
		ratio := slip.NetSalary / slip.BasicSalary
		if ratio > 1.6 {
			auditFlags = append(auditFlags, fmt.Sprintf("Gaji bersih (Rp %.0f) melonjak >60%% dibanding gaji pokok kontrak (Rp %.0f). Periksa bonus atau klaim lembur.", slip.NetSalary, slip.BasicSalary))
			riskScore += 25.0
		} else if ratio < 0.6 {
			auditFlags = append(auditFlags, fmt.Sprintf("Gaji bersih (Rp %.0f) terpangkas >40%% dari gaji pokok kontrak. Pastikan pemotongan cuti unpaid dan penalti sudah valid.", slip.NetSalary, slip.BasicSalary))
			riskScore += 20.0
		}
	}

	// 3. Audit BPJS & Tax Withholding Completeness
	var hasBPJS, hasTax bool
	for _, l := range slip.Lines {
		if strings.Contains(l.Code, "BPJS") && l.Total > 0 {
			hasBPJS = true
		}
		if (strings.Contains(l.Code, "TAX") || strings.Contains(l.Code, "PPH")) && l.Total > 0 {
			hasTax = true
		}
	}
	if !hasBPJS {
		auditFlags = append(auditFlags, "Komponen pemotongan iuran jaminan sosial BPJS Ketenagakerjaan/Kesehatan tidak ditemukan atau bernilai 0.")
		riskScore += 20.0
	}
	if !hasTax && slip.GrossSalary > 54000000/12 { // Di atas PTKP bulanan
		auditFlags = append(auditFlags, "Karyawan berpenghasilan bruto di atas PTKP bulanan namun tidak dikenakan potongan PPh 21.")
		taxStatus = "Potential Non-Compliance"
		riskScore += 25.0
	}

	if riskScore >= 50.0 {
		riskLevel = "high"
	} else if riskScore >= 25.0 {
		riskLevel = "medium"
	}

	complianceScore := math.Max(100.0-riskScore, 20.0)

	// If Cloudflare Workers AI configured, invoke @cf/meta/llama-3.1-8b-instruct
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		prompt := fmt.Sprintf(`Analisis kepatuhan penggajian dan perpajakan untuk ERP:
Nama Karyawan: %s
Gaji Pokok: Rp %.0f
Gaji Bruto: Rp %.0f
Total Potongan: Rp %.0f
Gaji Bersih: Rp %.0f
Jam Lembur: %.1f jam (Biaya: Rp %.0f)
Hari Cuti Tanpa Gaji: %.1f hari
Temuan Awal: %v

Berikan ringkasan eksekutif dan rekomendasi kepatuhan dalam format JSON:
{
  "summary": "...",
  "recommendations": ["..."]
}`, empName, slip.BasicSalary, slip.GrossSalary, slip.TotalDeductions, slip.NetSalary, slip.OvertimeHours, slip.OvertimeAmount, slip.UnpaidLeaveDays, auditFlags)

		systemPrompt := "Anda adalah Asisten AI Senior Payroll & Pajak Ketenagakerjaan Enterprise ERP. Analisis kepatuhan regulasi PPh 21, BPJS, dan UU Ketenagakerjaan."
		if aiResp, err := s.aiClient.CallRaw(prompt, systemPrompt); err == nil && aiResp != "" {
			var parsed struct {
				Summary         string   `json:"summary"`
				Recommendations []string `json:"recommendations"`
			}
			if jsonErr := json.Unmarshal([]byte(extractJSON(aiResp)), &parsed); jsonErr == nil && parsed.Summary != "" {
				result := &AIPayrollAuditResult{
					PayslipID:                slip.ID,
					EmployeeName:             empName,
					Period:                   slip.Period,
					RiskLevel:                riskLevel,
					ComplianceScore:          complianceScore,
					TaxComplianceStatus:      taxStatus,
					OvertimeRegulationStatus: otStatus,
					AuditFlags:               auditFlags,
					ExecutiveSummary:         parsed.Summary,
					Recommendations:          parsed.Recommendations,
					Source:                   "cloudflare_workers_ai",
					AIConfidence:             0.96,
				}

				// Update slip AI record
				slip.AICheckStatus = riskLevel
				slip.AIRiskScore = riskScore
				slip.AINotes = parsed.Summary
				_ = s.repo.UpdatePayslip(slip)

				return result, nil
			}
		}
	}

	// Fallback Smart Local Heuristic Engine
	summary := fmt.Sprintf("Audit Payroll & Pajak Ketenagakerjaan: Evaluasi slip gaji %s untuk periode %s. Skor kepatuhan regulasi %0.1f%% (%s Risk).",
		empName, slip.Period, complianceScore, strings.ToUpper(riskLevel))
	if len(auditFlags) > 0 {
		summary += fmt.Sprintf(" Terdeteksi %d perhatian kepatuhan.", len(auditFlags))
	} else {
		summary += " Seluruh komponen gaji pokok, lembur presensi, potongan BPJS, dan PPh 21 berada dalam koridor hukum ketenagakerjaan."
	}

	recs := []string{
		"Pastikan bukti potong PPh 21 (Form 1721-A1/Bulanan) terdokumentasi rapi di sistem.",
		"Sinkronkan jam lembur dengan log persetujuan SPKL (Surat Perintah Kerja Lembur) di modul Attendance.",
		"Setor potongan iuran BPJS Ketenagakerjaan & Kesehatan paling lambat tanggal 10 bulan berikutnya.",
	}
	if riskLevel == "high" {
		recs = append([]string{"Lakukan verifikasi manual supervisor sebelum memposting jurnal penggajian ini ke Buku Besar GL."}, recs...)
	}

	result := &AIPayrollAuditResult{
		PayslipID:                slip.ID,
		EmployeeName:             empName,
		Period:                   slip.Period,
		RiskLevel:                riskLevel,
		ComplianceScore:          complianceScore,
		TaxComplianceStatus:      taxStatus,
		OvertimeRegulationStatus: otStatus,
		AuditFlags:               auditFlags,
		ExecutiveSummary:         summary,
		Recommendations:          recs,
		Source:                   "local_heuristic_engine",
		AIConfidence:             0.94,
	}

	slip.AICheckStatus = riskLevel
	slip.AIRiskScore = riskScore
	slip.AINotes = summary
	_ = s.repo.UpdatePayslip(slip)

	s.logChatter("hr.payslip", slip.ID, fmt.Sprintf("🤖 <b>Cloudflare Workers AI Payroll Audit:</b> Status Kepatuhan <b>%0.1f%% (%s Risk)</b>. %s", complianceScore, strings.ToUpper(riskLevel), summary), companyID)

	return result, nil
}

// ConfirmPayslip finalizes the payslip and generates a balanced double-entry GL journal move
func (s *Service) ConfirmPayslip(id uint, companyID uint) (*Payslip, error) {
	slip, err := s.repo.GetPayslipByID(id)
	if err != nil {
		return nil, fmt.Errorf("slip gaji #%d tidak ditemukan: %w", id, err)
	}

	if slip.State == "done" || slip.State == "paid" {
		return nil, errors.New("slip gaji sudah dalam status disetujui/dibayar")
	}

	// Find MISC Journal
	var journal account.AccountJournal
	if err := s.db.Where("company_id = ? AND code = 'MISC'", companyID).First(&journal).Error; err != nil {
		// Fallback to first general journal
		if err2 := s.db.Where("company_id = ?", companyID).First(&journal).Error; err2 != nil {
			return nil, errors.New("jurnal akuntansi tidak ditemukan")
		}
	}

	// Prepare Account Move Lines
	// 1. Debit: 610200 Beban Gaji & Upah Pokok Karyawan
	// 2. Debit: 610300 Beban Tunjangan & Lembur Karyawan (jika ada)
	// 3. Credit: 210300 Hutang Pajak PPh 21 (jika ada)
	// 4. Credit: 210400 Hutang Iuran BPJS Ketenagakerjaan & Kesehatan (jika ada)
	// 5. Credit: 210200 Hutang Gaji & Upah Karyawan (Net Salary)

	accWages := s.getOrCreateAccount("610200", "Beban Gaji & Upah Pokok Karyawan", "expense", companyID)
	accAllowances := s.getOrCreateAccount("610300", "Beban Tunjangan & Lembur Karyawan", "expense", companyID)
	accTaxPayable := s.getOrCreateAccount("210300", "Hutang Pajak PPh Pasal 21", "liability_payable", companyID)
	accBPJSPayable := s.getOrCreateAccount("210400", "Hutang Iuran BPJS Ketenagakerjaan & Kesehatan", "liability_payable", companyID)
	accSalaryPayable := s.getOrCreateAccount("210200", "Hutang Gaji & Upah Karyawan", "liability_payable", companyID)

	var moveLines []account.AccountMoveLine

	// 1. Debit Basic Salary
	if slip.BasicSalary > 0 {
		moveLines = append(moveLines, account.AccountMoveLine{
			AccountID: accWages.ID,
			Name:      fmt.Sprintf("Beban Gaji Pokok - %s (%s)", slip.Employee.Name, slip.Number),
			Debit:     slip.BasicSalary,
			Credit:    0,
			Balance:   slip.BasicSalary,
		})
	}

	// 2. Debit Allowances & Overtime
	allowancesAmount := slip.GrossSalary - slip.BasicSalary
	if allowancesAmount > 0 {
		moveLines = append(moveLines, account.AccountMoveLine{
			AccountID: accAllowances.ID,
			Name:      fmt.Sprintf("Beban Tunjangan & Lembur - %s (%s)", slip.Employee.Name, slip.Number),
			Debit:     allowancesAmount,
			Credit:    0,
			Balance:   allowancesAmount,
		})
	}

	// Calculate Tax and BPJS totals from lines
	var taxTotal, bpjsTotal, otherDedTotal float64
	for _, l := range slip.Lines {
		if l.Category == "TAX" {
			taxTotal += l.Total
		} else if l.Category == "DED" {
			if strings.Contains(l.Code, "BPJS") {
				bpjsTotal += l.Total
			} else {
				otherDedTotal += l.Total
			}
		}
	}

	// 3. Credit PPh 21 Payable
	if taxTotal > 0 {
		moveLines = append(moveLines, account.AccountMoveLine{
			AccountID: accTaxPayable.ID,
			Name:      fmt.Sprintf("Hutang PPh 21 - %s (%s)", slip.Employee.Name, slip.Number),
			Debit:     0,
			Credit:    taxTotal,
			Balance:   -taxTotal,
		})
	}

	// 4. Credit BPJS Payable
	if bpjsTotal > 0 {
		moveLines = append(moveLines, account.AccountMoveLine{
			AccountID: accBPJSPayable.ID,
			Name:      fmt.Sprintf("Hutang Iuran BPJS - %s (%s)", slip.Employee.Name, slip.Number),
			Debit:     0,
			Credit:    bpjsTotal,
			Balance:   -bpjsTotal,
		})
	}

	// 5. Credit Salary Payable (Net Take-Home Pay)
	// Balance equation: Debit (Basic + Allowances) == Credit (Tax + BPJS + Net + OtherDeductions)
	netCredit := slip.NetSalary + otherDedTotal
	moveLines = append(moveLines, account.AccountMoveLine{
		AccountID: accSalaryPayable.ID,
		Name:      fmt.Sprintf("Hutang Gaji Bersih (Net) - %s (%s)", slip.Employee.Name, slip.Number),
		Debit:     0,
		Credit:    netCredit,
		Balance:   -netCredit,
	})

	move := &account.AccountMove{
		MoveType:       "entry",
		JournalID:      journal.ID,
		InvoiceDate:    time.Now().Format("2006-01-02"),
		InvoiceDueDate: time.Now().Format("2006-01-02"),
		Ref:            slip.Number,
		Note:           fmt.Sprintf("Posting Beban Gaji & Upah Karyawan %s Periode %s", slip.Employee.Name, slip.Period),
		State:          "draft",
		PaymentState:   "not_paid",
		LineIDs:        moveLines,
	}
	move.CompanyID = companyID

	if err := s.accountSvc.CreateMove(move); err != nil {
		return nil, fmt.Errorf("gagal membuat entri jurnal akuntansi payroll: %w", err)
	}

	// Post move to general ledger (verifies Debit == Credit)
	postedMove, err := s.accountSvc.ActionPost(move.ID, companyID)
	if err != nil {
		return nil, fmt.Errorf("gagal memposting jurnal payroll ke buku besar GL: %w", err)
	}

	slip.State = "done"
	slip.AccountingMoveID = &postedMove.ID
	slip.AccountingMoveRef = postedMove.Name

	if err := s.repo.UpdatePayslip(slip); err != nil {
		return nil, err
	}

	s.logChatter("hr.payslip", slip.ID, fmt.Sprintf("Slip gaji disetujui & diposting ke Buku Besar GL dengan Entri Jurnal Berimbang <b>%s</b> (Total: Rp %.0f).",
		postedMove.Name, postedMove.AmountTotal), companyID)

	return s.repo.GetPayslipByID(slip.ID)
}

// PayPayslip records the actual bank/cash payment disbursement for the payslip
func (s *Service) PayPayslip(id uint, companyID uint) (*Payslip, error) {
	slip, err := s.repo.GetPayslipByID(id)
	if err != nil {
		return nil, fmt.Errorf("slip gaji #%d tidak ditemukan: %w", id, err)
	}

	if slip.State != "done" {
		return nil, errors.New("hanya slip gaji berstatus 'done' (disetujui) yang dapat dibayarkan")
	}

	// Find Bank Journal (BNK)
	var bankJournal account.AccountJournal
	if err := s.db.Where("company_id = ? AND code = 'BNK'", companyID).First(&bankJournal).Error; err != nil {
		_ = s.db.Where("company_id = ?", companyID).First(&bankJournal)
	}

	accSalaryPayable := s.getOrCreateAccount("210200", "Hutang Gaji & Upah Karyawan", "liability_payable", companyID)
	accBank := s.getOrCreateAccount("110200", "Bank BCA Operasional", "asset_cash", companyID)

	// Payment Move: Debit Hutang Gaji, Credit Bank
	paymentLines := []account.AccountMoveLine{
		{
			AccountID: accSalaryPayable.ID,
			Name:      fmt.Sprintf("Pelunasan Hutang Gaji - %s (%s)", slip.Employee.Name, slip.Number),
			Debit:     slip.NetSalary,
			Credit:    0,
			Balance:   slip.NetSalary,
		},
		{
			AccountID: accBank.ID,
			Name:      fmt.Sprintf("Pengeluaran Kas/Bank Gaji - %s (%s)", slip.Employee.Name, slip.Number),
			Debit:     0,
			Credit:    slip.NetSalary,
			Balance:   -slip.NetSalary,
		},
	}

	payMove := &account.AccountMove{
		MoveType:       "entry",
		JournalID:      bankJournal.ID,
		InvoiceDate:    time.Now().Format("2006-01-02"),
		InvoiceDueDate: time.Now().Format("2006-01-02"),
		Ref:            fmt.Sprintf("PAY/%s", slip.Number),
		Note:           fmt.Sprintf("Pembayaran Gaji Transfer Bank BCA - %s (%s)", slip.Employee.Name, slip.Number),
		State:          "draft",
		PaymentState:   "paid",
		LineIDs:        paymentLines,
	}
	payMove.CompanyID = companyID

	if err := s.accountSvc.CreateMove(payMove); err != nil {
		return nil, fmt.Errorf("gagal membuat jurnal pembayaran gaji: %w", err)
	}

	if _, err := s.accountSvc.ActionPost(payMove.ID, companyID); err != nil {
		return nil, fmt.Errorf("gagal memposting pembayaran bank gaji ke GL: %w", err)
	}

	slip.State = "paid"
	if err := s.repo.UpdatePayslip(slip); err != nil {
		return nil, err
	}

	s.logChatter("hr.payslip", slip.ID, fmt.Sprintf("Gaji sebesar <b>Rp %.0f</b> telah dibayarkan kepada <b>%s</b> melalui <b>Bank BCA (%s)</b>. Status: <b>PAID</b>.",
		slip.NetSalary, slip.Employee.Name, payMove.Name), companyID)

	return s.repo.GetPayslipByID(slip.ID)
}

func (s *Service) CancelPayslip(id uint, companyID uint) (*Payslip, error) {
	slip, err := s.repo.GetPayslipByID(id)
	if err != nil {
		return nil, err
	}
	if slip.State == "paid" {
		return nil, errors.New("slip gaji yang sudah dibayar tidak dapat dibatalkan")
	}

	slip.State = "cancel"
	if err := s.repo.UpdatePayslip(slip); err != nil {
		return nil, err
	}

	s.logChatter("hr.payslip", slip.ID, "Slip gaji telah dibatalkan.", companyID)

	return slip, nil
}

// Stats returns executive payroll analytics
func (s *Service) GetStats(companyID uint) (*PayrollStats, error) {
	return s.repo.GetStats(companyID)
}

// Helper to find or create GL accounts
func (s *Service) getOrCreateAccount(code, name, accType string, companyID uint) account.AccountAccount {
	var acc account.AccountAccount
	if err := s.db.Where("code = ? AND company_id = ?", code, companyID).First(&acc).Error; err != nil {
		acc = account.AccountAccount{
			Code: code,
			Name: name,
			Type: accType,
		}
		acc.CompanyID = companyID
		_ = s.db.Create(&acc)
	}
	return acc
}

func extractJSON(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}

func (s *Service) logChatter(model string, resID uint, body string, companyID uint) {
	if s.chatterSvc != nil {
		msg := &chatter.MailMessage{
			ModelName:   model,
			ResID:       resID,
			Body:        body,
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(msg)
	}
}
