package vendor_portal

import (
	"fmt"
	"time"

	"erp-golang/internal/modules/purchase"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(
		&VendorPortalAccount{},
		&VendorQuotationBid{},
		&VendorQuotationBidLine{},
		&VendorASN{},
		&VendorASNLine{},
		&VendorInvoiceSubmission{},
	)
}

// Sequence generators
func (r *Repository) GenerateBidNumber() (string, error) {
	year := time.Now().Format("2006")
	var count int64
	err := r.db.Model(&VendorQuotationBid{}).
		Where("bid_number LIKE ?", fmt.Sprintf("BID/%s/%%", year)).
		Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("BID/%s/%05d", year, count+1), nil
}

func (r *Repository) GenerateASNNumber() (string, error) {
	year := time.Now().Format("2006")
	var count int64
	err := r.db.Model(&VendorASN{}).
		Where("asn_number LIKE ?", fmt.Sprintf("ASN/%s/%%", year)).
		Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("ASN/%s/%05d", year, count+1), nil
}

func (r *Repository) GenerateInvoiceSubmissionNumber() (string, error) {
	year := time.Now().Format("2006")
	var count int64
	err := r.db.Model(&VendorInvoiceSubmission{}).
		Where("submission_number LIKE ?", fmt.Sprintf("VINV/%s/%%", year)).
		Count(&count).Error
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("VINV/%s/%05d", year, count+1), nil
}

// Vendor Portal Accounts
func (r *Repository) CreateAccount(account *VendorPortalAccount) error {
	return r.db.Create(account).Error
}

func (r *Repository) GetAccountByID(id uint) (*VendorPortalAccount, error) {
	var acc VendorPortalAccount
	if err := r.db.First(&acc, id).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) GetAccountByToken(token string) (*VendorPortalAccount, error) {
	var acc VendorPortalAccount
	if err := r.db.Where("access_token = ? AND is_active = ?", token, true).First(&acc).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) GetAccountByPartnerID(partnerID uint) (*VendorPortalAccount, error) {
	var acc VendorPortalAccount
	if err := r.db.Where("partner_id = ?", partnerID).First(&acc).Error; err != nil {
		return nil, err
	}
	return &acc, nil
}

