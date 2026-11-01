package landed_cost

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
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"

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

// CreateLandedCost initializes a new landed cost calculation document
func (s *Service) CreateLandedCost(req CreateLandedCostRequest, companyID, userID uint) (*StockLandedCost, error) {
	if req.PickingID == 0 {
		return nil, errors.New("dokumen penerimaan barang (picking_id) wajib dipilih")
	}

	picking, err := s.repo.GetIncomingPicking(req.PickingID, companyID)
	if err != nil {
		return nil, fmt.Errorf("penerimaan barang (picking) tidak ditemukan: %w", err)
	}

	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	// Generate sequence LC/2026/0001
	year := time.Now().Format("2006")
	var count int64
	s.db.Model(&StockLandedCost{}).Where("company_id = ?", companyID).Count(&count)
	name := fmt.Sprintf("LC/%s/%04d", year, count+1)

	// Ensure clearing account default
	defaultClearingAccID := s.GetOrCreateClearingAccountID(companyID)

	var totalAmount float64
	var costLines []StockLandedCostLine
	for _, l := range req.CostLines {
		if l.Name == "" || l.PriceUnit <= 0 {
			continue
		}
		split := l.SplitMethod
		if split == "" {
			split = SplitByValue
		}
		accID := l.AccountID
		if accID == 0 {
			accID = defaultClearingAccID
		}
		totalAmount += l.PriceUnit
		costLines = append(costLines, StockLandedCostLine{
			Name:        l.Name,
			SplitMethod: split,
			PriceUnit:   l.PriceUnit,
			AccountID:   accID,
		})
	}

	cost := &StockLandedCost{
		Name:             name,
		Date:             dateStr,
		PickingID:        picking.ID,
		State:            StateDraft,
		CompanyID:        companyID,
		Description:      req.Description,
		TotalAmount:      totalAmount,
		CostLines:        costLines,
		AccountJournalID: 5, // MISC journal
	}

	if err := s.repo.Create(cost); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := fmt.Sprintf("Draft Landed Cost <b>%s</b> dibuat untuk penerimaan barang <b>%s</b> dengan total biaya Rp %.2f.",
			cost.Name, picking.Name, cost.TotalAmount)
		_ = s.chatterSvc.LogTracking("stock.landed.cost", cost.ID, userID, "Landed Cost Engine", msg, "create", companyID)
	}

	return s.repo.GetByID(cost.ID, companyID)
}

// GetLandedCost retrieves a landed cost document by ID
func (s *Service) GetLandedCost(id, companyID uint) (*StockLandedCost, error) {
	return s.repo.GetByID(id, companyID)
}

// ListLandedCosts retrieves list of landed costs
func (s *Service) ListLandedCosts(companyID uint, state string) ([]StockLandedCost, error) {
	return s.repo.List(companyID, state)
}

