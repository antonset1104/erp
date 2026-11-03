package ai

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/purchase"

	"gorm.io/gorm"
)

// Tender and Bidding Data Structures
type CreateTenderRequest struct {
	Title           string  `json:"title" binding:"required"`
	ProductID       uint    `json:"product_id" binding:"required"`
	TargetQuantity  float64 `json:"target_quantity" binding:"required"`
	BudgetUnitPrice float64 `json:"budget_unit_price"`
	Deadline        string  `json:"deadline"`
	VendorIDs       []uint  `json:"vendor_ids"`
}

type VendorInvite struct {
	VendorID   uint   `json:"vendor_id"`
	VendorName string `json:"vendor_name"`
	Token      string `json:"token"`
	BidURL     string `json:"bid_url"`
	Status     string `json:"status"` // "invited", "submitted"
}

type RFQBidTender struct {
	TenderID        string         `json:"tender_id"`
	Title           string         `json:"title"`
	ProductID       uint           `json:"product_id"`
	ProductName     string         `json:"product_name"`
	ProductSKU      string         `json:"product_sku"`
	TargetQuantity  float64        `json:"target_quantity"`
	BudgetUnitPrice float64        `json:"budget_unit_price"`
	Status          string         `json:"status"` // "open", "evaluating", "awarded"
	Deadline        string         `json:"deadline"`
	CompanyID       uint           `json:"company_id"`
	InvitedVendors  []VendorInvite `json:"invited_vendors"`
	AwardedVendorID *uint          `json:"awarded_vendor_id,omitempty"`
	AwardedPOName   string         `json:"awarded_po_name,omitempty"`
	CreatedAt       string         `json:"created_at"`
}

type VendorBidSubmission struct {
	TenderID         string  `json:"tender_id" binding:"required"`
	Token            string  `json:"token"`
	VendorID         uint    `json:"vendor_id" binding:"required"`
	VendorName       string  `json:"vendor_name,omitempty"`
	OfferedUnitPrice float64 `json:"offered_unit_price" binding:"required"`
	LeadTimeDays     int     `json:"lead_time_days"`
	PaymentTermsDays int     `json:"payment_terms_days"` // e.g. 30, 60, 90 days
	WarrantyMonths   int     `json:"warranty_months"`
	DiscountPct      float64 `json:"discount_pct"`
	Notes            string  `json:"notes"`
	SubmittedAt      string  `json:"submitted_at"`
}

type EvaluatedBid struct {
	VendorID               uint    `json:"vendor_id"`
	VendorName             string  `json:"vendor_name"`
	OfferedUnitPrice       float64 `json:"offered_unit_price"`
	EffectivePrice         float64 `json:"effective_price"`
	LeadTimeDays           int     `json:"lead_time_days"`
	PaymentTermsDays       int     `json:"payment_terms_days"`
	WarrantyMonths         int     `json:"warranty_months"`
	DiscountPct            float64 `json:"discount_pct"`
	PriceScore             float64 `json:"price_score"`       // Weight 40%
	LeadTimeScore          float64 `json:"lead_time_score"`   // Weight 25%
	TermsScore             float64 `json:"terms_score"`       // Weight 20%
	WarrantyScore          float64 `json:"warranty_score"`    // Weight 15%
	CompositeScore         float64 `json:"composite_score"`   // 0 - 100
	Rank                   int     `json:"rank"`
	AIRationale            string  `json:"ai_rationale"`
	CounterOfferSuggestion string  `json:"counter_offer_suggestion"`
}

type BidEvaluationScorecard struct {
	TenderID                string         `json:"tender_id"`
	Title                   string         `json:"title"`
	ProductName             string         `json:"product_name"`
	TargetQuantity          float64        `json:"target_quantity"`
	BudgetUnitPrice         float64        `json:"budget_unit_price"`
	TotalBids               int            `json:"total_bids"`
	Bids                    []EvaluatedBid `json:"bids"`
	WinningBid              *EvaluatedBid  `json:"winning_bid,omitempty"`
	TotalEstimatedSavings   float64        `json:"total_estimated_savings"`
	SavingsPct              float64        `json:"savings_pct"`
	StrategicRecommendation string         `json:"strategic_recommendation"`
	EvaluatedAt             string         `json:"evaluated_at"`
}

type AwardTenderRequest struct {
	VendorID uint `json:"vendor_id"`
}

// In-Memory Thread-Safe Store for Bidding
var biddingStore = struct {
	sync.RWMutex
	tenders map[string]*RFQBidTender
	bids    map[string][]VendorBidSubmission
}{
	tenders: make(map[string]*RFQBidTender),
	bids:    make(map[string][]VendorBidSubmission),
}

