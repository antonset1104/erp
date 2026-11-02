package esignature

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
)

type Module struct {
	db         *gorm.DB
	chatterSvc *chatter.Service
	aiClient   *ai.Client
	repo       Repository
	service    Service
	handler    *Handler
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	svc := NewService(repo, chatterSvc, aiClient)
	handler := NewHandler(svc)

	return &Module{
		db:         db,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
		repo:       repo,
		service:    svc,
		handler:    handler,
	}
}

func (m *Module) Name() string {
	return "esignature"
}

func (m *Module) Init(db *gorm.DB) error {
	m.db = db
	return nil
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&DocumentSignRequest{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) Service() Service {
	return m.service
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&DocumentSignRequest{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Digital Signature & Watermarking Sample Data...")

	now := time.Now()
	signedTime := now.Add(-3 * time.Hour)
	expiryTime := now.AddDate(0, 0, 7)

	auditLog1 := []SignerAuditLogEntry{
		{
			Timestamp: now.Add(-5 * time.Hour).Format(time.RFC3339),
			Action:    "created",
			Actor:     "System Admin",
			IP:        "127.0.0.1",
			UserAgent: "GoERP Automated Billing",
			Details:   "Permohonan TTD elektronik resmi diterbitkan untuk Faktur Penjualan INV/2026/00001",
		},
		{
			Timestamp: now.Add(-4 * time.Hour).Format(time.RFC3339),
			Action:    "viewed",
			Actor:     "direktur@holding.co.id",
			IP:        "182.253.120.45",
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/128.0.0.0",
			Details:   "Dokumen faktur ditinjau oleh Direktur Keuangan melalui secure portal",
		},
		{
			Timestamp: signedTime.Format(time.RFC3339),
			Action:    "signed",
			Actor:     "Dr. Ir. Hendra Gunawan, M.B.A. <direktur@holding.co.id>",
			IP:        "182.253.120.45",
			UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/128.0.0.0",
			Details:   "Dokumen resmi ditandatangani dengan sertifikat digital SHA-256",
		},
	}
	auditJSON1, _ := json.Marshal(auditLog1)

	sampleRequests := []DocumentSignRequest{
		{
			RequestNumber:           "ESIGN/2026/00001",
			CompanyID:               1,
			DocumentType:            "invoice",
			DocumentID:              1,
			DocumentNumber:          "INV/2026/00001",
			Title:                   "Faktur Komersial & Tagihan Proyek IT Holding",
			Description:             "Penandatanganan faktur penagihan termin 1 termin proyek ERP multi-company.",
			SignerName:              "Dr. Ir. Hendra Gunawan, M.B.A.",
			SignerEmail:             "direktur@holding.co.id",
			SignerRole:              "Direktur Keuangan (CFO)",
			Status:                  "signed",
			SecurityToken:           "a1b2c3d4e5f60718293a4b5c6d7e8f90",
			ExpiryDate:              &expiryTime,
			SignatureType:           "draw",
			SignatureData:           "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAK8AAAA8CAYAAADt7z10AAAFeklEQVR4Xu2dz4tcVRTHP/fe7unpZCYz",
			SignatureHash:           "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			CertificateSerialNumber: "CERT-ID-2026-e3b0c44298fc",
			SignedAt:                &signedTime,
			SignedIP:                "182.253.120.45",
			SignedUserAgent:         "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/128.0.0.0",
			WatermarkEnabled:        true,
			WatermarkText:           "CONFIDENTIAL & DIGITALLY VERIFIED - PT MULTI PERUSAHAAN HOLDING",
			WatermarkQRData:         "https://erp.internal/verify/esign?token=a1b2c3d4e5f60718293a4b5c6d7e8f90&hash=e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			AuditTrailLog:           string(auditJSON1),
		},
		{
			RequestNumber:    "ESIGN/2026/00002",
			CompanyID:        1,
			DocumentType:     "purchase_order",
			DocumentID:       1,
			DocumentNumber:   "PO/2026/00001",
			Title:            "Surat Pesanan Pembelian Bahan Baku Baja & Komponen",
			Description:      "Otorisasi pesanan pengadaan bahan baku baja lembaran untuk lini permesinan CNC.",
			SignerName:       "Bambang Soedjarwo, S.T.",
			SignerEmail:      "procurement.head@holding.co.id",
			SignerRole:       "Kepala Pengadaan & Supply Chain",
			Status:           "pending",
			SecurityToken:    "f1e2d3c4b5a60718293a4b5c6d7e8f99",
			ExpiryDate:       &expiryTime,
			WatermarkEnabled: true,
			WatermarkText:    "DRAFT PO - AWAITING DIGITAL APPROVAL",
			WatermarkQRData:  "https://erp.internal/verify/esign?token=f1e2d3c4b5a60718293a4b5c6d7e8f99&doc=PO/2026/00001",
			AuditTrailLog:    `[{"timestamp":"2026-10-10T08:00:00Z","action":"created","actor":"Procurement Officer","ip":"127.0.0.1","user_agent":"ERP Admin","details":"Permohonan tanda tangan PO/2026/00001 dibuat"}]`,
		},
	}

	for _, req := range sampleRequests {
		if err := db.Create(&req).Error; err != nil {
			return err
		}
	}

	log.Printf("✅ Digital Signature seeded successfully: %d Requests (1 Signed, 1 Pending)", len(sampleRequests))
	return nil
}
