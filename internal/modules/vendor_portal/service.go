package vendor_portal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/purchase"
	"erp-golang/internal/modules/stock"
	"gorm.io/gorm"
)

type Service struct {
	repo        *Repository
	db          *gorm.DB
	purchaseSvc *purchase.Service
	stockSvc    *stock.Service
	accountSvc  *account.Service
	chatterSvc  *chatter.Service
	aiClient    *ai.Client
}

func NewService(
	repo *Repository,
	db *gorm.DB,
	purchaseSvc *purchase.Service,
	stockSvc *stock.Service,
	accountSvc *account.Service,
	chatterSvc *chatter.Service,
	aiClient *ai.Client,
) *Service {
	return &Service{
		repo:        repo,
		db:          db,
		purchaseSvc: purchaseSvc,
		stockSvc:    stockSvc,
		accountSvc:  accountSvc,
		chatterSvc:  chatterSvc,
		aiClient:    aiClient,
	}
}

// Generate random secure 32-char token
func generateSecureToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("token_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func (s *Service) logChatter(modelName string, resID uint, author string, text string, companyID uint) {
	if s.chatterSvc != nil {
		_ = s.chatterSvc.PostMessage(&chatter.MailMessage{
			ModelName:   modelName,
			ResID:       resID,
			AuthorName:  author,
			Body:        text,
			MessageType: "comment",
		})
	}
}

// Vendor Account Management
func (s *Service) CreateAccount(req CreateVendorAccountRequest) (*VendorPortalAccount, error) {
	// Check existing account
	existing, _ := s.repo.GetAccountByPartnerID(req.PartnerID)
	if existing != nil {
		return existing, nil
	}

	token := generateSecureToken()
	account := &VendorPortalAccount{
		PartnerID:   req.PartnerID,
		VendorName:  req.VendorName,
		ContactName: req.ContactName,
		Email:       req.Email,
		Phone:       req.Phone,
		AccessToken: token,
		IsActive:    true,
	}

	if err := s.repo.CreateAccount(account); err != nil {
		return nil, fmt.Errorf("gagal membuat akun portal vendor: %w", err)
	}

	s.logChatter(
		"res.partner",
		req.PartnerID,
		"System",
		fmt.Sprintf("🔑 Akun Vendor Portal dibuat untuk %s (%s). Token akses telah diaktifkan.", req.VendorName, req.Email),
		1,
	)

	return account, nil
}

func (s *Service) GetAccountByToken(token string) (*VendorPortalAccount, error) {
	acc, err := s.repo.GetAccountByToken(token)
	if err != nil {
		return nil, fmt.Errorf("token portal vendor tidak valid atau tidak aktif")
	}
	now := time.Now()
	_ = s.repo.UpdateAccountLogin(acc.ID)
	acc.LastLoginAt = &now
	acc.TotalLogins++
	return acc, nil
}

func (s *Service) ListAccounts() ([]VendorPortalAccount, error) {
	return s.repo.ListAccounts()
}

func (s *Service) GetAccountByID(id uint) (*VendorPortalAccount, error) {
	return s.repo.GetAccountByID(id)
}

// RFQ & Quotation Bidding
func (s *Service) ListOpenRFQs(partnerID uint) ([]VendorPortalRFQView, error) {
	return s.repo.ListOpenRFQsForVendor(partnerID)
}

