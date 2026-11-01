package base

import (
	"erp-golang/internal/core/models"
)

// Company corresponds to Odoo's res.company
type Company struct {
	models.ModelBase
	Name       string `gorm:"size:255;not null" json:"name"`
	Email      string `gorm:"size:128" json:"email"`
	Phone      string `gorm:"size:64" json:"phone"`
	Street     string `gorm:"size:255" json:"street"`
	City       string `gorm:"size:128" json:"city"`
	Country    string `gorm:"size:64;default:'Indonesia'" json:"country"`
	CurrencyID uint   `gorm:"default:1" json:"currency_id"`
	Currency   string `gorm:"size:10;default:'IDR'" json:"currency"`
	TaxID      string `gorm:"size:64" json:"tax_id"`
}

// Currency corresponds to Odoo's res.currency
type Currency struct {
	models.ModelBase
	Name   string  `gorm:"size:10;not null;unique" json:"name"`   // IDR, USD, EUR
	Symbol string  `gorm:"size:10;not null" json:"symbol"`        // Rp, $, €
	Rate   float64 `gorm:"default:1.0" json:"rate"`               // Exchange rate against base
	Digits int     `gorm:"default:2" json:"digits"`
}

// User corresponds to Odoo's res.users
type User struct {
	models.ModelBase
	Name         string `gorm:"size:128;not null" json:"name"`
	Email        string `gorm:"size:128;not null;uniqueIndex" json:"email"`
	PasswordHash string `gorm:"size:255;not null" json:"-"`
	Role         string `gorm:"size:32;default:'user'" json:"role"` // admin, manager, user
	PartnerID    uint   `json:"partner_id"`
}

// Partner corresponds to Odoo's res.partner (Customers, Vendors, Contacts)
type Partner struct {
	models.ModelBase
	Name         string `gorm:"size:255;not null;index" json:"name"`
	IsCompany    bool   `gorm:"default:false" json:"is_company"`
	ParentID     *uint  `json:"parent_id,omitempty"` // For contacts belonging to a company
	Email        string `gorm:"size:128;index" json:"email"`
	Phone        string `gorm:"size:64" json:"phone"`
	Mobile       string `gorm:"size:64" json:"mobile"`
	Street       string `gorm:"size:255" json:"street"`
	City         string `gorm:"size:128" json:"city"`
	State        string `gorm:"size:128" json:"state"`
	Zip          string `gorm:"size:32" json:"zip"`
	Country      string `gorm:"size:64;default:'Indonesia'" json:"country"`
	CustomerRank int    `gorm:"default:0" json:"customer_rank"` // >0 if customer
	SupplierRank int    `gorm:"default:0" json:"supplier_rank"` // >0 if supplier
	TaxID        string `gorm:"size:64" json:"tax_id"`
	Website      string `gorm:"size:255" json:"website"`
	CreditLimit  float64 `gorm:"default:0" json:"credit_limit"`
}
