package traceability

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/barcode"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"

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
// 1. Lot Movement Recording Engine
// ---------------------------------------------------------------------------

func (s *Service) RecordMovement(req RecordLotMovementRequest, companyID uint, userID uint) (*LotTraceRecord, error) {
	if req.Quantity <= 0 {
		return nil, errors.New("jumlah pergerakan lot harus lebih besar dari 0")
	}

	lot, err := s.repo.GetLotByID(req.LotID, companyID)
	if err != nil {
		return nil, fmt.Errorf("lot dengan ID %d tidak ditemukan: %w", req.LotID, err)
	}

	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02 15:04:05")
	}

	record := &LotTraceRecord{
		LotID:          lot.ID,
		LotName:        lot.Name,
		ProductID:      req.ProductID,
		ProductName:    lot.Product.Name,
		EventType:      req.EventType,
		Reference:      req.Reference,
		SourceDocument: req.SourceDocument,
		PartnerID:      req.PartnerID,
		PartnerName:    req.PartnerName,
		PickingID:      req.PickingID,
		ProductionID:   req.ProductionID,
		ParentLotID:    req.ParentLotID,
		ProducedLotID:  req.ProducedLotID,
		ConsumedLotID:  req.ConsumedLotID,
		Quantity:       req.Quantity,
		UoM:            req.UoM,
		LocationSrcID:  req.LocationSrcID,
		LocationDestID: req.LocationDestID,
		Date:           dateStr,
		ExpirationDate: lot.ExpirationDate,
		Notes:          req.Notes,
	}
	record.CompanyID = companyID

	if req.ParentLotID != nil {
		if parentLot, err := s.repo.GetLotByID(*req.ParentLotID, companyID); err == nil {
			record.ParentLotName = parentLot.Name
		}
	}
	if req.ProducedLotID != nil {
		if prodLot, err := s.repo.GetLotByID(*req.ProducedLotID, companyID); err == nil {
			record.ProducedLotName = prodLot.Name
		}
	}
	if req.ConsumedLotID != nil {
		if consLot, err := s.repo.GetLotByID(*req.ConsumedLotID, companyID); err == nil {
			record.ConsumedLotName = consLot.Name
		}
	}

	if err := s.repo.CreateTraceRecord(record); err != nil {
		return nil, fmt.Errorf("gagal mencatat rekam jejak lot: %w", err)
	}

	return record, nil
}

func (s *Service) ListTraceRecords(lotID uint, companyID uint, eventType string, search string) ([]LotTraceRecord, error) {
	return s.repo.ListTraceRecords(lotID, companyID, eventType, search)
}

// ---------------------------------------------------------------------------
// 2. Bidirectional Upstream & Downstream Genealogical Traceability Tree
// ---------------------------------------------------------------------------