func generateSecureToken() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// CreateBiddingTender initializes a multi-vendor RFQ bidding tender
func (c *Client) CreateBiddingTender(db *gorm.DB, req *CreateTenderRequest, companyID uint) (*RFQBidTender, error) {
	biddingStore.Lock()
	defer biddingStore.Unlock()

	var prod product.Product
	if err := db.Where("id = ?", req.ProductID).First(&prod).Error; err != nil {
		return nil, fmt.Errorf("product ID %d not found: %w", req.ProductID, err)
	}

	budget := req.BudgetUnitPrice
	if budget <= 0 {
		budget = prod.StandardPrice
	}

	tenderID := fmt.Sprintf("TND-%s-%04d", time.Now().Format("20060102"), len(biddingStore.tenders)+1)

	var invites []VendorInvite
	for _, vid := range req.VendorIDs {
		var p base.Partner
		vName := fmt.Sprintf("Vendor #%d", vid)
		if err := db.Where("id = ?", vid).First(&p).Error; err == nil && p.Name != "" {
			vName = p.Name
		}
		tok := generateSecureToken()
		invites = append(invites, VendorInvite{
			VendorID:   vid,
			VendorName: vName,
			Token:      tok,
			BidURL:     fmt.Sprintf("/portal/bids/%s?token=%s", tenderID, tok),
			Status:     "invited",
		})
	}

	deadline := req.Deadline
	if deadline == "" {
		deadline = time.Now().AddDate(0, 0, 7).Format("2006-01-02 18:00:00")
	}

	tender := &RFQBidTender{
		TenderID:        tenderID,
		Title:           req.Title,
		ProductID:       req.ProductID,
		ProductName:     prod.Name,
		ProductSKU:      prod.DefaultCode,
		TargetQuantity:  req.TargetQuantity,
		BudgetUnitPrice: budget,
		Status:          "open",
		Deadline:        deadline,
		CompanyID:       companyID,
		InvitedVendors:  invites,
		CreatedAt:       time.Now().Format("2006-01-02 15:04:05"),
	}

	biddingStore.tenders[tenderID] = tender
	return tender, nil
}

// SubmitVendorBid accepts an intake bid from a vendor
func (c *Client) SubmitVendorBid(db *gorm.DB, req *VendorBidSubmission, companyID uint) (*EvaluatedBid, error) {
	biddingStore.Lock()
	defer biddingStore.Unlock()

	tender, exists := biddingStore.tenders[req.TenderID]
	if !exists {
		return nil, fmt.Errorf("tender %s not found", req.TenderID)
	}

	// Resolve vendor name
	vName := req.VendorName
	if vName == "" {
		var p base.Partner
		if err := db.Where("id = ?", req.VendorID).First(&p).Error; err == nil && p.Name != "" {
			vName = p.Name
		} else {
			vName = fmt.Sprintf("Vendor #%d", req.VendorID)
		}
	}
	req.VendorName = vName
	req.SubmittedAt = time.Now().Format("2006-01-02 15:04:05")

	// Mark invite as submitted if found
	for i := range tender.InvitedVendors {
		if tender.InvitedVendors[i].VendorID == req.VendorID {
			tender.InvitedVendors[i].Status = "submitted"
			break
		}
	}

	// Add or replace bid
	existingList := biddingStore.bids[req.TenderID]
	replaced := false
	for i, b := range existingList {
		if b.VendorID == req.VendorID {
			existingList[i] = *req
			replaced = true
			break
		}
	}
	if !replaced {
		existingList = append(existingList, *req)
	}
	biddingStore.bids[req.TenderID] = existingList

	effPrice := req.OfferedUnitPrice * (1.0 - (req.DiscountPct / 100.0))

	return &EvaluatedBid{
		VendorID:         req.VendorID,
		VendorName:       vName,
		OfferedUnitPrice: req.OfferedUnitPrice,
		EffectivePrice:   effPrice,
		LeadTimeDays:     req.LeadTimeDays,
		PaymentTermsDays: req.PaymentTermsDays,
		WarrantyMonths:   req.WarrantyMonths,
		DiscountPct:      req.DiscountPct,
	}, nil
}