// ComputeValuationLines simulates and allocates landed costs across all stock moves
func (s *Service) ComputeValuationLines(id, companyID, userID uint) (*StockLandedCost, error) {
	cost, err := s.repo.GetByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen landed cost tidak ditemukan")
	}

	if cost.State != StateDraft && cost.State != StateComputed {
		return nil, fmt.Errorf("hanya dokumen bertatus draft atau computed yang dapat dihitung ulang (status saat ini: %s)", cost.State)
	}

	if len(cost.CostLines) == 0 {
		return nil, errors.New("tidak ada rincian biaya tambahan (cost lines) untuk dialokasikan")
	}

	picking, err := s.repo.GetIncomingPicking(cost.PickingID, companyID)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat incoming picking: %w", err)
	}

	if len(picking.MoveLines) == 0 {
		return nil, errors.New("tidak ada item barang (stock move) pada penerimaan barang ini")
	}

	var valuationLines []StockValuationAdjustmentLine

	// Recompute total amount
	var totalCostAmount float64
	for _, cl := range cost.CostLines {
		totalCostAmount += cl.PriceUnit
	}
	cost.TotalAmount = totalCostAmount

	for _, costLine := range cost.CostLines {
		if costLine.PriceUnit <= 0 {
			continue
		}

		// Calculate total base weights across all moves for this split method
		var totalWeight float64
		type MoveWeight struct {
			Move   stock.StockMove
			Qty    float64
			Weight float64
			Volume float64
			Value  float64
			Metric float64
		}

		var moveWeights []MoveWeight
		for _, move := range picking.MoveLines {
			qty := move.QuantityDone
			if qty <= 0 {
				qty = move.ProductUomQty
			}
			if qty <= 0 {
				qty = 1.0
			}

			// Product details
			prodWeight := move.Product.Weight
			if prodWeight <= 0 {
				prodWeight = 1.0 // fallback 1kg per unit
			}
			prodVolume := move.Product.Volume
			if prodVolume <= 0 {
				prodVolume = 0.005 // fallback 0.005 m3 per unit
			}
			unitPrice := move.Product.StandardPrice
			if unitPrice <= 0 {
				unitPrice = 1000.0 // fallback
			}

			itemWeight := qty * prodWeight
			itemVolume := qty * prodVolume
			itemValue := qty * unitPrice

			var metric float64
			switch costLine.SplitMethod {
			case SplitByQuantity:
				metric = qty
			case SplitByValue:
				metric = itemValue
			case SplitByWeight:
				metric = itemWeight
			case SplitByVolume:
				metric = itemVolume
			case SplitEqual:
				metric = 1.0
			default:
				metric = itemValue
			}

			totalWeight += metric
			moveWeights = append(moveWeights, MoveWeight{
				Move:   move,
				Qty:    qty,
				Weight: itemWeight,
				Volume: itemVolume,
				Value:  itemValue,
				Metric: metric,
			})
		}

		if totalWeight <= 0 {
			totalWeight = float64(len(moveWeights))
			for i := range moveWeights {
				moveWeights[i].Metric = 1.0
			}
		}

		// Allocate proportionally
		var allocatedSoFar float64
		for i, mw := range moveWeights {
			var allocated float64
			if i == len(moveWeights)-1 {
				// Last item absorbs penny rounding differences
				allocated = math.Round((costLine.PriceUnit-allocatedSoFar)*100) / 100
			} else {
				ratio := mw.Metric / totalWeight
				allocated = math.Round((costLine.PriceUnit*ratio)*100) / 100
				allocatedSoFar += allocated
			}

			formerCost := mw.Value
			formerPerUnit := formerCost / mw.Qty
			finalCost := formerCost + allocated
			finalPerUnit := finalCost / mw.Qty

			valuationLines = append(valuationLines, StockValuationAdjustmentLine{
				CostID:               cost.ID,
				CostLineID:           costLine.ID,
				CostLineName:         costLine.Name,
				MoveID:               mw.Move.ID,
				ProductID:            mw.Move.ProductID,
				ProductName:          mw.Move.ProductName,
				Quantity:             mw.Qty,
				Weight:               mw.Weight,
				Volume:               mw.Volume,
				FormerCost:           formerCost,
				FormerCostPerUnit:    formerPerUnit,
				AdditionalLandedCost: allocated,
				FinalCost:            finalCost,
				FinalCostPerUnit:     finalPerUnit,
			})
		}
	}

	if err := s.repo.SaveValuationLines(cost.ID, valuationLines); err != nil {
		return nil, fmt.Errorf("gagal menyimpan hasil alokasi penilaian stok: %w", err)
	}

	cost.State = StateComputed
	if err := s.repo.Update(cost); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := fmt.Sprintf("Kalkulasi alokasi Landed Cost <b>%s</b> berhasil disimulasikan (%d baris penyesuaian nilai persediaan).",
			cost.Name, len(valuationLines))
		_ = s.chatterSvc.LogTracking("stock.landed.cost", cost.ID, userID, "Landed Cost Engine", msg, "compute", companyID)
	}

	return s.repo.GetByID(cost.ID, companyID)
}

