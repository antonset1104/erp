package subscription

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/chatter"

	"gorm.io/gorm"
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

// ---------------------------------------------------------------------------
// 1. Calculations & Revenue Helpers
// ---------------------------------------------------------------------------

// CalculateMRRandARR normalizes billing cycle recurring amounts into standard SaaS MRR & ARR
func CalculateMRRandARR(amount float64, interval string, intervalCount int) (float64, float64) {
	if intervalCount <= 0 {
		intervalCount = 1
	}

	var mrr float64
	switch strings.ToLower(interval) {
	case IntervalDaily:
		// 365 days / 12 months = ~30.4166 days per month
		mrr = (amount * 365.0) / (12.0 * float64(intervalCount))
	case IntervalWeekly:
		// 52 weeks / 12 months = ~4.3333 weeks per month
		mrr = (amount * 52.0) / (12.0 * float64(intervalCount))
	case IntervalMonthly:
		mrr = amount / float64(intervalCount)
	case IntervalQuarterly:
		// 3 months per quarter
		mrr = amount / (3.0 * float64(intervalCount))
	case IntervalYearly:
		// 12 months per year
		mrr = amount / (12.0 * float64(intervalCount))
	default:
		mrr = amount / float64(intervalCount)
	}

	// Round to 2 decimal places
	mrr = math.Round(mrr*100) / 100
	arr := math.Round(mrr*12.0*100) / 100

	return mrr, arr
}

// CalculateNextInvoiceDate advances date string (YYYY-MM-DD) based on interval and count
func CalculateNextInvoiceDate(currentDateStr string, interval string, intervalCount int) (string, error) {
	if intervalCount <= 0 {
		intervalCount = 1
	}

	curr, err := time.Parse("2006-01-02", currentDateStr)
	if err != nil {
		curr = time.Now()
	}

	var next time.Time
	switch strings.ToLower(interval) {
	case IntervalDaily:
		next = curr.AddDate(0, 0, intervalCount)
	case IntervalWeekly:
		next = curr.AddDate(0, 0, intervalCount*7)
	case IntervalMonthly:
		next = curr.AddDate(0, intervalCount, 0)
	case IntervalQuarterly:
		next = curr.AddDate(0, intervalCount*3, 0)
	case IntervalYearly:
		next = curr.AddDate(intervalCount, 0, 0)
	default:
		next = curr.AddDate(0, intervalCount, 0)
	}

	return next.Format("2006-01-02"), nil
}

// ---------------------------------------------------------------------------
// 2. Subscription Templates CRUD
// ---------------------------------------------------------------------------

func (s *Service) CreateTemplate(req CreateTemplateRequest, companyID uint) (*SubscriptionTemplate, error) {
	if req.Code == "" || req.Name == "" {
		return nil, errors.New("kode dan nama template wajib diisi")
	}

	interval := strings.ToLower(req.RecurringInterval)
	if interval != IntervalDaily && interval != IntervalWeekly && interval != IntervalMonthly && interval != IntervalQuarterly && interval != IntervalYearly {
		return nil, fmt.Errorf("interval tidak valid: %s (gunakan daily, weekly, monthly, quarterly, yearly)", req.RecurringInterval)
	}

	ruleCount := req.RecurringRuleCount
	if ruleCount <= 0 {
		ruleCount = 1
	}

	journalID := req.InvoiceJournalID
	if journalID == 0 {
		journalID = 1
	}

	autoPost := true
	if req.AutoPostInvoice != nil {
		autoPost = *req.AutoPostInvoice
	}

	tmpl := &SubscriptionTemplate{
		Code:               req.Code,
		Name:               req.Name,
		RecurringInterval:  interval,
		RecurringRuleCount: ruleCount,
		InvoiceJournalID:   journalID,
		AutoPostInvoice:    autoPost,
		Description:        req.Description,
		Active:             true,
	}
	tmpl.CompanyID = companyID

	if err := s.repo.CreateTemplate(tmpl); err != nil {
		return nil, fmt.Errorf("gagal membuat template: %w", err)
	}
	return tmpl, nil
}

func (s *Service) ListTemplates(companyID uint) ([]SubscriptionTemplate, error) {
	return s.repo.ListTemplates(companyID)
}

func (s *Service) GetTemplate(id uint, companyID uint) (*SubscriptionTemplate, error) {
	return s.repo.GetTemplateByID(id, companyID)
}

// ---------------------------------------------------------------------------
// 3. Subscription CRUD & Lifecycle Management
// ---------------------------------------------------------------------------