func (s *Service) BuildTraceabilityTree(lotID uint, direction string, companyID uint) (*TraceabilityTreeReport, error) {
	lot, err := s.repo.GetLotByID(lotID, companyID)
	if err != nil {
		return nil, fmt.Errorf("lot dengan ID %d tidak ditemukan: %w", lotID, err)
	}

	dir := strings.ToLower(strings.TrimSpace(direction))
	if dir == "" {
		dir = "full"
	}

	report := &TraceabilityTreeReport{
		TargetLotID:          lot.ID,
		TargetLotName:        lot.Name,
		TargetProductID:      lot.ProductID,
		TargetProductName:    lot.Product.Name,
		Direction:            dir,
		GeneratedAt:          time.Now().Format("2006-01-02 15:04:05"),
		CurrentLotQty:        lot.Qty,
		ExpirationDate:       lot.ExpirationDate,
		ImpactedCustomers:    []string{},
		ImpactedLocations:    []string{},
		ImpactedFinishedLots: []string{},
	}

	// Fetch all trace records related to this lot
	directTraces, err := s.repo.GetTracesByLotID(lot.ID, companyID)
	if err != nil {
		return nil, err
	}

	custMap := make(map[string]bool)
	locMap := make(map[string]bool)
	finLotMap := make(map[string]bool)
	suppMap := make(map[string]bool)
	moMap := make(map[string]bool)
	delMap := make(map[string]bool)

	// Build Upstream (Where did it come from?)
	if dir == "upstream" || dir == "full" {
		rootUpstream := &TraceabilityNode{
			ID:             fmt.Sprintf("target-lot-%d", lot.ID),
			NodeType:       "lot",
			Title:          fmt.Sprintf("Lot: %s (%s)", lot.Name, lot.Product.Name),
			Reference:      lot.Name,
			LotID:          &lot.ID,
			LotName:        lot.Name,
			ProductID:      lot.ProductID,
			ProductName:    lot.Product.Name,
			Quantity:       lot.Qty,
			ExpirationDate: lot.ExpirationDate,
			Date:           lot.CreatedAt.Format("2006-01-02 15:04:05"),
			Status:         "current_target",
			Children:       []*TraceabilityNode{},
		}

		// 1. Direct Receipt from Supplier
		for _, tr := range directTraces {
			if tr.EventType == EventReceipt {
				suppNode := &TraceabilityNode{
					ID:          fmt.Sprintf("receipt-%d", tr.ID),
					NodeType:    "receipt",
					Title:       fmt.Sprintf("Penerimaan Supplier: %s (%s)", tr.Reference, tr.PartnerName),
					Reference:   tr.Reference,
					Quantity:    tr.Quantity,
					PartnerName: tr.PartnerName,
					Date:        tr.Date,
					Status:      "supplier_origin",
				}
				rootUpstream.Children = append(rootUpstream.Children, suppNode)
				if tr.PartnerName != "" {
					suppMap[tr.PartnerName] = true
				}
			}

			// 2. Production Finished Good -> trace back to raw material lots consumed
			if tr.EventType == EventMOProduce {
				moNode := &TraceabilityNode{
					ID:        fmt.Sprintf("mo-%d", tr.ID),
					NodeType:  "production",
					Title:     fmt.Sprintf("Perintah Produksi: %s", tr.Reference),
					Reference: tr.Reference,
					Quantity:  tr.Quantity,
					Date:      tr.Date,
					Status:    "manufactured",
					Children:  []*TraceabilityNode{},
				}
				moMap[tr.Reference] = true

				// Find raw material trace records that were consumed into this produced lot
				consumedTraces, _ := s.repo.GetTracesByProducedLotID(lot.ID, companyID)
				for _, cTr := range consumedTraces {
					rawLotNode := &TraceabilityNode{
						ID:             fmt.Sprintf("raw-lot-%d", cTr.ID),
						NodeType:       "raw_lot",
						Title:          fmt.Sprintf("Bahan Baku: %s (Lot %s)", cTr.ProductName, cTr.LotName),
						Reference:      cTr.LotName,
						LotID:          &cTr.LotID,
						LotName:        cTr.LotName,
						ProductID:      cTr.ProductID,
						ProductName:    cTr.ProductName,
						Quantity:       cTr.Quantity,
						Date:           cTr.Date,
						ExpirationDate: cTr.ExpirationDate,
						Status:         "consumed_component",
						Children:       []*TraceabilityNode{},
					}

					// Trace origin of raw material lot (supplier receipt)
					rawReceipts, _ := s.repo.GetTracesByLotID(cTr.LotID, companyID)
					for _, rr := range rawReceipts {
						if rr.EventType == EventReceipt {
							rawSuppNode := &TraceabilityNode{
								ID:          fmt.Sprintf("raw-receipt-%d", rr.ID),
								NodeType:    "receipt",
								Title:       fmt.Sprintf("Penerimaan Supplier Bahan Baku: %s (%s)", rr.Reference, rr.PartnerName),
								Reference:   rr.Reference,
								Quantity:    rr.Quantity,
								PartnerName: rr.PartnerName,
								Date:        rr.Date,
								Status:      "vendor_origin",
							}
							rawLotNode.Children = append(rawLotNode.Children, rawSuppNode)
							if rr.PartnerName != "" {
								suppMap[rr.PartnerName] = true
							}
						}
					}

					moNode.Children = append(moNode.Children, rawLotNode)
				}

				rootUpstream.Children = append(rootUpstream.Children, moNode)
			}
		}

		report.UpstreamTree = rootUpstream
	}

	// Build Downstream (Where did it go / Product Recall Path)
	if dir == "downstream" || dir == "full" {
		rootDownstream := &TraceabilityNode{
			ID:             fmt.Sprintf("target-lot-%d", lot.ID),
			NodeType:       "lot",
			Title:          fmt.Sprintf("Lot: %s (%s)", lot.Name, lot.Product.Name),
			Reference:      lot.Name,
			LotID:          &lot.ID,
			LotName:        lot.Name,
			ProductID:      lot.ProductID,
			ProductName:    lot.Product.Name,
			Quantity:       lot.Qty,
			ExpirationDate: lot.ExpirationDate,
			Date:           lot.CreatedAt.Format("2006-01-02 15:04:05"),
			Status:         "current_target",
			Children:       []*TraceabilityNode{},
		}

		// 1. Direct Delivery to Customers
		for _, tr := range directTraces {
			if tr.EventType == EventDelivery {
				delNode := &TraceabilityNode{
					ID:          fmt.Sprintf("del-%d", tr.ID),
					NodeType:    "delivery",
					Title:       fmt.Sprintf("Pengiriman Pelanggan: %s (%s)", tr.Reference, tr.PartnerName),
					Reference:   tr.Reference,
					Quantity:    tr.Quantity,
					PartnerName: tr.PartnerName,
					Date:        tr.Date,
					Status:      "delivered_to_customer",
				}
				rootDownstream.Children = append(rootDownstream.Children, delNode)
				if tr.PartnerName != "" {
					custMap[tr.PartnerName] = true
				}
				delMap[tr.Reference] = true
			}

			// 2. Consumed in MO -> produced finished good lots -> shipped to customers!
			if tr.EventType == EventMOConsume {
				moNode := &TraceabilityNode{
					ID:        fmt.Sprintf("mo-consume-%d", tr.ID),
					NodeType:  "production",
					Title:     fmt.Sprintf("Dikonsumsi di Perintah Produksi: %s", tr.Reference),
					Reference: tr.Reference,
					Quantity:  tr.Quantity,
					Date:      tr.Date,
					Status:    "consumed_in_manufacturing",
					Children:  []*TraceabilityNode{},
				}
				moMap[tr.Reference] = true

				// Check what finished good lots were produced from this consumed raw lot
				producedTraces, _ := s.repo.GetTracesByConsumedLotID(lot.ID, companyID)
				for _, pTr := range producedTraces {
					finLotNode := &TraceabilityNode{
						ID:             fmt.Sprintf("fin-lot-%d", pTr.ID),
						NodeType:       "finished_lot",
						Title:          fmt.Sprintf("Produk Jadi: %s (Lot %s)", pTr.ProductName, pTr.LotName),
						Reference:      pTr.LotName,
						LotID:          &pTr.LotID,
						LotName:        pTr.LotName,
						ProductID:      pTr.ProductID,
						ProductName:    pTr.ProductName,
						Quantity:       pTr.Quantity,
						Date:           pTr.Date,
						ExpirationDate: pTr.ExpirationDate,
						Status:         "finished_batch",
						Children:       []*TraceabilityNode{},
					}
					finLotMap[pTr.LotName] = true

					// Check downstream customer deliveries of this finished lot!
					finDeliveries, _ := s.repo.GetTracesByLotID(pTr.LotID, companyID)
					for _, fd := range finDeliveries {
						if fd.EventType == EventDelivery {
							fdNode := &TraceabilityNode{
								ID:          fmt.Sprintf("fin-del-%d", fd.ID),
								NodeType:    "delivery",
								Title:       fmt.Sprintf("Pengiriman Produk Jadi: %s (%s)", fd.Reference, fd.PartnerName),
								Reference:   fd.Reference,
								Quantity:    fd.Quantity,
								PartnerName: fd.PartnerName,
								Date:        fd.Date,
								Status:      "delivered_to_customer",
							}
							finLotNode.Children = append(finLotNode.Children, fdNode)
							if fd.PartnerName != "" {
								custMap[fd.PartnerName] = true
							}
							delMap[fd.Reference] = true
						}
					}

					moNode.Children = append(moNode.Children, finLotNode)
				}

				rootDownstream.Children = append(rootDownstream.Children, moNode)
			}
		}

		report.DownstreamTree = rootDownstream
	}

	// Consolidate metrics
	for c := range custMap {
		report.ImpactedCustomers = append(report.ImpactedCustomers, c)
	}
	sort.Strings(report.ImpactedCustomers)

	for fl := range finLotMap {
		report.ImpactedFinishedLots = append(report.ImpactedFinishedLots, fl)
	}
	sort.Strings(report.ImpactedFinishedLots)

	if lot.Location != nil && lot.Location.Name != "" {
		locMap[lot.Location.Name] = true
	}
	for loc := range locMap {
		report.ImpactedLocations = append(report.ImpactedLocations, loc)
	}

	report.TotalSuppliersInvolved = len(suppMap)
	report.TotalManufacturingOrdersInvolved = len(moMap)
	report.TotalCustomerDeliveriesInvolved = len(delMap)

	return report, nil
}

