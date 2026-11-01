package stock_inventory

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/stock"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(db *gorm.DB, repo *Repository, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// ---------------------------------------------------------------------------
// Inventory Opname Lifecycle
// ---------------------------------------------------------------------------

func (s *Service) ListInventories(companyID uint, state string, warehouseID uint, locationID uint, search string) ([]StockInventory, error) {
	return s.repo.ListInventories(companyID, state, warehouseID, locationID, search)
}

func (s *Service) GetInventory(id, companyID uint) (*StockInventory, error) {
	return s.repo.GetInventoryByID(id, companyID)
}

func (s *Service) CreateInventory(req *CreateInventoryRequest, companyID, userID uint) (*StockInventory, error) {
	if req.LocationID == 0 {
		return nil, errors.New("lokasi gudang (location_id) wajib dipilih")
	}

	year := time.Now().Format("2006")
	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	name := req.Name
	if name == "" {
		var count int64
		s.db.Model(&StockInventory{}).Where("company_id = ?", companyID).Count(&count)
		name = fmt.Sprintf("OPNAME/%s/%05d", year, count+1)
	}

	inv := &StockInventory{
		Name:        name,
		WarehouseID: req.WarehouseID,
		LocationID:  req.LocationID,
		Date:        dateStr,
		State:       "draft",
		AIRiskLevel: "none",
		Notes:       req.Notes,
	}
	inv.CompanyID = companyID

	if err := s.repo.CreateInventory(inv); err != nil {
		return nil, err
	}

	// Fetch location name for chatter
	locName := fmt.Sprintf("Lokasi ID #%d", inv.LocationID)
	if loc, err := s.repo.GetLocationByID(inv.LocationID, companyID); err == nil {
		locName = loc.Name
	}

	s.logChatter(inv.ID, fmt.Sprintf("Membuat draf Stock Opname fisik '%s' untuk %s.", inv.Name, locName), companyID, userID)

	return s.repo.GetInventoryByID(inv.ID, companyID)
}

// StartInventory snapshots current StockQuants for LocationID into InventoryLines
func (s *Service) StartInventory(id, companyID, userID uint) (*StockInventory, error) {
	inv, err := s.repo.GetInventoryByID(id, companyID)
	if err != nil {
		return nil, errors.New("data stock opname tidak ditemukan")
	}

	if inv.State != "draft" {
		return nil, fmt.Errorf("hanya stock opname berstatus 'draft' yang dapat dimulai (status saat ini: %s)", inv.State)
	}

	// Fetch quants at this location
	quants, err := s.repo.GetQuantsByLocation(inv.LocationID, companyID)
	if err != nil {
		return nil, err
	}

	// If no quants exist yet, load active storable products so inspector has lines to count
	if len(quants) == 0 {
		var products []product.Product
		s.db.Where("company_id = ? AND type = 'product'", companyID).Find(&products)
		for _, p := range products {
			line := StockInventoryLine{
				InventoryID:    inv.ID,
				ProductID:      p.ID,
				TheoreticalQty: 0,
				CountedQty:     0,
				DifferenceQty:  0,
				CostPrice:      p.StandardPrice,
				VarianceValue:  0,
				Reason:         "",
				State:          "draft",
			}
			line.CompanyID = companyID
			_ = s.repo.CreateLine(&line)
		}
	} else {
		for _, q := range quants {
			cost := float64(0)
			if q.Product.ID > 0 {
				cost = q.Product.StandardPrice
			} else {
				var p product.Product
				if err := s.db.Where("id = ?", q.ProductID).First(&p).Error; err == nil {
					cost = p.StandardPrice
				}
			}

			line := StockInventoryLine{
				InventoryID:    inv.ID,
				ProductID:      q.ProductID,
				TheoreticalQty: q.Quantity,
				CountedQty:     q.Quantity, // default counted equals book
				DifferenceQty:  0,
				CostPrice:      cost,
				VarianceValue:  0,
				Reason:         "",
				State:          "draft",
			}
			line.CompanyID = companyID
			_ = s.repo.CreateLine(&line)
		}
	}

	inv.State = "confirm" // In Progress
	s.recalculateInventoryTotals(inv)
	if err := s.repo.UpdateInventory(inv); err != nil {
		return nil, err
	}

	s.logChatter(inv.ID, fmt.Sprintf("Memulai penghitungan fisik lapangan Stock Opname '%s'. Saldo buku berhasil di-snapshot ke formulir inspeksi.", inv.Name), companyID, userID)

	return s.repo.GetInventoryByID(inv.ID, companyID)
}

