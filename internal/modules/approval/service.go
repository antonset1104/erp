package approval

import (
	"errors"
	"fmt"
	"time"

	"erp-golang/internal/modules/chatter"
	"gorm.io/gorm"
)

type Service struct {
	repo       *Repository
	chatterSvc *chatter.Service
}

func NewService(db *gorm.DB, chatterSvc *chatter.Service) *Service {
	return &Service{
		repo:       NewRepository(db),
		chatterSvc: chatterSvc,
	}
}

func (s *Service) logChatter(resID uint, body string, msgType string, authorID uint, companyID uint) {
	if s.chatterSvc == nil {
		return
	}
	msg := chatter.MailMessage{
		ModelName:   "approval.request",
		ResID:       resID,
		Body:        body,
		MessageType: msgType,
		AuthorID:    authorID,
	}
	msg.CompanyID = companyID
	_ = s.chatterSvc.PostMessage(&msg)
}

// Category Management
func (s *Service) ListCategories(companyID uint) ([]ApprovalCategory, error) {
	return s.repo.ListCategories(companyID)
}

func (s *Service) GetCategoryByID(id, companyID uint) (*ApprovalCategory, error) {
	return s.repo.GetCategoryByID(id, companyID)
}

func (s *Service) CreateCategory(cat *ApprovalCategory) error {
	if cat.SequenceCode == "" {
		cat.SequenceCode = "APPR"
	}
	if cat.MinApprovalsRequired <= 0 {
		cat.MinApprovalsRequired = 1
	}
	return s.repo.CreateCategory(cat)
}

// Request Management
type CreateRequestInput struct {
	Name           string              `json:"name" binding:"required"`
	CategoryID     uint                `json:"category_id" binding:"required"`
	RequestOwnerID uint                `json:"request_owner_id"`
	Reason         string              `json:"reason"`
	Amount         float64             `json:"amount"`
	CurrencyID     uint                `json:"currency_id"`
	DateStart      *time.Time          `json:"date_start"`
	DateEnd        *time.Time          `json:"date_end"`
	Reference      string              `json:"reference"`
	CompanyID      uint                `json:"company_id"`
	Approvers      []ApproverInputItem `json:"approvers"`
}

type ApproverInputItem struct {
	UserID   uint `json:"user_id"`
	Sequence int  `json:"sequence"`
	Required bool `json:"required"`
}

func (s *Service) CreateRequest(input CreateRequestInput, creatorID uint) (*ApprovalRequest, error) {
	cat, err := s.repo.GetCategoryByID(input.CategoryID, input.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("kategori persetujuan tidak ditemukan: %w", err)
	}

	code, err := s.repo.NextSequence(input.CompanyID, cat.SequenceCode)
	if err != nil {
		return nil, err
	}

	ownerID := input.RequestOwnerID
	if ownerID == 0 {
		ownerID = creatorID
	}

	req := &ApprovalRequest{
		Name:           input.Name,
		Code:           code,
		CategoryID:     input.CategoryID,
		RequestOwnerID: ownerID,
		Reason:         input.Reason,
		Amount:         input.Amount,
		CurrencyID:     input.CurrencyID,
		DateStart:      input.DateStart,
		DateEnd:        input.DateEnd,
		Reference:      input.Reference,
		Stage:          "new",
	}
	req.CompanyID = input.CompanyID
	req.CreateUID = creatorID
	req.WriteUID = creatorID

	if err := s.repo.CreateRequest(req); err != nil {
		return nil, err
	}

	// Create approvers if provided
	var approverEntities []ApprovalApprover
	if len(input.Approvers) > 0 {
		for _, app := range input.Approvers {
			item := ApprovalApprover{
				RequestID: req.ID,
				UserID:    app.UserID,
				Sequence:  app.Sequence,
				Required:  app.Required,
				Status:    "new",
			}
			item.CompanyID = input.CompanyID
			item.CreateUID = creatorID
			item.WriteUID = creatorID
			approverEntities = append(approverEntities, item)
		}
	} else {
		// Default tier 1 approver (admin / manager)
		app1 := ApprovalApprover{
			RequestID: req.ID,
			UserID:    1, // Default Admin Manager
			Sequence:  1,
			Required:  true,
			Status:    "new",
		}
		app1.CompanyID = input.CompanyID
		app1.CreateUID = creatorID
		app1.WriteUID = creatorID
		approverEntities = append(approverEntities, app1)

		// If high amount exceeding category min_amount, add Tier 2 Executive approval
		if cat.HasAmount && cat.MinAmount > 0 && input.Amount >= cat.MinAmount {
			app2 := ApprovalApprover{
				RequestID: req.ID,
				UserID:    1, // Executive Approver
				Sequence:  2,
				Required:  true,
				Status:    "new",
			}
			app2.CompanyID = input.CompanyID
			app2.CreateUID = creatorID
			app2.WriteUID = creatorID
			approverEntities = append(approverEntities, app2)
		}
	}

	if len(approverEntities) > 0 {
		_ = s.repo.CreateApprovers(approverEntities)
	}

	// Log chatter
	s.logChatter(
		req.ID,
		fmt.Sprintf("Draf permohonan persetujuan <b>%s</b> dibuat (Kategori: %s, Ref: %s, Nilai: Rp %.2f).", req.Code, cat.Name, req.Reference, req.Amount),
		"comment", creatorID, input.CompanyID,
	)

	return s.repo.GetRequestByID(req.ID, input.CompanyID)
}