func (s *Service) CreateSubscription(req CreateSubscriptionRequest, companyID uint, userID uint) (*Subscription, error) {
	if len(req.Lines) == 0 {
		return nil, errors.New("langganan harus memiliki minimal satu baris produk/layanan")
	}

	var partner base.Partner
	if err := s.db.Where("id = ? AND company_id = ?", req.PartnerID, companyID).First(&partner).Error; err != nil {
		return nil, fmt.Errorf("pelanggan dengan ID %d tidak ditemukan: %w", req.PartnerID, err)
	}

	tmpl, err := s.repo.GetTemplateByID(req.TemplateID, companyID)
	if err != nil {
		return nil, fmt.Errorf("template langganan dengan ID %d tidak ditemukan: %w", req.TemplateID, err)
	}

	code, err := s.repo.GetNextSubscriptionCode(companyID)
	if err != nil {
		return nil, fmt.Errorf("gagal menghasilkan nomor kontrak langganan: %w", err)
	}

	interval := req.RecurringInterval
	if interval == "" {
		interval = tmpl.RecurringInterval
	}
	intervalCount := req.RecurringIntervalCount
	if intervalCount <= 0 {
		intervalCount = tmpl.RecurringRuleCount
	}
	if intervalCount <= 0 {
		intervalCount = 1
	}

	startDate := req.StartDate
	if startDate == "" {
		startDate = time.Now().Format("2006-01-02")
	}

	nextInvoice := req.NextInvoiceDate
	if nextInvoice == "" {
		nextInvoice = startDate
	}

	currency := req.Currency
	if currency == "" {
		currency = "IDR"
	}

	payMethod := req.PaymentMethod
	if payMethod == "" {
		payMethod = "bank_transfer"
	}

	autoRenew := true
	if req.AutoRenew != nil {
		autoRenew = *req.AutoRenew
	}

	// Process lines and calculate total recurring amount
	var totalRecurring float64
	var subLines []SubscriptionLine

	for _, lineReq := range req.Lines {
		if lineReq.Quantity <= 0 {
			return nil, errors.New("kuantitas baris langganan harus lebih besar dari 0")
		}
		if lineReq.UnitPrice < 0 {
			return nil, errors.New("harga satuan tidak boleh bernilai negatif")
		}

		subtotal := lineReq.Quantity * lineReq.UnitPrice * (1.0 - (lineReq.Discount / 100.0))
		subtotal = math.Round(subtotal*100) / 100
		totalRecurring += subtotal

		lineName := lineReq.Name
		if lineName == "" {
			lineName = fmt.Sprintf("Layanan Langganan (Produk #%d)", lineReq.ProductID)
		}

		uom := lineReq.UoM
		if uom == "" {
			uom = "Units"
		}

		subLine := SubscriptionLine{
			ProductID: lineReq.ProductID,
			Name:      lineName,
			Quantity:  lineReq.Quantity,
			UnitPrice: lineReq.UnitPrice,
			Discount:  lineReq.Discount,
			Subtotal:  subtotal,
			UoM:       uom,
		}
		subLine.CompanyID = companyID
		subLines = append(subLines, subLine)
	}

	totalRecurring = math.Round(totalRecurring*100) / 100
	mrr, arr := CalculateMRRandARR(totalRecurring, interval, intervalCount)

	sub := &Subscription{
		Code:                   code,
		Name:                   req.Name,
		PartnerID:              req.PartnerID,
		TemplateID:             req.TemplateID,
		State:                  StateDraft,
		StartDate:              startDate,
		EndDate:                req.EndDate,
		NextInvoiceDate:        nextInvoice,
		RecurringInterval:      interval,
		RecurringIntervalCount: intervalCount,
		Currency:               currency,
		RecurringAmount:        totalRecurring,
		TotalMRR:               mrr,
		TotalARR:               arr,
		InvoicedCount:          0,
		TotalInvoicedAmount:    0,
		AutoRenew:              autoRenew,
		PaymentMethod:          payMethod,
		Notes:                  req.Notes,
		Lines:                  subLines,
	}
	sub.CompanyID = companyID

	if err := s.repo.CreateSubscription(sub); err != nil {
		return nil, fmt.Errorf("gagal menyimpan data langganan: %w", err)
	}

	// Universal Chatter Logging
	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"sale.subscription",
			sub.ID,
			userID,
			"Subscription Engine",
			fmt.Sprintf("Kontrak langganan <b>%s</b> (%s) berhasil dibuat dalam status <i>Draft</i> senilai <b>Rp %.2f</b> / siklus (MRR: Rp %.2f).",
				sub.Code, sub.Name, sub.RecurringAmount, sub.TotalMRR),
			"subscription_created",
			companyID,
		)
	}

	return s.repo.GetSubscriptionByID(sub.ID, companyID)
}

