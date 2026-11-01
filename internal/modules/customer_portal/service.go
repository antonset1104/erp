package customer_portal

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/sale"
	"erp-golang/internal/modules/stock"
	"gorm.io/gorm"
)

type Service struct {
	repo       *Repository
	db         *gorm.DB
	saleSvc    *sale.Service
	stockSvc   *stock.Service
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(
	repo *Repository,
	db *gorm.DB,
	saleSvc *sale.Service,
	stockSvc *stock.Service,
	accountSvc *account.Service,
	chatterSvc *chatter.Service,
	aiClient *ai.Client,
) *Service {
	return &Service{
		repo:       repo,
		db:         db,
		saleSvc:    saleSvc,
		stockSvc:   stockSvc,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// -------------------------------------------------------------
// Customer Account Management
// -------------------------------------------------------------

func (s *Service) CreateAccount(req CreateCustomerAccountRequest) (*CustomerPortalAccount, error) {
	// Check if already exists for partner
	existing, _ := s.repo.GetAccountByPartnerID(req.PartnerID)
	if existing != nil {
		return existing, nil
	}

	// Generate secure token
	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	token := fmt.Sprintf("cp_tok_%s", hex.EncodeToString(tokenBytes))

	acc := &CustomerPortalAccount{
		PartnerID:      req.PartnerID,
		CustomerName:   req.CustomerName,
		ContactName:    req.ContactName,
		Email:          req.Email,
		Phone:          req.Phone,
		AccessToken:    token,
		IsActive:       true,
		CreditLimit:    req.CreditLimit,
		AllowedReorder: req.AllowedReorder,
	}

	if err := s.repo.CreateAccount(acc); err != nil {
		return nil, fmt.Errorf("gagal membuat akun portal pelanggan: %w", err)
	}

	s.logChatter(
		"res.partner",
		acc.PartnerID,
		"Customer Portal",
		fmt.Sprintf("🔑 Akun Customer Portal mandiri diaktifkan untuk %s (%s). Token akses telah diterbitkan.", acc.CustomerName, acc.Email),
		1,
	)

	return acc, nil
}

func (s *Service) GetAccountByToken(token string) (*CustomerPortalAccount, error) {
	acc, err := s.repo.GetAccountByToken(token)
	if err != nil {
		return nil, fmt.Errorf("token portal pelanggan tidak valid atau akun tidak aktif")
	}

	now := time.Now()
	acc.LastLoginAt = &now
	acc.TotalLogins++
	_ = s.repo.UpdateAccount(acc)

	return acc, nil
}

func (s *Service) GetAccountByID(id uint) (*CustomerPortalAccount, error) {
	return s.repo.GetAccountByID(id)
}

func (s *Service) ListAccounts() ([]CustomerPortalAccount, error) {
	return s.repo.ListAccounts()
}

// -------------------------------------------------------------
// Customer Portal Dashboard & B2B Orders
// -------------------------------------------------------------

type CustomerDashboardSummary struct {
	Account          *CustomerPortalAccount `json:"account"`
	TotalOrdersCount int                    `json:"total_orders_count"`
	RecentOrders     []sale.SaleOrder       `json:"recent_orders"`
	OutstandingDue   float64                `json:"outstanding_due"`
	CreditLimit      float64                `json:"credit_limit"`
	CreditRemaining  float64                `json:"credit_remaining"`
	PendingRMAsCount int                    `json:"pending_rmas_count"`
}

func (s *Service) GetCustomerDashboard(accountID uint) (*CustomerDashboardSummary, error) {
	acc, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return nil, err
	}

	// 1. Orders
	var orders []sale.SaleOrder
	s.db.Preload("OrderLines.Product").Where("partner_id = ?", acc.PartnerID).Order("id desc").Limit(5).Find(&orders)
	var totalOrders int64
	s.db.Model(&sale.SaleOrder{}).Where("partner_id = ?", acc.PartnerID).Count(&totalOrders)

	// 2. AR Balance
	stmt, _ := s.GetCustomerStatement(acc.PartnerID)
	outstanding := 0.0
	if stmt != nil {
		outstanding = stmt.TotalOutstanding
	}
	remainingCredit := acc.CreditLimit - outstanding
	if remainingCredit < 0 {
		remainingCredit = 0
	}

	// 3. RMAs
	rmas, _ := s.repo.ListRMAs(acc.ID)
	pendingRMA := 0
	for _, r := range rmas {
		if r.Status == "submitted" || r.Status == "under_review" {
			pendingRMA++
		}
	}

	return &CustomerDashboardSummary{
		Account:          acc,
		TotalOrdersCount: int(totalOrders),
		RecentOrders:     orders,
		OutstandingDue:   outstanding,
		CreditLimit:      acc.CreditLimit,
		CreditRemaining:  remainingCredit,
		PendingRMAsCount: pendingRMA,
	}, nil
}

func (s *Service) ListCustomerSaleOrders(partnerID uint) ([]sale.SaleOrder, error) {
	var orders []sale.SaleOrder
	err := s.db.Preload("OrderLines.Product").Where("partner_id = ?", partnerID).Order("id desc").Find(&orders).Error
	return orders, err
}

func (s *Service) CreateB2BOrderRequest(accountID uint, req CreateB2BOrderRequestPayload) (*B2BOrderRequest, error) {
	acc, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("akun portal tidak ditemukan: %w", err)
	}

	reqNum, err := s.repo.GenerateOrderRequestNumber()
	if err != nil {
		return nil, err
	}

	var reqDelivery *time.Time
	if req.RequestedDeliveryDate != "" {
		if t, err := time.Parse("2006-01-02", req.RequestedDeliveryDate); err == nil {
			reqDelivery = &t
		}
	}

	var totalUntaxed float64
	var lines []B2BOrderLine
	var saleOrderLines []sale.SaleOrderLine

	for _, l := range req.Lines {
		var prod product.Product
		if err := s.db.First(&prod, l.ProductID).Error; err != nil {
			return nil, fmt.Errorf("produk ID %d tidak ditemukan", l.ProductID)
		}

		unitPrice := l.UnitPrice
		if unitPrice <= 0 {
			unitPrice = prod.ListPrice
		}
		subtotal := l.Quantity * unitPrice
		totalUntaxed += subtotal

		lines = append(lines, B2BOrderLine{
			ProductID: prod.ID,
			Quantity:  l.Quantity,
			UnitPrice: unitPrice,
			Subtotal:  subtotal,
			Notes:     l.Notes,
		})

		saleOrderLines = append(saleOrderLines, sale.SaleOrderLine{
			ProductID:     prod.ID,
			ProductUomQty: l.Quantity,
			PriceUnit:     unitPrice,
			PriceSubtotal: subtotal,
		})
	}

	taxAmount := totalUntaxed * 0.11 // PPN 11%
	totalAmount := totalUntaxed + taxAmount

	payMethod := req.PaymentMethod
	if payMethod == "" {
		payMethod = "credit_terms"
	}

	orderReq := &B2BOrderRequest{
		RequestNumber:         reqNum,
		PortalAccountID:       acc.ID,
		PartnerID:             acc.PartnerID,
		CompanyID:             1,
		SubtotalAmount:        totalUntaxed,
		TaxAmount:             taxAmount,
		TotalAmount:           totalAmount,
		PaymentMethod:         payMethod,
		DeliveryAddress:       req.DeliveryAddress,
		RequestedDeliveryDate: reqDelivery,
		CustomerNotes:         req.CustomerNotes,
		Status:                "submitted",
		Lines:                 lines,
	}

	if err := s.repo.CreateOrderRequest(orderReq); err != nil {
		return nil, fmt.Errorf("gagal membuat pesanan B2B: %w", err)
	}

	// Auto-create draft Sale Order in ERP
	saleOrder := &sale.SaleOrder{
		PartnerID:  acc.PartnerID,
		State:      "draft",
		DateOrder:  time.Now().Format("2006-01-02 15:04:05"),
		TaxRate:    11.0,
		OrderLines: saleOrderLines,
		Note:       fmt.Sprintf("Customer Portal Order Ref: %s", reqNum),
	}
	saleOrder.CompanyID = 1
	if err := s.saleSvc.CreateOrder(saleOrder); err == nil {
		orderReq.SaleOrderID = &saleOrder.ID
		orderReq.Status = "approved"
		_ = s.repo.UpdateOrderRequest(orderReq)

		s.logChatter(
			"sale.order",
			saleOrder.ID,
			"Customer Portal",
			fmt.Sprintf("🛒 Pesanan B2B mandiri (%s) diterima dari pelanggan %s (Nilai: Rp %.2f). Draf SO dibuat otomatis.",
				reqNum, acc.CustomerName, totalAmount),
			1,
		)
	}

	return orderReq, nil
}

func (s *Service) ApproveB2BOrder(requestID uint, autoConfirm bool) (*B2BOrderRequest, error) {
	orderReq, err := s.repo.GetOrderRequestByID(requestID)
	if err != nil {
		return nil, fmt.Errorf("pesanan B2B tidak ditemukan")
	}

	if orderReq.SaleOrderID != nil && autoConfirm {
		_, _, err := s.saleSvc.ActionConfirm(*orderReq.SaleOrderID, orderReq.CompanyID)
		if err != nil {
			return nil, fmt.Errorf("gagal konfirmasi pesanan penjualan: %w", err)
		}
		orderReq.Status = "converted_to_so"
	} else {
		orderReq.Status = "approved"
	}

	if err := s.repo.UpdateOrderRequest(orderReq); err != nil {
		return nil, err
	}

	return orderReq, nil
}

func (s *Service) ListOrderRequests(portalAccountID uint) ([]B2BOrderRequest, error) {
	return s.repo.ListOrderRequests(portalAccountID)
}

func (s *Service) ListAllOrderRequests() ([]B2BOrderRequest, error) {
	return s.repo.ListAllOrderRequests()
}

// -------------------------------------------------------------
// Invoices & Customer AR Statement
// -------------------------------------------------------------

func (s *Service) ListCustomerInvoices(partnerID uint) ([]account.AccountMove, error) {
	var invoices []account.AccountMove
	err := s.db.Where("partner_id = ? AND move_type = ? AND state = ?", partnerID, "out_invoice", "posted").
		Order("id desc").Find(&invoices).Error
	return invoices, err
}

func (s *Service) GetCustomerStatement(partnerID uint) (*CustomerStatementSummary, error) {
	acc, _ := s.repo.GetAccountByPartnerID(partnerID)
	custName := fmt.Sprintf("Customer #%d", partnerID)
	creditLimit := 0.0
	if acc != nil {
		custName = acc.CustomerName
		creditLimit = acc.CreditLimit
	}

	var invoices []account.AccountMove
	s.db.Where("partner_id = ? AND move_type = ? AND state = ? AND payment_state IN ?",
		partnerID, "out_invoice", "posted", []string{"not_paid", "partial"}).
		Order("invoice_date asc").Find(&invoices)

	summary := &CustomerStatementSummary{
		PartnerID:        partnerID,
		CustomerName:     custName,
		CreditLimit:      creditLimit,
		Invoices:         invoices,
		TotalOutstanding: 0,
		AgingCurrent:     0,
		Aging1To30:       0,
		Aging31To60:      0,
		AgingOver60:      0,
	}

	now := time.Now()
	for _, inv := range invoices {
		dueAmount := inv.AmountResidual
		if dueAmount <= 0 {
			dueAmount = inv.AmountTotal
		}
		summary.TotalOutstanding += dueAmount

		// Calculate aging based on InvoiceDueDate or InvoiceDate
		dueDate := now
		if inv.InvoiceDueDate != "" {
			if t, err := time.Parse("2006-01-02", inv.InvoiceDueDate); err == nil {
				dueDate = t
			}
		} else if inv.InvoiceDate != "" {
			if t, err := time.Parse("2006-01-02", inv.InvoiceDate); err == nil {
				dueDate = t
			}
		}

		daysOverdue := int(now.Sub(dueDate).Hours() / 24)
		if daysOverdue <= 0 {
			summary.AgingCurrent += dueAmount
		} else if daysOverdue <= 30 {
			summary.Aging1To30 += dueAmount
		} else if daysOverdue <= 60 {
			summary.Aging31To60 += dueAmount
		} else {
			summary.AgingOver60 += dueAmount
		}
	}

	return summary, nil
}

// -------------------------------------------------------------
// Return Merchandise Authorization (RMA)
// -------------------------------------------------------------

func (s *Service) SubmitRMARequest(accountID uint, req SubmitRMARequestPayload) (*RMARequest, error) {
	acc, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("akun portal pelanggan tidak ditemukan: %w", err)
	}

	rmaNum, err := s.repo.GenerateRMANumber()
	if err != nil {
		return nil, err
	}

	resolution := req.ResolutionRequested
	if resolution == "" {
		resolution = "replace_item"
	}

	rma := &RMARequest{
		RMANumber:           rmaNum,
		PortalAccountID:     acc.ID,
		PartnerID:           acc.PartnerID,
		SaleOrderID:         req.SaleOrderID,
		InvoiceID:           req.InvoiceID,
		ProductID:           req.ProductID,
		Quantity:            req.Quantity,
		ReasonCode:          req.ReasonCode,
		Description:         req.Description,
		EvidenceURL:         req.EvidenceURL,
		ResolutionRequested: resolution,
		Status:              "submitted",
	}

	if err := s.repo.CreateRMA(rma); err != nil {
		return nil, fmt.Errorf("gagal mengajukan RMA: %w", err)
	}

	if req.SaleOrderID != nil {
		s.logChatter(
			"sale.order",
			*req.SaleOrderID,
			"Customer Portal",
			fmt.Sprintf("⚠️ Klaim Retur/RMA (%s) diajukan oleh pelanggan %s untuk produk ID #%d (Qty: %.2f). Alasan: %s.",
				rmaNum, acc.CustomerName, req.ProductID, req.Quantity, req.ReasonCode),
			1,
		)
	}

	return rma, nil
}

