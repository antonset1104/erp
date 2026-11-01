package treasury

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/purchase"
	"erp-golang/internal/modules/sale"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(db *gorm.DB, repo *Repository, accountSvc *account.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// ============================================================================
// 1. BANK ACCOUNTS
// ============================================================================

func (s *Service) CreateBankAccount(req CreateBankAccountRequest, userID, companyID uint) (*TreasuryBankAccount, error) {
	year := time.Now().Year()
	code, err := s.repo.GenerateBankAccountCode(year)
	if err != nil {
		return nil, err
	}

	accType := req.AccountType
	if accType == "" {
		accType = BankTypeChecking
	}

	curr := req.Currency
	if curr == "" {
		curr = "IDR"
	}

	acc := &TreasuryBankAccount{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			CreateUID: userID,
			Active:    true,
		},
		AccountCode:     code,
		BankName:        req.BankName,
		AccountNumber:   req.AccountNumber,
		AccountHolder:   req.AccountHolder,
		AccountType:     accType,
		Currency:        curr,
		CurrentBalance:  req.CurrentBalance,
		MinimumBalance:  req.MinimumBalance,
		OverdraftLimit:  req.OverdraftLimit,
		InterestRatePct: req.InterestRatePct,
		JournalID:       req.JournalID,
		GLAccountID:     req.GLAccountID,
		Status:          "active",
		IsPrimaryPool:   req.IsPrimaryPool,
		Notes:           req.Notes,
	}

	if err := s.repo.CreateBankAccount(acc); err != nil {
		return nil, fmt.Errorf("gagal membuat rekening kas/bank perbendaharaan: %w", err)
	}

	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			"treasury.bank_account", acc.ID, userID, "Treasury Admin",
			fmt.Sprintf("Rekening Bank Perbendaharaan '%s' (%s - %s) berhasil didaftarkan dengan saldo awal Rp %.2f",
				acc.AccountCode, acc.BankName, acc.AccountNumber, acc.CurrentBalance),
			"", companyID,
		)
	}

	return acc, nil
}

func (s *Service) ListBankAccounts(companyID uint) ([]TreasuryBankAccount, error) {
	return s.repo.ListBankAccounts(companyID)
}

func (s *Service) GetBankAccount(id uint, companyID uint) (*TreasuryBankAccount, error) {
	return s.repo.GetBankAccountByID(id, companyID)
}

// ============================================================================
// 2. CASH FLOW FORECASTING ENGINE
// ============================================================================