func (s *Service) GetSubscription(id uint, companyID uint) (*Subscription, error) {
	return s.repo.GetSubscriptionByID(id, companyID)
}

func (s *Service) ListSubscriptions(companyID uint, state string, partnerID uint, templateID uint, search string) ([]Subscription, error) {
	return s.repo.ListSubscriptions(companyID, state, partnerID, templateID, search)
}

func (s *Service) UpdateSubscription(id uint, req UpdateSubscriptionRequest, companyID uint, userID uint) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionByID(id, companyID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		sub.Name = req.Name
	}
	if req.NextInvoiceDate != "" {
		sub.NextInvoiceDate = req.NextInvoiceDate
	}
	if req.EndDate != "" {
		sub.EndDate = req.EndDate
	}
	if req.PaymentMethod != "" {
		sub.PaymentMethod = req.PaymentMethod
	}
	if req.AutoRenew != nil {
		sub.AutoRenew = *req.AutoRenew
	}
	if req.Notes != "" {
		sub.Notes = req.Notes
	}

	if err := s.repo.UpdateSubscription(sub); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"sale.subscription",
			sub.ID,
			userID,
			"Subscription Engine",
			"Data kontrak langganan diperbarui.",
			"subscription_updated",
			companyID,
		)
	}

	return sub, nil
}

// StartSubscription moves contract from draft to in_progress
func (s *Service) StartSubscription(id uint, companyID uint, userID uint) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionByID(id, companyID)
	if err != nil {
		return nil, err
	}

	if sub.State != StateDraft {
		return nil, fmt.Errorf("hanya kontrak dalam status draft yang dapat diaktifkan (status saat ini: %s)", sub.State)
	}

	oldState := sub.State
	sub.State = StateInProgress
	if sub.NextInvoiceDate == "" {
		sub.NextInvoiceDate = time.Now().Format("2006-01-02")
	}

	if err := s.repo.UpdateSubscription(sub); err != nil {
		return nil, err
	}

	// Stage log
	stageLog := &SubscriptionStageLog{
		SubscriptionID: sub.ID,
		OldState:       oldState,
		NewState:       StateInProgress,
		Reason:         "Aktivasi langganan",
		MRRChange:      sub.TotalMRR,
		ChangedAt:      time.Now().Format("2006-01-02 15:04:05"),
		UserID:         userID,
	}
	stageLog.CompanyID = companyID
	s.repo.CreateStageLog(stageLog)

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"sale.subscription",
			sub.ID,
			userID,
			"Subscription Engine",
			fmt.Sprintf("Kontrak diaktifkan (In Progress). MRR aktif: <b>Rp %.2f</b>. Jadwal faktur berikutnya: <b>%s</b>.", sub.TotalMRR, sub.NextInvoiceDate),
			"subscription_started",
			companyID,
		)
	}

	return sub, nil
}

// PauseSubscription temporarily pauses an active subscription
func (s *Service) PauseSubscription(id uint, reason string, companyID uint, userID uint) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionByID(id, companyID)
	if err != nil {
		return nil, err
	}

	if sub.State != StateInProgress {
		return nil, fmt.Errorf("hanya langganan aktif yang dapat dijeda (status saat ini: %s)", sub.State)
	}

	oldState := sub.State
	sub.State = StatePaused

	if err := s.repo.UpdateSubscription(sub); err != nil {
		return nil, err
	}

	stageLog := &SubscriptionStageLog{
		SubscriptionID: sub.ID,
		OldState:       oldState,
		NewState:       StatePaused,
		Reason:         reason,
		MRRChange:      -sub.TotalMRR,
		ChangedAt:      time.Now().Format("2006-01-02 15:04:05"),
		UserID:         userID,
	}
	stageLog.CompanyID = companyID
	s.repo.CreateStageLog(stageLog)

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"sale.subscription",
			sub.ID,
			userID,
			"Subscription Engine",
			fmt.Sprintf("Langganan dijeda (Paused). Alasan: <i>%s</i>.", reason),
			"subscription_paused",
			companyID,
		)
	}

	return sub, nil
}

