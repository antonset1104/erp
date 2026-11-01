package crm

import (
	"errors"
	"fmt"
	"time"

	"erp-golang/internal/modules/base"
	"erp-golang/internal/modules/sale"

	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	saleSvc *sale.Service
}

func NewService(db *gorm.DB, saleSvc *sale.Service) *Service {
	return &Service{db: db, saleSvc: saleSvc}
}

func (s *Service) ListLeads(companyID uint, stage string) ([]CrmLead, error) {
	var leads []CrmLead
	query := s.db.Preload("Partner").Where("company_id = ?", companyID)
	if stage != "" {
		query = query.Where("stage = ?", stage)
	}
	if err := query.Order("id desc").Find(&leads).Error; err != nil {
		return nil, err
	}
	return leads, nil
}

func (s *Service) GetLead(id uint, companyID uint) (*CrmLead, error) {
	var lead CrmLead
	if err := s.db.Preload("Partner").Where("id = ? AND company_id = ?", id, companyID).First(&lead).Error; err != nil {
		return nil, err
	}
	return &lead, nil
}

func (s *Service) CreateLead(lead *CrmLead) error {
	if lead.Stage == "" {
		lead.Stage = "new"
	}
	if lead.Probability == 0 {
		switch lead.Stage {
		case "new":
			lead.Probability = 10
		case "qualified":
			lead.Probability = 35
		case "proposition":
			lead.Probability = 70
		case "won":
			lead.Probability = 100
		case "lost":
			lead.Probability = 0
		}
	}
	return s.db.Create(lead).Error
}

func (s *Service) UpdateStage(id uint, stage string, lostReason string, companyID uint) (*CrmLead, error) {
	lead, err := s.GetLead(id, companyID)
	if err != nil {
		return nil, errors.New("crm lead not found")
	}

	lead.Stage = stage
	switch stage {
	case "new":
		lead.Probability = 10
	case "qualified":
		lead.Probability = 35
	case "proposition":
		lead.Probability = 70
	case "won":
		lead.Probability = 100
	case "lost":
		lead.Probability = 0
		lead.LostReason = lostReason
	}

	if err := s.db.Save(lead).Error; err != nil {
		return nil, err
	}
	return lead, nil
}

// ConvertToQuotation converts a qualified CRM lead into a draft Sales Order in Sale module
func (s *Service) ConvertToQuotation(id uint, companyID uint) (*sale.SaleOrder, *CrmLead, error) {
	lead, err := s.GetLead(id, companyID)
	if err != nil {
		return nil, nil, errors.New("crm lead not found")
	}

	if lead.SaleOrderID != nil {
		return nil, nil, errors.New("lead has already been converted to quotation")
	}

	// 1. Resolve or create customer partner
	partnerID := uint(1)
	if lead.PartnerID != nil && *lead.PartnerID > 0 {
		partnerID = *lead.PartnerID
	} else {
		// Create a customer partner from lead data
		partnerName := lead.PartnerName
		if partnerName == "" {
			partnerName = lead.ContactName
		}
		if partnerName == "" {
			partnerName = fmt.Sprintf("Customer (%s)", lead.Name)
		}

		p := base.Partner{
			Name:         partnerName,
			Email:        lead.Email,
			Phone:        lead.Phone,
			CustomerRank: 1,
			IsCompany:    lead.PartnerName != "",
		}
		p.CompanyID = companyID
		if err := s.db.Create(&p).Error; err == nil {
			partnerID = p.ID
			lead.PartnerID = &partnerID
		}
	}

	// 2. Build Sales Order
	priceUnit := lead.ExpectedRevenue
	if priceUnit <= 0 {
		priceUnit = 10000000
	}
	subtotal := priceUnit
	tax := subtotal * 0.11
	total := subtotal + tax

	so := sale.SaleOrder{
		PartnerID:     partnerID,
		DateOrder:     time.Now().Format("2006-01-02 15:04:05"),
		State:         "draft",
		AmountUntaxed: subtotal,
		AmountTax:     tax,
		AmountTotal:   total,
		TaxRate:       11.0,
		Note:          fmt.Sprintf("Converted from CRM Opportunity: %s", lead.Name),
		OrderLines: []sale.SaleOrderLine{
			{
				ProductID:     1, // Default to primary product or line
				Name:          fmt.Sprintf("Deal: %s", lead.Name),
				ProductUomQty: 1,
				PriceUnit:     priceUnit,
				PriceSubtotal: subtotal,
				PriceTotal:    total,
			},
		},
	}
	so.CompanyID = companyID

	if err := s.saleSvc.CreateOrder(&so); err != nil {
		return nil, nil, fmt.Errorf("failed to create sales order: %w", err)
	}

	// 3. Mark Lead as Won and link quotation
	lead.SaleOrderID = &so.ID
	lead.Stage = "won"
	lead.Probability = 100
	if err := s.db.Save(lead).Error; err != nil {
		return nil, nil, err
	}

	return &so, lead, nil
}
