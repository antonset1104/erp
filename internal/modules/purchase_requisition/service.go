package purchase_requisition

import (
	"errors"
	"fmt"
	"time"

	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"
	"erp-golang/internal/modules/purchase"

	"gorm.io/gorm"
)

type Service struct {
	db          *gorm.DB
	repo        *Repository
	purchaseSvc *purchase.Service
	chatterSvc  *chatter.Service
}

func NewService(db *gorm.DB, repo *Repository, purchaseSvc *purchase.Service, chatterSvc *chatter.Service) *Service {
	return &Service{
		db:          db,
		repo:        repo,
		purchaseSvc: purchaseSvc,
		chatterSvc:  chatterSvc,
	}
}

// CreateRequisition creates a new Purchase Requisition or Blanket Order agreement
func (s *Service) CreateRequisition(req CreateRequisitionRequest, companyID, userID uint) (*PurchaseRequisition, error) {
	if len(req.Lines) == 0 {
		return nil, errors.New("daftar barang permintaan tidak boleh kosong")
	}

	reqType := req.Type
	if reqType == "" {
		reqType = TypePurchaseTender
	}

	seqName, err := s.repo.GetNextSequence(reqType, companyID)
	if err != nil {
		return nil, fmt.Errorf("gagal menghasilkan nomor urut dokumen: %w", err)
	}

	ordDate := req.OrderingDate
	if ordDate == "" {
		ordDate = time.Now().Format("2006-01-02")
	}

	doc := &PurchaseRequisition{
		Name:         seqName,
		Origin:       req.Origin,
		Type:         reqType,
		State:        StateDraft,
		UserID:       userID,
		VendorID:     req.VendorID,
		OrderingDate: ordDate,
		DateEnd:      req.DateEnd,
		ScheduleDate: req.ScheduleDate,
		CompanyID:    companyID,
		Description:  req.Description,
	}

	var totalEst float64
	for _, l := range req.Lines {
		var prod product.Product
		if err := s.db.First(&prod, l.ProductID).Error; err != nil {
			return nil, fmt.Errorf("produk ID %d tidak ditemukan", l.ProductID)
		}

		priceUnit := l.PriceUnit
		if priceUnit <= 0 {
			priceUnit = prod.StandardPrice
		}
		subtotal := l.ProductQty * priceUnit
		totalEst += subtotal

		desc := l.Description
		if desc == "" {
			desc = prod.Name
		}

		doc.Lines = append(doc.Lines, PurchaseRequisitionLine{
			ProductID:         l.ProductID,
			Description:       desc,
			ProductQty:        l.ProductQty,
			PriceUnit:         priceUnit,
			Subtotal:          subtotal,
			AccountAnalyticID: l.AccountAnalyticID,
		})
	}
	doc.TotalAmount = totalEst

	if err := s.repo.Create(doc); err != nil {
		return nil, fmt.Errorf("gagal menyimpan purchase requisition: %w", err)
	}

	// Audit Chatter
	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"purchase.requisition",
			doc.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Dokumen '%s' (%s) berhasil dibuat dengan estimasi total Rp %.2f", doc.Name, doc.Type, doc.TotalAmount),
			"state:draft",
			companyID,
		)
	}

	return s.GetByID(doc.ID, companyID)
}

// ActionConfirm approves internal requisition from draft to confirmed
func (s *Service) ActionConfirm(id, companyID, userID uint) (*PurchaseRequisition, error) {
	doc, err := s.repo.GetByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen purchase requisition tidak ditemukan")
	}
	if doc.State != StateDraft {
		return nil, fmt.Errorf("hanya dokumen draf yang dapat dikonfirmasi, status saat ini: %s", doc.State)
	}

	doc.State = StateConfirmed
	if err := s.repo.Update(doc); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"purchase.requisition",
			doc.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Permintaan pembelian '%s' disetujui atasan. Siap untuk proses tender/bidding vendor.", doc.Name),
			"state:confirmed",
			companyID,
		)
	}

	return doc, nil
}

// ActionOpenTender opens competitive bidding process
func (s *Service) ActionOpenTender(id, companyID, userID uint) (*PurchaseRequisition, error) {
	doc, err := s.repo.GetByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen purchase requisition tidak ditemukan")
	}
	if doc.State != StateConfirmed {
		return nil, fmt.Errorf("dokumen harus berstatus 'confirmed' sebelum membuka tender, status saat ini: %s", doc.State)
	}

	doc.State = StateInProgress
	if err := s.repo.Update(doc); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"purchase.requisition",
			doc.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Tender pengadaan terbuka untuk penawaran rekanan (Bidding In Progress)."),
			"state:in_progress",
			companyID,
		)
	}

	return doc, nil
}