// ResumeSubscription reactivates a paused subscription
func (s *Service) ResumeSubscription(id uint, companyID uint, userID uint) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionByID(id, companyID)
	if err != nil {
		return nil, err
	}

	if sub.State != StatePaused {
		return nil, fmt.Errorf("hanya langganan yang sedang dijeda yang dapat dilanjutkan (status saat ini: %s)", sub.State)
	}

	oldState := sub.State
	sub.State = StateInProgress

	if err := s.repo.UpdateSubscription(sub); err != nil {
		return nil, err
	}

	stageLog := &SubscriptionStageLog{
		SubscriptionID: sub.ID,
		OldState:       oldState,
		NewState:       StateInProgress,
		Reason:         "Melanjutkan langganan",
		MRRChange:      sub.TotalMRR,
		ChangedAt:      time.Now().Format("2006-01-02 15:04:05"),
		UserID:         userID,
	}
	stageLog.CompanyID = companyID
	s.repo.CreateStageLog(stageLog)

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"sale.subscription",
			sub.ID,
			userID,
			"Subscription Engine",
			fmt.Sprintf("Langganan aktif kembali (Resumed). MRR aktif: <b>Rp %.2f</b>.", sub.TotalMRR),
			"subscription_resumed",
			companyID,
		)
	}

	return sub, nil
}

// CancelSubscription marks contract as churned with churn reason
func (s *Service) CancelSubscription(id uint, reason string, companyID uint, userID uint) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionByID(id, companyID)
	if err != nil {
		return nil, err
	}

	if sub.State == StateChurned || sub.State == StateClosed {
		return nil, fmt.Errorf("langganan sudah dalam status non-aktif (%s)", sub.State)
	}

	oldState := sub.State
	sub.State = StateChurned
	sub.ChurnReason = reason
	sub.ChurnDate = time.Now().Format("2006-01-02")

	if err := s.repo.UpdateSubscription(sub); err != nil {
		return nil, err
	}

	stageLog := &SubscriptionStageLog{
		SubscriptionID: sub.ID,
		OldState:       oldState,
		NewState:       StateChurned,
		Reason:         reason,
		MRRChange:      -sub.TotalMRR,
		ChangedAt:      time.Now().Format("2006-01-02 15:04:05"),
		UserID:         userID,
	}
	stageLog.CompanyID = companyID
	s.repo.CreateStageLog(stageLog)

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"sale.subscription",
			sub.ID,
			userID,
			"Subscription Engine",
			fmt.Sprintf("Kontrak langganan dibatalkan (Churn). Kehilangan MRR: <b>-Rp %.2f</b>. Alasan pembatalan: <i>%s</i>.", sub.TotalMRR, reason),
			"subscription_churned",
			companyID,
		)
	}

	return sub, nil
}

// CloseSubscription completes contract upon expiration
func (s *Service) CloseSubscription(id uint, reason string, companyID uint, userID uint) (*Subscription, error) {
	sub, err := s.repo.GetSubscriptionByID(id, companyID)
	if err != nil {
		return nil, err
	}

	oldState := sub.State
	sub.State = StateClosed
	if sub.ChurnReason == "" {
		sub.ChurnReason = reason
	}
	if sub.ChurnDate == "" {
		sub.ChurnDate = time.Now().Format("2006-01-02")
	}

	if err := s.repo.UpdateSubscription(sub); err != nil {
		return nil, err
	}

	stageLog := &SubscriptionStageLog{
		SubscriptionID: sub.ID,
		OldState:       oldState,
		NewState:       StateClosed,
		Reason:         reason,
		MRRChange:      -sub.TotalMRR,
		ChangedAt:      time.Now().Format("2006-01-02 15:04:05"),
		UserID:         userID,
	}
	stageLog.CompanyID = companyID
	s.repo.CreateStageLog(stageLog)

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"sale.subscription",
			sub.ID,
			userID,
			"Subscription Engine",
			fmt.Sprintf("Kontrak langganan ditutup/selesai (Closed). Alasan: <i>%s</i>.", reason),
			"subscription_closed",
			companyID,
		)
	}

	return sub, nil
}

// ---------------------------------------------------------------------------
// 4. Automated Recurring Invoicing Engine
// ---------------------------------------------------------------------------

