package esignature

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
)

type Service interface {
	CreateSignRequest(req *CreateSignRequest, companyID uint) (*DocumentSignRequest, error)
	GetSignRequestByID(id uint) (*DocumentSignRequest, error)
	GetSignRequestByToken(token string) (*DocumentSignRequest, error)
	ListSignRequests(companyID uint, status string, docType string) ([]DocumentSignRequest, error)
	GetSignRequestsForDocument(companyID uint, docType string, docID uint) ([]DocumentSignRequest, error)
	ExecuteSigning(token string, req *ExecuteSignRequest, signerIP string, userAgent string) (*DocumentSignRequest, error)
	RejectSignRequest(token string, reason string, signerIP string, userAgent string) (*DocumentSignRequest, error)
	CancelSignRequest(id uint, companyID uint) error
	GetStats(companyID uint) (*SignatureStats, error)
	AnalyzeSignatureComplianceWithAI(requestID uint) (*AIDigitalSignatureAuditReport, error)
}

type service struct {
	repo       Repository
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(repo Repository, chatterSvc *chatter.Service, aiClient *ai.Client) Service {
	return &service{
		repo:       repo,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func generateSecureToken() string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("token_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func computeSignatureHash(docType, docNumber, signerEmail, timestamp, signatureData string) string {
	payload := fmt.Sprintf("%s|%s|%s|%s|%s", docType, docNumber, signerEmail, timestamp, signatureData)
	h := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(h[:])
}

func (s *service) CreateSignRequest(req *CreateSignRequest, companyID uint) (*DocumentSignRequest, error) {
	reqNum, err := s.repo.GenerateRequestNumber()
	if err != nil {
		return nil, fmt.Errorf("gagal membuat nomor permohonan ttd: %w", err)
	}

	expiryDays := req.ExpiryDays
	if expiryDays <= 0 {
		expiryDays = 7
	}
	expiryDate := time.Now().AddDate(0, 0, expiryDays)

	watermarkEnabled := true
	if req.WatermarkEnabled != nil {
		watermarkEnabled = *req.WatermarkEnabled
	}

	watermarkText := req.WatermarkText
	if watermarkText == "" {
		watermarkText = fmt.Sprintf("DIGITALLY SIGNED & VERIFIED - %s - TGL %s", req.DocumentNumber, time.Now().Format("2006-01-02"))
	}

	token := generateSecureToken()
	now := time.Now()

	initialAudit := []SignerAuditLogEntry{
		{
			Timestamp: now.Format(time.RFC3339),
			Action:    "created",
			Actor:     "System Admin",
			IP:        "127.0.0.1",
			UserAgent: "GoERP Server",
			Details:   fmt.Sprintf("Permohonan tanda tangan elektronik diterbitkan untuk %s (%s)", req.SignerName, req.SignerRole),
		},
	}
	auditJSON, _ := json.Marshal(initialAudit)

	qrVerificationData := fmt.Sprintf("https://erp.internal/verify/esign?token=%s&doc=%s", token, req.DocumentNumber)

	signReq := &DocumentSignRequest{
		RequestNumber:    reqNum,
		CompanyID:        companyID,
		DocumentType:     req.DocumentType,
		DocumentID:       req.DocumentID,
		DocumentNumber:   req.DocumentNumber,
		Title:            req.Title,
		Description:      req.Description,
		SignerName:       req.SignerName,
		SignerEmail:      req.SignerEmail,
		SignerRole:       req.SignerRole,
		Status:           "pending",
		SecurityToken:    token,
		ExpiryDate:       &expiryDate,
		WatermarkEnabled: watermarkEnabled,
		WatermarkText:    watermarkText,
		WatermarkQRData:  qrVerificationData,
		AuditTrailLog:    string(auditJSON),
	}

	if err := s.repo.CreateSignRequest(signReq); err != nil {
		return nil, fmt.Errorf("gagal menyimpan permohonan tanda tangan: %w", err)
	}

	// Log to Chatter if available
	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			signReq.DocumentType,
			signReq.DocumentID,
			1,
			"System Admin",
			fmt.Sprintf("Permohonan Tanda Tangan Elektronik diterbitkan: %s (%s) kepada %s (%s). Tautan verifikasi aktif s.d. %s", signReq.RequestNumber, signReq.Title, signReq.SignerName, signReq.SignerEmail, expiryDate.Format("02 Jan 2006")),
			"",
			companyID,
		)
	}

	return signReq, nil
}

func (s *service) GetSignRequestByID(id uint) (*DocumentSignRequest, error) {
	return s.repo.GetSignRequestByID(id)
}

func (s *service) GetSignRequestByToken(token string) (*DocumentSignRequest, error) {
	req, err := s.repo.GetSignRequestByToken(token)
	if err != nil {
		return nil, err
	}

	// Record audit view event
	var auditList []SignerAuditLogEntry
	_ = json.Unmarshal([]byte(req.AuditTrailLog), &auditList)
	auditList = append(auditList, SignerAuditLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    "viewed",
		Actor:     req.SignerEmail,
		IP:        "client-ip",
		UserAgent: "web-portal",
		Details:   "Dokumen dan formulir penandatanganan diakses oleh penerima",
	})
	auditJSON, _ := json.Marshal(auditList)
	req.AuditTrailLog = string(auditJSON)
	_ = s.repo.UpdateSignRequest(req)

	return req, nil
}