func (s *Service) GenerateCashFlowForecast(req GenerateForecastRequest, userID, companyID uint) (*CashFlowForecast, error) {
	year := time.Now().Year()
	code, err := s.repo.GenerateForecastCode(year)
	if err != nil {
		return nil, err
	}

	horizon := req.HorizonDays
	if horizon <= 0 {
		horizon = 30
	}

	baseDate := req.BaseDate
	if baseDate.IsZero() {
		baseDate = time.Now()
	}
	endDate := baseDate.AddDate(0, 0, horizon)

	// 1. Calculate Opening Cash Balance from active bank accounts
	accs, err := s.repo.ListBankAccounts(companyID)
	if err != nil {
		return nil, err
	}
	var openingBalance float64
	for _, a := range accs {
		if a.Status == "active" {
			openingBalance += a.CurrentBalance
		}
	}

	minSafetyBuffer := req.MinimumSafetyBuffer
	if minSafetyBuffer <= 0 {
		minSafetyBuffer = openingBalance * 0.20 // Default 20% safety buffer
	}

	var items []CashFlowForecastItem

	// 2. SCAN FUTURE CASH INFLOWS
	// A. Customer Receivables (Open AR Invoices)
	var openInvoices []account.AccountMove
	s.db.Where("company_id = ? AND move_type = 'out_invoice' AND state = 'posted' AND payment_state != 'paid'", companyID).
		Find(&openInvoices)

	for _, inv := range openInvoices {
		dueDate, parseErr := time.Parse("2006-01-02", inv.InvoiceDueDate)
		if parseErr != nil || dueDate.IsZero() {
			dueDate, _ = time.Parse("2006-01-02", inv.InvoiceDate)
			dueDate = dueDate.AddDate(0, 0, 30) // default +30 days
		}

		if dueDate.Before(baseDate) {
			// Overdue invoice: expected to be collected within 7 days from baseDate
			dueDate = baseDate.AddDate(0, 0, 7)
		}

		if dueDate.Before(endDate) {
			certainty := 0.95
			if dueDate.Before(baseDate.AddDate(0, 0, 14)) {
				certainty = 0.90
			}
			residual := inv.AmountResidual
			if residual <= 0 {
				residual = inv.AmountTotal
			}

			items = append(items, CashFlowForecastItem{
				Category:        CategoryARReceivables,
				Direction:       DirectionInflow,
				ExpectedDate:    dueDate,
				GrossAmount:     residual,
				CertaintyFactor: certainty,
				AdjustedAmount:  math.Round(residual*certainty*100) / 100,
				SourceDocType:   "account.move",
				SourceDocID:     inv.ID,
				SourceDocRef:    inv.Name,
				PartnerName:     fmt.Sprintf("Pelanggan (Partner #%d)", inv.PartnerID),
				Notes:           fmt.Sprintf("Faktur Penjualan '%s' jatuh tempo %s", inv.Name, dueDate.Format("2006-01-02")),
			})
		}
	}

	// B. Confirmed Sales Orders (Pipeline SO)
	if req.IncludePipelineSO {
		var confirmedSOs []sale.SaleOrder
		s.db.Where("company_id = ? AND state = 'sale'", companyID).Find(&confirmedSOs)

		for _, so := range confirmedSOs {
			// Expected collection 30 days after confirmation
			soDate := so.CreatedAt
			if soDate.IsZero() {
				soDate = baseDate
			}
			expDate := soDate.AddDate(0, 0, 30)
			if expDate.Before(baseDate) {
				expDate = baseDate.AddDate(0, 0, 15)
			}

			if expDate.Before(endDate) {
				certainty := 0.75
				items = append(items, CashFlowForecastItem{
					Category:        CategorySalesOrder,
					Direction:       DirectionInflow,
					ExpectedDate:    expDate,
					GrossAmount:     so.AmountTotal,
					CertaintyFactor: certainty,
					AdjustedAmount:  math.Round(so.AmountTotal*certainty*100) / 100,
					SourceDocType:   "sale.order",
					SourceDocID:     so.ID,
					SourceDocRef:    so.Name,
					PartnerName:     fmt.Sprintf("Customer SO #%d", so.PartnerID),
					Notes:           fmt.Sprintf("Pesanan Penjualan '%s' estimasi penagihan termin", so.Name),
				})
			}
		}
	}

	// C. Recurring Subscriptions (MRR)
	if req.IncludeMRRRenewal {
		var activeSubs []struct {
			ID              uint
			SubscriptionCode string
			Name            string
			PartnerID       uint
			RecurringAmount float64
			NextInvoiceDate time.Time
		}
		s.db.Table("subscriptions").
			Select("id, subscription_code, name, partner_id, recurring_amount, next_invoice_date").
			Where("company_id = ? AND state = 'in_progress'", companyID).
			Find(&activeSubs)

		for _, sub := range activeSubs {
			expDate := sub.NextInvoiceDate
			if expDate.IsZero() || expDate.Before(baseDate) {
				expDate = baseDate.AddDate(0, 0, 15)
			}

			if expDate.Before(endDate) {
				certainty := 0.90
				items = append(items, CashFlowForecastItem{
					Category:        CategoryRecurringMRR,
					Direction:       DirectionInflow,
					ExpectedDate:    expDate,
					GrossAmount:     sub.RecurringAmount,
					CertaintyFactor: certainty,
					AdjustedAmount:  math.Round(sub.RecurringAmount*certainty*100) / 100,
					SourceDocType:   "subscription",
					SourceDocID:     sub.ID,
					SourceDocRef:    sub.SubscriptionCode,
					Notes:           fmt.Sprintf("Perpanjangan Langganan '%s' (%s)", sub.Name, sub.SubscriptionCode),
				})
			}
		}
	}

	// 3. SCAN FUTURE CASH OUTFLOWS
	// A. Vendor Bills (Open AP)
	var openBills []account.AccountMove
	s.db.Where("company_id = ? AND move_type = 'in_invoice' AND state = 'posted' AND payment_state != 'paid'", companyID).
		Find(&openBills)

	for _, bill := range openBills {
		dueDate, parseErr := time.Parse("2006-01-02", bill.InvoiceDueDate)
		if parseErr != nil || dueDate.IsZero() {
			dueDate, _ = time.Parse("2006-01-02", bill.InvoiceDate)
			dueDate = dueDate.AddDate(0, 0, 30)
		}

		if dueDate.Before(baseDate) {
			dueDate = baseDate.AddDate(0, 0, 3) // Need immediate settlement
		}

		if dueDate.Before(endDate) {
			certainty := 1.0 // Firm contractual commitment
			residual := bill.AmountResidual
			if residual <= 0 {
				residual = bill.AmountTotal
			}

			items = append(items, CashFlowForecastItem{
				Category:        CategoryAPPayables,
				Direction:       DirectionOutflow,
				ExpectedDate:    dueDate,
				GrossAmount:     residual,
				CertaintyFactor: certainty,
				AdjustedAmount:  math.Round(residual*certainty*100) / 100,
				SourceDocType:   "account.move",
				SourceDocID:     bill.ID,
				SourceDocRef:    bill.Name,
				PartnerName:     fmt.Sprintf("Pemasok (Partner #%d)", bill.PartnerID),
				Notes:           fmt.Sprintf("Tagihan Pemasok '%s' wajib bayar %s", bill.Name, dueDate.Format("2006-01-02")),
			})
		}
	}

	// B. Approved Purchase Orders (CapEx / OpEx commitments)
	if req.IncludeApprovedPO {
		var confirmedPOs []purchase.PurchaseOrder
		s.db.Where("company_id = ? AND state = 'purchase'", companyID).Find(&confirmedPOs)

		for _, po := range confirmedPOs {
			expDate := po.CreatedAt.AddDate(0, 0, 20)
			if expDate.Before(baseDate) {
				expDate = baseDate.AddDate(0, 0, 10)
			}

			if expDate.Before(endDate) {
				certainty := 0.90
				items = append(items, CashFlowForecastItem{
					Category:        CategoryCapexPurchase,
					Direction:       DirectionOutflow,
					ExpectedDate:    expDate,
					GrossAmount:     po.AmountTotal,
					CertaintyFactor: certainty,
					AdjustedAmount:  math.Round(po.AmountTotal*certainty*100) / 100,
					SourceDocType:   "purchase.order",
					SourceDocID:     po.ID,
					SourceDocRef:    po.Name,
					PartnerName:     fmt.Sprintf("Vendor PO #%d", po.PartnerID),
					Notes:           fmt.Sprintf("Pesanan Pembelian '%s' komitmen pengadaan barang", po.Name),
				})
			}
		}
	}

	// C. Monthly Payroll Outflow Commitment
	// Estimate monthly payroll total from hr_employees or past salary slips
	var employeeCount int64
	s.db.Table("hr_employees").Where("company_id = ? AND active = ?", companyID, true).Count(&employeeCount)
	if employeeCount == 0 {
		employeeCount = 10 // Baseline workforce
	}
	avgSalaryPerEmp := 7500000.0 // Rp 7.5jt baseline
	estMonthlyPayroll := float64(employeeCount) * avgSalaryPerEmp

	// Add monthly salary payouts (typically 25th of month)
	payrollMonths := (horizon / 30) + 1
	for m := 0; m < payrollMonths; m++ {
		payDate := time.Date(baseDate.Year(), baseDate.Month()+time.Month(m), 25, 0, 0, 0, 0, baseDate.Location())
		if payDate.After(baseDate) && payDate.Before(endDate) {
			items = append(items, CashFlowForecastItem{
				Category:        CategoryPayrollSalaries,
				Direction:       DirectionOutflow,
				ExpectedDate:    payDate,
				GrossAmount:     estMonthlyPayroll,
				CertaintyFactor: 1.0,
				AdjustedAmount:  estMonthlyPayroll,
				SourceDocType:   "hr.payroll",
				Notes:           fmt.Sprintf("Estimasi Gaji & BPJS %d Karyawan Periode %s", employeeCount, payDate.Format("Jan 2006")),
			})
		}
	}

	// 4. AGGREGATE TOTALS & CALCULATE CASH TRAJECTORY
	var totalInflow, totalOutflow float64
	for _, it := range items {
		if it.Direction == DirectionInflow {
			totalInflow += it.AdjustedAmount
		} else {
			totalOutflow += it.AdjustedAmount
		}
	}

	projectedNet := totalInflow - totalOutflow
	projectedEnding := openingBalance + projectedNet

	// Simulate daily trajectory to find lowest cash point
	lowestPoint := openingBalance
	runningCash := openingBalance

	// Sort and step by days
	for d := 1; d <= horizon; d++ {
		currentDay := baseDate.AddDate(0, 0, d)
		for _, it := range items {
			if it.ExpectedDate.Year() == currentDay.Year() &&
				it.ExpectedDate.YearDay() == currentDay.YearDay() {
				if it.Direction == DirectionInflow {
					runningCash += it.AdjustedAmount
				} else {
					runningCash -= it.AdjustedAmount
				}
			}
		}
		if runningCash < lowestPoint {
			lowestPoint = runningCash
		}
	}

	// Assess Deficit Risk
	deficitRisk := RiskNone
	if lowestPoint < 0 {
		deficitRisk = RiskCritical
	} else if lowestPoint < minSafetyBuffer {
		deficitRisk = RiskModerate
	}

	// Calculate Runway in days
	dailyBurn := totalOutflow / float64(horizon)
	runwayDays := 365
	if dailyBurn > 0 {
		runwayDays = int(openingBalance / dailyBurn)
		if runwayDays > 365 {
			runwayDays = 365
		}
	}

	forecast := &CashFlowForecast{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			CreateUID: userID,
			Active:    true,
		},
		ForecastCode:        code,
		Name:                req.Name,
		BaseDate:            baseDate,
		HorizonDays:         horizon,
		OpeningCashBalance:  openingBalance,
		ProjectedInflow:     math.Round(totalInflow*100) / 100,
		ProjectedOutflow:    math.Round(totalOutflow*100) / 100,
		ProjectedNetCash:    math.Round(projectedNet*100) / 100,
		ProjectedEndingCash: math.Round(projectedEnding*100) / 100,
		MinimumSafetyBuffer: minSafetyBuffer,
		LowestPointCash:     math.Round(lowestPoint*100) / 100,
		DeficitRisk:         deficitRisk,
		EstimatedRunwayDays: runwayDays,
		Status:              "active",
		Notes:               req.Notes,
	}

	if err := s.repo.CreateForecast(forecast); err != nil {
		return nil, fmt.Errorf("gagal menyimpan proyeksi arus kas: %w", err)
	}

	// Associate items with forecast ID and save
	for i := range items {
		items[i].CompanyID = companyID
		items[i].CreateUID = userID
		items[i].Active = true
		items[i].ForecastID = forecast.ID
	}
	if err := s.repo.CreateForecastItems(items); err != nil {
		return nil, fmt.Errorf("gagal menyimpan rincian baris proyeksi: %w", err)
	}

	forecast.Items = items

	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			"treasury.forecast", forecast.ID, userID, "AI Cash Flow Engine",
			fmt.Sprintf("Proyeksi Arus Kas '%s' (%s, Horizon %d hari) dihitung: Kas Masuk Rp %.2f, Kas Keluar Rp %.2f, Saldo Akhir Rp %.2f (Status Risiko: %s)",
				forecast.ForecastCode, forecast.Name, forecast.HorizonDays, forecast.ProjectedInflow, forecast.ProjectedOutflow, forecast.ProjectedEndingCash, forecast.DeficitRisk),
			"", companyID,
		)
	}

	return forecast, nil
}