func (r *Repository) ListAccounts() ([]VendorPortalAccount, error) {
	var accounts []VendorPortalAccount
	if err := r.db.Order("id DESC").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (r *Repository) UpdateAccountLogin(id uint) error {
	now := time.Now()
	return r.db.Model(&VendorPortalAccount{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_login_at": &now,
			"total_logins":  gorm.Expr("total_logins + 1"),
		}).Error
}

// Vendor Quotation Bids
func (r *Repository) CreateBid(bid *VendorQuotationBid) error {
	return r.db.Create(bid).Error
}

func (r *Repository) GetBidByID(id uint) (*VendorQuotationBid, error) {
	var bid VendorQuotationBid
	if err := r.db.Preload("Lines").First(&bid, id).Error; err != nil {
		return nil, err
	}
	return &bid, nil
}

func (r *Repository) ListBids() ([]VendorQuotationBid, error) {
	var bids []VendorQuotationBid
	if err := r.db.Preload("Lines").Order("id DESC").Find(&bids).Error; err != nil {
		return nil, err
	}
	return bids, nil
}

func (r *Repository) ListBidsByPartnerID(partnerID uint) ([]VendorQuotationBid, error) {
	var bids []VendorQuotationBid
	if err := r.db.Preload("Lines").Where("partner_id = ?", partnerID).Order("id DESC").Find(&bids).Error; err != nil {
		return nil, err
	}
	return bids, nil
}

func (r *Repository) UpdateBidStatus(id uint, status string) error {
	return r.db.Model(&VendorQuotationBid{}).Where("id = ?", id).Update("status", status).Error
}

// Vendor ASNs
func (r *Repository) CreateASN(asn *VendorASN) error {
	return r.db.Create(asn).Error
}

func (r *Repository) GetASNByID(id uint) (*VendorASN, error) {
	var asn VendorASN
	if err := r.db.Preload("Lines").First(&asn, id).Error; err != nil {
		return nil, err
	}
	return &asn, nil
}

func (r *Repository) ListASNs() ([]VendorASN, error) {
	var asns []VendorASN
	if err := r.db.Preload("Lines").Order("id DESC").Find(&asns).Error; err != nil {
		return nil, err
	}
	return asns, nil
}

func (r *Repository) ListASNsByPartnerID(partnerID uint) ([]VendorASN, error) {
	var asns []VendorASN
	if err := r.db.Preload("Lines").Where("partner_id = ?", partnerID).Order("id DESC").Find(&asns).Error; err != nil {
		return nil, err
	}
	return asns, nil
}

func (r *Repository) UpdateASNStatus(id uint, status string, actualArrival *time.Time) error {
	updates := map[string]interface{}{"status": status}
	if actualArrival != nil {
		updates["actual_arrival_date"] = actualArrival
	}
	return r.db.Model(&VendorASN{}).Where("id = ?", id).Updates(updates).Error
}

func (r *Repository) UpdateASNLineReceived(lineID uint, qtyReceived float64) error {
	return r.db.Model(&VendorASNLine{}).Where("id = ?", lineID).Update("quantity_received", qtyReceived).Error
}

// Vendor Invoices
func (r *Repository) CreateInvoiceSubmission(inv *VendorInvoiceSubmission) error {
	return r.db.Create(inv).Error
}

func (r *Repository) GetInvoiceSubmissionByID(id uint) (*VendorInvoiceSubmission, error) {
	var inv VendorInvoiceSubmission
	if err := r.db.First(&inv, id).Error; err != nil {
		return nil, err
	}
	return &inv, nil
}

func (r *Repository) ListInvoiceSubmissions() ([]VendorInvoiceSubmission, error) {
	var invoices []VendorInvoiceSubmission
	if err := r.db.Order("id DESC").Find(&invoices).Error; err != nil {
		return nil, err
	}
	return invoices, nil
}

func (r *Repository) ListInvoiceSubmissionsByPartnerID(partnerID uint) ([]VendorInvoiceSubmission, error) {
	var invoices []VendorInvoiceSubmission
	if err := r.db.Where("partner_id = ?", partnerID).Order("id DESC").Find(&invoices).Error; err != nil {
		return nil, err
	}
	return invoices, nil
}

func (r *Repository) UpdateInvoiceMatching(id uint, matchingStatus, discrepancyNotes, status string, billID *uint) error {
	updates := map[string]interface{}{
		"matching_status":   matchingStatus,
		"discrepancy_notes": discrepancyNotes,
		"status":            status,
	}
	if billID != nil {
		updates["account_move_id"] = *billID
	}
	return r.db.Model(&VendorInvoiceSubmission{}).Where("id = ?", id).Updates(updates).Error
}

// Queries for RFQs and POs
func (r *Repository) ListOpenRFQsForVendor(partnerID uint) ([]VendorPortalRFQView, error) {
	var pos []purchase.PurchaseOrder
	query := r.db.Preload("OrderLines").Where("state IN ?", []string{"draft", "sent"})
	if partnerID > 0 {
		query = query.Where("partner_id = ?", partnerID)
	}
	if err := query.Order("id DESC").Find(&pos).Error; err != nil {
		return nil, err
	}

	var results []VendorPortalRFQView
	for _, po := range pos {
		var existingBid VendorQuotationBid
		hasBid := r.db.Where("purchase_order_id = ? AND partner_id = ?", po.ID, partnerID).First(&existingBid).Error == nil

		bidNum := ""
		if hasBid {
			bidNum = existingBid.BidNumber
		}

		results = append(results, VendorPortalRFQView{
			ID:             po.ID,
			Name:           po.Name,
			DateOrder:      po.DateOrder,
			State:          po.State,
			Notes:          po.Note,
			Lines:          po.OrderLines,
			AlreadyBid:     hasBid,
			ExistingBidNum: bidNum,
		})
	}
	return results, nil
}

func (r *Repository) ListConfirmedPOsForVendor(partnerID uint) ([]purchase.PurchaseOrder, error) {
	var pos []purchase.PurchaseOrder
	query := r.db.Preload("OrderLines").Where("state = ?", "purchase")
	if partnerID > 0 {
		query = query.Where("partner_id = ?", partnerID)
	}
	if err := query.Order("id DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	return pos, nil
}

func (r *Repository) GetStats() (*VendorPortalStats, error) {
	var stats VendorPortalStats

	r.db.Model(&VendorPortalAccount{}).Count(&stats.TotalVendorAccounts)
	r.db.Model(&VendorPortalAccount{}).Where("is_active = ?", true).Count(&stats.ActiveVendorAccounts)
	r.db.Model(&VendorQuotationBid{}).Count(&stats.TotalBidsSubmitted)
	r.db.Model(&VendorQuotationBid{}).Where("status = ?", "accepted").Count(&stats.AcceptedBids)
	r.db.Model(&VendorASN{}).Where("status = ?", "in_transit").Count(&stats.TotalASNsInTransit)
	r.db.Model(&VendorASN{}).Where("status = ?", "received").Count(&stats.ReceivedASNs)
	r.db.Model(&VendorInvoiceSubmission{}).Where("matching_status = ?", "pending_match").Count(&stats.PendingInvoiceMatches)
	r.db.Model(&VendorInvoiceSubmission{}).Where("matching_status = ?", "matched_3way").Count(&stats.Matched3WayInvoices)

	var totalBilled float64
	r.db.Model(&VendorInvoiceSubmission{}).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalBilled)
	stats.TotalBilledAmount = totalBilled

	return &stats, nil
}