// ActionValidate posts journal entries and updates product valuation/standard cost
func (s *Service) ActionValidate(id, companyID, userID uint) (*StockLandedCost, error) {
	cost, err := s.repo.GetByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen landed cost tidak ditemukan")
	}

	if cost.State == StateDone {
		return nil, errors.New("dokumen landed cost ini telah divalidasi sebelumnya")
	}

	if cost.State == StateCancel {
		return nil, errors.New("dokumen landed cost yang dibatalkan tidak dapat divalidasi")
	}

	// If in draft or no valuation lines, compute first
	if cost.State == StateDraft || len(cost.ValuationLines) == 0 {
		cost, err = s.ComputeValuationLines(id, companyID, userID)
		if err != nil {
			return nil, err
		}
	}

	// 1. Post General Ledger Entry (Double-entry journal move)
	// Debit: Persediaan Barang Dagang (Stock Valuation Account: 130000)
	// Credit: Akun Antara Biaya Impor (Clearing Account: 130090 or costLine.AccountID)
	stockValAccountID := s.GetOrCreateStockValuationAccountID(companyID)

	// Consolidate debit lines per product/move
	productAllocations := make(map[uint]float64)
	productQty := make(map[uint]float64)
	productName := make(map[uint]string)
	productFormer := make(map[uint]float64)

	for _, vl := range cost.ValuationLines {
		productAllocations[vl.ProductID] += vl.AdditionalLandedCost
		productQty[vl.ProductID] += vl.Quantity
		productName[vl.ProductID] = vl.ProductName
		productFormer[vl.ProductID] += vl.FormerCost
	}

	var moveLines []account.AccountMoveLine
	var totalDebit, totalCredit float64

	// Debit lines: Stock Valuation for each product
	for prodID, addCost := range productAllocations {
		if addCost <= 0 {
			continue
		}
		pID := prodID
		line := account.AccountMoveLine{
			AccountID: stockValAccountID,
			Name:      fmt.Sprintf("Penambahan HPP Impor: %s (%s)", productName[prodID], cost.Name),
			Debit:     addCost,
			Credit:    0,
			ProductID: &pID,
		}
		moveLines = append(moveLines, line)
		totalDebit += addCost
	}

	// Credit lines: Landed Cost Clearing for each cost line
	for _, cl := range cost.CostLines {
		if cl.PriceUnit <= 0 {
			continue
		}
		line := account.AccountMoveLine{
			AccountID: cl.AccountID,
			Name:      fmt.Sprintf("Kliring Biaya Tambahan: %s (%s)", cl.Name, cost.Name),
			Debit:     0,
			Credit:    cl.PriceUnit,
		}
		moveLines = append(moveLines, line)
		totalCredit += cl.PriceUnit
	}

	// Check balance precision
	diff := math.Abs(totalDebit - totalCredit)
	if diff > 0.01 {
		return nil, fmt.Errorf("jurnal akuntansi landed cost tidak seimbang: Total Debit (Rp %.2f) != Total Credit (Rp %.2f)", totalDebit, totalCredit)
	}

	// Find default MISC journal
	var journal account.AccountJournal
	if err := s.db.Where("code = ? AND company_id = ?", "MISC", companyID).First(&journal).Error; err != nil {
		if err := s.db.Where("company_id = ?", companyID).First(&journal).Error; err != nil {
			journal.ID = 1
		}
	}

	// Find system partner (e.g. Internal / Company)
	var partner base.Partner
	if err := s.db.Where("company_id = ?", companyID).First(&partner).Error; err != nil {
		partner.ID = 1
	}

	entryMove := &account.AccountMove{
		MoveType:       "entry",
		JournalID:      journal.ID,
		PartnerID:      partner.ID,
		InvoiceDate:    cost.Date,
		InvoiceDueDate: cost.Date,
		Ref:            fmt.Sprintf("Landed Cost %s (%s)", cost.Name, cost.Picking.Name),
		Note:           fmt.Sprintf("Penyesuaian nilai persediaan atas penerimaan barang %s", cost.Picking.Name),
		LineIDs:        moveLines,
	}
	entryMove.CompanyID = companyID

	if err := s.accountSvc.CreateMove(entryMove); err != nil {
		return nil, fmt.Errorf("gagal membuat jurnal akun landed cost: %w", err)
	}

	postedMove, err := s.accountSvc.ActionPost(entryMove.ID, companyID)
	if err != nil {
		return nil, fmt.Errorf("gagal posting jurnal akun landed cost: %w", err)
	}

	// 2. Update Product Standard Cost (Weighted Average Cost Price)
	for prodID, addCost := range productAllocations {
		qty := productQty[prodID]
		if qty > 0 {
			formerCost := productFormer[prodID]
			newTotalCost := formerCost + addCost
			newUnitCost := math.Round((newTotalCost/qty)*100) / 100

			// Update product standard price
			_ = s.db.Model(&product.Product{}).
				Where("id = ? AND company_id = ?", prodID, companyID).
				Update("standard_price", newUnitCost).Error
		}
	}

	// 3. Mark Landed Cost as Done
	cost.State = StateDone
	cost.AccountMoveID = &postedMove.ID
	if err := s.repo.Update(cost); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := fmt.Sprintf("Landed Cost <b>%s</b> berhasil divalidasi. Jurnal GL <b>%s</b> telah dibukukan dan harga pokok persediaan telah diperbarui.",
			cost.Name, postedMove.Name)
		_ = s.chatterSvc.LogTracking("stock.landed.cost", cost.ID, userID, "Landed Cost Engine", msg, "validate", companyID)
	}

	return s.repo.GetByID(cost.ID, companyID)
}