func (s *Service) ApproveRMA(rmaID uint, resolutionAction string, adminNotes string) (*RMARequest, error) {
	rma, err := s.repo.GetRMAByID(rmaID)
	if err != nil {
		return nil, fmt.Errorf("pengajuan RMA tidak ditemukan: %w", err)
	}

	rma.Status = "approved"
	rma.ResolutionAction = resolutionAction
	rma.AdminNotes = adminNotes
	now := time.Now()
	rma.ResolvedAt = &now

	if err := s.repo.UpdateRMA(rma); err != nil {
		return nil, err
	}

	if rma.SaleOrderID != nil {
		s.logChatter(
			"sale.order",
			*rma.SaleOrderID,
			"Customer Portal",
			fmt.Sprintf("✅ RMA %s disetujui. Tindakan: %s. Catatan Admin: %s.",
				rma.RMANumber, resolutionAction, adminNotes),
			1,
		)
	}

	return rma, nil
}

func (s *Service) RejectRMA(rmaID uint, reason string) (*RMARequest, error) {
	rma, err := s.repo.GetRMAByID(rmaID)
	if err != nil {
		return nil, fmt.Errorf("pengajuan RMA tidak ditemukan: %w", err)
	}

	rma.Status = "rejected"
	rma.AdminNotes = reason
	now := time.Now()
	rma.ResolvedAt = &now

	if err := s.repo.UpdateRMA(rma); err != nil {
		return nil, err
	}

	return rma, nil
}