// GenerateRFQ generates a RFQ / Purchase Order from the requisition for a specific vendor
func (s *Service) GenerateRFQ(reqID uint, req GenerateRFQRequest, companyID, userID uint) (*purchase.PurchaseOrder, error) {
	doc, err := s.repo.GetByID(reqID, companyID)
	if err != nil {
		return nil, errors.New("dokumen purchase requisition tidak ditemukan")
	}

	if doc.State != StateConfirmed && doc.State != StateInProgress {
		return nil, fmt.Errorf("RFQ hanya dapat diterbitkan pada status 'confirmed' atau 'in_progress', status saat ini: %s", doc.State)
	}

	// Auto-promote to in_progress if confirmed
	if doc.State == StateConfirmed {
		doc.State = StateInProgress
		_ = s.repo.Update(doc)
	}

	var vendor base.Partner
	if err := s.db.First(&vendor, req.VendorID).Error; err != nil {
		return nil, fmt.Errorf("vendor ID %d tidak ditemukan", req.VendorID)
	}

	// Create RFQ
	po := &purchase.PurchaseOrder{
		PartnerID:     req.VendorID,
		RequisitionID: &reqID,
		Note:          req.Notes,
		State:         "draft",
	}
	po.CompanyID = companyID

	for _, line := range doc.Lines {
		po.OrderLines = append(po.OrderLines, purchase.PurchaseOrderLine{
			ProductID:     line.ProductID,
			Name:          line.Description,
			ProductUomQty: line.ProductQty,
			PriceUnit:     line.PriceUnit,
		})
	}

	if err := s.purchaseSvc.CreateOrder(po); err != nil {
		return nil, fmt.Errorf("gagal membuat RFQ vendor: %w", err)
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"purchase.requisition",
			doc.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("RFQ '%s' berhasil diterbitkan untuk vendor '%s' (Total Estimasi: Rp %.2f)", po.Name, vendor.Name, po.AmountTotal),
			fmt.Sprintf("rfq_created:%s", po.Name),
			companyID,
		)
	}

	return po, nil
}

// SelectWinningBid closes the tender, approves the winning PO, and cancels competitor bids
func (s *Service) SelectWinningBid(reqID, winningPOID, companyID, userID uint) (*PurchaseRequisition, error) {
	doc, err := s.repo.GetByID(reqID, companyID)
	if err != nil {
		return nil, errors.New("dokumen purchase requisition tidak ditemukan")
	}

	if doc.State != StateInProgress {
		return nil, fmt.Errorf("pemilihan tender hanya dapat dilakukan saat status 'in_progress', status saat ini: %s", doc.State)
	}

	// Verify winning PO
	winningPO, err := s.purchaseSvc.GetOrder(winningPOID, companyID)
	if err != nil || winningPO.RequisitionID == nil || *winningPO.RequisitionID != reqID {
		return nil, errors.New("PO pemenang tidak valid atau tidak terdaftar pada tender ini")
	}

	// 1. Confirm winning PO into approved Purchase Order + generate double-entry stock receipts
	if winningPO.State == "draft" || winningPO.State == "sent" {
		if _, _, err := s.purchaseSvc.ActionConfirm(winningPOID, companyID); err != nil {
			return nil, fmt.Errorf("gagal mengonfirmasi PO pemenang '%s': %w", winningPO.Name, err)
		}
	}

	// 2. Automatically cancel other competing RFQs for this requisition
	var competingPOs []purchase.PurchaseOrder
	s.db.Where("requisition_id = ? AND id != ? AND state IN ?", reqID, winningPOID, []string{"draft", "sent"}).
		Find(&competingPOs)

	for _, comp := range competingPOs {
		comp.State = "cancel"
		s.db.Save(&comp)
	}

	// 3. Mark Requisition as done and update lines
	doc.SelectedPOID = &winningPOID
	doc.State = StateDone
	if doc.VendorID == nil {
		doc.VendorID = &winningPO.PartnerID
	}

	for i := range doc.Lines {
		doc.Lines[i].QtyOrdered = doc.Lines[i].ProductQty
		_ = s.repo.UpdateLine(&doc.Lines[i])
	}
	_ = s.repo.Update(doc)

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"purchase.requisition",
			doc.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Tender ditutup! Vendor '%s' terpilih sebagai pemenang melalui PO '%s'. %d penawaran rekanan lainnya otomatis dibatalkan.",
				winningPO.Partner.Name, winningPO.Name, len(competingPOs)),
			"bid_selected:done",
			companyID,
		)
	}

	return s.GetByID(doc.ID, companyID)
}