// ---------------------------------------------------------------------------
// 3. FEFO (First-Expired, First-Out) Picking Strategy Engine
// ---------------------------------------------------------------------------

func parseLotExpiryDate(dateStr string) (time.Time, bool) {
	str := strings.TrimSpace(dateStr)
	if str == "" {
		return time.Time{}, false
	}

	layouts := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"2006/01/02",
		"02-01-2006",
	}

	for _, l := range layouts {
		if t, err := time.Parse(l, str); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func (s *Service) AllocateFEFO(req FEFOAllocationRequest, companyID uint) (*FEFOAllocationResult, error) {
	if req.RequestedQty <= 0 {
		return nil, errors.New("jumlah permintaan picking harus lebih besar dari 0")
	}

	var prod product.Product
	if err := s.db.Where("id = ? AND company_id = ?", req.ProductID, companyID).First(&prod).Error; err != nil {
		return nil, fmt.Errorf("produk ID %d tidak ditemukan: %w", req.ProductID, err)
	}

	cutoffDays := req.ExpiryCutoffDays
	if cutoffDays <= 0 {
		cutoffDays = 30 // Default 30-day warning window
	}

	lots, err := s.repo.FindLotsForProduct(req.ProductID, companyID, req.LocationID)
	if err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)

	// Pre-process and categorize candidate lots
	type candidateLot struct {
		lot             barcode.StockProductionLot
		expDate         time.Time
		hasExpiry       bool
		daysUntilExpiry int
		status          string
	}

	var candidates []candidateLot
	for _, l := range lots {
		if l.Qty <= 0 {
			continue
		}

		expTime, hasExp := parseLotExpiryDate(l.ExpirationDate)
		daysUntil := 99999
		status := "fresh"

		if hasExp {
			daysUntil = int(expTime.Sub(today).Hours() / 24)
			if daysUntil < 0 {
				status = "expired"
			} else if daysUntil <= cutoffDays {
				status = "warning_near_expiry"
			}
		}

		// Filter out expired lots unless explicitly permitted
		if status == "expired" && !req.AllowNearExpiry {
			continue
		}

		candidates = append(candidates, candidateLot{
			lot:             l,
			expDate:         expTime,
			hasExpiry:       hasExp,
			daysUntilExpiry: daysUntil,
			status:          status,
		})
	}

	// Sort candidates according to strict FEFO:
	// Lots with earliest expiration date first. Lots without expiration date come last.
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].hasExpiry && candidates[j].hasExpiry {
			return candidates[i].expDate.Before(candidates[j].expDate)
		}
		if candidates[i].hasExpiry && !candidates[j].hasExpiry {
			return true
		}
		if !candidates[i].hasExpiry && candidates[j].hasExpiry {
			return false
		}
		return candidates[i].lot.ID < candidates[j].lot.ID
	})

	result := &FEFOAllocationResult{
		ProductID:         prod.ID,
		ProductName:       prod.Name,
		RequestedQty:      req.RequestedQty,
		StrategyApplied:   "FEFO (First-Expired, First-Out Strategy)",
		Allocations:       []FEFOAllocationItem{},
		TotalAllocatedQty: 0,
	}

	remaining := req.RequestedQty

	for _, c := range candidates {
		if remaining <= 0 {
			break
		}

		allocQty := math.Min(c.lot.Qty, remaining)
		allocQty = math.Round(allocQty*1000) / 1000 // 3 decimals precision

		locName := ""
		if c.lot.Location != nil {
			locName = c.lot.Location.Name
		}

		item := FEFOAllocationItem{
			LotID:           c.lot.ID,
			LotName:         c.lot.Name,
			Barcode:         c.lot.Barcode,
			ExpirationDate:  c.lot.ExpirationDate,
			RemovalDate:     c.lot.RemovalDate,
			AvailableQty:    c.lot.Qty,
			AllocatedQty:    allocQty,
			DaysUntilExpiry: c.daysUntilExpiry,
			Status:          c.status,
			LocationID:      c.lot.LocationID,
			LocationName:    locName,
		}

		result.Allocations = append(result.Allocations, item)
		result.TotalAllocatedQty += allocQty
		remaining -= allocQty
	}

	result.TotalAllocatedQty = math.Round(result.TotalAllocatedQty*1000) / 1000
	result.ShortageQty = math.Max(0, math.Round((req.RequestedQty-result.TotalAllocatedQty)*1000)/1000)
	result.IsFullyAllocated = result.ShortageQty <= 0.0001

	return result, nil
}