func (s *Service) SubmitQuotationBid(partnerID uint, req SubmitBidRequest) (*VendorQuotationBid, error) {
	bidNumber, err := s.repo.GenerateBidNumber()
	if err != nil {
		return nil, err
	}

	var vendorName string
	acc, _ := s.repo.GetAccountByPartnerID(partnerID)
	if acc != nil {
		vendorName = acc.VendorName
	}

	totalAmount := 0.0
	var lines []VendorQuotationBidLine
	for _, l := range req.Lines {
		subtotal := l.Quantity * l.UnitPrice
		totalAmount += subtotal
		lines = append(lines, VendorQuotationBidLine{
			ProductID:   l.ProductID,
			ProductName: l.ProductName,
			ProductCode: l.ProductCode,
			Quantity:    l.Quantity,
			UnitPrice:   l.UnitPrice,
			Subtotal:    subtotal,
			Notes:       l.Notes,
		})
	}

	leadTime := req.DeliveryLeadTimeDays
	if leadTime <= 0 {
		leadTime = 7
	}
	payTerms := req.PaymentTerms
	if payTerms == "" {
		payTerms = "NET 30"
	}

	bid := &VendorQuotationBid{
		BidNumber:            bidNumber,
		RFQReference:         req.RFQReference,
		PurchaseOrderID:      req.PurchaseOrderID,
		PartnerID:            partnerID,
		VendorName:           vendorName,
		TotalBidAmount:       totalAmount,
		DeliveryLeadTimeDays: leadTime,
		PaymentTerms:         payTerms,
		Status:               "submitted",
		Notes:                req.Notes,
		SubmittedAt:          time.Now(),
		Lines:                lines,
	}

	if err := s.repo.CreateBid(bid); err != nil {
		return nil, fmt.Errorf("gagal mencatat penawaran harga vendor: %w", err)
	}

	if req.PurchaseOrderID != nil {
		s.logChatter(
			"purchase.order",
			*req.PurchaseOrderID,
			"Vendor Portal",
			fmt.Sprintf("📥 Penawaran harga baru diajukan oleh %s (%s). Total Penawaran: Rp %.2f (Lead Time: %d hari).",
				vendorName, bidNumber, totalAmount, leadTime),
			1,
		)
	}

	return bid, nil
}

func (s *Service) AcceptQuotationBid(bidID uint) (*VendorQuotationBid, error) {
	bid, err := s.repo.GetBidByID(bidID)
	if err != nil {
		return nil, fmt.Errorf("penawaran harga tidak ditemukan")
	}

	if err := s.repo.UpdateBidStatus(bidID, "accepted"); err != nil {
		return nil, err
	}
	bid.Status = "accepted"

	// If linked to PO, update lines and confirm PO
	if bid.PurchaseOrderID != nil {
		var po purchase.PurchaseOrder
		if err := s.db.Preload("OrderLines").First(&po, *bid.PurchaseOrderID).Error; err == nil {
			// Update prices based on bid lines
			bidLineMap := make(map[uint]float64)
			for _, bl := range bid.Lines {
				bidLineMap[bl.ProductID] = bl.UnitPrice
			}

			newTotal := 0.0
			for i := range po.OrderLines {
				if price, exists := bidLineMap[po.OrderLines[i].ProductID]; exists && price > 0 {
					po.OrderLines[i].PriceUnit = price
					po.OrderLines[i].PriceSubtotal = float64(po.OrderLines[i].ProductUomQty) * price
					s.db.Save(&po.OrderLines[i])
				}
				newTotal += po.OrderLines[i].PriceSubtotal
			}

			po.AmountTotal = newTotal
			s.db.Save(&po)

			// Action confirm PO if still in draft/sent
			if po.State == "draft" || po.State == "sent" {
				if s.purchaseSvc != nil {
					_, _, _ = s.purchaseSvc.ActionConfirm(po.ID, po.CompanyID)
				}
			}

			s.logChatter(
				"purchase.order",
				po.ID,
				"Procurement Team",
				fmt.Sprintf("✅ Penawaran %s diterima! Harga PO diperbarui menjadi Rp %.2f dan PO dikonfirmasi.",
					bid.BidNumber, newTotal),
				po.CompanyID,
			)
		}
	}

	return bid, nil
}

func (s *Service) ListBids() ([]VendorQuotationBid, error) {
	return s.repo.ListBids()
}

func (s *Service) ListBidsByPartnerID(partnerID uint) ([]VendorQuotationBid, error) {
	return s.repo.ListBidsByPartnerID(partnerID)
}

func (s *Service) GetBidByID(id uint) (*VendorQuotationBid, error) {
	return s.repo.GetBidByID(id)
}