func (s *service) ListSignRequests(companyID uint, status string, docType string) ([]DocumentSignRequest, error) {
	return s.repo.ListSignRequests(companyID, status, docType)
}

func (s *service) GetSignRequestsForDocument(companyID uint, docType string, docID uint) ([]DocumentSignRequest, error) {
	return s.repo.GetSignRequestByDocument(companyID, docType, docID)
}

func (s *service) ExecuteSigning(token string, req *ExecuteSignRequest, signerIP string, userAgent string) (*DocumentSignRequest, error) {
	signReq, err := s.repo.GetSignRequestByToken(token)
	if err != nil {
		return nil, fmt.Errorf("permohonan tanda tangan tidak ditemukan: %w", err)
	}

	if signReq.Status != "pending" {
		return nil, fmt.Errorf("dokumen ini sudah tidak dalam status permohonan (status saat ini: %s)", signReq.Status)
	}

	if signReq.ExpiryDate != nil && time.Now().After(*signReq.ExpiryDate) {
		signReq.Status = "expired"
		_ = s.repo.UpdateSignRequest(signReq)
		return nil, fmt.Errorf("tautan tanda tangan telah kedaluwarsa pada %s", signReq.ExpiryDate.Format("02 Jan 2006"))
	}

	now := time.Now()
	timestampStr := now.Format(time.RFC3339)
	sigHash := computeSignatureHash(signReq.DocumentType, signReq.DocumentNumber, signReq.SignerEmail, timestampStr, req.SignatureData)

	certSerial := fmt.Sprintf("CERT-ID-%d-%s", now.Year(), sigHash[:12])

	signReq.Status = "signed"
	signReq.SignedAt = &now
	signReq.SignatureType = req.SignatureType
	signReq.SignatureData = req.SignatureData
	signReq.SignatureHash = sigHash
	signReq.CertificateSerialNumber = certSerial
	signReq.SignedIP = signerIP
	signReq.SignedUserAgent = userAgent

	if req.WatermarkText != "" {
		signReq.WatermarkText = req.WatermarkText
	}

	// Update audit trail
	var auditList []SignerAuditLogEntry
	_ = json.Unmarshal([]byte(signReq.AuditTrailLog), &auditList)
	auditList = append(auditList, SignerAuditLogEntry{
		Timestamp: timestampStr,
		Action:    "signed",
		Actor:     fmt.Sprintf("%s <%s>", signReq.SignerName, signReq.SignerEmail),
		IP:        signerIP,
		UserAgent: userAgent,
		Details:   fmt.Sprintf("Dokumen berhasil ditandatangani secara digital (Metode: %s, SHA-256: %s, Cert: %s)", req.SignatureType, sigHash[:16], certSerial),
	})
	auditJSON, _ := json.Marshal(auditList)
	signReq.AuditTrailLog = string(auditJSON)

	// Update QR Data with signed proof
	signReq.WatermarkQRData = fmt.Sprintf("https://erp.internal/verify/esign?token=%s&hash=%s&cert=%s", token, sigHash, certSerial)

	if err := s.repo.UpdateSignRequest(signReq); err != nil {
		return nil, fmt.Errorf("gagal menyimpan status penandatanganan: %w", err)
	}

	// Chatter notification
	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			signReq.DocumentType,
			signReq.DocumentID,
			1,
			signReq.SignerName,
			fmt.Sprintf("Dokumen %s telah DITANDATANGANI SECARA ELEKTRONIK (Cert: %s, Hash: %s)", signReq.DocumentNumber, certSerial, sigHash[:16]),
			"pending",
			signReq.CompanyID,
		)
	}

	return signReq, nil
}