func (s *Service) UpdateLine(invID, lineID uint, req *UpdateInventoryLineRequest, companyID, userID uint) (*StockInventory, error) {
	inv, err := s.repo.GetInventoryByID(invID, companyID)
	if err != nil {
		return nil, errors.New("data stock opname tidak ditemukan")
	}

	if inv.State != "confirm" && inv.State != "draft" {
		return nil, fmt.Errorf("tidak dapat mengedit baris pada opname berstatus '%s'", inv.State)
	}

	line, err := s.repo.GetLineByID(lineID)
	if err != nil || line.InventoryID != inv.ID {
		return nil, errors.New("baris opname tidak ditemukan")
	}

	line.CountedQty = req.CountedQty
	line.DifferenceQty = line.CountedQty - line.TheoreticalQty
	line.VarianceValue = line.DifferenceQty * line.CostPrice
	line.Reason = req.Reason

	if err := s.repo.UpdateLine(line); err != nil {
		return nil, err
	}

	// Recalculate inventory totals
	s.recalculateInventoryTotals(inv)
	if err := s.repo.UpdateInventory(inv); err != nil {
		return nil, err
	}

	return s.repo.GetInventoryByID(inv.ID, companyID)
}

func (s *Service) AddLine(invID uint, req *AddInventoryLineRequest, companyID, userID uint) (*StockInventory, error) {
	inv, err := s.repo.GetInventoryByID(invID, companyID)
	if err != nil {
		return nil, errors.New("data stock opname tidak ditemukan")
	}

	if inv.State != "confirm" && inv.State != "draft" {
		return nil, fmt.Errorf("tidak dapat menambah item pada opname berstatus '%s'", inv.State)
	}

	var prod product.Product
	if err := s.db.Where("id = ? AND company_id = ?", req.ProductID, companyID).First(&prod).Error; err != nil {
		return nil, errors.New("produk tidak ditemukan")
	}

	// Check if already in lines
	for _, l := range inv.Lines {
		if l.ProductID == req.ProductID {
			return nil, errors.New("produk ini sudah terdaftar dalam daftar opname, silakan perbarui baris yang ada")
		}
	}

	line := StockInventoryLine{
		InventoryID:    inv.ID,
		ProductID:      prod.ID,
		TheoreticalQty: 0,
		CountedQty:     req.CountedQty,
		DifferenceQty:  req.CountedQty,
		CostPrice:      prod.StandardPrice,
		VarianceValue:  req.CountedQty * prod.StandardPrice,
		Reason:         req.Reason,
		State:          "draft",
	}
	line.CompanyID = companyID

	if err := s.repo.CreateLine(&line); err != nil {
		return nil, err
	}

	s.recalculateInventoryTotals(inv)
	if err := s.repo.UpdateInventory(inv); err != nil {
		return nil, err
	}

	return s.repo.GetInventoryByID(inv.ID, companyID)
}

// ---------------------------------------------------------------------------
// Cloudflare Workers AI Discrepancy & Shrinkage Auditor
// ---------------------------------------------------------------------------

func (s *Service) AuditWithAI(id, companyID, userID uint) (*AIAuditResult, error) {
	inv, err := s.repo.GetInventoryByID(id, companyID)
	if err != nil {
		return nil, errors.New("data stock opname tidak ditemukan")
	}

	s.recalculateInventoryTotals(inv)

	// Collect discrepancy lines
	var discLines []StockInventoryLine
	for _, l := range inv.Lines {
		if math.Abs(l.DifferenceQty) > 0.0001 {
			discLines = append(discLines, l)
		}
	}

	// If no discrepancies at all
	if len(discLines) == 0 {
		res := &AIAuditResult{
			InventoryID:        inv.ID,
			InventoryName:      inv.Name,
			RiskLevel:          "low",
			Summary:            "Pemeriksaan fisik menunjukkan kecocokan 100% sempurna dengan saldo buku sistem (Zero Discrepancy). Tidak ditemukan risiko kehilangan atau penyusutan inventaris.",
			DiscrepancyCount:   0,
			TotalLossValue:     0,
			TotalGainValue:     0,
			SuspiciousItems:    []string{},
			RecommendedActions: []string{"Pertahankan SOP penyimpanan dan akurasi pencatatan barang saat ini."},
			EngineSource:       "local_heuristic_engine",
		}
		inv.AIRiskLevel = "low"
		inv.AIAuditSummary = res.Summary
		_ = s.repo.UpdateInventory(inv)
		return res, nil
	}

	// Discrepancies exist: Run AI analysis
	result := s.executeAIAudit(inv, discLines)

	// Persist AI insights to inventory record
	inv.AIRiskLevel = result.RiskLevel
	inv.AIAuditSummary = result.Summary
	_ = s.repo.UpdateInventory(inv)

	s.logChatter(
		inv.ID,
		fmt.Sprintf("✨ Cloudflare Workers AI Audit Selesai: Tingkat Risiko '%s', %d item selisih (Kerugian: Rp %.2f, Surplus: Rp %.2f).",
			strings.ToUpper(result.RiskLevel), result.DiscrepancyCount, result.TotalLossValue, result.TotalGainValue),
		companyID,
		userID,
	)

	return result, nil
}