func (s *Service) ListRMAs(portalAccountID uint) ([]RMARequest, error) {
	return s.repo.ListRMAs(portalAccountID)
}

func (s *Service) ListAllRMAs() ([]RMARequest, error) {
	return s.repo.ListAllRMAs()
}

// -------------------------------------------------------------
// Payment Notification & Auto-Reconcile
// -------------------------------------------------------------

func (s *Service) SubmitPaymentNotification(accountID uint, req SubmitPaymentNotificationPayload) (*CustomerPaymentNotification, error) {
	acc, err := s.repo.GetAccountByID(accountID)
	if err != nil {
		return nil, fmt.Errorf("akun portal pelanggan tidak ditemukan: %w", err)
	}

	notifNum, err := s.repo.GeneratePaymentNotificationNumber()
	if err != nil {
		return nil, err
	}

	payDate := time.Now()
	if req.PaymentDate != "" {
		if t, err := time.Parse("2006-01-02", req.PaymentDate); err == nil {
			payDate = t
		}
	}

	notif := &CustomerPaymentNotification{
		NotificationNumber: notifNum,
		PortalAccountID:    acc.ID,
		PartnerID:          acc.PartnerID,
		InvoiceID:          req.InvoiceID,
		AmountPaid:         req.AmountPaid,
		PaymentDate:        payDate,
		SenderBank:         req.SenderBank,
		SenderAccountName:  req.SenderAccountName,
		SenderAccountNo:    req.SenderAccountNo,
		TargetJournalID:    1,
		ReferenceNumber:    req.ReferenceNumber,
		ProofURL:           req.ProofURL,
		Status:             "pending_verification",
		Notes:              req.Notes,
	}

	if err := s.repo.CreatePaymentNotification(notif); err != nil {
		return nil, fmt.Errorf("gagal mencatat konfirmasi pembayaran: %w", err)
	}

	s.logChatter(
		"account.move",
		req.InvoiceID,
		"Customer Portal",
		fmt.Sprintf("💳 Konfirmasi Pembayaran Pelanggan (%s) diterima dari %s sejumlah Rp %.2f (Ref: %s). Menunggu verifikasi keuangan.",
			notifNum, acc.CustomerName, req.AmountPaid, req.ReferenceNumber),
		1,
	)

	return notif, nil
}

