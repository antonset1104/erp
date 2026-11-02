package account

import (
	"errors"
	"fmt"
	"math"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListMoves(companyID uint, moveType string, state string) ([]AccountMove, error) {
	var moves []AccountMove
	query := s.db.Preload("Partner").Preload("Journal").Preload("LineIDs.Account").
		Where("company_id = ?", companyID)

	if moveType != "" {
		query = query.Where("move_type = ?", moveType)
	}
	if state != "" {
		query = query.Where("state = ?", state)
	}

	if err := query.Order("id desc").Find(&moves).Error; err != nil {
		return nil, err
	}
	return moves, nil
}

func (s *Service) GetMove(id uint, companyID uint) (*AccountMove, error) {
	var move AccountMove
	if err := s.db.Preload("Partner").Preload("Journal").Preload("LineIDs.Account").
		Where("id = ? AND company_id = ?", id, companyID).First(&move).Error; err != nil {
		return nil, err
	}
	return &move, nil
}

func (s *Service) CreateMove(move *AccountMove) error {
	year := time.Now().Format("2006")
	if move.Name == "" {
		var count int64
		s.db.Model(&AccountMove{}).Where("move_type = ?", move.MoveType).Count(&count)
		prefix := "INV"
		if move.MoveType == "in_invoice" {
			prefix = "BILL"
		} else if move.MoveType == "entry" {
			prefix = "MISC"
		}
		for {
			count++
			candidate := fmt.Sprintf("%s/%s/%05d", prefix, year, count)
			var exists int64
			s.db.Model(&AccountMove{}).Where("name = ?", candidate).Count(&exists)
			if exists == 0 {
				move.Name = candidate
				break
			}
		}
	}

	if move.InvoiceDate == "" {
		move.InvoiceDate = time.Now().Format("2006-01-02")
	}
	if move.State == "" {
		move.State = "draft"
	}
	if move.PaymentState == "" {
		move.PaymentState = "not_paid"
	}

	// Calculate line balances
	var totalDebit, totalCredit float64
	for i := range move.LineIDs {
		line := &move.LineIDs[i]
		line.Balance = line.Debit - line.Credit
		totalDebit += line.Debit
		totalCredit += line.Credit
	}

	if move.AmountTotal == 0 {
		move.AmountTotal = math.Max(totalDebit, totalCredit)
	}
	move.AmountResidual = move.AmountTotal

	return s.db.Create(move).Error
}

// ActionPost validates double-entry balance (Debit == Credit) and posts the move to the general ledger
func (s *Service) ActionPost(id uint, companyID uint) (*AccountMove, error) {
	move, err := s.GetMove(id, companyID)
	if err != nil {
		return nil, errors.New("invoice/move not found")
	}

	if move.State == "posted" {
		return nil, errors.New("move is already posted")
	}
	if move.State == "cancel" {
		return nil, errors.New("cannot post a cancelled move")
	}

	// Validate Debit == Credit (Enterprise ERP core accounting rule)
	var sumDebit, sumCredit float64
	for _, line := range move.LineIDs {
		sumDebit += line.Debit
		sumCredit += line.Credit
	}

	// Allow tiny floating point epsilon
	if math.Abs(sumDebit-sumCredit) > 0.01 {
		return nil, fmt.Errorf("cannot post unbalanced entry: total debit (%.2f) != total credit (%.2f)", sumDebit, sumCredit)
	}

	move.State = "posted"
	move.AmountResidual = move.AmountTotal
	if err := s.db.Save(move).Error; err != nil {
		return nil, err
	}

	return move, nil
}

// RegisterPayment registers a payment against an invoice and updates its payment state
func (s *Service) RegisterPayment(moveID uint, amount float64, journalID uint, memo string, companyID uint) (*AccountPayment, *AccountMove, error) {
	move, err := s.GetMove(moveID, companyID)
	if err != nil {
		return nil, nil, errors.New("invoice not found")
	}

	if move.State != "posted" {
		return nil, nil, errors.New("you can only register payment for posted invoices")
	}
	if move.PaymentState == "paid" {
		return nil, nil, errors.New("invoice is already fully paid")
	}
	if amount <= 0 {
		return nil, nil, errors.New("payment amount must be greater than zero")
	}

	var count int64
	s.db.Model(&AccountPayment{}).Count(&count)
	payName := fmt.Sprintf("PAY/%s/%05d", time.Now().Format("2006"), count+1)

	paymentType := "inbound"
	if move.MoveType == "in_invoice" {
		paymentType = "outbound"
	}

	payment := &AccountPayment{
		Name:        payName,
		MoveID:      move.ID,
		PartnerID:   move.PartnerID,
		JournalID:   journalID,
		Amount:      amount,
		PaymentDate: time.Now().Format("2006-01-02 15:04:05"),
		PaymentType: paymentType,
		Memo:        memo,
	}
	payment.CompanyID = companyID

	if err := s.db.Create(payment).Error; err != nil {
		return nil, nil, err
	}

	// Update invoice residual
	move.AmountResidual -= amount
	if move.AmountResidual <= 0.01 {
		move.AmountResidual = 0
		move.PaymentState = "paid"
	}

	if err := s.db.Save(move).Error; err != nil {
		return nil, nil, err
	}

	return payment, move, nil
}

func (s *Service) ListAccounts(companyID uint) ([]AccountAccount, error) {
	var accounts []AccountAccount
	if err := s.db.Where("company_id = ?", companyID).Order("code asc").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

func (s *Service) ListJournals(companyID uint) ([]AccountJournal, error) {
	var journals []AccountJournal
	if err := s.db.Where("company_id = ?", companyID).Order("code asc").Find(&journals).Error; err != nil {
		return nil, err
	}
	return journals, nil
}
