package base

import (
	"errors"
	"fmt"

	"erp-golang/internal/core/middleware"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db        *gorm.DB
	jwtSecret string
}

func NewService(db *gorm.DB, jwtSecret string) *Service {
	return &Service{db: db, jwtSecret: jwtSecret}
}

func (s *Service) Authenticate(email, password string) (*User, string, error) {
	var user User
	if err := s.db.Where("email = ? AND active = ?", email, true).First(&user).Error; err != nil {
		return nil, "", errors.New("invalid email or password")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", errors.New("invalid email or password")
	}

	token, err := middleware.GenerateToken(user.ID, user.Email, user.Name, user.Role, user.CompanyID, s.jwtSecret)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	return &user, token, nil
}

func (s *Service) Register(name, email, password, role string, companyID uint) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Create partner entry for user
	partner := Partner{
		Name:      name,
		Email:     email,
		IsCompany: false,
	}
	partner.CompanyID = companyID
	if err := s.db.Create(&partner).Error; err != nil {
		return nil, err
	}

	user := User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         role,
		PartnerID:    partner.ID,
	}
	user.CompanyID = companyID

	if err := s.db.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Service) ListPartners(companyID uint, partnerType string) ([]Partner, error) {
	var partners []Partner
	query := s.db.Where("company_id = ? AND active = ?", companyID, true)

	if partnerType == "customer" {
		query = query.Where("customer_rank > 0")
	} else if partnerType == "supplier" || partnerType == "vendor" {
		query = query.Where("supplier_rank > 0")
	}

	if err := query.Order("name asc").Find(&partners).Error; err != nil {
		return nil, err
	}
	return partners, nil
}

func (s *Service) CreatePartner(p *Partner) error {
	return s.db.Create(p).Error
}

func (s *Service) GetPartner(id uint, companyID uint) (*Partner, error) {
	var p Partner
	if err := s.db.Where("id = ? AND company_id = ?", id, companyID).First(&p).Error; err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) UpdatePartner(p *Partner) error {
	return s.db.Save(p).Error
}