func (s *Service) executeAIAudit(inv *StockInventory, discLines []StockInventoryLine) *AIAuditResult {
	// Build prompt for Cloudflare Workers AI
	var itemsDesc []string
	for _, l := range discLines {
		pName := fmt.Sprintf("Product ID #%d", l.ProductID)
		if l.Product != nil && l.Product.Name != "" {
			pName = l.Product.Name
		}
		itemsDesc = append(itemsDesc, fmt.Sprintf("- %s: Buku=%.2f, Fisik=%.2f, Selisih=%.2f, Nilai Varians=Rp %.2f (Alasan: %s)",
			pName, l.TheoreticalQty, l.CountedQty, l.DifferenceQty, l.VarianceValue, l.Reason))
	}

	prompt := fmt.Sprintf(`Sebagai AI Warehouse & Inventory Loss Prevention Auditor, analisis hasil Stock Opname berikut:
Nomor Dokumen: %s
Lokasi: ID #%d
Total Kerugian (Loss/Shrinkage): Rp %.2f
Total Surplus (Gain): Rp %.2f
Item dengan Selisih Fisik:
%s

Tugas Anda:
1. Tentukan risk_level ("low", "medium", atau "high").
2. Buat executive summary (2-3 kalimat) mengenai pola kehilangan dan penyebab potensial.
3. Sebutkan suspicious_items (nama barang yang mencurigakan atau nilai selisihnya besar).
4. Berikan 2-3 recommended_actions konkrit untuk manajemen gudang (misal investigasi CCTV, rekonsiliasi PO vendor, perbaikan barcode scanning).

Jawab HANYA dalam format JSON valid tanpa markdown formatting tambahan:
{
  "risk_level": "low|medium|high",
  "summary": "...",
  "suspicious_items": ["..."],
  "recommended_actions": ["..."]
}`, inv.Name, inv.LocationID, inv.TotalLossValue, inv.TotalGainValue, strings.Join(itemsDesc, "\n"))

	type AIResponse struct {
		RiskLevel          string   `json:"risk_level"`
		Summary            string   `json:"summary"`
		SuspiciousItems    []string `json:"suspicious_items"`
		RecommendedActions []string `json:"recommended_actions"`
	}

	// Try Cloudflare Workers AI if configured
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "You are an enterprise warehouse inventory loss prevention auditor. Respond only in valid JSON format."
		respText, err := s.aiClient.CallRaw(prompt, systemPrompt)
		if err == nil && respText != "" {
			cleanJSON := strings.TrimSpace(respText)
			if strings.HasPrefix(cleanJSON, "```") {
				lines := strings.Split(cleanJSON, "\n")
				if len(lines) > 2 {
					cleanJSON = strings.Join(lines[1:len(lines)-1], "\n")
				}
			}
			var parsed AIResponse
			if json.Unmarshal([]byte(cleanJSON), &parsed) == nil && parsed.RiskLevel != "" {
				return &AIAuditResult{
					InventoryID:        inv.ID,
					InventoryName:      inv.Name,
					RiskLevel:          strings.ToLower(parsed.RiskLevel),
					Summary:            parsed.Summary,
					DiscrepancyCount:   len(discLines),
					TotalLossValue:     inv.TotalLossValue,
					TotalGainValue:     inv.TotalGainValue,
					SuspiciousItems:    parsed.SuspiciousItems,
					RecommendedActions: parsed.RecommendedActions,
					EngineSource:       "cloudflare_workers_ai",
				}
			}
		}
	}

	// Smart Local Heuristic Fallback Engine
	riskLevel := "low"
	if inv.TotalLossValue > 5000000 || len(discLines) >= 3 {
		riskLevel = "high"
	} else if inv.TotalLossValue > 1000000 || len(discLines) >= 2 {
		riskLevel = "medium"
	}

	var suspicious []string
	for _, l := range discLines {
		if l.DifferenceQty < 0 && (math.Abs(l.VarianceValue) > 500000 || math.Abs(l.DifferenceQty) >= 5) {
			pName := fmt.Sprintf("Produk ID #%d", l.ProductID)
			if l.Product != nil && l.Product.Name != "" {
				pName = l.Product.Name
			}
			suspicious = append(suspicious, fmt.Sprintf("%s (Hilang %.1f unit, kerugian Rp %.2f)", pName, math.Abs(l.DifferenceQty), math.Abs(l.VarianceValue)))
		}
	}

	summary := fmt.Sprintf("Audit Stock Opname mendeteksi %d produk mengalami selisih fisik dengan total nilai kerugian Rp %.2f dan surplus Rp %.2f. Tingkat risiko dinilai %s.",
		len(discLines), inv.TotalLossValue, inv.TotalGainValue, strings.ToUpper(riskLevel))

	var recommendations []string
	if riskLevel == "high" {
		recommendations = append(recommendations, "Lakukan audit rekonsiliasi penerimaan vendor (WH/IN) dan pengiriman (WH/OUT) dalam 30 hari terakhir.")
		recommendations = append(recommendations, "Periksa rekaman CCTV area penyimpanan rak dan verifikasi akses personel gudang.")
		recommendations = append(recommendations, "Wajibkan verifikasi ganda (double-check barcode scanner) sebelum barang dikeluarkan dari gudang.")
	} else if riskLevel == "medium" {
		recommendations = append(recommendations, "Investigasi kemungkinan barang rusak/pecah yang belum dilaporkan atau salah hitung kemasan satuan.")
		recommendations = append(recommendations, "Perbarui pencatatan picking gudang dan lakukan cycle count mingguan pada produk bernilai tinggi.")
	} else {
		recommendations = append(recommendations, "Selisih berada dalam batas wajar toleransi operasional gudang.")
		recommendations = append(recommendations, "Lakukan penyesuaian stok sistem dan terus pantau kebersihan administrasi picking.")
	}

	return &AIAuditResult{
		InventoryID:        inv.ID,
		InventoryName:      inv.Name,
		RiskLevel:          riskLevel,
		Summary:            summary,
		DiscrepancyCount:   len(discLines),
		TotalLossValue:     inv.TotalLossValue,
		TotalGainValue:     inv.TotalGainValue,
		SuspiciousItems:    suspicious,
		RecommendedActions: recommendations,
		EngineSource:       "local_heuristic_engine",
	}
}