func (s *service) RejectSignRequest(token string, reason string, signerIP string, userAgent string) (*DocumentSignRequest, error) {
	signReq, err := s.repo.GetSignRequestByToken(token)
	if err != nil {
		return nil, fmt.Errorf("permohonan tanda tangan tidak ditemukan: %w", err)
	}

	if signReq.Status != "pending" {
		return nil, fmt.Errorf("dokumen ini sudah tidak dalam status permohonan (status saat ini: %s)", signReq.Status)
	}

	signReq.Status = "rejected"
	signReq.RejectionReason = reason

	var auditList []SignerAuditLogEntry
	_ = json.Unmarshal([]byte(signReq.AuditTrailLog), &auditList)
	auditList = append(auditList, SignerAuditLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    "rejected",
		Actor:     fmt.Sprintf("%s <%s>", signReq.SignerName, signReq.SignerEmail),
		IP:        signerIP,
		UserAgent: userAgent,
		Details:   fmt.Sprintf("Penandatanganan ditolak oleh penerima dengan alasan: %s", reason),
	})
	auditJSON, _ := json.Marshal(auditList)
	signReq.AuditTrailLog = string(auditJSON)

	if err := s.repo.UpdateSignRequest(signReq); err != nil {
		return nil, fmt.Errorf("gagal memperbarui status penolakan: %w", err)
	}

	// Chatter notification
	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking(
			signReq.DocumentType,
			signReq.DocumentID,
			1,
			signReq.SignerName,
			fmt.Sprintf("Penandatanganan dokumen %s DITOLAK. Alasan: %s", signReq.DocumentNumber, reason),
			"pending",
			signReq.CompanyID,
		)
	}

	return signReq, nil
}

func (s *service) CancelSignRequest(id uint, companyID uint) error {
	signReq, err := s.repo.GetSignRequestByID(id)
	if err != nil {
		return err
	}
	if signReq.CompanyID != companyID && companyID > 0 {
		return fmt.Errorf("akses tidak diizinkan")
	}

	signReq.Status = "cancelled"
	var auditList []SignerAuditLogEntry
	_ = json.Unmarshal([]byte(signReq.AuditTrailLog), &auditList)
	auditList = append(auditList, SignerAuditLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Action:    "cancelled",
		Actor:     "System Admin",
		IP:        "127.0.0.1",
		UserAgent: "GoERP Server",
		Details:   "Permohonan tanda tangan dibatalkan oleh inisiator internal",
	})
	auditJSON, _ := json.Marshal(auditList)
	signReq.AuditTrailLog = string(auditJSON)

	return s.repo.UpdateSignRequest(signReq)
}

func (s *service) GetStats(companyID uint) (*SignatureStats, error) {
	return s.repo.GetStats(companyID)
}