func (s *Service) VerifyPaymentNotification(notifID uint, verifyUserID uint, autoPost bool) (*CustomerPaymentNotification, error) {
	notif, err := s.repo.GetPaymentNotificationByID(notifID)
	if err != nil {
		return nil, fmt.Errorf("data konfirmasi pembayaran tidak ditemukan: %w", err)
	}

	var move account.AccountMove
	if err := s.db.First(&move, notif.InvoiceID).Error; err != nil {
		return nil, fmt.Errorf("tagihan faktur terkait tidak ditemukan: %w", err)
	}

	now := time.Now()
	notif.Status = "confirmed"
	notif.VerifiedByUserID = &verifyUserID
	notif.VerifiedAt = &now

	if autoPost {
		memo := fmt.Sprintf("Pelunasan Portal %s - Inv %s", notif.NotificationNumber, move.Name)
		pmt, _, err := s.accountSvc.RegisterPayment(move.ID, notif.AmountPaid, notif.TargetJournalID, memo, move.CompanyID)
		if err == nil && pmt != nil {
			notif.AccountPaymentID = &pmt.ID
		}
	}

	if err := s.repo.UpdatePaymentNotification(notif); err != nil {
		return nil, err
	}

	s.logChatter(
		"account.move",
		move.ID,
		"Finance Officer",
		fmt.Sprintf("✅ Pembayaran Faktur %s telah diverifikasi & dialokasikan ke jurnal pelunasan (Nilai: Rp %.2f).",
			move.Name, notif.AmountPaid),
		move.CompanyID,
	)

	return notif, nil
}

