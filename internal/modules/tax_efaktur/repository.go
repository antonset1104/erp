package tax_efaktur

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// CreateNSFP stores a new NSFP quota range
func (r *Repository) CreateNSFP(nsfp *TaxNSFP) error {
	return r.db.Create(nsfp).Error
}

// ListNSFP retrieves all NSFP ranges
func (r *Repository) ListNSFP(companyID uint) ([]TaxNSFP, error) {
	var list []TaxNSFP
	err := r.db.Where("company_id = ?", companyID).Order("year desc, id desc").Find(&list).Error
	return list, err
}

// GetActiveNSFP finds the current usable NSFP range
func (r *Repository) GetActiveNSFP(prefix string, year int, companyID uint) (*TaxNSFP, error) {
	var nsfp TaxNSFP
	err := r.db.Where("company_id = ? AND prefix = ? AND year = ? AND state = ?",
		companyID, prefix, year, "active").
		Order("id asc").First(&nsfp).Error
	if err != nil {
		return nil, err
	}
	return &nsfp, nil
}

// AllocateNextNSFP atomically generates and claims the next sequential NSFP number
func (r *Repository) AllocateNextNSFP(prefix string, year int, companyID uint) (string, error) {
	var fullNSFP string

	err := r.db.Transaction(func(tx *gorm.DB) error {
		var nsfp TaxNSFP
		err := tx.Where("company_id = ? AND prefix = ? AND year = ? AND state = ?",
			companyID, prefix, year, "active").
			Order("id asc").First(&nsfp).Error
		if err != nil {
			return errors.New("tidak ada jatah Nomor Seri Faktur Pajak (NSFP) aktif untuk tahun dan prefix ini. Silakan input jatah NSFP baru dari DJP")
		}

		if nsfp.RemainingQuota <= 0 {
			nsfp.State = "depleted"
			tx.Save(&nsfp)
			return errors.New("jatah NSFP telah habis. Silakan daftarkan surat alokasi nomor faktur baru dari DJP")
		}

		currNumInt, err := strconv.ParseInt(nsfp.CurrentNumber, 10, 64)
		if err != nil {
			return fmt.Errorf("format CurrentNumber NSFP tidak valid: %v", err)
		}

		// Format standard DJP 16-digit: "010.001-26.00000001"
		fullNSFP = fmt.Sprintf("%s.001-%02d.%08d", nsfp.Prefix, nsfp.Year%100, currNumInt)

		// Increment sequence
		nextNumInt := currNumInt + 1
		nsfp.CurrentNumber = fmt.Sprintf("%08d", nextNumInt)
		nsfp.UsedQuota++
		nsfp.RemainingQuota--

		if nsfp.RemainingQuota <= 0 {
			nsfp.State = "depleted"
		}

		return tx.Save(&nsfp).Error
	})

	return fullNSFP, err
}

// CreateTaxInvoice creates a tax invoice document with its line items
func (r *Repository) CreateTaxInvoice(inv *TaxInvoice) error {
	return r.db.Create(inv).Error
}

