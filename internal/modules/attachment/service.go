package attachment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"path/filepath"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/storage"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/expense"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       Repository
	storage    storage.Provider
	aiClient   *ai.Client
	chatterSvc *chatter.Service
	expenseSvc *expense.Service
}

func NewService(
	db *gorm.DB,
	repo Repository,
	storage storage.Provider,
	aiClient *ai.Client,
	chatterSvc *chatter.Service,
	expenseSvc *expense.Service,
) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		storage:    storage,
		aiClient:   aiClient,
		chatterSvc: chatterSvc,
		expenseSvc: expenseSvc,
	}
}

func (s *Service) SetExpenseService(expenseSvc *expense.Service) {
	s.expenseSvc = expenseSvc
}

func (s *Service) UploadAttachment(
	companyID, userID uint,
	userName string,
	req *UploadAttachmentRequest,
	fileBytes []byte,
	mimeType string,
) (*Attachment, error) {
	if len(fileBytes) == 0 {
		return nil, errors.New("file content cannot be empty")
	}

	name := req.Name
	if name == "" {
		name = "attachment_" + time.Now().Format("20060102_150405")
	}

	// Guess mime type if missing
	if mimeType == "" || mimeType == "application/octet-stream" {
		ext := filepath.Ext(name)
		if ext != "" {
			mimeType = mime.TypeByExtension(ext)
		}
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
	}

	// Calculate SHA256 checksum
	hash := sha256.Sum256(fileBytes)
	checksum := hex.EncodeToString(hash[:])

	// Storage key format: YYYY/MM/timestamp_sanitized_name
	now := time.Now()
	cleanName := strings.ReplaceAll(name, " ", "_")
	storageKey := fmt.Sprintf("%04d/%02d/%d_%s", now.Year(), now.Month(), now.UnixNano(), cleanName)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	savedKey, err := s.storage.Save(ctx, storageKey, mimeType, fileBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to save file to storage: %w", err)
	}

	resModel := req.ResModel
	if resModel == "" {
		resModel = "document"
	}

	att := &Attachment{
		Name:          name,
		ResModel:      resModel,
		ResID:         req.ResID,
		ResName:       req.ResName,
		MimeType:      mimeType,
		FileSize:      int64(len(fileBytes)),
		StorageType:   s.storage.Type(),
		StoragePath:   savedKey,
		Checksum:      checksum,
		Description:   req.Description,
		OCRStatus:     "none",
		CompanyID:     companyID,
		CreatedByID:   userID,
		CreatedByName: userName,
	}

	if err := s.repo.Create(att); err != nil {
		_ = s.storage.Delete(context.Background(), savedKey)
		return nil, fmt.Errorf("failed to persist attachment record: %w", err)
	}

	// Update download URL
	att.FileURL = fmt.Sprintf("/api/v1/attachments/%d/download", att.ID)
	_ = s.repo.Update(att)

	// Log chatter if linked to a specific record
	if s.chatterSvc != nil && att.ResModel != "" && att.ResID > 0 {
		msg := fmt.Sprintf("📎 Lampiran baru diunggah: <strong>%s</strong> (Ukuran: %.2f KB, Tipe: %s)",
			att.Name, float64(att.FileSize)/1024, att.MimeType)
		chatterMsg := &chatter.MailMessage{
			ModelName:   att.ResModel,
			ResID:       att.ResID,
			Body:        msg,
			MessageType: "comment",
			AuthorID:    userID,
			AuthorName:  userName,
		}
		chatterMsg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(chatterMsg)
	}

	// Auto OCR if requested
	if req.AutoOCR && s.aiClient != nil {
		_, _ = s.RunOCR(att.ID, req.TextHint)
		// reload updated OCR data
		updated, _ := s.repo.FindByID(att.ID)
		if updated != nil {
			att = updated
		}
	}

	return att, nil
}