func (s *Service) ListPaymentNotifications(portalAccountID uint) ([]CustomerPaymentNotification, error) {
	return s.repo.ListPaymentNotifications(portalAccountID)
}

func (s *Service) ListAllPaymentNotifications() ([]CustomerPaymentNotification, error) {
	return s.repo.ListAllPaymentNotifications()
}

// -------------------------------------------------------------
// AI Customer Health & Churn Risk Scorecard
// -------------------------------------------------------------

func (s *Service) EvaluateAICustomerHealth(partnerID uint) (*AICustomerHealthScorecard, error) {
	custName := fmt.Sprintf("Customer #%d", partnerID)
	acc, _ := s.repo.GetAccountByPartnerID(partnerID)
	if acc != nil {
		custName = acc.CustomerName
	}

	// 1. Order History
	var orders []sale.SaleOrder
	s.db.Where("partner_id = ?", partnerID).Find(&orders)
	ordersCount := len(orders)
	lifetimeValue := 0.0
	for _, o := range orders {
		lifetimeValue += o.AmountTotal
	}
	avgOrderVal := 0.0
	if ordersCount > 0 {
		avgOrderVal = lifetimeValue / float64(ordersCount)
	}

	// 2. AR Statement & Payment Punctuality
	stmt, _ := s.GetCustomerStatement(partnerID)
	paymentPunctuality := 90.0
	if stmt != nil && stmt.TotalOutstanding > 0 {
		overdueRatio := (stmt.Aging31To60 + stmt.AgingOver60*2.0) / stmt.TotalOutstanding
		paymentPunctuality = math.Max(30.0, 100.0-(overdueRatio*60.0))
	}

	// 3. Return Rate
	rmas, _ := s.repo.ListRMAs(0)
	rmaCountForCust := 0
	for _, r := range rmas {
		if r.PartnerID == partnerID {
			rmaCountForCust++
		}
	}
	returnRate := 0.0
	if ordersCount > 0 {
		returnRate = (float64(rmaCountForCust) / float64(ordersCount)) * 100.0
	}

	// 4. Composite Health Score (0-100)
	// Punctuality (40%) + Order Volume Score (35%) + Low Return Rate (25%)
	volumeScore := math.Min(100.0, float64(ordersCount)*20.0)
	returnScore := math.Max(0.0, 100.0-(returnRate*5.0))
	overallHealth := (paymentPunctuality * 0.40) + (volumeScore * 0.35) + (returnScore * 0.25)

	tier := "REGULAR"
	churnRisk := "LOW"

	if overallHealth >= 85 {
		tier = "VIP PLATINUM"
		churnRisk = "LOW"
	} else if overallHealth >= 70 {
		tier = "GOLD"
		churnRisk = "LOW"
	} else if overallHealth >= 50 {
		tier = "REGULAR"
		churnRisk = "MEDIUM"
	} else {
		tier = "AT_RISK"
		churnRisk = "HIGH"
	}

	// Executive Summary & AI Advice
	execSummary := fmt.Sprintf("Evaluasi Kesehatan Pelanggan B2B: %s berstatus %s (Skor Kesehatan: %.1f/100, Risiko Churn: %s). LTV Tercatat: Rp %.2f dari %d pesanan. Rasio ketepatan pembayaran: %.1f%% dengan tingkat retur RMA %.1f%%.",
		custName, tier, overallHealth, churnRisk, lifetimeValue, ordersCount, paymentPunctuality, returnRate)

	var recs []string
	if churnRisk == "HIGH" || tier == "AT_RISK" {
		recs = []string{
			"Segera lakukan kontak proaktif account executive untuk mengidentifikasi kendala pembayaran piutang dan kepuasan produk",
			"Tinjau limit kredit dan jadwalkan restrukturisasi termin pembayaran sebelum menerbitkan pesanan B2B baru",
			"Lakukan audit mutu pada produk yang sering diajukan RMA untuk memulihkan kepercayaan pelanggan",
		}
	} else if churnRisk == "MEDIUM" {
		recs = []string{
			"Tawarkan program loyalitas diskon volume bertingkat (Tiered Rebate) untuk meningkatkan frekuensi pesanan",
			"Kirimkan e-Statement otomatis pada H-3 jatuh tempo guna mempercepat perputaran piutang (DSO)",
			"Optimalkan fitur B2B Re-order mandiri di portal dengan rekomendasi stok berdasarkan siklus konsumsi historis",
		}
	} else {
		recs = []string{
			"Pertahankan status VIP Platinum dengan memberikan prioritas alokasi stok gudang tercepat (Fast-Track Picking)",
			"Tawarkan kontrak tahunan (Blanket Sales Order) dengan komitmen harga stabil guna mengunci retensi jangka panjang",
			"Berikan apresiasi fasilitas termin kredit fleksibel NET 60 sebagai mitra strategis korporat",
		}
	}

	return &AICustomerHealthScorecard{
		PartnerID:               partnerID,
		CustomerName:            custName,
		TotalOrdersCount:        ordersCount,
		TotalOrderLifetimeValue: lifetimeValue,
		AverageOrderValue:       avgOrderVal,
		PaymentPunctualityScore: paymentPunctuality,
		ReturnRatePercentage:    returnRate,
		OverallHealthScore:      overallHealth,
		ChurnRiskLevel:          churnRisk,
		TierBadge:               tier,
		ExecutiveSummary:        execSummary,
		RecommendedActions:      recs,
		AIProvider:              "Local Heuristic Engine (Customer Retention & AR Analytics)",
		EvaluatedAt:             time.Now(),
	}, nil
}

// -------------------------------------------------------------
// Stats & Helpers
// -------------------------------------------------------------

func (s *Service) GetStats() (*CustomerPortalStats, error) {
	return s.repo.GetStats()
}

func (s *Service) logChatter(modelName string, resID uint, authorName string, text string, companyID uint) {
	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(modelName, resID, 1, authorName, text, "Customer Portal", companyID)
	}
}