// ---------------------------------------------------------------------------
// Validation, Double-Entry Moves & General Ledger Accounting Post
// ---------------------------------------------------------------------------

func (s *Service) ValidateInventory(id, companyID, userID uint) (*StockInventory, error) {
	inv, err := s.repo.GetInventoryByID(id, companyID)
	if err != nil {
		return nil, errors.New("data stock opname tidak ditemukan")
	}

	if inv.State != "confirm" {
		return nil, fmt.Errorf("hanya stock opname berstatus 'confirm' yang dapat divalidasi (status saat ini: %s)", inv.State)
	}

	s.recalculateInventoryTotals(inv)

	// Fetch Virtual Inventory Adjustment Location
	var adjLoc stock.StockLocation
	if err := s.db.Where("name = ? AND company_id = ?", "Virtual Locations/Inventory Adjustment", companyID).First(&adjLoc).Error; err != nil {
		// Fallback create virtual inventory location
		adjLoc = stock.StockLocation{
			Name:         "Virtual Locations/Inventory Adjustment",
			Usage:        "inventory",
			CompleteName: "Virtual Locations/Inventory Adjustment",
		}
		adjLoc.CompanyID = companyID
		s.db.Create(&adjLoc)
	}

	// Begin atomic transaction
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	nowStr := time.Now().Format("2006-01-02 15:04:05")

	// 1. Process Stock Moves and Quant Updates
	for i := range inv.Lines {
		line := &inv.Lines[i]
		if math.Abs(line.DifferenceQty) > 0.0001 {
			var srcLocID, destLocID uint
			moveQty := math.Abs(line.DifferenceQty)

			if line.DifferenceQty > 0 {
				// Surplus (Gain): Move from Virtual Adjustment to Stock Location
				srcLocID = adjLoc.ID
				destLocID = inv.LocationID
			} else {
				// Shrinkage (Loss): Move from Stock Location to Virtual Adjustment
				srcLocID = inv.LocationID
				destLocID = adjLoc.ID
			}

			// Create StockMove
			move := stock.StockMove{
				ProductID:      line.ProductID,
				ProductName:    fmt.Sprintf("Adj: %s", inv.Name),
				ProductUomQty:  moveQty,
				QuantityDone:   moveQty,
				LocationID:     srcLocID,
				LocationDestID: destLocID,
				State:          "done",
			}
			move.CompanyID = companyID
			if err := tx.Create(&move).Error; err != nil {
				tx.Rollback()
				return nil, err
			}

			// Update or Insert StockQuant at LocationID to match CountedQty exactly
			var quant stock.StockQuant
			err := tx.Where("product_id = ? AND location_id = ? AND company_id = ?", line.ProductID, inv.LocationID, companyID).First(&quant).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					quant = stock.StockQuant{
						ProductID:  line.ProductID,
						LocationID: inv.LocationID,
						Quantity:   line.CountedQty,
					}
					quant.CompanyID = companyID
					if err := tx.Create(&quant).Error; err != nil {
						tx.Rollback()
						return nil, err
					}
				} else {
					tx.Rollback()
					return nil, err
				}
			} else {
				quant.Quantity = line.CountedQty
				if err := tx.Save(&quant).Error; err != nil {
					tx.Rollback()
					return nil, err
				}
			}
		}

		line.State = "done"
		if err := tx.Save(line).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// 2. Financial General Ledger Posting (account.move)
	var createdMove *account.AccountMove
	if inv.TotalLossValue > 0 || inv.TotalGainValue > 0 {
		var miscJournal account.AccountJournal
		if err := tx.Where("code = ? AND company_id = ?", "MISC", companyID).First(&miscJournal).Error; err != nil {
			// Find any general journal
			_ = tx.Where("company_id = ?", companyID).First(&miscJournal).Error
		}

		// Accounts:
		// 130000: Persediaan Barang Dagang (Stock Valuation)
		// 620700: Beban Selisih Kurang Stok (Inventory Shrinkage/Loss)
		// 710200: Pendapatan Selisih Lebih Stok (Inventory Gain)
		var accStock, accLoss, accGain account.AccountAccount
		if err := tx.Where("code = ? AND company_id = ?", "130000", companyID).First(&accStock).Error; err != nil {
			accStock.ID = 4 // Fallback
		}
		if err := tx.Where("code = ? AND company_id = ?", "620700", companyID).First(&accLoss).Error; err != nil {
			// fallback to COGS 510000
			if err2 := tx.Where("code = ? AND company_id = ?", "510000", companyID).First(&accLoss).Error; err2 != nil {
				accLoss.ID = 8
			}
		}
		if err := tx.Where("code = ? AND company_id = ?", "710200", companyID).First(&accGain).Error; err != nil {
			// fallback to 410000
			if err2 := tx.Where("code = ? AND company_id = ?", "410000", companyID).First(&accGain).Error; err2 != nil {
				accGain.ID = 7
			}
		}

		year := time.Now().Format("2006")
		var miscCount int64
		tx.Model(&account.AccountMove{}).Where("company_id = ?", companyID).Count(&miscCount)
		moveName := fmt.Sprintf("MISC/%s/%05d", year, miscCount+1)

		totalMoveAmount := inv.TotalLossValue + inv.TotalGainValue
		move := account.AccountMove{
			Name:           moveName,
			MoveType:       "entry",
			JournalID:      miscJournal.ID,
			PartnerID:      1, // Internal/Company
			InvoiceDate:    time.Now().Format("2006-01-02"),
			InvoiceDueDate: time.Now().Format("2006-01-02"),
			State:          "posted",
			PaymentState:   "paid",
			AmountUntaxed:  totalMoveAmount,
			AmountTotal:    totalMoveAmount,
			Ref:            inv.Name,
			Note:           fmt.Sprintf("Penyesuaian Nilai Persediaan dari Stock Opname %s (Loss: Rp %.2f, Gain: Rp %.2f)", inv.Name, inv.TotalLossValue, inv.TotalGainValue),
		}
		move.CompanyID = companyID
		if err := tx.Create(&move).Error; err != nil {
			tx.Rollback()
			return nil, err
		}

		var moveLines []account.AccountMoveLine

		// If Loss (Shrinkage): Debit Beban Selisih Kurang Stok, Credit Persediaan
		if inv.TotalLossValue > 0 {
			moveLines = append(moveLines, account.AccountMoveLine{
				MoveID:    move.ID,
				AccountID: accLoss.ID,
				Name:      fmt.Sprintf("Selisih Kurang Stok (Shrinkage Loss) %s", inv.Name),
				Debit:     inv.TotalLossValue,
				Credit:    0,
				Balance:   inv.TotalLossValue,
			})
			moveLines = append(moveLines, account.AccountMoveLine{
				MoveID:    move.ID,
				AccountID: accStock.ID,
				Name:      fmt.Sprintf("Pengurangan Persediaan Barang %s", inv.Name),
				Debit:     0,
				Credit:    inv.TotalLossValue,
				Balance:   -inv.TotalLossValue,
			})
		}

		// If Gain (Surplus): Debit Persediaan, Credit Pendapatan Selisih Lebih
		if inv.TotalGainValue > 0 {
			moveLines = append(moveLines, account.AccountMoveLine{
				MoveID:    move.ID,
				AccountID: accStock.ID,
				Name:      fmt.Sprintf("Penambahan Persediaan Barang %s", inv.Name),
				Debit:     inv.TotalGainValue,
				Credit:    0,
				Balance:   inv.TotalGainValue,
			})
			moveLines = append(moveLines, account.AccountMoveLine{
				MoveID:    move.ID,
				AccountID: accGain.ID,
				Name:      fmt.Sprintf("Selisih Lebih Stok (Surplus Gain) %s", inv.Name),
				Debit:     0,
				Credit:    inv.TotalGainValue,
				Balance:   -inv.TotalGainValue,
			})
		}

		for _, ml := range moveLines {
			ml.CompanyID = companyID
			if err := tx.Create(&ml).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}

		createdMove = &move
		inv.AccountingMoveID = &move.ID
		inv.AccountingMoveName = move.Name
	}

	inv.State = "done"
	if err := tx.Save(inv).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Post-commit Chatter
	msg := fmt.Sprintf("✅ Stock Opname '%s' telah divalidasi dan selesai (Done). Saldo quant fisik telah disinkronkan ke seluruh sistem.", inv.Name)
	if createdMove != nil {
		msg += fmt.Sprintf(" Entri Jurnal Akuntansi '%s' berhasil diposting ke Buku Besar.", createdMove.Name)
	}
	s.logChatter(inv.ID, msg, companyID, userID)

	_ = nowStr
	return s.repo.GetInventoryByID(inv.ID, companyID)
}

