package chatter

import (
	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListMessages(modelName string, resID uint, companyID uint) ([]MailMessage, error) {
	var messages []MailMessage
	if err := s.db.Where("model_name = ? AND res_id = ? AND company_id = ?", modelName, resID, companyID).
		Order("id asc").Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *Service) PostMessage(msg *MailMessage) error {
	if msg.MessageType == "" {
		msg.MessageType = "comment"
	}
	return s.db.Create(msg).Error
}

func (s *Service) LogTracking(modelName string, resID uint, authorID uint, authorName string, text string, tracking string, companyID uint) error {
	msg := MailMessage{
		ModelName:   modelName,
		ResID:       resID,
		AuthorID:    authorID,
		AuthorName:  authorName,
		Body:        text,
		MessageType: "tracking",
		Tracking:    tracking,
	}
	msg.CompanyID = companyID
	return s.PostMessage(&msg)
}