// GetTaxInvoiceByID fetches invoice by primary key with relations
func (r *Repository) GetTaxInvoiceByID(id uint, companyID uint) (*TaxInvoice, error) {
	var inv TaxInvoice
	err := r.db.Preload("Partner").Preload("Move").Preload("Lines").
		Where("id = ? AND company_id = ?", id, companyID).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// GetTaxInvoiceByMoveID fetches tax invoice linked to commercial invoice
func (r *Repository) GetTaxInvoiceByMoveID(moveID uint, companyID uint) (*TaxInvoice, error) {
	var inv TaxInvoice
	err := r.db.Preload("Partner").Preload("Lines").
		Where("move_id = ? AND company_id = ?", moveID, companyID).First(&inv).Error
	if err != nil {
		return nil, err
	}
	return &inv, nil
}

// ListTaxInvoices retrieves tax invoices with filtering
func (r *Repository) ListTaxInvoices(taxType string, period int, year int, state string, companyID uint) ([]TaxInvoice, error) {
	var list []TaxInvoice
	query := r.db.Preload("Partner").Preload("Lines").Where("company_id = ?", companyID)

	if taxType != "" {
		query = query.Where("tax_invoice_type = ?", taxType)
	}
	if period > 0 {
		query = query.Where("tax_period = ?", period)
	}
	if year > 0 {
		query = query.Where("tax_year = ?", year)
	}
	if state != "" {
		query = query.Where("state = ?", state)
	}

	err := query.Order("invoice_date desc, id desc").Find(&list).Error
	return list, err
}

// UpdateTaxInvoiceState updates document state
func (r *Repository) UpdateTaxInvoiceState(id uint, state string, companyID uint) error {
	return r.db.Model(&TaxInvoice{}).
		Where("id = ? AND company_id = ?", id, companyID).
		Update("state", state).Error
}

// GetStats calculates executive tax analytics
func (r *Repository) GetStats(companyID uint) (*TaxStats, error) {
	now := time.Now()
	currentPeriod := int(now.Month())
	currentYear := now.Year()

	stats := &TaxStats{
		CurrentTaxPeriod: currentPeriod,
		CurrentTaxYear:   currentYear,
	}

	// NSFP Quotas
	var totalAllocated, remainingQuota int64
	r.db.Model(&TaxNSFP{}).Where("company_id = ?", companyID).Select("COALESCE(SUM(total_quota), 0)").Scan(&totalAllocated)
	r.db.Model(&TaxNSFP{}).Where("company_id = ? AND state = 'active'", companyID).Select("COALESCE(SUM(remaining_quota), 0)").Scan(&remainingQuota)
	stats.TotalNSFPAllocated = int(totalAllocated)
	stats.RemainingNSFPQuota = int(remainingQuota)

	// Invoices count and totals (excluding cancelled)
	var outCount, inCount int64
	r.db.Model(&TaxInvoice{}).Where("company_id = ? AND tax_invoice_type = 'out_tax' AND state != 'cancelled'", companyID).Count(&outCount)
	r.db.Model(&TaxInvoice{}).Where("company_id = ? AND tax_invoice_type = 'in_tax' AND state != 'cancelled'", companyID).Count(&inCount)
	stats.TotalOutTaxInvoices = int(outCount)
	stats.TotalInTaxInvoices = int(inCount)

	// PPN Totals
	r.db.Model(&TaxInvoice{}).Where("company_id = ? AND tax_invoice_type = 'out_tax' AND state != 'cancelled'", companyID).
		Select("COALESCE(SUM(dpp), 0) as total_dpp, COALESCE(SUM(ppn), 0) as total_ppn").
		Row().Scan(&stats.TotalDPPOutTax, &stats.TotalPPNKeluaran)

	r.db.Model(&TaxInvoice{}).Where("company_id = ? AND tax_invoice_type = 'in_tax' AND state != 'cancelled'", companyID).
		Select("COALESCE(SUM(dpp), 0) as total_dpp, COALESCE(SUM(ppn), 0) as total_ppn").
		Row().Scan(&stats.TotalDPPInTax, &stats.TotalPPNMasukan)

	stats.NetPPNKurangBayar = stats.TotalPPNKeluaran - stats.TotalPPNMasukan

	return stats, nil
}

// GetSPTPeriodSummary computes PPN SPT 1111 periodic fiscal summary
func (r *Repository) GetSPTPeriodSummary(period int, year int, companyID uint) (*SPTPeriodSummary, error) {
	summary := &SPTPeriodSummary{
		TaxPeriod: period,
		TaxYear:   year,
	}

	// Faktur Keluaran (Penyerahan BKP/JKP)
	var outCount int64
	r.db.Model(&TaxInvoice{}).
		Where("company_id = ? AND tax_period = ? AND tax_year = ? AND tax_invoice_type = 'out_tax' AND state != 'cancelled'", companyID, period, year).
		Count(&outCount)
	summary.TotalOutTaxInvoices = int(outCount)

	r.db.Model(&TaxInvoice{}).
		Where("company_id = ? AND tax_period = ? AND tax_year = ? AND tax_invoice_type = 'out_tax' AND state != 'cancelled'", companyID, period, year).
		Select("COALESCE(SUM(dpp), 0), COALESCE(SUM(ppn), 0)").
		Row().Scan(&summary.DPPPenyerahanKeluaran, &summary.PPNKeluaranDipungut)

	// Faktur Masukan (Perolehan BKP/JKP)
	var inCount int64
	r.db.Model(&TaxInvoice{}).
		Where("company_id = ? AND tax_period = ? AND tax_year = ? AND tax_invoice_type = 'in_tax' AND state != 'cancelled'", companyID, period, year).
		Count(&inCount)
	summary.TotalInTaxInvoices = int(inCount)

	r.db.Model(&TaxInvoice{}).
		Where("company_id = ? AND tax_period = ? AND tax_year = ? AND tax_invoice_type = 'in_tax' AND state != 'cancelled'", companyID, period, year).
		Select("COALESCE(SUM(dpp), 0), COALESCE(SUM(ppn), 0)").
		Row().Scan(&summary.DPPPerolehanMasukan, &summary.PPNMasukanDikreditkan)

	summary.PPNKurangLebihBayar = summary.PPNKeluaranDipungut - summary.PPNMasukanDikreditkan
	if summary.PPNKurangLebihBayar > 0 {
		summary.StatusFiskal = "Kurang Bayar"
	} else if summary.PPNKurangLebihBayar < 0 {
		summary.StatusFiskal = "Lebih Bayar"
	} else {
		summary.StatusFiskal = "Nihil"
	}

	return summary, nil
}