// ActionCancel cancels landed cost document
func (s *Service) ActionCancel(id, companyID, userID uint) (*StockLandedCost, error) {
	cost, err := s.repo.GetByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen landed cost tidak ditemukan")
	}

	if cost.State == StateDone {
		return nil, errors.New("dokumen landed cost yang sudah divalidasi (done) tidak dapat dibatalkan langsung. Silakan buat entri penyesuaian pembalik.")
	}

	cost.State = StateCancel
	if err := s.repo.Update(cost); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := fmt.Sprintf("Dokumen Landed Cost <b>%s</b> dibatalkan.", cost.Name)
		_ = s.chatterSvc.LogTracking("stock.landed.cost", cost.ID, userID, "Landed Cost Engine", msg, "cancel", companyID)
	}

	return s.repo.GetByID(cost.ID, companyID)
}

// GetStats returns executive metrics on landed costs
func (s *Service) GetStats(companyID uint) (*LandedCostStats, error) {
	return s.repo.GetStats(companyID)
}

// AILandedCostCopilot performs supply chain profitability audit and margin stress-testing
func (s *Service) AILandedCostCopilot(id, companyID uint) (*AILandedCostReport, error) {
	cost, err := s.repo.GetByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen landed cost tidak ditemukan")
	}

	if len(cost.ValuationLines) == 0 {
		return nil, errors.New("dokumen belum dihitung. Jalankan 'Hitung Alokasi' terlebih dahulu sebelum analisis AI")
	}

	// Consolidate product-level figures
	type ProdSummary struct {
		ID          uint
		Name        string
		Qty         float64
		FormerCost  float64
		AddCost     float64
		FinalCost   float64
		SellPrice   float64
	}

	prodMap := make(map[uint]*ProdSummary)
	var totalFormerCost, totalAddCost float64

	for _, vl := range cost.ValuationLines {
		ps, exists := prodMap[vl.ProductID]
		if !exists {
			var p product.Product
			_ = s.db.Where("id = ?", vl.ProductID).First(&p)
			ps = &ProdSummary{
				ID:        vl.ProductID,
				Name:      vl.ProductName,
				Qty:       vl.Quantity,
				SellPrice: p.ListPrice,
			}
			prodMap[vl.ProductID] = ps
		}
		ps.FormerCost += vl.FormerCost
		ps.AddCost += vl.AdditionalLandedCost
		ps.FinalCost += vl.FinalCost

		totalFormerCost += vl.FormerCost
		totalAddCost += vl.AdditionalLandedCost
	}

	overallOverheadPct := 0.0
	if totalFormerCost > 0 {
		overallOverheadPct = (totalAddCost / totalFormerCost) * 100.0
	}

	var productImpacts []AIProductImpact
	var tightMargins, negativeMargins int

	for _, ps := range prodMap {
		oldUnit := 0.0
		newUnit := 0.0
		if ps.Qty > 0 {
			oldUnit = ps.FormerCost / ps.Qty
			newUnit = ps.FinalCost / ps.Qty
		}

		costIncPct := 0.0
		if oldUnit > 0 {
			costIncPct = ((newUnit - oldUnit) / oldUnit) * 100.0
		}

		grossMarginPct := 0.0
		status := "HEALTHY"
		if ps.SellPrice > 0 {
			grossMarginPct = ((ps.SellPrice - newUnit) / ps.SellPrice) * 100.0
			if grossMarginPct < 0 {
				status = "NEGATIVE"
				negativeMargins++
			} else if grossMarginPct < 20 {
				status = "TIGHT"
				tightMargins++
			}
		}

		productImpacts = append(productImpacts, AIProductImpact{
			ProductID:               ps.ID,
			ProductName:             ps.Name,
			Quantity:                ps.Qty,
			OldCostPerUnit:          math.Round(oldUnit*100) / 100,
			NewCostPerUnit:          math.Round(newUnit*100) / 100,
			CostIncreasePct:         math.Round(costIncPct*10) / 10,
			SellingPrice:            ps.SellPrice,
			EstimatedGrossMarginPct: math.Round(grossMarginPct*10) / 10,
			MarginStatus:            status,
		})
	}

	riskLevel := "LOW"
	if negativeMargins > 0 || overallOverheadPct > 35.0 {
		riskLevel = "HIGH"
	} else if tightMargins > 0 || overallOverheadPct > 20.0 {
		riskLevel = "MEDIUM"
	}

	// Base heuristics recommendations
	var recommendations []string
	if negativeMargins > 0 {
		recommendations = append(recommendations, fmt.Sprintf("⚠️ PERINGATAN: Terdapat %d produk dengan Margin Kotor NEGATIF setelah memperhitungkan landed cost. Segera tinjau ulang harga jual (List Price) atau negosiasikan ulang ongkos angkut.", negativeMargins))
	}
	if overallOverheadPct > 25.0 {
		recommendations = append(recommendations, fmt.Sprintf("Biaya impor & logistik mencapai %.1f%% dari nilai barang. Pertimbangkan konsolidasi pengiriman (FCL/Full Container Load) untuk menekan unit shipping cost.", overallOverheadPct))
	} else {
		recommendations = append(recommendations, fmt.Sprintf("Rasio biaya logistik sebesar %.1f%% masih dalam batas wajar industri (<20-25%%).", overallOverheadPct))
	}
	recommendations = append(recommendations, "Gunakan metode alokasi 'by_volume' jika komponen biaya didominasi oleh Sea Freight, atau 'by_weight' untuk Air Freight.")
	recommendations = append(recommendations, "Pastikan akun kliring landed cost (130090) dicocokkan (reconciled) dengan Faktur Pajak Bea Masuk dan Vendor Bill Ekspedisi.")

	report := &AILandedCostReport{
		Summary: fmt.Sprintf("Analisis Landed Cost %s: Total biaya pengadaan tambahan sebesar Rp %.2f dialokasikan ke %d produk. Rasio beban logistik terhadap harga beli adalah %.1f%% dengan profil risiko %s.",
			cost.Name, totalAddCost, len(productImpacts), overallOverheadPct, riskLevel),
		TotalAdditionalCost: totalAddCost,
		OverallOverheadPct:   math.Round(overallOverheadPct*10) / 10,
		ProductImpacts:      productImpacts,
		Recommendations:     recommendations,
		RiskLevel:           riskLevel,
		AIProvider:          "Local Heuristic Supply Chain Engine",
	}

	// Try Cloudflare Workers AI Edge Inference if configured
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		prompt := fmt.Sprintf(`Analisis dokumen Landed Cost impor berikut:
Dokumen: %s, Total Biaya Tambahan: Rp %.2f, Rasio Beban Pengadaan: %.1f%%, Jumlah Produk: %d.
Dampak Produk: %+v.
Berikan JSON response dengan schema:
{
  "summary": "penjelasan eksekutif dalam bahasa indonesia",
  "risk_level": "LOW" | "MEDIUM" | "HIGH",
  "recommendations": ["rekomendasi 1", "rekomendasi 2", "rekomendasi 3"]
}`, cost.Name, totalAddCost, overallOverheadPct, len(productImpacts), productImpacts)

		systemPrompt := "You are a professional CFO and senior international supply chain & landed cost pricing auditor. Return ONLY a valid JSON object."
		if aiResp, err := s.aiClient.CallRaw(prompt, systemPrompt); err == nil && aiResp != "" {
			var parsed struct {
				Summary         string   `json:"summary"`
				RiskLevel       string   `json:"risk_level"`
				Recommendations []string `json:"recommendations"`
			}
			jsonStr := strings.TrimSpace(aiResp)
			if idx := strings.Index(jsonStr, "{"); idx != -1 {
				if endIdx := strings.LastIndex(jsonStr, "}"); endIdx != -1 {
					jsonStr = jsonStr[idx : endIdx+1]
				}
			}
			if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
				if parsed.Summary != "" {
					report.Summary = parsed.Summary
				}
				if parsed.RiskLevel != "" {
					report.RiskLevel = parsed.RiskLevel
				}
				if len(parsed.Recommendations) > 0 {
					report.Recommendations = parsed.Recommendations
				}
				report.AIProvider = "Cloudflare Workers AI (@cf/meta/llama-3.3-70b-instruct)"
			}
		}
	}

	return report, nil
}