// ReleaseBlanketPO releases partial or full PO releases against an approved Blanket Order agreement
func (s *Service) ReleaseBlanketPO(reqID uint, req ReleaseBlanketPORequest, companyID, userID uint) (*purchase.PurchaseOrder, error) {
	doc, err := s.repo.GetByID(reqID, companyID)
	if err != nil {
		return nil, errors.New("dokumen blanket order tidak ditemukan")
	}

	if doc.Type != TypeBlanketOrder {
		return nil, errors.New("dokumen ini bukan bertipe Blanket Order")
	}

	if doc.State != StateConfirmed && doc.State != StateInProgress {
		return nil, fmt.Errorf("rilis blanket order hanya dapat dilakukan pada status 'confirmed' atau 'in_progress', status saat ini: %s", doc.State)
	}

	vendorID := doc.VendorID
	if req.VendorID > 0 {
		vendorID = &req.VendorID
	}
	if vendorID == nil || *vendorID == 0 {
		return nil, errors.New("rekanan (vendor) untuk blanket order belum ditentukan")
	}

	po := &purchase.PurchaseOrder{
		PartnerID:     *vendorID,
		RequisitionID: &reqID,
		Note:          req.Notes,
		State:         "draft",
	}
	po.CompanyID = companyID

	// Map requested lines and validate quotas
	for _, rLine := range req.Lines {
		var reqLine *PurchaseRequisitionLine
		if rLine.RequisitionLineID > 0 {
			var err error
			reqLine, err = s.repo.GetLineByID(rLine.RequisitionLineID)
			if err != nil || reqLine.RequisitionID != reqID {
				return nil, fmt.Errorf("baris perjanjian ID %d tidak ditemukan", rLine.RequisitionLineID)
			}
		} else if rLine.ProductID > 0 {
			for i := range doc.Lines {
				if doc.Lines[i].ProductID == rLine.ProductID {
					reqLine = &doc.Lines[i]
					break
				}
			}
			if reqLine == nil {
				return nil, fmt.Errorf("produk ID %d tidak ditemukan pada perjanjian ini", rLine.ProductID)
			}
		} else {
			return nil, errors.New("identitas baris perjanjian atau produk wajib disertakan")
		}

		remainingQuota := reqLine.ProductQty - reqLine.QtyOrdered
		if rLine.Qty > remainingQuota {
			return nil, fmt.Errorf("kuantitas rilis (%.2f) melebihi sisa kuota blanket order (%.2f) untuk produk '%s'",
				rLine.Qty, remainingQuota, reqLine.Product.Name)
		}

		priceUnit := reqLine.PriceUnit
		if rLine.PriceUnit != nil && *rLine.PriceUnit > 0 {
			priceUnit = *rLine.PriceUnit
		}

		po.OrderLines = append(po.OrderLines, purchase.PurchaseOrderLine{
			ProductID:     reqLine.ProductID,
			Name:          reqLine.Description,
			ProductUomQty: rLine.Qty,
			PriceUnit:     priceUnit,
		})

		// Increment released quota
		reqLine.QtyOrdered += rLine.Qty
		_ = s.repo.UpdateLine(reqLine)
	}

	// Create and confirm the PO release
	if err := s.purchaseSvc.CreateOrder(po); err != nil {
		return nil, fmt.Errorf("gagal membuat PO release: %w", err)
	}

	// Check if all lines are fully fulfilled
	allFulfilled := true
	lines, _ := s.repo.GetByID(reqID, companyID)
	for _, l := range lines.Lines {
		if l.QtyOrdered < l.ProductQty {
			allFulfilled = false
			break
		}
	}

	if allFulfilled {
		doc.State = StateDone
	} else {
		doc.State = StateInProgress
	}
	_ = s.repo.Update(doc)

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"purchase.requisition",
			doc.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Rilis Blanket PO '%s' diterbitkan senilai Rp %.2f.", po.Name, po.AmountTotal),
			fmt.Sprintf("blanket_released:%s", po.Name),
			companyID,
		)
	}

	return po, nil
}

// ActionCancel cancels requisition and its draft RFQs
func (s *Service) ActionCancel(id, companyID, userID uint) (*PurchaseRequisition, error) {
	doc, err := s.repo.GetByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen purchase requisition tidak ditemukan")
	}

	if doc.State == StateDone {
		return nil, errors.New("dokumen yang sudah selesai (done) tidak dapat dibatalkan")
	}

	doc.State = StateCancelled
	if err := s.repo.Update(doc); err != nil {
		return nil, err
	}

	// Cancel related draft POs
	s.db.Model(&purchase.PurchaseOrder{}).
		Where("requisition_id = ? AND state IN ?", id, []string{"draft", "sent"}).
		Update("state", "cancel")

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"purchase.requisition",
			doc.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Dokumen '%s' dibatalkan.", doc.Name),
			"state:cancel",
			companyID,
		)
	}

	return doc, nil
}

// ActionResetDraft resets cancelled document back to draft
func (s *Service) ActionResetDraft(id, companyID, userID uint) (*PurchaseRequisition, error) {
	doc, err := s.repo.GetByID(id, companyID)
	if err != nil {
		return nil, errors.New("dokumen purchase requisition tidak ditemukan")
	}

	doc.State = StateDraft
	if err := s.repo.Update(doc); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"purchase.requisition",
			doc.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Status dokumen '%s' dikembalikan ke Draf.", doc.Name),
			"state:draft",
			companyID,
		)
	}

	return doc, nil
}

func (s *Service) GetByID(id, companyID uint) (*PurchaseRequisition, error) {
	return s.repo.GetByID(id, companyID)
}

func (s *Service) List(companyID uint, state, reqType, search string) ([]PurchaseRequisition, error) {
	return s.repo.List(companyID, state, reqType, search)
}

func (s *Service) GetStats(companyID uint) (*RequisitionStats, error) {
	return s.repo.GetStats(companyID)
}