func (s *Service) CancelInventory(id, companyID, userID uint) (*StockInventory, error) {
	inv, err := s.repo.GetInventoryByID(id, companyID)
	if err != nil {
		return nil, errors.New("data stock opname tidak ditemukan")
	}

	if inv.State == "done" {
		return nil, errors.New("tidak dapat membatalkan stock opname yang sudah selesai (done)")
	}

	inv.State = "cancel"
	if err := s.repo.UpdateInventory(inv); err != nil {
		return nil, err
	}

	s.logChatter(inv.ID, fmt.Sprintf("Stock Opname '%s' telah dibatalkan.", inv.Name), companyID, userID)

	return s.repo.GetInventoryByID(inv.ID, companyID)
}

func (s *Service) logChatter(invID uint, body string, companyID, userID uint) {
	if s.chatterSvc == nil {
		return
	}
	msg := chatter.MailMessage{
		ModelName:   "stock.inventory",
		ResID:       invID,
		AuthorID:    userID,
		AuthorName:  "Administrator",
		Body:        body,
		MessageType: "comment",
	}
	msg.CompanyID = companyID
	_ = s.chatterSvc.PostMessage(&msg)
}

func (s *Service) GetStats(companyID uint) (*InventoryStats, error) {
	return s.repo.GetStats(companyID)
}

func (s *Service) recalculateInventoryTotals(inv *StockInventory) {
	var totalTheo, totalCounted, totalDiff, totalLoss, totalGain float64
	for _, l := range inv.Lines {
		totalTheo += l.TheoreticalQty
		totalCounted += l.CountedQty
		diff := l.CountedQty - l.TheoreticalQty
		totalDiff += diff
		varVal := diff * l.CostPrice
		if varVal < 0 {
			totalLoss += math.Abs(varVal)
		} else {
			totalGain += varVal
		}
	}

	inv.TotalTheoreticalQty = totalTheo
	inv.TotalCountedQty = totalCounted
	inv.TotalDifferenceQty = totalDiff
	inv.TotalLossValue = totalLoss
	inv.TotalGainValue = totalGain
}