// Advance Shipping Notice (ASN)
func (s *Service) SubmitASN(partnerID uint, req SubmitASNRequest) (*VendorASN, error) {
	var po purchase.PurchaseOrder
	if err := s.db.First(&po, req.PurchaseOrderID).Error; err != nil {
		return nil, fmt.Errorf("dokumen Purchase Order #%d tidak ditemukan", req.PurchaseOrderID)
	}

	asnNumber, err := s.repo.GenerateASNNumber()
	if err != nil {
		return nil, err
	}

	var vendorName string
	acc, _ := s.repo.GetAccountByPartnerID(partnerID)
	if acc != nil {
		vendorName = acc.VendorName
	}

	estArrival, err := time.Parse("2006-01-02", req.EstimatedArrivalDate)
	if err != nil {
		estArrival = time.Now().AddDate(0, 0, 3)
	}

	var lines []VendorASNLine
	for _, l := range req.Lines {
		lines = append(lines, VendorASNLine{
			ProductID:       l.ProductID,
			ProductName:     l.ProductName,
			QuantityShipped: l.QuantityShipped,
			LotNumber:       l.LotNumber,
			Notes:           l.Notes,
		})
	}

	carrier := req.CarrierName
	if carrier == "" {
		carrier = "Ekspedisi Logistik"
	}

	asn := &VendorASN{
		ASNNumber:            asnNumber,
		PurchaseOrderID:      po.ID,
		PurchaseOrderRef:     po.Name,
		PartnerID:            partnerID,
		VendorName:           vendorName,
		CarrierName:          carrier,
		TrackingRef:          req.TrackingRef,
		DriverName:           req.DriverName,
		VehicleNumber:        req.VehicleNumber,
		EstimatedArrivalDate: estArrival,
		Status:               "in_transit",
		Notes:                req.Notes,
		Lines:                lines,
	}

	if err := s.repo.CreateASN(asn); err != nil {
		return nil, fmt.Errorf("gagal membuat ASN: %w", err)
	}

	s.logChatter(
		"purchase.order",
		po.ID,
		"Vendor Portal",
		fmt.Sprintf("🚚 Surat Jalan Pemasok (%s) diterbitkan oleh %s via %s (Resi: %s). Estimasi Tiba: %s.",
			asnNumber, vendorName, carrier, req.TrackingRef, estArrival.Format("02/01/2006")),
		po.CompanyID,
	)

	return asn, nil
}

func (s *Service) ReceiveASN(asnID uint, actualReceivedLines map[uint]float64) (*VendorASN, error) {
	asn, err := s.repo.GetASNByID(asnID)
	if err != nil {
		return nil, fmt.Errorf("dokumen ASN tidak ditemukan")
	}

	now := time.Now()
	for _, l := range asn.Lines {
		qtyRec, exists := actualReceivedLines[l.ID]
		if !exists {
			qtyRec = l.QuantityShipped // default to full receipt if not specified
		}
		_ = s.repo.UpdateASNLineReceived(l.ID, qtyRec)
	}

	if err := s.repo.UpdateASNStatus(asnID, "received", &now); err != nil {
		return nil, err
	}
	asn.Status = "received"
	asn.ActualArrivalDate = &now

	// If linked to PO, find incoming picking and validate if available
	var pickings []stock.StockPicking
	if err := s.db.Where("origin = ? AND picking_type = ?", asn.PurchaseOrderRef, "incoming").Find(&pickings).Error; err == nil && len(pickings) > 0 {
		for _, p := range pickings {
			if p.State == "draft" || p.State == "assigned" {
				if s.stockSvc != nil {
					_, _ = s.stockSvc.ValidatePicking(p.ID, p.CompanyID)
				}
			}
		}
	}

	s.logChatter(
		"purchase.order",
		asn.PurchaseOrderID,
		"Warehouse Team",
		fmt.Sprintf("📦 Penerimaan barang berdasarkan ASN %s selesai diperiksa dan dicatat di gudang.", asn.ASNNumber),
		1,
	)

	return asn, nil
}

func (s *Service) ListASNs() ([]VendorASN, error) {
	return s.repo.ListASNs()
}

func (s *Service) ListASNsByPartnerID(partnerID uint) ([]VendorASN, error) {
	return s.repo.ListASNsByPartnerID(partnerID)
}

func (s *Service) GetASNByID(id uint) (*VendorASN, error) {
	return s.repo.GetASNByID(id)
}

