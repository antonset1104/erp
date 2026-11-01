package landed_cost

import (
	"time"

	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/stock"

	"gorm.io/gorm"
)

// SplitMethod represents the algorithm used to allocate landed cost lines
type SplitMethod string

const (
	SplitByQuantity SplitMethod = "by_quantity" // Alokasi proporsional berdasarkan kuantitas fisik
	SplitByValue    SplitMethod = "by_value"    // Alokasi proporsional berdasarkan nilai/harga beli
	SplitByWeight   SplitMethod = "by_weight"   // Alokasi proporsional berdasarkan bobot kotor (kg)
	SplitByVolume   SplitMethod = "by_volume"   // Alokasi proporsional berdasarkan kubikasi/volume (m3)
	SplitEqual      SplitMethod = "equal"       // Alokasi dibagi rata sama banyak antar produk
)

// LandedCostState represents the lifecycle state of a landed cost document
type LandedCostState string

const (
	StateDraft    LandedCostState = "draft"    // Konfigurasi & biaya komponen diinput
	StateComputed LandedCostState = "computed" // Simulasi alokasi harga pokok telah dihitung
	StateDone     LandedCostState = "done"     // Tervalidasi & jurnal akuntansi terbentuk
	StateCancel   LandedCostState = "cancel"   // Dibatalkan
)

// StockLandedCost represents Odoo's stock.landed.cost (Alokasi Biaya Tambahan Pengadaan & Impor)
type StockLandedCost struct {
	ID               uint            `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"-"`
	Name             string          `gorm:"type:varchar(64);uniqueIndex;not null" json:"name"` // e.g. LC/2026/0001
	Date             string          `gorm:"type:varchar(32);not null" json:"date"`             // Tanggal alokasi
	PickingID        uint            `gorm:"index;not null" json:"picking_id"`                  // Penerimaan barang terkait (Incoming Receipt)
	Picking          *stock.StockPicking `gorm:"foreignKey:PickingID" json:"picking,omitempty"`
	VendorBillID     *uint           `json:"vendor_bill_id,omitempty"`                          // Tagihan biaya vendor eksternal (optional)
	AccountMoveID    *uint           `json:"account_move_id,omitempty"`                         // Jurnal penyesuaian nilai persediaan (GL Entry)
	AccountMove      *account.AccountMove `gorm:"foreignKey:AccountMoveID" json:"account_move,omitempty"`
	AccountJournalID uint            `gorm:"default:5" json:"account_journal_id"`               // Default Miscellaneous Journal
	State            LandedCostState `gorm:"type:varchar(32);default:'draft';index" json:"state"`
	CompanyID        uint            `gorm:"default:1" json:"company_id"`
	Description      string          `gorm:"type:text" json:"description"`
	TotalAmount      float64         `gorm:"type:decimal(15,2);default:0" json:"total_amount"` // Total akumulasi biaya landed

	CostLines      []StockLandedCostLine          `gorm:"foreignKey:CostID" json:"cost_lines"`
	ValuationLines []StockValuationAdjustmentLine `gorm:"foreignKey:CostID" json:"valuation_lines"`
}

// StockLandedCostLine represents Odoo's stock.landed.cost.line (Rincian komponen biaya)
type StockLandedCostLine struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
	CostID      uint           `gorm:"not null;index" json:"cost_id"`
	Name        string         `gorm:"type:varchar(128);not null" json:"name"` // misal: Freight, Bea Masuk, THC Pelabuhan
	SplitMethod SplitMethod    `gorm:"type:varchar(32);default:'by_value'" json:"split_method"`
	PriceUnit   float64        `gorm:"type:decimal(15,2);not null;default:0" json:"price_unit"` // Nilai biaya (Rp)
	AccountID   uint           `gorm:"not null;default:7" json:"account_id"`                    // Akun antara biaya (Clearing Account)
	AccountName string         `gorm:"type:varchar(128)" json:"account_name,omitempty"`
}

// StockValuationAdjustmentLine represents Odoo's stock.valuation.adjustment.lines (Hasil simulasi alokasi ke tiap item)
type StockValuationAdjustmentLine struct {
	ID                   uint           `gorm:"primaryKey" json:"id"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
	CostID               uint           `gorm:"not null;index" json:"cost_id"`
	CostLineID           uint           `gorm:"not null;index" json:"cost_line_id"`
	CostLineName         string         `gorm:"type:varchar(128)" json:"cost_line_name"`
	MoveID               uint           `gorm:"not null;index" json:"move_id"`
	ProductID            uint           `gorm:"not null;index" json:"product_id"`
	ProductName          string         `gorm:"type:varchar(255)" json:"product_name"`
	Quantity             float64        `gorm:"type:decimal(12,2);default:0" json:"quantity"`
	Weight               float64        `gorm:"type:decimal(12,2);default:0" json:"weight"`
	Volume               float64        `gorm:"type:decimal(12,4);default:0" json:"volume"`
	FormerCost           float64        `gorm:"type:decimal(15,2);default:0" json:"former_cost"`            // Total harga awal (Qty * Harga Beli)
	FormerCostPerUnit    float64        `gorm:"type:decimal(15,2);default:0" json:"former_cost_per_unit"`   // Harga awal per unit
	AdditionalLandedCost float64        `gorm:"type:decimal(15,2);default:0" json:"additional_landed_cost"` // Beban landed yang dialokasikan
	FinalCost            float64        `gorm:"type:decimal(15,2);default:0" json:"final_cost"`             // Total harga akhir (Former + Additional)
	FinalCostPerUnit     float64        `gorm:"type:decimal(15,2);default:0" json:"final_cost_per_unit"`    // HPP final per unit
}

// DTOs
type CreateLandedCostRequest struct {
	Date        string                 `json:"date"`
	PickingID   uint                   `json:"picking_id"`
	Description string                 `json:"description"`
	CostLines   []CreateCostLineRequest `json:"cost_lines"`
}

type CreateCostLineRequest struct {
	Name        string      `json:"name"`
	SplitMethod SplitMethod `json:"split_method"`
	PriceUnit   float64     `json:"price_unit"`
	AccountID   uint        `json:"account_id"`
}

type LandedCostStats struct {
	TotalLandedCosts   int64   `json:"total_landed_costs"`
	DraftCount         int64   `json:"draft_count"`
	DoneCount          int64   `json:"done_count"`
	TotalCostAllocated float64 `json:"total_cost_allocated"`
	AverageCostRatio   float64 `json:"average_cost_ratio"` // Rasio rata-rata landed cost terhadap nilai barang (%)
}

type AILandedCostReport struct {
	Summary              string            `json:"summary"`
	TotalAdditionalCost  float64           `json:"total_additional_cost"`
	OverallOverheadPct   float64           `json:"overall_overhead_pct"`
	ProductImpacts       []AIProductImpact `json:"product_impacts"`
	Recommendations      []string          `json:"recommendations"`
	RiskLevel            string            `json:"risk_level"`
	AIProvider           string            `json:"ai_provider"`
}

type AIProductImpact struct {
	ProductID               uint    `json:"product_id"`
	ProductName             string  `json:"product_name"`
	Quantity                float64 `json:"quantity"`
	OldCostPerUnit          float64 `json:"old_cost_per_unit"`
	NewCostPerUnit          float64 `json:"new_cost_per_unit"`
	CostIncreasePct         float64 `json:"cost_increase_pct"`
	SellingPrice            float64 `json:"selling_price"`
	EstimatedGrossMarginPct float64 `json:"estimated_gross_margin_pct"`
	MarginStatus            string  `json:"margin_status"` // "HEALTHY", "TIGHT", "NEGATIVE"
}