// EvaluateBiddingTender runs Multi-Attribute Utility Theory (MAUT) AI scoring
func (c *Client) EvaluateBiddingTender(db *gorm.DB, tenderID string, companyID uint) (*BidEvaluationScorecard, error) {
	biddingStore.Lock()
	defer biddingStore.Unlock()

	tender, exists := biddingStore.tenders[tenderID]
	if !exists {
		return nil, fmt.Errorf("tender %s not found", tenderID)
	}

	rawBids := biddingStore.bids[tenderID]
	if len(rawBids) == 0 {
		return &BidEvaluationScorecard{
			TenderID:                tenderID,
			Title:                   tender.Title,
			ProductName:             tender.ProductName,
			TargetQuantity:          tender.TargetQuantity,
			BudgetUnitPrice:         tender.BudgetUnitPrice,
			TotalBids:               0,
			StrategicRecommendation: "No vendor bids submitted yet. Tender remains open.",
			EvaluatedAt:             time.Now().Format("2006-01-02 15:04:05"),
		}, nil
	}

	// Find min/max ranges for normalization
	minPrice := math.MaxFloat64
	maxPrice := 0.0
	minLead := 999999
	maxLead := 0
	maxTerms := 0
	maxWarranty := 0

	type bidPrep struct {
		b        VendorBidSubmission
		effPrice float64
	}
	preps := make([]bidPrep, len(rawBids))
	for i, b := range rawBids {
		eff := b.OfferedUnitPrice * (1.0 - (b.DiscountPct / 100.0))
		if eff < minPrice {
			minPrice = eff
		}
		if eff > maxPrice {
			maxPrice = eff
		}
		if b.LeadTimeDays < minLead {
			minLead = b.LeadTimeDays
		}
		if b.LeadTimeDays > maxLead {
			maxLead = b.LeadTimeDays
		}
		if b.PaymentTermsDays > maxTerms {
			maxTerms = b.PaymentTermsDays
		}
		if b.WarrantyMonths > maxWarranty {
			maxWarranty = b.WarrantyMonths
		}
		preps[i] = bidPrep{b: b, effPrice: eff}
	}

	var evaluated []EvaluatedBid
	for _, p := range preps {
		// Price score: Lower is better (40% weight)
		priceScore := 100.0
		if maxPrice > minPrice {
			priceScore = 100.0 * (1.0 - ((p.effPrice - minPrice) / (maxPrice - minPrice)))
		}

		// Lead time: Shorter is better (25% weight)
		leadScore := 100.0
		if maxLead > minLead {
			leadScore = 100.0 * (1.0 - float64(p.b.LeadTimeDays-minLead)/float64(maxLead-minLead))
		}

		// Payment terms: Longer is better for buyer cashflow (20% weight)
		termsScore := 50.0
		if maxTerms > 0 {
			termsScore = math.Min(100.0, (float64(p.b.PaymentTermsDays)/float64(maxTerms))*100.0)
		}

		// Warranty: Longer is better (15% weight)
		warrantyScore := 50.0
		if maxWarranty > 0 {
			warrantyScore = math.Min(100.0, (float64(p.b.WarrantyMonths)/float64(maxWarranty))*100.0)
		}

		compScore := (priceScore * 0.40) + (leadScore * 0.25) + (termsScore * 0.20) + (warrantyScore * 0.15)

		rationale := fmt.Sprintf("Effective unit price IDR %.2f with %d days lead time and Net %d payment terms.",
			p.effPrice, p.b.LeadTimeDays, p.b.PaymentTermsDays)

		counterSuggestion := ""
		if compScore < 85.0 {
			counterSuggestion = fmt.Sprintf("Propose counter-offer at IDR %.2f (-5%%) in exchange for Net 30 payment terms.", p.effPrice*0.95)
		} else {
			counterSuggestion = "Strong bid terms; recommended for immediate direct award."
		}

		evaluated = append(evaluated, EvaluatedBid{
			VendorID:               p.b.VendorID,
			VendorName:             p.b.VendorName,
			OfferedUnitPrice:       p.b.OfferedUnitPrice,
			EffectivePrice:         p.effPrice,
			LeadTimeDays:           p.b.LeadTimeDays,
			PaymentTermsDays:       p.b.PaymentTermsDays,
			WarrantyMonths:         p.b.WarrantyMonths,
			DiscountPct:            p.b.DiscountPct,
			PriceScore:             math.Round(priceScore*10) / 10,
			LeadTimeScore:          math.Round(leadScore*10) / 10,
			TermsScore:             math.Round(termsScore*10) / 10,
			WarrantyScore:          math.Round(warrantyScore*10) / 10,
			CompositeScore:         math.Round(compScore*10) / 10,
			AIRationale:            rationale,
			CounterOfferSuggestion: counterSuggestion,
		})
	}

	// Sort by CompositeScore descending
	sort.Slice(evaluated, func(i, j int) bool {
		return evaluated[i].CompositeScore > evaluated[j].CompositeScore
	})

	for i := range evaluated {
		evaluated[i].Rank = i + 1
	}

	winner := evaluated[0]
	budgetTotal := tender.BudgetUnitPrice * tender.TargetQuantity
	winningTotal := winner.EffectivePrice * tender.TargetQuantity
	savings := budgetTotal - winningTotal
	savingsPct := 0.0
	if budgetTotal > 0 && savings > 0 {
		savingsPct = math.Round((savings/budgetTotal)*10000) / 100
	}

	strat := fmt.Sprintf("Autonomous Recommendation: Award Tender to %s (Composite Score: %.1f/100, Effective Price: IDR %.2f). Expected procurement savings: IDR %.2f (%.1f%% below baseline budget).",
		winner.VendorName, winner.CompositeScore, winner.EffectivePrice, savings, savingsPct)

	tender.Status = "evaluating"

	return &BidEvaluationScorecard{
		TenderID:                tenderID,
		Title:                   tender.Title,
		ProductName:             tender.ProductName,
		TargetQuantity:          tender.TargetQuantity,
		BudgetUnitPrice:         tender.BudgetUnitPrice,
		TotalBids:               len(evaluated),
		Bids:                    evaluated,
		WinningBid:              &winner,
		TotalEstimatedSavings:   math.Round(savings*100) / 100,
		SavingsPct:              savingsPct,
		StrategicRecommendation: strat,
		EvaluatedAt:             time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// AwardBiddingTender awards the tender and generates a Draft Purchase Order
func (c *Client) AwardBiddingTender(db *gorm.DB, tenderID string, vendorID uint, companyID uint) (*purchase.PurchaseOrder, error) {
	biddingStore.Lock()
	defer biddingStore.Unlock()

	tender, exists := biddingStore.tenders[tenderID]
	if !exists {
		return nil, fmt.Errorf("tender %s not found", tenderID)
	}

	bids := biddingStore.bids[tenderID]
	var winningBid *VendorBidSubmission
	for _, b := range bids {
		if b.VendorID == vendorID {
			winningBid = &b
			break
		}
	}
	if winningBid == nil && len(bids) > 0 {
		winningBid = &bids[0]
		vendorID = winningBid.VendorID
	}
	if winningBid == nil {
		return nil, fmt.Errorf("no valid bids available to award for tender %s", tenderID)
	}

	effPrice := winningBid.OfferedUnitPrice * (1.0 - (winningBid.DiscountPct / 100.0))
	untaxed := effPrice * tender.TargetQuantity
	tax := untaxed * 0.11 // PPN 11%
	total := untaxed + tax

	poNumber := fmt.Sprintf("PO/AWARD/%s/%04d", time.Now().Format("200601"), time.Now().Unix()%10000)
	po := purchase.PurchaseOrder{
		Name:          poNumber,
		PartnerID:     vendorID,
		DateOrder:     time.Now().Format("2006-01-02 15:04:05"),
		State:         "draft",
		AmountUntaxed: untaxed,
		AmountTax:     tax,
		AmountTotal:   total,
		TaxRate:       11.0,
		Note:          fmt.Sprintf("Auto-generated from AI Autonomous RFQ Bidding Tender %s. Awarded Vendor: %s, Effective Price: IDR %.2f, Lead Time: %d days.", tenderID, winningBid.VendorName, effPrice, winningBid.LeadTimeDays),
	}
	po.CompanyID = companyID

	if err := db.Create(&po).Error; err != nil {
		return nil, fmt.Errorf("failed to create awarded purchase order: %w", err)
	}

	line := purchase.PurchaseOrderLine{
		OrderID:       po.ID,
		ProductID:     tender.ProductID,
		Name:          fmt.Sprintf("%s (Tender Award: %s)", tender.ProductName, tenderID),
		ProductUomQty: tender.TargetQuantity,
		PriceUnit:     effPrice,
		PriceSubtotal: untaxed,
	}
	line.CompanyID = companyID

	if err := db.Create(&line).Error; err != nil {
		return nil, fmt.Errorf("failed to create awarded purchase order line: %w", err)
	}

	tender.Status = "awarded"
	tender.AwardedVendorID = &vendorID
	tender.AwardedPOName = po.Name

	po.OrderLines = []purchase.PurchaseOrderLine{line}
	return &po, nil
}