// GenerateSingleInvoice creates an account invoice for the current billing cycle
func (s *Service) GenerateSingleInvoice(subscriptionID uint, companyID uint, userID uint) (*SubscriptionInvoiceLog, error) {
	sub, err := s.repo.GetSubscriptionByID(subscriptionID, companyID)
	if err != nil {
		return nil, fmt.Errorf("langganan ID %d tidak ditemukan: %w", subscriptionID, err)
	}

	if sub.State != StateInProgress {
		return nil, fmt.Errorf("tidak dapat membuat faktur untuk langganan yang tidak aktif (status: %s)", sub.State)
	}

	invoiceDate := sub.NextInvoiceDate
	if invoiceDate == "" {
		invoiceDate = time.Now().Format("2006-01-02")
	}

	// Calculate due date (default +30 days)
	tInv, _ := time.Parse("2006-01-02", invoiceDate)
	dueDate := tInv.AddDate(0, 0, 30).Format("2006-01-02")

	// Determine Receivable and Income Accounts
	var receivableAcc account.AccountAccount
	if err := s.db.Where("company_id = ? AND (code = ? OR type = ?)", companyID, "120000", "asset_receivable").First(&receivableAcc).Error; err != nil {
		receivableAcc.ID = 3 // Standard seeded account ID
	}

	var incomeAcc account.AccountAccount
	if err := s.db.Where("company_id = ? AND (code = ? OR type = ?)", companyID, "410000", "income").First(&incomeAcc).Error; err != nil {
		incomeAcc.ID = 7 // Standard seeded account ID
	}

	journalID := sub.Template.InvoiceJournalID
	if journalID == 0 {
		var jrn account.AccountJournal
		if err := s.db.Where("company_id = ? AND (code = ? OR type = ?)", companyID, "INV", "sale").First(&jrn).Error; err == nil {
			journalID = jrn.ID
		} else {
			journalID = 1
		}
	}

	// Construct Account Move Lines with perfect debit = credit balance
	partnerID := sub.PartnerID
	partnerName := sub.Partner.Name
	if partnerName == "" {
		partnerName = fmt.Sprintf("Customer #%d", sub.PartnerID)
	}

	// Debit line: Accounts Receivable (120000)
	receivableLine := account.AccountMoveLine{
		AccountID: receivableAcc.ID,
		PartnerID: &partnerID,
		Name:      fmt.Sprintf("Piutang Langganan %s - %s", sub.Code, partnerName),
		Debit:     sub.RecurringAmount,
		Credit:    0,
		Balance:   sub.RecurringAmount,
	}
	receivableLine.CompanyID = companyID

	// Credit lines: Revenue (410000) for each subscription line
	var creditLines []account.AccountMoveLine
	for _, line := range sub.Lines {
		prodID := line.ProductID
		crLine := account.AccountMoveLine{
			AccountID: incomeAcc.ID,
			PartnerID: &partnerID,
			ProductID: &prodID,
			Name:      fmt.Sprintf("%s (%s - %s)", line.Name, sub.Code, invoiceDate),
			Debit:     0,
			Credit:    line.Subtotal,
			Balance:   -line.Subtotal,
			Quantity:  line.Quantity,
			PriceUnit: line.UnitPrice,
		}
		crLine.CompanyID = companyID
		creditLines = append(creditLines, crLine)
	}

	// If no lines found (safety fallback), create single credit line
	if len(creditLines) == 0 {
		crLine := account.AccountMoveLine{
			AccountID: incomeAcc.ID,
			PartnerID: &partnerID,
			Name:      fmt.Sprintf("Pendapatan Langganan %s - %s", sub.Code, invoiceDate),
			Debit:     0,
			Credit:    sub.RecurringAmount,
			Balance:   -sub.RecurringAmount,
			Quantity:  1,
			PriceUnit: sub.RecurringAmount,
		}
		crLine.CompanyID = companyID
		creditLines = append(creditLines, crLine)
	}

	allLines := append([]account.AccountMoveLine{receivableLine}, creditLines...)

	move := &account.AccountMove{
		MoveType:       "out_invoice",
		JournalID:      journalID,
		PartnerID:      partnerID,
		InvoiceDate:    invoiceDate,
		InvoiceDueDate: dueDate,
		Ref:            sub.Code,
		Note:           fmt.Sprintf("Faktur Tagihan Otomatis Kontrak Langganan %s (%s). Siklus: %s.", sub.Code, sub.Name, sub.RecurringInterval),
		AmountUntaxed:  sub.RecurringAmount,
		AmountTotal:    sub.RecurringAmount,
		AmountResidual: sub.RecurringAmount,
		LineIDs:        allLines,
	}
	move.CompanyID = companyID

	if err := s.accountSvc.CreateMove(move); err != nil {
		return nil, fmt.Errorf("gagal membuat faktur pelanggan: %w", err)
	}

	// Auto post if configured
	if sub.Template.AutoPostInvoice {
		if _, err := s.accountSvc.ActionPost(move.ID, companyID); err != nil {
			// Log warning but continue if already created
			fmt.Printf("[Subscription] Auto-post invoice %s failed: %v\n", move.Name, err)
		} else {
			move.State = "posted"
		}
	}

	// Calculate and set next invoice date
	nextDate, err := CalculateNextInvoiceDate(invoiceDate, sub.RecurringInterval, sub.RecurringIntervalCount)
	if err != nil {
		nextDate = time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	}

	// Create invoice log
	invLog := &SubscriptionInvoiceLog{
		SubscriptionID:     sub.ID,
		InvoiceID:          move.ID,
		InvoiceNumber:      move.Name,
		InvoiceDate:        invoiceDate,
		Amount:             sub.RecurringAmount,
		Status:             move.State,
		NextInvoiceDateSet: nextDate,
	}
	invLog.CompanyID = companyID

	if err := s.repo.CreateInvoiceLog(invLog); err != nil {
		return nil, fmt.Errorf("gagal mencatat riwayat faktur langganan: %w", err)
	}

	// Update subscription counters & next invoice date
	sub.InvoicedCount++
	sub.TotalInvoicedAmount += sub.RecurringAmount
	sub.NextInvoiceDate = nextDate

	// Check if contract has end date and is now expired
	if sub.EndDate != "" && nextDate > sub.EndDate {
		if sub.AutoRenew {
			// Auto renew: extend end date by 1 year or interval
			currEnd, _ := time.Parse("2006-01-02", sub.EndDate)
			sub.EndDate = currEnd.AddDate(1, 0, 0).Format("2006-01-02")
		} else {
			sub.State = StateClosed
			sub.ChurnReason = "Kontrak berakhir secara wajar (Expired End Date)"
		}
	}

	if err := s.repo.UpdateSubscription(sub); err != nil {
		return nil, fmt.Errorf("gagal memperbarui status kontrak langganan: %w", err)
	}

	// Universal Chatter Logging
	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"sale.subscription",
			sub.ID,
			userID,
			"Subscription Engine",
			fmt.Sprintf("Faktur Berulang <b>%s</b> berhasil diterbitkan otomatis senilai <b>Rp %.2f</b>. Jadwal penagihan berikutnya: <b>%s</b>.",
				move.Name, sub.RecurringAmount, sub.NextInvoiceDate),
			"invoice_generated",
			companyID,
		)
	}

	return invLog, nil
}