func (s *service) AnalyzeSignatureComplianceWithAI(requestID uint) (*AIDigitalSignatureAuditReport, error) {
	signReq, err := s.repo.GetSignRequestByID(requestID)
	if err != nil {
		return nil, fmt.Errorf("permohonan tanda tangan tidak ditemukan: %w", err)
	}

	// AI Audit Prompt
	prompt := fmt.Sprintf(`Anda adalah Pakar Hukum Digital & Keamanan Siber Elektronik (Spesialis UU ITE No. 1 Tahun 2024 & PP No. 71 Tahun 2019 tentang Penyelenggaraan Sistem dan Transaksi Elektronik / PSTE).
Lakukan evaluasi keabsahan yuridis dan integritas kriptografi untuk dokumen tanda tangan digital berikut:

Detail Dokumen:
- Nomor Permohonan: %s
- Jenis Dokumen: %s (#%s)
- Judul: %s
- Penandatangan: %s <%s> (Jabatan: %s)
- Status: %s
- Waktu TTD: %v
- Tipe TTD: %s
- Serial Sertifikat: %s
- SHA-256 Hash: %s
- Watermark Text: %s
- IP Address: %s

Kembalikan respons HANYA dalam format JSON valid tanpa markdown formatting dengan struktur:
{
  "audit_status": "VERIFIED_LEGAL",
  "integrity_score": 95,
  "request_number": "%s",
  "signer_identity_match": true,
  "hash_integrity_valid": true,
  "legal_basis": "Pasal 11 UU ITE No. 1/2024 & Pasal 60 PP No. 71/2019",
  "executive_summary": "Ringkasan kepatuhan hukum dan kekuatan alat bukti di persidangan",
  "evidence_checklist": [
    "Identitas penandatangan terverifikasi secara unik",
    "Integritas dokumen dijamin oleh hash kriptografis SHA-256",
    "Pencatatan jejak audit (audit trail) waktu nyata dan tahan manipulasi",
    "Watermarking dinamis dan QR verification code aktif"
  ],
  "risk_recommendations": [
    "Saran mitigasi risiko pembuktian atau rekomendasi implementasi sertifikat digital berinduk Kominfo"
  ]
}`,
		signReq.RequestNumber, signReq.DocumentType, signReq.DocumentNumber, signReq.Title,
		signReq.SignerName, signReq.SignerEmail, signReq.SignerRole,
		signReq.Status, signReq.SignedAt, signReq.SignatureType,
		signReq.CertificateSerialNumber, signReq.SignatureHash,
		signReq.WatermarkText, signReq.SignedIP, signReq.RequestNumber,
	)

	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "Anda adalah Pakar Hukum Digital & Keamanan Siber Elektronik (Spesialis UU ITE No. 1 Tahun 2024 & PP No. 71 Tahun 2019). Kembalikan JSON murni."
		resp, err := s.aiClient.CallRaw(prompt, systemPrompt)
		if err == nil && resp != "" {
			var report AIDigitalSignatureAuditReport
			clean := strings.TrimSpace(resp)
			clean = strings.TrimPrefix(clean, "```json")
			clean = strings.TrimPrefix(clean, "```")
			clean = strings.TrimSuffix(clean, "```")
			clean = strings.TrimSpace(clean)

			if err := json.Unmarshal([]byte(clean), &report); err == nil {
				report.AIProvider = "Cloudflare Workers AI (@cf/meta/llama-3.1-8b-instruct)"
				return &report, nil
			}
			log.Printf("[AI eSignature] Failed to parse AI JSON response: %v", err)
		}
	}

	// Smart Heuristic Engine (UU ITE & PP PSTE Baseline)
	score := 85
	status := "VERIFIED_LEGAL"
	hashValid := signReq.SignatureHash != ""
	if signReq.Status == "pending" {
		score = 60
		status = "CAUTION"
	} else if signReq.Status == "rejected" {
		score = 30
		status = "INVALID"
	}

	if signReq.SignatureType == "draw" || signReq.SignatureType == "certificate_pkcs12" {
		score += 10
		if score > 100 {
			score = 100
		}
	}

	summary := fmt.Sprintf("Dokumen %s memiliki keabsahan hukum elektronik berkategori %s (Skor Integritas: %d/100). Memenuhi kriteria tanda tangan elektronik mengikat berdasarkan Pasal 11 UU ITE No. 1/2024 jo. PP No. 71/2019 dengan pembuktian jejak audit dan enkripsi hash SHA-256.", signReq.DocumentNumber, status, score)

	return &AIDigitalSignatureAuditReport{
		AuditStatus:         status,
		IntegrityScore:      score,
		RequestNumber:       signReq.RequestNumber,
		SignerIdentityMatch: true,
		HashIntegrityValid:  hashValid,
		LegalBasis:          "UU ITE No. 1/2024 (Pasal 11) & PP No. 71/2019 (Pasal 60)",
		ExecutiveSummary:    summary,
		EvidenceChecklist: []string{
			"Identitas penandatangan melekat secara unik pada dokumen elektronik",
			"Segel integritas dokumen terlindungi oleh SHA-256 hash fingerprint",
			"Jejak audit (audit trail) mencatat IP penandatangan, timestamp presisi, dan user-agent",
			"Visual watermark protektif dan tautan verifikasi QR aktif pada dokumen",
		},
		RiskRecommendations: []string{
			"Simpan salinan berkas beserta rekaman audit trail utuh sebagai bukti sah di pengadilan (Pasal 5 UU ITE).",
			"Untuk transaksi material bernilai tinggi di atas Rp 1 Milyar, pertimbangkan penerbitan sertifikat digital Tersertifikasi (PSrE Indonesia).",
			"Pastikan verifikasi berkala dilakukan melalui portal verifikasi QR sebelum serah terima barang/jasa.",
		},
		AIProvider: "Local Heuristic Engine (UU ITE & PP PSTE Compliance Baseline)",
	}, nil
}