// ---------------------------------------------------------------------------
// 4. Product Recall & Downstream Customer Tracing Engine
// ---------------------------------------------------------------------------

func (s *Service) InitiateRecall(req ProductRecallRequest, companyID uint, userID uint) (*ProductRecall, error) {
	lot, err := s.repo.GetLotByID(req.LotID, companyID)
	if err != nil {
		return nil, fmt.Errorf("lot dengan ID %d tidak ditemukan: %w", req.LotID, err)
	}

	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errors.New("alasan penarikan produk (recall reason) wajib diisi")
	}

	severity := strings.ToLower(strings.TrimSpace(req.Severity))
	if severity == "" {
		severity = "critical"
	}

	// 1. Run Downstream Genealogical Traceability to find all impacted customers & deliveries
	treeReport, err := s.BuildTraceabilityTree(lot.ID, "downstream", companyID)
	if err != nil {
		return nil, fmt.Errorf("gagal menganalisis silsilah downstream: %w", err)
	}

	// Generate sequence RCL/YYYY/00001
	year := time.Now().Format("2006")
	var count int64
	s.db.Model(&ProductRecall{}).Where("company_id = ?", companyID).Count(&count)
	recallCode := fmt.Sprintf("RCL/%s/%05d", year, count+1)

	// Financial Risk Calculation = (Current Quarantined Qty + Shipped Qty) * Standard Price
	unitCost := lot.Product.StandardPrice
	if unitCost <= 0 {
		unitCost = lot.Product.ListPrice * 0.7 // heuristic estimate
	}
	finRisk := (lot.Qty) * unitCost

	recall := &ProductRecall{
		RecallCode:               recallCode,
		TargetLotID:              lot.ID,
		TargetLotName:            lot.Name,
		ProductID:                lot.ProductID,
		ProductName:              lot.Product.Name,
		Reason:                   reason,
		Severity:                 severity,
		State:                    RecallStateActive,
		AffectedStockQuarantined: lot.Qty,
		AffectedDeliveriesCount:  treeReport.TotalCustomerDeliveriesInvolved,
		AffectedCustomersCount:   len(treeReport.ImpactedCustomers),
		FinancialRisk:            finRisk,
		AIProvider:               "Rule-Based Regulatory Heuristics",
	}
	recall.CompanyID = companyID

	// 2. Draft Customer Notice & Regulatory Notice using AI Copilot (with smart fallback)
	s.generateRecallNotices(recall, treeReport, lot)

	// 3. Quarantine warehouse stock of target lot
	if lot.Qty > 0 {
		lot.Notes = fmt.Sprintf("[QUARANTINED BY RECALL %s: %s] %s", recall.RecallCode, reason, lot.Notes)
		s.db.Save(lot)
		recall.State = RecallStateQuarantined
	}

	if err := s.repo.CreateRecall(recall); err != nil {
		return nil, fmt.Errorf("gagal membuat dokumen penarikan produk: %w", err)
	}

	// 4. Chatter Audit Trail
	if s.chatterSvc != nil {
		msg := fmt.Sprintf("🚨 <b>PENARIKAN PRODUK (PRODUCT RECALL) DITERBITKAN</b>: Kode <b>%s</b> untuk Lot <b>%s</b> (%s). "+
			"Alasan: %s. Pelanggan terdampak: %d. Stok dikarantina: %.2f unit. Estimasi Risiko Keuangan: Rp %.2f.",
			recall.RecallCode, recall.TargetLotName, recall.ProductName, recall.Reason,
			recall.AffectedCustomersCount, recall.AffectedStockQuarantined, recall.FinancialRisk)
		_ = s.chatterSvc.LogTracking("stock.lot.recall", recall.ID, userID, "Recall Safety Copilot", msg, "recall_issued", companyID)
	}

	return recall, nil
}