func (s *Service) GetRequestByID(id, companyID uint) (*ApprovalRequest, error) {
	return s.repo.GetRequestByID(id, companyID)
}

func (s *Service) ListRequests(companyID uint, categoryID uint, stage string, userID uint) ([]ApprovalRequest, error) {
	return s.repo.ListRequests(companyID, categoryID, stage, userID)
}

// ActionSubmit moves request from new -> pending and activates level 1 approver(s)
func (s *Service) ActionSubmit(requestID, userID, companyID uint) (*ApprovalRequest, error) {
	req, err := s.repo.GetRequestByID(requestID, companyID)
	if err != nil {
		return nil, err
	}
	if req.Stage != "new" {
		return nil, errors.New("hanya permohonan berstatus 'new' yang dapat diajukan")
	}

	if len(req.Approvers) == 0 {
		return nil, errors.New("permohonan harus memiliki minimal 1 approver")
	}

	req.Stage = "pending"
	req.WriteUID = userID

	// Activate lowest sequence approver(s)
	lowestSeq := 9999
	for _, app := range req.Approvers {
		if app.Sequence < lowestSeq {
			lowestSeq = app.Sequence
		}
	}

	for i := range req.Approvers {
		if req.Approvers[i].Sequence == lowestSeq {
			req.Approvers[i].Status = "waiting"
			_ = s.repo.UpdateApprover(&req.Approvers[i])
		}
	}

	if err := s.repo.UpdateRequest(req); err != nil {
		return nil, err
	}

	s.logChatter(
		req.ID,
		fmt.Sprintf("Permohonan persetujuan diajukan untuk ditinjau. Menunggu evaluasi approver Level %d.", lowestSeq),
		"stage_change", userID, companyID,
	)

	return s.repo.GetRequestByID(req.ID, companyID)
}