// RunBillingBatch processes all due subscriptions up to targetDate
func (s *Service) RunBillingBatch(req BillingBatchRequest, companyID uint, userID uint) (*BillingBatchResult, error) {
	targetDate := req.TargetDate
	if targetDate == "" {
		targetDate = time.Now().Format("2006-01-02")
	}

	dueSubs, err := s.repo.GetDueSubscriptions(companyID, targetDate)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data langganan jatuh tempo: %w", err)
	}

	res := &BillingBatchResult{
		TargetDate:             targetDate,
		ProcessedSubscriptions: len(dueSubs),
		GeneratedInvoices:      []SubscriptionInvoiceLog{},
		Errors:                 []string{},
	}

	for _, sub := range dueSubs {
		invLog, err := s.GenerateSingleInvoice(sub.ID, companyID, userID)
		if err != nil {
			errStr := fmt.Sprintf("Kontrak %s (%s): %v", sub.Code, sub.Name, err)
			res.Errors = append(res.Errors, errStr)
			continue
		}

		res.InvoicesGenerated++
		res.TotalBilledRevenue += invLog.Amount
		res.GeneratedInvoices = append(res.GeneratedInvoices, *invLog)
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// 5. Cloudflare Workers AI SaaS Retention & Churn Risk Copilot
// ---------------------------------------------------------------------------

func (s *Service) AnalyzeChurnRisk(subscriptionID uint, companyID uint) (*AIChurnRiskReport, error) {
	sub, err := s.repo.GetSubscriptionByID(subscriptionID, companyID)
	if err != nil {
		return nil, fmt.Errorf("langganan ID %d tidak ditemukan: %w", subscriptionID, err)
	}

	// Compute contract age in days
	startDate, _ := time.Parse("2006-01-02", sub.StartDate)
	ageDays := int(time.Since(startDate).Hours() / 24)
	if ageDays < 0 {
		ageDays = 0
	}

	// Check overdue invoice count
	var overdueInvoices int64
	today := time.Now().Format("2006-01-02")
	s.db.Model(&account.AccountMove{}).
		Where("partner_id = ? AND move_type = 'out_invoice' AND state = 'posted' AND payment_state = 'not_paid' AND invoice_due_date < ?", sub.PartnerID, today).
		Count(&overdueInvoices)

	partnerName := sub.Partner.Name
	if partnerName == "" {
		partnerName = fmt.Sprintf("Customer #%d", sub.PartnerID)
	}

	report := &AIChurnRiskReport{
		SubscriptionID:   sub.ID,
		SubscriptionCode: sub.Code,
		PartnerName:      partnerName,
		ContractAgeDays:  ageDays,
		AIProvider:       "Local SaaS Retention Heuristic Engine",
	}

	// Smart Heuristic Analysis
	var riskLevel string
	var churnProb float64
	var keyDrivers []string
	var actionPlan []string
	var discountPct float64

	switch sub.State {
	case StateChurned:
		riskLevel = "CRITICAL"
		churnProb = 100.0
		keyDrivers = append(keyDrivers, fmt.Sprintf("Kontrak telah dibatalkan (Churned). Alasan: %s", sub.ChurnReason))
		actionPlan = append(actionPlan, "Kirimkan survei 'Exit Interview' dan tawarkan promo 'Win-Back' dengan diskon 25% setelah masa cooling-off 30 hari")
		discountPct = 25.0
	case StatePaused:
		riskLevel = "HIGH"
		churnProb = 75.0
		keyDrivers = append(keyDrivers, "Langganan saat ini dijeda (Paused), indikator awal risiko kehilangan pelanggan permanen")
		actionPlan = append(actionPlan, "Hubungi penanggung jawab akun (PIC) dalam 48 jam untuk mendiskusikan penyesuaian kuota atau opsi paket fleksibel")
		discountPct = 15.0
	case StateClosed:
		riskLevel = "MEDIUM"
		churnProb = 50.0
		keyDrivers = append(keyDrivers, "Kontrak telah mencapai tanggal berakhir dan ditutup")
		actionPlan = append(actionPlan, "Hubungi pelanggan untuk penawaran perpanjangan kontrak multi-tahun dengan garansi harga tetap")
		discountPct = 10.0
	default:
		// Active In Progress
		churnScore := 10.0 // Base healthy probability

		if !sub.AutoRenew {
			churnScore += 30.0
			keyDrivers = append(keyDrivers, "Fitur perpanjangan otomatis (Auto-Renew) non-aktif, berpotensi tidak diperpanjang")
		}

		if overdueInvoices > 0 {
			churnScore += float64(overdueInvoices) * 15.0
			keyDrivers = append(keyDrivers, fmt.Sprintf("Terdapat %d faktur pelanggan yang telah melewati batas jatuh tempo", overdueInvoices))
		}

		if sub.EndDate != "" {
			endT, _ := time.Parse("2006-01-02", sub.EndDate)
			daysUntilEnd := int(endT.Sub(time.Now()).Hours() / 24)
			if daysUntilEnd <= 30 && daysUntilEnd >= 0 {
				churnScore += 25.0
				keyDrivers = append(keyDrivers, fmt.Sprintf("Kontrak akan berakhir dalam %d hari", daysUntilEnd))
			}
		}

		if ageDays > 365 {
			// Loyal customer bonus
			churnScore = math.Max(5.0, churnScore-10.0)
		}

		churnProb = math.Min(95.0, math.Max(5.0, churnScore))

		if churnProb >= 70.0 {
			riskLevel = "HIGH"
			discountPct = 15.0
			actionPlan = append(actionPlan, "Jadwalkan Quarterly Business Review (QBR) darurat bersama Customer Success Manager")
			actionPlan = append(actionPlan, "Tawarkan diskon retensi 15% untuk komitmen perpanjangan 1 tahun berikutnya")
			actionPlan = append(actionPlan, "Verifikasi kendala teknis atau komplain helpdesk yang belum terselesaikan")
		} else if churnProb >= 40.0 {
			riskLevel = "MEDIUM"
			discountPct = 10.0
			actionPlan = append(actionPlan, "Kirimkan email check-in proaktif mengenai adopsi fitur dan kepuasan layanan")
			actionPlan = append(actionPlan, "Tawarkan sesi training gratis tambahan bagi staf operasional pengguna")
		} else {
			riskLevel = "LOW"
			discountPct = 0.0
			keyDrivers = append(keyDrivers, "Pola pembayaran dan kelangsungan kontrak pelanggan stabil dan sehat")
			actionPlan = append(actionPlan, "Akun berada dalam kondisi sehat. Peluang upsell atau cross-sell modul add-on")
			actionPlan = append(actionPlan, "Ajak pelanggan untuk berpartisipasi dalam program Customer Advocacy & Case Study")
		}
	}

	report.RiskLevel = riskLevel
	report.ChurnProbabilityPct = math.Round(churnProb*10) / 10
	report.KeyRiskDrivers = keyDrivers
	report.RetentionActionPlan = actionPlan
	report.RecommendedDiscount = discountPct
	report.ExecutiveSummary = fmt.Sprintf("Pelanggan %s pada kontrak %s memiliki estimasi risiko churn %s (%.1f%%). Diperlukan %d tindakan strategis untuk mengamankan pendapatan MRR Rp %.2f.",
		partnerName, sub.Code, riskLevel, report.ChurnProbabilityPct, len(actionPlan), sub.TotalMRR)

	// Try Cloudflare Workers AI if available
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		prompt := fmt.Sprintf(`Analyze subscription churn risk and return strict JSON:
Subscription Code: %s
Customer: %s
Plan/Template: %s
State: %s
Interval: %s
Recurring Amount: Rp %.2f
MRR: Rp %.2f
Contract Age: %d days
Auto-Renew: %v
Overdue Invoices: %d
End Date: %s

Respond with valid JSON format:
{
  "risk_level": "LOW|MEDIUM|HIGH|CRITICAL",
  "churn_probability_pct": 25.5,
  "key_risk_drivers": ["driver 1", "driver 2"],
  "retention_action_plan": ["action 1", "action 2"],
  "recommended_discount_pct": 10.0,
  "executive_summary": "Summary..."
}`, sub.Code, partnerName, sub.Template.Name, sub.State, sub.RecurringInterval, sub.RecurringAmount, sub.TotalMRR, ageDays, sub.AutoRenew, overdueInvoices, sub.EndDate)

		systemPrompt := "You are a Chief Revenue Officer & Customer Success AI Expert specializing in B2B SaaS retention, ARR expansion, and churn prevention."

		if raw, err := s.aiClient.CallRaw(prompt, systemPrompt); err == nil && raw != "" {
			var aiRes struct {
				RiskLevel           string   `json:"risk_level"`
				ChurnProbabilityPct float64  `json:"churn_probability_pct"`
				KeyRiskDrivers      []string `json:"key_risk_drivers"`
				RetentionActionPlan []string `json:"retention_action_plan"`
				RecommendedDiscount float64  `json:"recommended_discount_pct"`
				ExecutiveSummary    string   `json:"executive_summary"`
			}
			clean := strings.TrimSpace(raw)
			if idx := strings.Index(clean, "{"); idx != -1 {
				if last := strings.LastIndex(clean, "}"); last != -1 && last > idx {
					clean = clean[idx : last+1]
				}
			}
			if err := json.Unmarshal([]byte(clean), &aiRes); err == nil && aiRes.RiskLevel != "" {
				report.RiskLevel = aiRes.RiskLevel
				report.ChurnProbabilityPct = aiRes.ChurnProbabilityPct
				if len(aiRes.KeyRiskDrivers) > 0 {
					report.KeyRiskDrivers = aiRes.KeyRiskDrivers
				}
				if len(aiRes.RetentionActionPlan) > 0 {
					report.RetentionActionPlan = aiRes.RetentionActionPlan
				}
				report.RecommendedDiscount = aiRes.RecommendedDiscount
				if aiRes.ExecutiveSummary != "" {
					report.ExecutiveSummary = aiRes.ExecutiveSummary
				}
				report.AIProvider = "Cloudflare Workers AI (Llama 3.3 70B)"
			}
		}
	}

	return report, nil
}

// ---------------------------------------------------------------------------
// 6. SaaS Analytics & KPI Stats
// ---------------------------------------------------------------------------

func (s *Service) GetStats(companyID uint) (*SaaSMetrics, error) {
	return s.repo.GetSaaSMetrics(companyID)
}

func (s *Service) ListInvoiceLogs(companyID uint, subscriptionID uint) ([]SubscriptionInvoiceLog, error) {
	return s.repo.ListInvoiceLogs(companyID, subscriptionID)
}