func (s *Service) generateRecallNotices(recall *ProductRecall, tree *TraceabilityTreeReport, lot *barcode.StockProductionLot) {
	// Prepare context for Cloudflare Workers AI
	customerListStr := strings.Join(tree.ImpactedCustomers, ", ")
	if customerListStr == "" {
		customerListStr = "Belum terkirim ke pelanggan eksternal (terisolasi di gudang internal)"
	}

	prompt := fmt.Sprintf(`Anda adalah Direktur Kepatuhan Regulasi Farmasi & Mutu Supply Chain (BPOM/FDA).
Buatkan draf resmi surat pemberitahuan penarikan produk darurat (Product Recall Notification) dan laporan audit regulasi untuk:
- Kode Recall: %s
- Nomor Lot Target: %s
- Produk: %s
- Tanggal Kadaluwarsa: %s
- Alasan Penarikan: %s
- Tingkat Keparahan: %s
- Pelanggan Terdampak (%d): %s
- Stok Gudang Dikarantina: %.2f unit

Hasilkan JSON dengan format tepat:
{
  "customer_notice": "Surat resmi untuk pelanggan/rumah sakit/distributor...",
  "regulatory_notice": "Laporan kepatuhan resmi untuk BPOM/Kementerian Kesehatan...",
  "resolution_notes": "Langkah mitigasi & SOP penanganan..."
}`, recall.RecallCode, recall.TargetLotName, recall.ProductName, lot.ExpirationDate, recall.Reason,
		recall.Severity, len(tree.ImpactedCustomers), customerListStr, recall.AffectedStockQuarantined)

	var aiResp struct {
		CustomerNotice   string `json:"customer_notice"`
		RegulatoryNotice string `json:"regulatory_notice"`
		ResolutionNotes  string `json:"resolution_notes"`
	}

	usedAI := false
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "Anda adalah Direktur Kepatuhan Regulasi Farmasi & Mutu Supply Chain (BPOM/FDA). Selalu berikan output dalam format JSON yang valid."
		if raw, err := s.aiClient.CallRaw(prompt, systemPrompt); err == nil && raw != "" {
			jsonStart := strings.Index(raw, "{")
			jsonEnd := strings.LastIndex(raw, "}")
			if jsonStart >= 0 && jsonEnd > jsonStart {
				jsonStr := raw[jsonStart : jsonEnd+1]
				if err := json.Unmarshal([]byte(jsonStr), &aiResp); err == nil && aiResp.CustomerNotice != "" {
					recall.CustomerNoticeText = aiResp.CustomerNotice
					recall.RegulatoryNoticeText = aiResp.RegulatoryNotice
					recall.ResolutionNotes = aiResp.ResolutionNotes
					recall.AIProvider = "Cloudflare Workers AI (Llama-3-8b-instruct)"
					usedAI = true
				}
			}
		}
	}

	// Smart Heuristic Fallback
	if !usedAI {
		recall.CustomerNoticeText = fmt.Sprintf(
			"SURAT EDARAN PENARIKAN PRODUK DARURAT (URGENT RECALL NOTICE)\n\n"+
				"Kepada Yth. Pelanggan / Rekanan (%s),\n\n"+
				"Dengan hormat,\n"+
				"Berdasarkan hasil pemantauan mutu dan kepatuhan standar keselamatan, bersama ini kami sampaikan penarikan resmi "+
				"produk dengan rincian sebagai berikut:\n\n"+
				"- Nama Produk: %s\n"+
				"- Nomor Bets / Lot: %s\n"+
				"- Tanggal Kadaluwarsa: %s\n"+
				"- Alasan Penarikan: %s\n"+
				"- Tingkat Urgensi: %s\n\n"+
				"TINDAKAN YANG HARUS DILAKUKAN SEGERA:\n"+
				"1. Mohon segera hentikan distribusi, penjualan, atau penggunaan lot tersebut.\n"+
				"2. Pisahkan dan beri tanda 'KARANTINA - DALAM PROSES PENARIKAN'.\n"+
				"3. Tim logistik kami akan segera menghubungi Anda untuk proses penjemputan fisik dan penggantian batch/kredit nota penuh.\n\n"+
				"Hotline Mutu: (021) 555-RECALL / quality-alert@erp-odoo.id\n\n"+
				"Hormat kami,\nManajemen Penjaminan Mutu (QA/QC)",
			customerListStr, recall.ProductName, recall.TargetLotName, lot.ExpirationDate, recall.Reason, strings.ToUpper(recall.Severity),
		)

		recall.RegulatoryNoticeText = fmt.Sprintf(
			"LAPORAN KEPATUHAN PENARIKAN BETS KEPADA OTORITAS REGULATORI (BPOM / KEMENKES)\n\n"+
				"Nomor Registrasi Laporan: %s\n"+
				"Tanggal: %s\n"+
				"Pelapor: PT Industri Manufaktur & Distribusi Terpadu\n\n"+
				"1. INFORMASI PRODUK TERDAMPAK:\n"+
				"   - Nama Produk: %s\n"+
				"   - Nomor Bets: %s\n"+
				"   - Masa Berlaku: %s\n"+
				"   - Klasifikasi Penarikan: Kelas %s\n\n"+
				"2. ANALISIS AKAR MASALAH:\n"+
				"   - %s\n\n"+
				"3. TINDAKAN PENGENDALIAN:\n"+
				"   - Total stok internal dikarantina: %.2f unit.\n"+
				"   - Jumlah pengiriman terdampak yang telah ditelusuri: %d pengiriman ke %d pelanggan.\n"+
				"   - Target penarikan 100%% selesai dalam 7x24 jam kalender.",
			recall.RecallCode, time.Now().Format("02 January 2006"),
			recall.ProductName, recall.TargetLotName, lot.ExpirationDate, strings.ToUpper(recall.Severity),
			recall.Reason, recall.AffectedStockQuarantined, tree.TotalCustomerDeliveriesInvolved, len(tree.ImpactedCustomers),
		)

		recall.ResolutionNotes = fmt.Sprintf(
			"SOP RESOLUSI & INVESTIGASI AKAR MASALAH:\n"+
				"1. Karantina fisik pada gudang karantina (Location usage: transit/inventory).\n"+
				"2. Uji laboratorium sampel pertinggal (retained sample test).\n"+
				"3. CAPA (Corrective and Preventive Action) pada lini produksi & sertifikasi pemasok.\n"+
				"4. Pemusnahan limbah bersertifikat (Certificate of Destruction) setelah verifikasi.",
		)
	}
}