func (s *Service) ListForecasts(companyID uint) ([]CashFlowForecast, error) {
	return s.repo.ListForecasts(companyID)
}

func (s *Service) GetForecast(id uint, companyID uint) (*CashFlowForecast, error) {
	return s.repo.GetForecastByID(id, companyID)
}

// ============================================================================
// 3. INTER-BANK FUND SWEEP & TRANSFER ENGINE
// ============================================================================

func (s *Service) ExecuteInterBankTransfer(req CreateInterBankTransferRequest, userID, companyID uint) (*InterBankTransfer, error) {
	if req.FromAccountID == req.ToAccountID {
		return nil, fmt.Errorf("rekening asal dan rekening tujuan tidak boleh sama")
	}

	fromAcc, err := s.repo.GetBankAccountByID(req.FromAccountID, companyID)
	if err != nil {
		return nil, fmt.Errorf("rekening pengirim (From ID #%d) tidak ditemukan: %w", req.FromAccountID, err)
	}

	toAcc, err := s.repo.GetBankAccountByID(req.ToAccountID, companyID)
	if err != nil {
		return nil, fmt.Errorf("rekening penerima (To ID #%d) tidak ditemukan: %w", req.ToAccountID, err)
	}

	totalDeduction := req.Amount + req.TransferFee
	availableFunds := fromAcc.CurrentBalance + fromAcc.OverdraftLimit
	if availableFunds < totalDeduction {
		return nil, fmt.Errorf("saldo rekening '%s' tidak mencukupi (Tersedia: Rp %.2f, Dibutuhkan: Rp %.2f)",
			fromAcc.BankName, availableFunds, totalDeduction)
	}

	txDate := req.TransferDate
	if txDate.IsZero() {
		txDate = time.Now()
	}

	year := time.Now().Year()
	txCode, err := s.repo.GenerateTransferCode(year)
	if err != nil {
		return nil, err
	}

	transfer := &InterBankTransfer{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			CreateUID: userID,
			Active:    true,
		},
		TransferCode:    txCode,
		FromAccountID:   fromAcc.ID,
		FromAccountName: fmt.Sprintf("%s (%s)", fromAcc.BankName, fromAcc.AccountNumber),
		ToAccountID:     toAcc.ID,
		ToAccountName:   fmt.Sprintf("%s (%s)", toAcc.BankName, toAcc.AccountNumber),
		TransferDate:    txDate,
		Amount:          req.Amount,
		TransferFee:     req.TransferFee,
		Reference:       req.Reference,
		State:           "posted",
		ExecutedByUID:   userID,
		Memo:            req.Memo,
	}

	// Atomically execute balance adjustment
	if err := s.repo.UpdateBankAccountBalance(fromAcc.ID, -totalDeduction); err != nil {
		return nil, fmt.Errorf("gagal mendebit rekening asal: %w", err)
	}
	if err := s.repo.UpdateBankAccountBalance(toAcc.ID, req.Amount); err != nil {
		// Rollback fromAcc
		s.repo.UpdateBankAccountBalance(fromAcc.ID, totalDeduction)
		return nil, fmt.Errorf("gagal mengkredit rekening tujuan: %w", err)
	}

	// Auto-post Balanced General Ledger Move if journals/accounts exist
	if fromAcc.GLAccountID > 0 && toAcc.GLAccountID > 0 {
		journalID := fromAcc.JournalID
		if journalID == 0 {
			journalID = toAcc.JournalID
		}
		if journalID == 0 {
			var anyBankJournal account.AccountJournal
			s.db.Where("company_id = ? AND type = 'bank'", companyID).First(&anyBankJournal)
			journalID = anyBankJournal.ID
		}

		if journalID > 0 {
			var moveCount int64
			s.db.Model(&account.AccountMove{}).Where("company_id = ?", companyID).Count(&moveCount)
			moveName := fmt.Sprintf("GL/TRF/%d/%04d", year, moveCount+1)

			glMove := account.AccountMove{
				ModelBase: models.ModelBase{
					CompanyID: companyID,
					CreateUID: userID,
					Active:    true,
				},
				Name:           moveName,
				MoveType:       "entry",
				JournalID:      journalID,
				PartnerID:      0,
				InvoiceDate:    txDate.Format("2006-01-02"),
				State:          "posted",
				PaymentState:   "not_paid",
				AmountTotal:    req.Amount,
				Ref:            txCode,
				Note:           fmt.Sprintf("Transfer Likuiditas Kas Antar-Bank: %s ➔ %s", transfer.FromAccountName, transfer.ToAccountName),
			}
			if err := s.db.Create(&glMove).Error; err == nil {
				// Debit receiving bank
				line1 := account.AccountMoveLine{
					ModelBase: models.ModelBase{CompanyID: companyID, Active: true},
					MoveID:    glMove.ID,
					AccountID: toAcc.GLAccountID,
					Name:      fmt.Sprintf("Penerimaan Transfer dari %s", fromAcc.BankName),
					Debit:     req.Amount,
					Credit:    0,
					Balance:   req.Amount,
				}
				// Credit sending bank
				line2 := account.AccountMoveLine{
					ModelBase: models.ModelBase{CompanyID: companyID, Active: true},
					MoveID:    glMove.ID,
					AccountID: fromAcc.GLAccountID,
					Name:      fmt.Sprintf("Pengiriman Transfer ke %s", toAcc.BankName),
					Debit:     0,
					Credit:    req.Amount,
					Balance:   -req.Amount,
				}
				s.db.Create(&line1)
				s.db.Create(&line2)

				transfer.JournalMoveID = glMove.ID
				transfer.JournalMoveName = glMove.Name
			}
		}
	}

	if err := s.repo.CreateTransfer(transfer); err != nil {
		return nil, fmt.Errorf("gagal mencatat transfer dana perbendaharaan: %w", err)
	}

	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			"treasury.transfer", transfer.ID, userID, "Treasury Operator",
			fmt.Sprintf("Transfer Antar-Bank '%s' berhasil diposting: Rp %.2f dipindahkan dari %s ke %s",
				transfer.TransferCode, transfer.Amount, transfer.FromAccountName, transfer.ToAccountName),
			"", companyID,
		)
	}

	return transfer, nil
}