// ActionApprove approves by the designated approver for the active level
func (s *Service) ActionApprove(requestID, userID, companyID uint, comment string) (*ApprovalRequest, error) {
	req, err := s.repo.GetRequestByID(requestID, companyID)
	if err != nil {
		return nil, err
	}
	if req.Stage != "pending" {
		return nil, errors.New("hanya permohonan berstatus 'pending' yang dapat disetujui")
	}

	// Find the approver entry for this user that is waiting or active
	var targetApprover *ApprovalApprover
	for i := range req.Approvers {
		if req.Approvers[i].UserID == userID && req.Approvers[i].Status == "waiting" {
			targetApprover = &req.Approvers[i]
			break
		}
	}

	// Fallback: If user is admin (ID 1), allow approving on behalf of the waiting level
	if targetApprover == nil && userID == 1 {
		for i := range req.Approvers {
			if req.Approvers[i].Status == "waiting" {
				targetApprover = &req.Approvers[i]
				break
			}
		}
	}

	if targetApprover == nil {
		return nil, errors.New("anda tidak terdaftar sebagai approver aktif pada tahapan permohonan ini")
	}

	now := time.Now()
	targetApprover.Status = "approved"
	targetApprover.Comment = comment
	targetApprover.ApprovalDate = &now
	targetApprover.WriteUID = userID
	if err := s.repo.UpdateApprover(targetApprover); err != nil {
		return nil, err
	}

	// Refresh request approvers
	req, err = s.repo.GetRequestByID(requestID, companyID)
	if err != nil {
		return nil, err
	}

	// Check if there are other waiting approvers in the same sequence
	hasWaitingInCurrent := false
	for _, app := range req.Approvers {
		if app.Sequence == targetApprover.Sequence && app.Status == "waiting" {
			hasWaitingInCurrent = true
			break
		}
	}

	if !hasWaitingInCurrent {
		// Find next higher sequence that is 'new'
		nextSeq := 9999
		for _, app := range req.Approvers {
			if app.Sequence > targetApprover.Sequence && app.Status == "new" && app.Sequence < nextSeq {
				nextSeq = app.Sequence
			}
		}

		if nextSeq < 9999 {
			// Activate next level
			for i := range req.Approvers {
				if req.Approvers[i].Sequence == nextSeq && req.Approvers[i].Status == "new" {
					req.Approvers[i].Status = "waiting"
					_ = s.repo.UpdateApprover(&req.Approvers[i])
				}
			}
			s.logChatter(
				req.ID,
				fmt.Sprintf("Persetujuan Level %d disetujui (Catatan: <i>'%s'</i>). Alur kerja beralih ke Level %d.", targetApprover.Sequence, comment, nextSeq),
				"stage_change", userID, companyID,
			)
		} else {
			// All tiers approved!
			req.Stage = "approved"
			req.WriteUID = userID
			if err := s.repo.UpdateRequest(req); err != nil {
				return nil, err
			}
			s.logChatter(
				req.ID,
				fmt.Sprintf("🎉 Permohonan persetujuan <b>%s</b> telah <b>DISETUJUI SEPENUHNYA</b> oleh seluruh level otoritas.", req.Code),
				"stage_change", userID, companyID,
			)
		}
	}

	return s.repo.GetRequestByID(req.ID, companyID)
}

// ActionRefuse rejects the request with a stated refusal reason
func (s *Service) ActionRefuse(requestID, userID, companyID uint, refusalReason string) (*ApprovalRequest, error) {
	req, err := s.repo.GetRequestByID(requestID, companyID)
	if err != nil {
		return nil, err
	}
	if req.Stage != "pending" && req.Stage != "new" {
		return nil, errors.New("hanya permohonan berstatus 'pending' atau 'new' yang dapat ditolak")
	}

	if refusalReason == "" {
		refusalReason = "Ditolak tanpa rincian alasan."
	}

	now := time.Now()
	for i := range req.Approvers {
		if req.Approvers[i].Status == "waiting" || req.Approvers[i].UserID == userID {
			req.Approvers[i].Status = "refused"
			req.Approvers[i].Comment = refusalReason
			req.Approvers[i].ApprovalDate = &now
			_ = s.repo.UpdateApprover(&req.Approvers[i])
		}
	}

	req.Stage = "refused"
	req.RefusalReason = refusalReason
	req.WriteUID = userID
	if err := s.repo.UpdateRequest(req); err != nil {
		return nil, err
	}

	s.logChatter(
		req.ID,
		fmt.Sprintf("❌ Permohonan persetujuan <b>%s</b> <b>DITOLAK</b>. Alasan: <i>'%s'</i>.", req.Code, refusalReason),
		"stage_change", userID, companyID,
	)

	return s.repo.GetRequestByID(req.ID, companyID)
}

// ActionCancel cancels the request
func (s *Service) ActionCancel(requestID, userID, companyID uint) (*ApprovalRequest, error) {
	req, err := s.repo.GetRequestByID(requestID, companyID)
	if err != nil {
		return nil, err
	}
	if req.Stage == "approved" {
		return nil, errors.New("permohonan yang sudah disetujui tidak dapat dibatalkan")
	}

	req.Stage = "cancelled"
	req.WriteUID = userID
	for i := range req.Approvers {
		if req.Approvers[i].Status != "approved" {
			req.Approvers[i].Status = "cancelled"
			_ = s.repo.UpdateApprover(&req.Approvers[i])
		}
	}

	if err := s.repo.UpdateRequest(req); err != nil {
		return nil, err
	}

	s.logChatter(
		req.ID,
		fmt.Sprintf("Permohonan persetujuan %s dibatalkan oleh pemohon.", req.Code),
		"stage_change", userID, companyID,
	)

	return s.repo.GetRequestByID(req.ID, companyID)
}

func (s *Service) GetStats(companyID uint) (map[string]int64, error) {
	return s.repo.GetStats(companyID)
}