// Helper: Ensure 130090 Landed Cost Clearing account exists
func (s *Service) GetOrCreateClearingAccountID(companyID uint) uint {
	var acc account.AccountAccount
	if err := s.db.Where("code = ? AND company_id = ?", "130090", companyID).First(&acc).Error; err == nil {
		return acc.ID
	}
	// Create it if missing
	newAcc := account.AccountAccount{
		Code: "130090",
		Name: "Akun Antara Biaya Impor & Logistik (Landed Cost Clearing)",
		Type: "liability_payable",
	}
	newAcc.CompanyID = companyID
	if err := s.db.Create(&newAcc).Error; err == nil {
		return newAcc.ID
	}
	// Fallback to accounts payable or cash
	var fallback account.AccountAccount
	if err := s.db.Where("company_id = ?", companyID).First(&fallback).Error; err == nil {
		return fallback.ID
	}
	return 1
}

// Helper: Ensure 130000 Stock Valuation account exists
func (s *Service) GetOrCreateStockValuationAccountID(companyID uint) uint {
	var acc account.AccountAccount
	if err := s.db.Where("code = ? AND company_id = ?", "130000", companyID).First(&acc).Error; err == nil {
		return acc.ID
	}
	// Fallback to first asset account or any account
	var fallback account.AccountAccount
	if err := s.db.Where("company_id = ?", companyID).First(&fallback).Error; err == nil {
		return fallback.ID
	}
	return 4
}