func (s *Service) ListRecalls(companyID uint, state string) ([]ProductRecall, error) {
	return s.repo.ListRecalls(companyID, state)
}

func (s *Service) GetRecall(id uint, companyID uint) (*ProductRecall, error) {
	return s.repo.GetRecallByID(id, companyID)
}

func (s *Service) CompleteRecall(id uint, companyID uint, resolutionNotes string, userID uint) (*ProductRecall, error) {
	recall, err := s.repo.GetRecallByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen penarikan produk tidak ditemukan")
	}

	recall.State = RecallStateCompleted
	if resolutionNotes != "" {
		recall.ResolutionNotes = resolutionNotes
	}

	if err := s.repo.UpdateRecall(recall); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := fmt.Sprintf("✅ Penarikan Produk <b>%s</b> telah diselesaikan (Completed). Resolusi: %s",
			recall.RecallCode, recall.ResolutionNotes)
		_ = s.chatterSvc.LogTracking("stock.lot.recall", recall.ID, userID, "Recall Safety Copilot", msg, "completed", companyID)
	}

	return recall, nil
}

// ---------------------------------------------------------------------------
// 5. Expiry Risk & Warehouse Batch Audit Engine
// ---------------------------------------------------------------------------

func (s *Service) AuditExpiryRisk(companyID uint) (*ExpiryRiskReport, error) {
	lots, err := s.repo.GetAllActiveLots(companyID)
	if err != nil {
		return nil, err
	}

	today := time.Now().Truncate(24 * time.Hour)
	report := &ExpiryRiskReport{
		TotalLotsTracked: len(lots),
		CriticalLots:     []ExpiryLotDetail{},
		AIProvider:       "Rule-Based Warehouse Expiry Heuristics",
	}

	for _, l := range lots {
		unitCost := l.Product.StandardPrice
		if unitCost <= 0 {
			unitCost = l.Product.ListPrice * 0.7
		}
		lossRisk := math.Round((l.Qty*unitCost)*100) / 100

		expTime, hasExp := parseLotExpiryDate(l.ExpirationDate)
		daysUntil := 99999
		status := "FRESH"

		if hasExp {
			daysUntil = int(expTime.Sub(today).Hours() / 24)
			if daysUntil < 0 {
				status = "EXPIRED"
				report.ExpiredLotsCount++
				report.ExpiredLotsValue += lossRisk
			} else if daysUntil <= 7 {
				status = "CRITICAL_7_DAYS"
				report.NearExpiryLotsCount++
				report.NearExpiryLotsValue += lossRisk
			} else if daysUntil <= 30 {
				status = "WARNING_30_DAYS"
				report.NearExpiryLotsCount++
				report.NearExpiryLotsValue += lossRisk
			} else {
				status = "FRESH"
				report.FreshLotsCount++
			}
		} else {
			report.FreshLotsCount++
		}

		locName := ""
		if l.Location != nil {
			locName = l.Location.Name
		}

		if status != "FRESH" {
			report.CriticalLots = append(report.CriticalLots, ExpiryLotDetail{
				LotID:          l.ID,
				LotName:        l.Name,
				ProductID:      l.ProductID,
				ProductName:    l.Product.Name,
				Qty:            l.Qty,
				StandardPrice:  unitCost,
				TotalLossRisk:  lossRisk,
				ExpirationDate: l.ExpirationDate,
				DaysRemaining:  daysUntil,
				Status:         status,
				LocationName:   locName,
			})
		}
	}

	// Determine overall warehouse risk level
	if report.ExpiredLotsCount > 0 {
		report.OverallRiskLevel = "CRITICAL"
	} else if report.NearExpiryLotsCount > 3 {
		report.OverallRiskLevel = "ELEVATED"
	} else if report.NearExpiryLotsCount > 0 {
		report.OverallRiskLevel = "MODERATE"
	} else {
		report.OverallRiskLevel = "HEALTHY"
	}

	// Generate AI Insights with fallback
	s.generateExpiryInsights(report)

	return report, nil
}