// Vendor Invoicing & Automated 3-Way Matching Engine
func (s *Service) SubmitVendorInvoice(partnerID uint, req SubmitVendorInvoiceRequest) (*VendorInvoiceSubmission, *ThreeWayMatchResult, error) {
	var po purchase.PurchaseOrder
	if err := s.db.Preload("OrderLines").First(&po, req.PurchaseOrderID).Error; err != nil {
		return nil, nil, fmt.Errorf("dokumen Purchase Order #%d tidak ditemukan", req.PurchaseOrderID)
	}

	subNumber, err := s.repo.GenerateInvoiceSubmissionNumber()
	if err != nil {
		return nil, nil, err
	}

	var vendorName string
	acc, _ := s.repo.GetAccountByPartnerID(partnerID)
	if acc != nil {
		vendorName = acc.VendorName
	}

	invDate, err := time.Parse("2006-01-02", req.InvoiceDate)
	if err != nil {
		invDate = time.Now()
	}
	dueDate, err := time.Parse("2006-01-02", req.DueDate)
	if err != nil {
		dueDate = invDate.AddDate(0, 0, 30)
	}

	inv := &VendorInvoiceSubmission{
		SubmissionNumber:      subNumber,
		PurchaseOrderID:       po.ID,
		PurchaseOrderRef:      po.Name,
		PartnerID:             partnerID,
		VendorName:            vendorName,
		InvoiceNumberSupplier: req.InvoiceNumberSupplier,
		InvoiceDate:           invDate,
		DueDate:               dueDate,
		Subtotal:              req.Subtotal,
		TaxAmount:             req.TaxAmount,
		TotalAmount:           req.TotalAmount,
		TaxInvoiceNumber:      req.TaxInvoiceNumber,
		FileAttachmentURL:     req.FileAttachmentURL,
		MatchingStatus:        "pending_match",
		Status:                "submitted",
		Notes:                 req.Notes,
	}

	if err := s.repo.CreateInvoiceSubmission(inv); err != nil {
		return nil, nil, fmt.Errorf("gagal mencatat tagihan vendor: %w", err)
	}

	// Execute 3-Way Matching immediately
	matchResult, _ := s.Execute3WayMatch(inv.ID)

	matchStatusText := "Lolos Rekonsiliasi 3-Way Matching"
	if !matchResult.IsMatch {
		matchStatusText = fmt.Sprintf("Terdapat Selisih: %s", strings.Join(matchResult.DiscrepancyDetail, "; "))
	}
	s.logChatter(
		"purchase.order",
		po.ID,
		"Vendor Portal",
		fmt.Sprintf("🧾 Tagihan Pemasok %s diajukan oleh %s (Nilai: Rp %.2f). Status Matching: %s.",
			req.InvoiceNumberSupplier, vendorName, req.TotalAmount, matchStatusText),
		po.CompanyID,
	)

	return inv, matchResult, nil
}

