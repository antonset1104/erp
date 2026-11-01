package product

import (
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListProducts(companyID uint) ([]Product, error) {
	var products []Product
	if err := s.db.Preload("Category").Preload("UoM").
		Where("company_id = ? AND active = ?", companyID, true).
		Order("name asc").
		Find(&products).Error; err != nil {
		return nil, err
	}
	return products, nil
}

func (s *Service) GetProduct(id uint, companyID uint) (*Product, error) {
	var p Product
	if err := s.db.Preload("Category").Preload("UoM").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) CreateProduct(p *Product) error {
	return s.db.Create(p).Error
}

func (s *Service) UpdateProduct(p *Product) error {
	return s.db.Save(p).Error
}

func (s *Service) ListCategories(companyID uint) ([]ProductCategory, error) {
	var cats []ProductCategory
	if err := s.db.Where("company_id = ? AND active = ?", companyID, true).Find(&cats).Error; err != nil {
		return nil, err
	}
	return cats, nil
}

func (s *Service) ListUoM() ([]UoM, error) {
	var uoms []UoM
	if err := s.db.Where("active = ?", true).Find(&uoms).Error; err != nil {
		return nil, err
	}
	return uoms, nil
}