func (s *Service) generateExpiryInsights(report *ExpiryRiskReport) {
	if report.OverallRiskLevel == "HEALTHY" {
		report.AIInsights = "Kondisi persediaan sangat prima. Semua batch dan lot yang tersimpan di gudang memiliki masa kadaluwarsa lebih dari 30 hari kalender. Rekomendasi: Pertahankan penerapan strategi picking FEFO (First-Expired, First-Out)."
		return
	}

	prompt := fmt.Sprintf(`Anda adalah Warehouse Inventory Optimization Copilot.
Analisis risiko kadaluwarsa stok lot berikut:
- Level Risiko: %s
- Jumlah Lot Kadaluwarsa: %d (Nilai Rugi: Rp %.2f)
- Jumlah Lot Mendekati Kadaluwarsa (<=30 hari): %d (Nilai Berisiko: Rp %.2f)
- Lot Segar: %d

Berikan ringkasan audit eksekutif dan 3 rekomendasi taktis dalam bahasa Indonesia yang lugas dan tajam.`,
		report.OverallRiskLevel, report.ExpiredLotsCount, report.ExpiredLotsValue,
		report.NearExpiryLotsCount, report.NearExpiryLotsValue, report.FreshLotsCount)

	usedAI := false
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "Anda adalah Warehouse Inventory Optimization Copilot. Berikan analisis risiko kadaluwarsa yang tajam dan taktis."
		if raw, err := s.aiClient.CallRaw(prompt, systemPrompt); err == nil && raw != "" {
			report.AIInsights = strings.TrimSpace(raw)
			report.AIProvider = "Cloudflare Workers AI (Llama-3-8b-instruct)"
			usedAI = true
		}
	}

	if !usedAI {
		report.AIInsights = fmt.Sprintf(
			"AUDIT PERINGATAN KADALUWARSA GUDANG:\n"+
				"Ditemukan %d lot kadaluwarsa (potensi kerugian Rp %.2f) dan %d lot mendekati kadaluwarsa dalam 30 hari (nilai berisiko Rp %.2f).\n\n"+
				"REKOMENDASI PRIORITAS:\n"+
				"1. SEGERA KARANTINA: Pindahkan %d lot yang telah expired ke lokasi scrap/karantina untuk mencegah kesalahan picking.\n"+
				"2. PRIORITAS PENJUALAN FEFO: Terapkan diskon cuci gudang (clearance sale) atau prioritaskan pengiriman pada pesanan penjualan (Sales Order) yang akan datang.\n"+
				"3. EVALUASI SAFETY STOCK: Turunkan parameter reordering rule dan minimalkan batch size pembelian pada supplier.",
			report.ExpiredLotsCount, report.ExpiredLotsValue,
			report.NearExpiryLotsCount, report.NearExpiryLotsValue,
			report.ExpiredLotsCount,
		)
	}
}

// ---------------------------------------------------------------------------
// 6. Executive Stats
// ---------------------------------------------------------------------------

func (s *Service) GetStats(companyID uint) (*TraceabilityStats, error) {
	return s.repo.GetStats(companyID)
}