func (s *Service) Execute3WayMatch(submissionID uint) (*ThreeWayMatchResult, error) {
	inv, err := s.repo.GetInvoiceSubmissionByID(submissionID)
	if err != nil {
		return nil, fmt.Errorf("dokumen tagihan vendor tidak ditemukan")
	}

	var po purchase.PurchaseOrder
	if err := s.db.Preload("OrderLines").First(&po, inv.PurchaseOrderID).Error; err != nil {
		return nil, fmt.Errorf("purchase order terkait tidak ditemukan")
	}

	expectedAmount := po.AmountTotal
	billedAmount := inv.TotalAmount
	variance := math.Abs(expectedAmount - billedAmount)

	var discrepancies []string
	matchingStatus := "matched_3way"
	isMatch := true

	// Check total amount variance tolerance (Rp 1.00)
	if variance > 1.0 {
		isMatch = false
		matchingStatus = "discrepancy_price"
		discrepancies = append(discrepancies, fmt.Sprintf("Total tagihan (Rp %.2f) berbeda dengan PO (Rp %.2f)", billedAmount, expectedAmount))
	}

	// Check Goods Received verification (Check ASNs or Stock Picking)
	var asns []VendorASN
	s.db.Where("purchase_order_id = ?", po.ID).Find(&asns)
	hasReceipt := false
	for _, a := range asns {
		if a.Status == "received" {
			hasReceipt = true
			break
		}
	}

	// Also check incoming stock picking state
	if !hasReceipt {
		var count int64
		s.db.Model(&stock.StockPicking{}).
			Where("origin = ? AND picking_type = 'incoming' AND state = 'done'", po.Name).
			Count(&count)
		if count > 0 {
			hasReceipt = true
		}
	}

	if !hasReceipt {
		discrepancies = append(discrepancies, "Barang pesanan belum tercatat diterima lengkap di gudang (Pending GR)")
		if isMatch {
			matchingStatus = "pending_match"
		}
	}

	status := "verified"
	var billID *uint

	// If 3-way matched, create draft vendor bill in account.move
	if isMatch && hasReceipt {
		matchingStatus = "matched_3way"
		status = "approved"

		if s.accountSvc != nil && inv.AccountMoveID == nil {
			var purchaseJournal account.AccountJournal
			_ = s.db.Where("type = 'purchase'").First(&purchaseJournal).Error
			jID := purchaseJournal.ID
			if jID == 0 {
				jID = 1
			}

			bill := &account.AccountMove{
				MoveType:       "in_invoice",
				JournalID:      jID,
				PartnerID:      inv.PartnerID,
				InvoiceDate:    inv.InvoiceDate.Format("2006-01-02"),
				InvoiceDueDate: inv.DueDate.Format("2006-01-02"),
				Ref:            fmt.Sprintf("%s (PO: %s)", inv.InvoiceNumberSupplier, po.Name),
				AmountUntaxed:  inv.Subtotal,
				AmountTax:      inv.TaxAmount,
				AmountTotal:    inv.TotalAmount,
				State:          "draft",
			}
			bill.CompanyID = po.CompanyID
			if err := s.accountSvc.CreateMove(bill); err == nil {
				billID = &bill.ID
			}
		}
	}

	discrepancyText := strings.Join(discrepancies, " | ")
	_ = s.repo.UpdateInvoiceMatching(inv.ID, matchingStatus, discrepancyText, status, billID)

	return &ThreeWayMatchResult{
		SubmissionID:      inv.ID,
		MatchingStatus:    matchingStatus,
		IsMatch:           isMatch && hasReceipt,
		ExpectedAmount:    expectedAmount,
		BilledAmount:      billedAmount,
		VarianceAmount:    variance,
		DiscrepancyDetail: discrepancies,
		GeneratedBillID:   billID,
	}, nil
}

func (s *Service) ListInvoiceSubmissions() ([]VendorInvoiceSubmission, error) {
	return s.repo.ListInvoiceSubmissions()
}

func (s *Service) ListInvoiceSubmissionsByPartnerID(partnerID uint) ([]VendorInvoiceSubmission, error) {
	return s.repo.ListInvoiceSubmissionsByPartnerID(partnerID)
}

func (s *Service) GetInvoiceSubmissionByID(id uint) (*VendorInvoiceSubmission, error) {
	return s.repo.GetInvoiceSubmissionByID(id)
}