func (s *Service) GetAttachment(id uint) (*Attachment, []byte, error) {
	att, err := s.repo.FindByID(id)
	if err != nil {
		return nil, nil, errors.New("attachment not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	data, err := s.storage.Get(ctx, att.StoragePath)
	if err != nil {
		return att, nil, fmt.Errorf("failed to retrieve file content from storage: %w", err)
	}

	return att, data, nil
}

func (s *Service) DeleteAttachment(id uint) error {
	att, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("attachment not found")
	}

	_ = s.storage.Delete(context.Background(), att.StoragePath)
	return s.repo.Delete(id)
}

func (s *Service) ListAttachments(companyID uint, filter AttachmentFilter) ([]Attachment, int64, error) {
	return s.repo.List(companyID, filter)
}

func (s *Service) RunOCR(id uint, textHint string) (*ai.ReceiptOCRResult, error) {
	att, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("attachment not found")
	}

	if s.aiClient == nil {
		return nil, errors.New("AI vision client is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	data, _ := s.storage.Get(ctx, att.StoragePath)

	if textHint == "" {
		textHint = att.Description
		if textHint == "" {
			textHint = att.Name
		}
	}

	result, err := s.aiClient.AnalyzeReceiptVision(data, att.MimeType, textHint)
	if err != nil {
		att.OCRStatus = "failed"
		_ = s.repo.Update(att)
		return nil, fmt.Errorf("OCR analysis failed: %w", err)
	}

	jsonBytes, _ := json.Marshal(result)
	att.OCRStatus = "processed"
	att.OCRData = string(jsonBytes)
	_ = s.repo.Update(att)

	// Log chatter on the linked document if applicable
	if s.chatterSvc != nil && att.ResModel != "" && att.ResID > 0 {
		msg := fmt.Sprintf("🤖 <strong>Cloudflare AI Vision OCR Selesai</strong>: Vendor: <em>%s</em>, Tanggal: %s, Total: Rp %.2f (Keyakinan: %.0f%%)",
			result.VendorName, result.InvoiceDate, result.TotalAmount, result.ConfidenceScore*100)
		chatterMsg := &chatter.MailMessage{
			ModelName:   att.ResModel,
			ResID:       att.ResID,
			Body:        msg,
			MessageType: "notification",
			AuthorName:  "AI Vision Engine",
		}
		chatterMsg.CompanyID = att.CompanyID
		_ = s.chatterSvc.PostMessage(chatterMsg)
	}

	return result, nil
}

func (s *Service) LinkToRecord(id uint, resModel string, resID uint, resName string) error {
	att, err := s.repo.FindByID(id)
	if err != nil {
		return errors.New("attachment not found")
	}

	att.ResModel = resModel
	att.ResID = resID
	att.ResName = resName
	return s.repo.Update(att)
}

func (s *Service) CreateExpenseFromReceipt(id uint, employeeID uint, companyID, userID uint) (*expense.Expense, error) {
	if s.expenseSvc == nil {
		return nil, errors.New("expense service is not available")
	}

	att, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("attachment not found")
	}

	var ocrResult ai.ReceiptOCRResult
	if att.OCRStatus != "processed" || att.OCRData == "" {
		res, err := s.RunOCR(id, "")
		if err != nil {
			return nil, err
		}
		ocrResult = *res
	} else {
		if err := json.Unmarshal([]byte(att.OCRData), &ocrResult); err != nil {
			return nil, fmt.Errorf("invalid saved OCR data: %w", err)
		}
	}

	// Match category in expense module
	var categoryID uint = 1
	var cat expense.ExpenseCategory
	if err := s.db.Where("(company_id = ? OR company_id = 0) AND LOWER(name) LIKE ?",
		companyID, "%"+strings.ToLower(ocrResult.CategorySuggestion)+"%").First(&cat).Error; err == nil {
		categoryID = cat.ID
	} else {
		// Fallback to first available category
		if err := s.db.Where("company_id = ? OR company_id = 0", companyID).First(&cat).Error; err == nil {
			categoryID = cat.ID
		}
	}

	expenseDate := time.Now()
	if ocrResult.InvoiceDate != "" {
		if t, err := time.Parse("2006-01-02", ocrResult.InvoiceDate); err == nil {
			expenseDate = t
		}
	}

	desc := fmt.Sprintf("[%s] %s", ocrResult.VendorName, ocrResult.CategorySuggestion)
	if len(ocrResult.LineItems) > 0 && ocrResult.LineItems[0].Description != "" {
		desc = fmt.Sprintf("[%s] %s", ocrResult.VendorName, ocrResult.LineItems[0].Description)
	}

	newExp := &expense.Expense{
		Name:        desc,
		EmployeeID:  employeeID,
		CategoryID:  categoryID,
		TotalAmount: ocrResult.TotalAmount,
		UnitPrice:   ocrResult.TotalAmount,
		Quantity:    1,
		Date:        expenseDate.Format("2006-01-02"),
		ReceiptRef:  fmt.Sprintf("STRUK-%d", att.ID),
		PaymentMode: "own_account",
		State:       "draft",
	}
	newExp.CompanyID = companyID

	if err := s.expenseSvc.CreateExpense(newExp, companyID, userID); err != nil {
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	// Link attachment to new expense
	att.ResModel = "hr.expense"
	att.ResID = newExp.ID
	att.ResName = newExp.Name
	_ = s.repo.Update(att)

	return newExp, nil
}

func (s *Service) GetStats(companyID uint) (map[string]interface{}, error) {
	return s.repo.GetStats(companyID)
}
