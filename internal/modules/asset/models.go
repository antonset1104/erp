package asset

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/base"
)

// Asset corresponds to standard enterprise account.asset (Fixed Asset Management)
type Asset struct {
	models.ModelBase
	Name                  string                  `gorm:"size:255;not null" json:"name"`
	Code                  string                  `gorm:"size:64;not null;uniqueIndex" json:"code"`
	Category              string                  `gorm:"size:64;default:'general'" json:"category"`
	PartnerID             *uint                   `json:"partner_id,omitempty"`
	Partner               *base.Partner           `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	PurchaseDate          string                  `gorm:"size:32;not null" json:"purchase_date"`
	OriginalValue         float64                 `gorm:"not null" json:"original_value"`
	SalvageValue          float64                 `gorm:"default:0" json:"salvage_value"`
	SalvableValue         float64                 `gorm:"default:0" json:"salvable_value"`
	Method                string                  `gorm:"size:32;default:'linear'" json:"method"` // "linear", "degressive"
	MethodNumber          int                     `gorm:"default:36" json:"method_number"`        // Number of periods (e.g. 36 months)
	MethodPeriod          string                  `gorm:"size:16;default:'months'" json:"method_period"`
	AssetAccountID        uint                    `gorm:"not null" json:"asset_account_id"`
	AssetAccount          *account.AccountAccount `gorm:"foreignKey:AssetAccountID" json:"asset_account,omitempty"`
	DepreciationAccountID uint                    `gorm:"not null" json:"depreciation_account_id"`
	DepreciationAccount   *account.AccountAccount `gorm:"foreignKey:DepreciationAccountID" json:"depreciation_account,omitempty"`
	AccumulatedAccountID  uint                    `gorm:"not null" json:"accumulated_account_id"`
	AccumulatedAccount    *account.AccountAccount `gorm:"foreignKey:AccumulatedAccountID" json:"accumulated_account,omitempty"`
	JournalID             uint                    `gorm:"not null" json:"journal_id"`
	Journal               *account.AccountJournal `gorm:"foreignKey:JournalID" json:"journal,omitempty"`
	State                 string                  `gorm:"size:32;default:'draft';index" json:"state"` // "draft", "running", "close"
	BookValue             float64                 `gorm:"default:0" json:"book_value"`
	ValueDepreciated      float64                 `gorm:"default:0" json:"value_depreciated"`
	DepreciationLines     []AssetDepreciationLine `gorm:"foreignKey:AssetID" json:"depreciation_lines"`
	Note                  string                  `gorm:"type:text" json:"note"`
}

// AssetDepreciationLine corresponds to standard enterprise account.asset.depreciation.line
type AssetDepreciationLine struct {
	models.ModelBase
	AssetID          uint                 `gorm:"not null;index" json:"asset_id"`
	Sequence         int                  `gorm:"not null" json:"sequence"`
	DepreciationDate string               `gorm:"size:32;not null" json:"depreciation_date"`
	Amount           float64              `gorm:"not null" json:"amount"`
	DepreciatedValue float64              `gorm:"not null" json:"depreciated_value"`
	RemainingValue   float64              `gorm:"not null" json:"remaining_value"`
	IsPosted         bool                 `gorm:"default:false;index" json:"is_posted"`
	MoveID           *uint                `json:"move_id,omitempty"`
	Move             *account.AccountMove `gorm:"foreignKey:MoveID" json:"move,omitempty"`
}