// AI Vendor Performance & Reliability Rating
func (s *Service) EvaluateAIVendorPerformance(partnerID uint) (*AIVendorPerformanceRating, error) {
	var partnerName string
	acc, _ := s.repo.GetAccountByPartnerID(partnerID)
	if acc != nil {
		partnerName = acc.VendorName
	}
	if partnerName == "" {
		partnerName = fmt.Sprintf("Vendor Partner #%d", partnerID)
	}

	// 1. Calculate On-Time Delivery Rate
	asns, _ := s.repo.ListASNsByPartnerID(partnerID)
	totalASNs := len(asns)
	onTimeCount := 0
	for _, a := range asns {
		if a.ActualArrivalDate != nil && !a.ActualArrivalDate.After(a.EstimatedArrivalDate.Add(24*time.Hour)) {
			onTimeCount++
		}
	}

	onTimeRate := 95.0
	if totalASNs > 0 {
		onTimeRate = (float64(onTimeCount) / float64(totalASNs)) * 100.0
	}

	// 2. Price Competitiveness
	bids, _ := s.repo.ListBidsByPartnerID(partnerID)
	totalBids := len(bids)
	acceptedBids := 0
	for _, b := range bids {
		if b.Status == "accepted" {
			acceptedBids++
		}
	}
	priceScore := 88.0
	if totalBids > 0 {
		priceScore = (float64(acceptedBids) / float64(totalBids)) * 100.0
		if priceScore < 60 {
			priceScore = 65.0
		}
	}

	// 3. Quality & Accuracy Rate
	invoices, _ := s.repo.ListInvoiceSubmissionsByPartnerID(partnerID)
	totalInvoices := len(invoices)
	matchedInvoices := 0
	for _, inv := range invoices {
		if inv.MatchingStatus == "matched_3way" {
			matchedInvoices++
		}
	}
	qualityRate := 98.0
	if totalInvoices > 0 {
		qualityRate = (float64(matchedInvoices) / float64(totalInvoices)) * 100.0
		if qualityRate < 70 {
			qualityRate = 75.0
		}
	}

	// Weighted overall score
	overallScore := (0.40 * onTimeRate) + (0.30 * priceScore) + (0.30 * qualityRate)
	tierBadge := "bronze"
	if overallScore >= 90.0 {
		tierBadge = "platinum"
	} else if overallScore >= 80.0 {
		tierBadge = "gold"
	} else if overallScore >= 70.0 {
		tierBadge = "silver"
	}

	report := &AIVendorPerformanceRating{
		PartnerID:                 partnerID,
		VendorName:                partnerName,
		TotalOrders:               totalASNs + totalBids,
		OnTimeDeliveryRate:        math.Round(onTimeRate*10) / 10,
		PriceCompetitivenessScore: math.Round(priceScore*10) / 10,
		QualityAcceptanceRate:     math.Round(qualityRate*10) / 10,
		OverallRatingScore:        math.Round(overallScore*10) / 10,
		TierBadge:                 tierBadge,
		ExecutiveSummary: fmt.Sprintf("Evaluasi Kinerja Pemasok AI: %s berperingkat %s dengan Skor Integritas Keseluruhan %.1f/100. Ketepatan waktu pengiriman berada di level %.1f%% dengan akurasi 3-way matching tagihan %.1f%%.",
			partnerName, strings.ToUpper(tierBadge), overallScore, onTimeRate, qualityRate),
		ActionableRecommendations: []string{
			"Prioritaskan vendor ini untuk alokasi Blanket Order semester depan berkat skor pengiriman tinggi",
			"Tingkatkan kolaborasi digital via portal mandiri untuk memangkas lead time konfirmasi PO hingga 24 jam",
			"Terapkan termin pembayaran NET 45 diskon 2% (2/10 NET 45) untuk mengoptimalkan working capital perusahaan",
		},
		AIProvider:  "Local Heuristic Engine (Procurement & Supplier Scorecard Baseline)",
		EvaluatedAt: time.Now(),
	}

	// Try Cloudflare Workers AI if configured
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		prompt := fmt.Sprintf(`Sebagai AI Procurement Specialist, berikan evaluasi skor performa vendor:
Vendor: %s, On-Time Rate: %.1f%%, Price Competitiveness: %.1f, Quality Rate: %.1f%%, Overall: %.1f, Tier: %s.
Format JSON valid:
{
  "executive_summary": "ringkasan eksekutif profesional",
  "actionable_recommendations": ["rekomendasi 1", "rekomendasi 2"]
}`, partnerName, onTimeRate, priceScore, qualityRate, overallScore, tierBadge)

		systemPrompt := "Anda adalah AI Strategic Sourcing & Supplier Relationship Manager. Berikan hanya JSON valid."
		resp, err := s.aiClient.CallRaw(prompt, systemPrompt)
		if err == nil && resp != "" {
			var aiRes struct {
				ExecutiveSummary          string   `json:"executive_summary"`
				ActionableRecommendations []string `json:"actionable_recommendations"`
			}
			cleanJSON := strings.TrimSpace(resp)
			cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
			cleanJSON = strings.TrimPrefix(cleanJSON, "```")
			cleanJSON = strings.TrimSuffix(cleanJSON, "```")
			cleanJSON = strings.TrimSpace(cleanJSON)

			if jErr := json.Unmarshal([]byte(cleanJSON), &aiRes); jErr == nil && aiRes.ExecutiveSummary != "" {
				report.ExecutiveSummary = aiRes.ExecutiveSummary
				if len(aiRes.ActionableRecommendations) > 0 {
					report.ActionableRecommendations = aiRes.ActionableRecommendations
				}
				report.AIProvider = "Cloudflare Workers AI (@cf/meta/llama-3.1-8b-instruct)"
			}
		} else {
			log.Printf("[Vendor Portal AI] Fallback triggered: %v", err)
		}
	}

	return report, nil
}

func (s *Service) GetStats() (*VendorPortalStats, error) {
	return s.repo.GetStats()
}