func (s *Service) ListTransfers(companyID uint) ([]InterBankTransfer, error) {
	return s.repo.ListTransfers(companyID)
}

func (s *Service) GetTransfer(id uint, companyID uint) (*InterBankTransfer, error) {
	return s.repo.GetTransferByID(id, companyID)
}

// ============================================================================
// 4. AI PREDICTIVE TREASURY COPILOT & WORKING CAPITAL ADVISOR
// ============================================================================

func (s *Service) AnalyzeTreasuryWithAI(forecastID uint, companyID uint) (*AITreasuryAdvisoryReport, error) {
	forecast, err := s.repo.GetForecastByID(forecastID, companyID)
	if err != nil {
		return nil, fmt.Errorf("proyeksi kas ID #%d tidak ditemukan: %w", forecastID, err)
	}

	// 1. Calculate Financial Working Capital Metrics
	runwayMonths := math.Round((float64(forecast.EstimatedRunwayDays)/30.0)*10) / 10
	if runwayMonths < 0.1 {
		runwayMonths = 0.5
	}

	healthScore := 90
	healthStatus := "HEALTHY"
	if forecast.DeficitRisk == RiskCritical {
		healthStatus = "DISTRESSED"
		healthScore = 45
	} else if forecast.DeficitRisk == RiskModerate {
		healthStatus = "CAUTION"
		healthScore = 72
	}

	// Calculate DSO (Days Sales Outstanding) & DPO (Days Payable Outstanding) approximations
	var totalAR, totalAP float64
	for _, it := range forecast.Items {
		if it.Category == CategoryARReceivables {
			totalAR += it.GrossAmount
		} else if it.Category == CategoryAPPayables {
			totalAP += it.GrossAmount
		}
	}

	dsoDays := 35.0
	if forecast.ProjectedInflow > 0 {
		dsoDays = math.Round((totalAR / forecast.ProjectedInflow) * float64(forecast.HorizonDays))
	}
	dpoDays := 45.0
	if forecast.ProjectedOutflow > 0 {
		dpoDays = math.Round((totalAP / forecast.ProjectedOutflow) * float64(forecast.HorizonDays))
	}

	metrics := []WorkingCapitalMetric{
		{
			MetricName:  "Days Sales Outstanding (DSO)",
			CurrentVal:  dsoDays,
			Benchmark:   45.0,
			Status:      "OPTIMAL",
			Description: "Rata-rata kecepatan penagihan piutang dari pelanggan.",
		},
		{
			MetricName:  "Days Payable Outstanding (DPO)",
			CurrentVal:  dpoDays,
			Benchmark:   30.0,
			Status:      "OPTIMAL",
			Description: "Rata-rata waktu pembayaran kewajiban kepada pemasok.",
		},
		{
			MetricName:  "Quick Ratio (Rasio Likuiditas Cepat)",
			CurrentVal:  math.Round(((forecast.OpeningCashBalance+totalAR)/math.Max(totalAP, 1.0))*100) / 100,
			Benchmark:   1.2,
			Status:      "OPTIMAL",
			Description: "Kemampuan aset likuid menutup kewajiban jangka pendek.",
		},
		{
			MetricName:  "Cash Buffer Coverage",
			CurrentVal:  math.Round((forecast.OpeningCashBalance/math.Max(forecast.MinimumSafetyBuffer, 1.0))*100),
			Benchmark:   100.0,
			Status:      "OPTIMAL",
			Description: "Persentase kecukupan kas terhadap batas pengaman darurat.",
		},
	}

	if metrics[2].CurrentVal < 1.0 {
		metrics[2].Status = "WARNING"
	}
	if metrics[3].CurrentVal < 100.0 {
		metrics[3].Status = "CRITICAL"
	}

	// Actionable advice heuristics
	advice := []string{
		"Jalankan program diskon pelunasan lebih awal (early settlement discount 2/10 net 30) untuk mempercepat perputaran kas masuk.",
		"Optimalkan jadwal pembayaran vendor mendekati batas due date guna mempertahankan likuiditas kas operasional harian.",
		"Lakukan sweep otomatis saldo mengendap di rekening cabang ke rekening perbendaharaan utama holding untuk memaksimalkan yield deposito.",
		"Pertahankan cadangan likuiditas darurat minimal sebesar 20% dari total pengeluaran operasional bulanan.",
	}

	execSummary := fmt.Sprintf(
		"Posisi likuiditas perbendaharaan perusahaan berstatus %s dengan skor kesehatan kas %d/100. Saldo kas saat ini Rp %.2f diproyeksikan mencukupi kebutuhan operasional selama %.1f bulan (Runway: %d hari) dengan estimasi kas akhir periode sebesar Rp %.2f.",
		healthStatus, healthScore, forecast.OpeningCashBalance, runwayMonths, forecast.EstimatedRunwayDays, forecast.ProjectedEndingCash,
	)

	report := &AITreasuryAdvisoryReport{
		ForecastCode:          forecast.ForecastCode,
		LiquidityHealth:       healthStatus,
		HealthScore:           healthScore,
		ExecutiveSummary:      execSummary,
		EstimatedRunwayMonths: runwayMonths,
		LowestCashProjection:  forecast.LowestPointCash,
		WorkingCapitalMetrics: metrics,
		ActionableAdvice:      advice,
		AIProvider:            "Local Heuristic Engine (Treasury & Cash Flow Management Baseline)",
	}

	// 2. Invoke Cloudflare Workers AI if configured
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "Bertindaklah sebagai Senior Corporate Treasurer & Chief Financial Officer (CFO). Analisis kesehatan arus kas dan berikan rekomendasi likuiditas dalam format JSON murni."
		userPrompt := fmt.Sprintf(`Lakukan analisis perbendaharaan dan proyeksi arus kas perusahaan berikut:
- Kode Proyeksi: %s
- Horizon Waktu: %d hari
- Saldo Kas Awal: Rp %.2f
- Proyeksi Kas Masuk: Rp %.2f
- Proyeksi Kas Keluar: Rp %.2f
- Proyeksi Saldo Kas Akhir: Rp %.2f
- Titik Kas Terendah: Rp %.2f
- Batas Kas Pengaman: Rp %.2f
- Status Risiko Defisit: %s
- Runway Operasional: %d hari

Berikan respons JSON murni dengan format persis:
{
  "liquidity_health": "HEALTHY",
  "health_score": 92,
  "executive_summary": "ringkasan eksekutif...",
  "actionable_advice": ["saran 1", "saran 2", "saran 3"]
}`, forecast.ForecastCode, forecast.HorizonDays, forecast.OpeningCashBalance, forecast.ProjectedInflow,
			forecast.ProjectedOutflow, forecast.ProjectedEndingCash, forecast.LowestPointCash,
			forecast.MinimumSafetyBuffer, forecast.DeficitRisk, forecast.EstimatedRunwayDays)

		respText, err := s.aiClient.CallRaw(userPrompt, systemPrompt)
		if err == nil && respText != "" {
			startIdx := strings.Index(respText, "{")
			endIdx := strings.LastIndex(respText, "}")
			if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
				var aiRes struct {
					LiquidityHealth  string   `json:"liquidity_health"`
					HealthScore      int      `json:"health_score"`
					ExecutiveSummary string   `json:"executive_summary"`
					ActionableAdvice []string `json:"actionable_advice"`
				}
				if errJson := json.Unmarshal([]byte(respText[startIdx:endIdx+1]), &aiRes); errJson == nil && aiRes.ExecutiveSummary != "" {
					report.LiquidityHealth = aiRes.LiquidityHealth
					report.HealthScore = aiRes.HealthScore
					report.ExecutiveSummary = aiRes.ExecutiveSummary
					if len(aiRes.ActionableAdvice) > 0 {
						report.ActionableAdvice = aiRes.ActionableAdvice
					}
					report.AIProvider = "Cloudflare Workers AI (@cf/meta/llama-3.1-8b-instruct)"
				}
			}
		}
	}

	return report, nil
}

// ============================================================================
// 5. STATS
// ============================================================================

func (s *Service) GetStats(companyID uint) (*TreasuryStats, error) {
	return s.repo.GetStats(companyID)
}
