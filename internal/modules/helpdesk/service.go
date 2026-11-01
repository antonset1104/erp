package helpdesk

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateTicket(t *HelpdeskTicket) error {
	if t.Name == "" {
		return errors.New("ticket subject is required")
	}

	if t.Stage == "" {
		t.Stage = "new"
	}
	if t.Priority == "" {
		t.Priority = "1"
	}
	if t.Category == "" {
		t.Category = "technical"
	}

	// Auto generate sequence TICKET/2026/00001
	if t.TicketNumber == "" {
		var count int64
		s.db.Model(&HelpdeskTicket{}).Count(&count)
		year := time.Now().Year()
		t.TicketNumber = fmt.Sprintf("TCK/%d/%05d", year, count+1)
	}

	return s.db.Create(t).Error
}

func (s *Service) ListTickets(companyID uint, stage, priority, category string, teamID uint) ([]HelpdeskTicket, error) {
	var tickets []HelpdeskTicket
	query := s.db.Where("company_id = ?", companyID)

	if stage != "" && stage != "all" {
		query = query.Where("stage = ?", stage)
	}
	if priority != "" && priority != "all" {
		query = query.Where("priority = ?", priority)
	}
	if category != "" && category != "all" {
		query = query.Where("category = ?", category)
	}
	if teamID > 0 {
		query = query.Where("team_id = ?", teamID)
	}

	err := query.
		Preload("Team").
		Preload("Partner").
		Preload("User").
		Order("id desc").
		Find(&tickets).Error
	return tickets, err
}

func (s *Service) GetTicket(id, companyID uint) (*HelpdeskTicket, error) {
	var t HelpdeskTicket
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("Team").
		Preload("Partner").
		Preload("User").
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Service) UpdateTicketStage(id, companyID, userID uint, stage string) (*HelpdeskTicket, error) {
	ticket, err := s.GetTicket(id, companyID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	ticket.Stage = stage
	ticket.WriteUID = userID
	if stage == "solved" {
		ticket.CloseDate = time.Now().Format("2006-01-02 15:04:05")
	}

	if err := s.db.Save(ticket).Error; err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *Service) AssignTicket(id, companyID, userID uint, assignUserID uint) (*HelpdeskTicket, error) {
	ticket, err := s.GetTicket(id, companyID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	ticket.UserID = &assignUserID
	ticket.WriteUID = userID
	if ticket.Stage == "new" {
		ticket.Stage = "in_progress"
	}

	if err := s.db.Save(ticket).Error; err != nil {
		return nil, err
	}
	return s.GetTicket(id, companyID)
}

func (s *Service) ResolveTicket(id, companyID, userID uint, resolution string) (*HelpdeskTicket, error) {
	ticket, err := s.GetTicket(id, companyID)
	if err != nil {
		return nil, errors.New("ticket not found")
	}

	ticket.Stage = "solved"
	ticket.Resolution = resolution
	ticket.CloseDate = time.Now().Format("2006-01-02 15:04:05")
	ticket.WriteUID = userID

	if err := s.db.Save(ticket).Error; err != nil {
		return nil, err
	}
	return s.GetTicket(id, companyID)
}

func (s *Service) ListTeams(companyID uint) ([]HelpdeskTeam, error) {
	var teams []HelpdeskTeam
	err := s.db.Where("company_id = ?", companyID).Preload("Leader").Order("id asc").Find(&teams).Error
	return teams, err
}

func (s *Service) CreateTeam(team *HelpdeskTeam) error {
	if team.Name == "" || team.Code == "" {
		return errors.New("team name and code are required")
	}
	return s.db.Create(team).Error
}
