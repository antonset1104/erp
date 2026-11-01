package inbound_webhook

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/crm"
	"erp-golang/internal/modules/stock"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(db *gorm.DB, repo *Repository, accountSvc *account.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// ---------------------------------------------------------------------------
// Security, Nonce & Signature Helpers
// ---------------------------------------------------------------------------

func GenerateRandomEndpointKey(prefix string) string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(bytes))
}

func GenerateRandomSecret() string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("whsec_in_%d", time.Now().UnixNano())
	}
	return "whsec_in_" + hex.EncodeToString(bytes)
}

func ComputeHMACSHA256(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func ComputeSHA512(input string) string {
	h := sha512.New()
	h.Write([]byte(input))
	return hex.EncodeToString(h.Sum(nil))
}

// ---------------------------------------------------------------------------
// Endpoint Management CRUD
// ---------------------------------------------------------------------------

func (s *Service) CreateEndpoint(req CreateInboundEndpointRequest, userID uint, companyID uint) (*InboundWebhookEndpoint, error) {
	code, err := s.repo.GenerateEndpointCode(companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate endpoint code: %w", err)
	}

	key := strings.TrimSpace(req.EndpointKey)
	if key == "" {
		providerSlug := strings.ToLower(strings.ReplaceAll(req.ProviderType, "_", ""))
		key = GenerateRandomEndpointKey("ep_" + providerSlug)
	}

	authType := strings.TrimSpace(req.AuthType)
	if authType == "" {
		authType = AuthHMACSHA256
	}

	secretKey := strings.TrimSpace(req.SecretKey)
	if secretKey == "" && authType != AuthNone {
		secretKey = GenerateRandomSecret()
	}

	autoAction := strings.TrimSpace(req.AutoAction)
	if autoAction == "" {
		autoAction = ActionLogOnly
	}

	endpoint := &InboundWebhookEndpoint{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			Active:    true,
			CreateUID: userID,
			WriteUID:  userID,
		},
		EndpointCode: code,
		EndpointKey:  key,
		Name:         req.Name,
		ProviderType: req.ProviderType,
		AuthType:     authType,
		SecretKey:    secretKey,
		TargetModel:  req.TargetModel,
		AutoAction:   autoAction,
		AllowedIPs:   req.AllowedIPs,
		IsActive:     true,
		Description:  req.Description,
	}

	if err := s.repo.CreateEndpoint(endpoint); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		s.chatterSvc.LogTracking("inbound.webhook.endpoint", endpoint.ID, userID, "System",
			fmt.Sprintf("Mendaftarkan endpoint inbound webhook '%s' (%s, Key: %s)", endpoint.Name, endpoint.EndpointCode, endpoint.EndpointKey),
			"", companyID)
	}

	return endpoint, nil
}

func (s *Service) GetEndpoint(id uint, companyID uint) (*InboundWebhookEndpoint, error) {
	return s.repo.GetEndpointByID(id, companyID)
}

func (s *Service) GetEndpointByKey(key string) (*InboundWebhookEndpoint, error) {
	return s.repo.GetEndpointByKey(key)
}

func (s *Service) ListEndpoints(companyID uint) ([]InboundWebhookEndpoint, error) {
	return s.repo.ListEndpoints(companyID)
}

func (s *Service) UpdateEndpoint(id uint, req UpdateInboundEndpointRequest, userID uint, companyID uint) (*InboundWebhookEndpoint, error) {
	endpoint, err := s.repo.GetEndpointByID(id, companyID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		endpoint.Name = req.Name
	}
	if req.ProviderType != "" {
		endpoint.ProviderType = req.ProviderType
	}
	if req.AuthType != "" {
		endpoint.AuthType = req.AuthType
	}
	if req.SecretKey != "" {
		endpoint.SecretKey = req.SecretKey
	}
	if req.TargetModel != "" {
		endpoint.TargetModel = req.TargetModel
	}
	if req.AutoAction != "" {
		endpoint.AutoAction = req.AutoAction
	}
	if req.AllowedIPs != "" {
		endpoint.AllowedIPs = req.AllowedIPs
	}
	if req.IsActive != nil {
		endpoint.IsActive = *req.IsActive
	}
	if req.Description != "" {
		endpoint.Description = req.Description
	}
	endpoint.WriteUID = userID

	if err := s.repo.UpdateEndpoint(endpoint); err != nil {
		return nil, err
	}

	return endpoint, nil
}

func (s *Service) DeleteEndpoint(id uint, companyID uint) error {
	return s.repo.DeleteEndpoint(id, companyID)
}

// ---------------------------------------------------------------------------
// Inbound Ingestion Engine & Automated Action Dispatcher
// ---------------------------------------------------------------------------

func (s *Service) ProcessInboundWebhook(endpointKey string, headers map[string]string, rawBody []byte, sourceIP string) (*InboundWebhookLog, error) {
	startTime := time.Now()

	// 1. Locate Endpoint
	endpoint, err := s.repo.GetEndpointByKey(endpointKey)
	if err != nil {
		return nil, fmt.Errorf("inbound webhook endpoint '%s' not found or inactive", endpointKey)
	}

	logCode, err := s.repo.GenerateLogCode(endpoint.CompanyID)
	if err != nil {
		logCode = fmt.Sprintf("INLOG/%d/%d", time.Now().Year(), time.Now().UnixNano()%100000)
	}

	headersJSON, _ := json.Marshal(headers)
	inboundLog := &InboundWebhookLog{
		ModelBase: models.ModelBase{
			CompanyID: endpoint.CompanyID,
			Active:    true,
		},
		LogCode:        logCode,
		EndpointID:     endpoint.ID,
		EndpointKey:    endpoint.EndpointKey,
		SourceIP:       sourceIP,
		HTTPMethod:     "POST",
		RequestHeaders: string(headersJSON),
		PayloadJSON:    string(rawBody),
		ProcessStatus:  StatusProcessed,
		HTTPStatusCode: 200,
	}

	// 2. IP Whitelisting Check (if configured)
	if endpoint.AllowedIPs != "" && sourceIP != "" {
		allowed := false
		for _, ip := range strings.Split(endpoint.AllowedIPs, ",") {
			if strings.TrimSpace(ip) == sourceIP || strings.TrimSpace(ip) == "127.0.0.1" || strings.TrimSpace(ip) == "::1" {
				allowed = true
				break
			}
		}
		if !allowed {
			inboundLog.ProcessStatus = StatusRejected
			inboundLog.HTTPStatusCode = 403
			inboundLog.ErrorMessage = fmt.Sprintf("Access Denied: Source IP '%s' is not in allowed whitelist", sourceIP)
			inboundLog.DurationMs = time.Since(startTime).Milliseconds()
			s.repo.CreateLog(inboundLog)
			s.repo.IncrementEndpointCall(endpoint.ID, StatusRejected)
			return inboundLog, fmt.Errorf("ip not allowed: %s", sourceIP)
		}
	}

	// 3. Signature & Authentication Verification
	sigValid, sigReceived := s.verifyInboundSignature(endpoint, headers, rawBody)
	inboundLog.SignatureReceived = sigReceived
	inboundLog.SignatureValid = sigValid

	if !sigValid && endpoint.AuthType != AuthNone {
		inboundLog.ProcessStatus = StatusRejected
		inboundLog.HTTPStatusCode = 401
		inboundLog.ErrorMessage = fmt.Sprintf("Signature verification failed for AuthType '%s'. Received signature mismatch.", endpoint.AuthType)
		inboundLog.DurationMs = time.Since(startTime).Milliseconds()
		s.repo.CreateLog(inboundLog)
		s.repo.IncrementEndpointCall(endpoint.ID, StatusRejected)
		return inboundLog, fmt.Errorf("invalid signature")
	}

	// 4. Parse JSON Payload
	var parsedPayload map[string]interface{}
	if err := json.Unmarshal(rawBody, &parsedPayload); err != nil {
		inboundLog.ProcessStatus = StatusError
		inboundLog.HTTPStatusCode = 400
		inboundLog.ErrorMessage = fmt.Sprintf("Malformed JSON payload: %v", err)
		inboundLog.DurationMs = time.Since(startTime).Milliseconds()
		s.repo.CreateLog(inboundLog)
		s.repo.IncrementEndpointCall(endpoint.ID, StatusError)
		return inboundLog, fmt.Errorf("invalid json payload: %w", err)
	}

	// 5. Automated Action Dispatcher
	actionResult, targetID, targetName, err := s.dispatchAutomatedAction(endpoint, parsedPayload)
	if err != nil {
		inboundLog.ProcessStatus = StatusError
		inboundLog.HTTPStatusCode = 500
		inboundLog.ErrorMessage = fmt.Sprintf("Action execution failed: %v", err)
		inboundLog.ActionTaken = actionResult
	} else {
		inboundLog.ProcessStatus = StatusProcessed
		inboundLog.HTTPStatusCode = 200
		inboundLog.ActionTaken = actionResult
		inboundLog.TargetResID = targetID
		inboundLog.TargetResName = targetName
	}

	inboundLog.DurationMs = time.Since(startTime).Milliseconds()
	if err := s.repo.CreateLog(inboundLog); err != nil {
		return nil, err
	}

	s.repo.IncrementEndpointCall(endpoint.ID, inboundLog.ProcessStatus)

	return inboundLog, nil
}

func (s *Service) verifyInboundSignature(endpoint *InboundWebhookEndpoint, headers map[string]string, rawBody []byte) (bool, string) {
	// Normalized header getter
	getHeader := func(names ...string) string {
		for _, name := range names {
			for k, v := range headers {
				if strings.EqualFold(k, name) {
					return strings.TrimSpace(v)
				}
			}
		}
		return ""
	}

	switch endpoint.AuthType {
	case AuthNone:
		return true, "none"

	case AuthBearer:
		authHdr := getHeader("Authorization", "X-Authorization", "X-Token")
		receivedToken := strings.TrimPrefix(authHdr, "Bearer ")
		return (receivedToken != "" && receivedToken == endpoint.SecretKey), receivedToken

	case AuthHMACSHA256:
		sigHdr := getHeader("X-Signature", "X-Webhook-Signature", "X-Hub-Signature-256")
		cleanSig := strings.TrimPrefix(sigHdr, "sha256=")
		expectedSig := ComputeHMACSHA256(rawBody, endpoint.SecretKey)
		return hmac.Equal([]byte(strings.ToLower(cleanSig)), []byte(strings.ToLower(expectedSig))), sigHdr

	case AuthSHA512:
		// Midtrans or SHA512 hash validation
		sigHdr := getHeader("X-Signature", "signature_key", "X-Midtrans-Signature")
		if sigHdr == "" {
			// Check if signature is embedded inside JSON body
			var bodyMap map[string]interface{}
			if err := json.Unmarshal(rawBody, &bodyMap); err == nil {
				if v, ok := bodyMap["signature_key"].(string); ok {
					sigHdr = v
				}
			}
		}

		// Try standard Midtrans format: sha512(order_id + status_code + gross_amount + ServerKey)
		var bodyMap map[string]interface{}
		if err := json.Unmarshal(rawBody, &bodyMap); err == nil {
			orderID, _ := bodyMap["order_id"].(string)
			statusCode, _ := bodyMap["status_code"].(string)
			grossAmount, _ := bodyMap["gross_amount"].(string)

			if orderID != "" && statusCode != "" && grossAmount != "" {
				midtransSignPayload := orderID + statusCode + grossAmount + endpoint.SecretKey
				expectedSig := ComputeSHA512(midtransSignPayload)
				if strings.EqualFold(sigHdr, expectedSig) {
					return true, sigHdr
				}
			}
		}

		// Fallback: direct sha512(rawBody + secret)
		directExpected := ComputeSHA512(string(rawBody) + endpoint.SecretKey)
		return strings.EqualFold(sigHdr, directExpected), sigHdr

	default:
		return true, "unspecified"
	}
}

func (s *Service) dispatchAutomatedAction(endpoint *InboundWebhookEndpoint, payload map[string]interface{}) (actionSummary string, targetID uint, targetName string, err error) {
	switch endpoint.AutoAction {

	// 1. Payment Gateway callback -> Auto mark invoice paid & double-entry payment posting
	case ActionMarkInvoicePaid:
		// Locate order/invoice identifier
		var invoiceRef string
		for _, key := range []string{"order_id", "invoice_id", "external_id", "invoice_number", "memo"} {
			if val, ok := payload[key]; ok {
				invoiceRef = fmt.Sprintf("%v", val)
				break
			}
		}

		if invoiceRef == "" {
			return "No invoice reference (order_id/invoice_id) detected in payload", 0, "", nil
		}

		// Check payment status from payload
		isPaidStatus := false
		for _, key := range []string{"transaction_status", "status", "payment_status", "event"} {
			if val, ok := payload[key]; ok {
				statusStr := strings.ToLower(fmt.Sprintf("%v", val))
				if statusStr == "settlement" || statusStr == "capture" || statusStr == "paid" || statusStr == "success" || statusStr == "completed" {
					isPaidStatus = true
					break
				}
			}
		}

		if !isPaidStatus {
			return fmt.Sprintf("Inbound payment status for '%s' is not settlement/paid (ignored)", invoiceRef), 0, invoiceRef, nil
		}

		// Locate Invoice in Database
		var move account.AccountMove
		errMove := s.db.Where("(id = ? OR name = ? OR ref = ?) AND company_id = ?",
			invoiceRef, invoiceRef, invoiceRef, endpoint.CompanyID).First(&move).Error
		if errMove != nil {
			return fmt.Sprintf("Invoice '%s' not found in database", invoiceRef), 0, invoiceRef, nil
		}

		if move.PaymentState == "paid" {
			return fmt.Sprintf("Faktur '%s' sudah berstatus lunas sebelumnya", move.Name), move.ID, move.Name, nil
		}

		// Locate Cash/Bank Journal
		var journal account.AccountJournal
		s.db.Where("company_id = ? AND type IN ('bank','cash')", endpoint.CompanyID).First(&journal)
		journalID := journal.ID
		if journalID == 0 {
			journalID = 1
		}

		paymentAmount := move.AmountTotal
		if amt, ok := payload["gross_amount"]; ok {
			var parsedAmt float64
			fmt.Sscanf(fmt.Sprintf("%v", amt), "%f", &parsedAmt)
			if parsedAmt > 0 {
				paymentAmount = parsedAmt
			}
		}

		// Register Payment through account service
		memo := fmt.Sprintf("Pembayaran Online Otomatis via %s (Ref: %s)", endpoint.Name, invoiceRef)
		if s.accountSvc != nil && move.State == "posted" {
			_, _, errPay := s.accountSvc.RegisterPayment(move.ID, paymentAmount, journalID, memo, endpoint.CompanyID)
			if errPay != nil {
				return fmt.Sprintf("Gagal registrasi pembayaran faktur: %v", errPay), move.ID, move.Name, errPay
			}
		} else {
			// Direct fallback update
			s.db.Model(&account.AccountMove{}).Where("id = ?", move.ID).Update("payment_state", "paid")
		}

		if s.chatterSvc != nil {
			s.chatterSvc.LogTracking("account.move", move.ID, 0, "Webhook Gateway",
				fmt.Sprintf("Faktur dilunasi otomatis melalui Inbound Webhook Callback '%s' (Nominal: Rp %.2f)", endpoint.Name, paymentAmount),
				"", endpoint.CompanyID)
		}

		return fmt.Sprintf("Faktur '%s' berhasil dilunasi otomatis sebesar Rp %.2f via %s", move.Name, paymentAmount, endpoint.Name), move.ID, move.Name, nil

	// 2. Logistics 3PL Tracking Updates
	case ActionUpdatePickingTracking:
		var pickingRef string
		for _, key := range []string{"picking_id", "picking_name", "tracking_number", "waybill", "resi", "order_id"} {
			if val, ok := payload[key]; ok {
				pickingRef = fmt.Sprintf("%v", val)
				break
			}
		}

		if pickingRef == "" {
			return "No picking or tracking identifier found in payload", 0, "", nil
		}

		var courierStatus string
		for _, key := range []string{"status", "tracking_status", "delivery_status", "shipping_status"} {
			if val, ok := payload[key]; ok {
				courierStatus = fmt.Sprintf("%v", val)
				break
			}
		}

		var picking stock.StockPicking
		errPick := s.db.Where("(id = ? OR name = ? OR origin = ?) AND company_id = ?",
			pickingRef, pickingRef, pickingRef, endpoint.CompanyID).First(&picking).Error
		if errPick != nil {
			return fmt.Sprintf("Dokumen pengiriman gudang '%s' tidak ditemukan", pickingRef), 0, pickingRef, nil
		}

		updateNote := fmt.Sprintf("Update Ekspedisi (%s): Status '%s' terverifikasi pada %s",
			endpoint.Name, courierStatus, time.Now().Format("2006-01-02 15:04"))

		s.db.Model(&stock.StockPicking{}).Where("id = ?", picking.ID).
			Update("note", gorm.Expr("COALESCE(note, '') || '\n' || ?", updateNote))

		if strings.EqualFold(courierStatus, "delivered") || strings.EqualFold(courierStatus, "done") {
			s.db.Model(&stock.StockPicking{}).Where("id = ?", picking.ID).Update("date_done", time.Now().Format("2006-01-02 15:04:05"))
		}

		if s.chatterSvc != nil {
			s.chatterSvc.LogTracking("stock.picking", picking.ID, 0, "Logistics Courier",
				fmt.Sprintf("Update resi/status kurir: %s", updateNote), "", endpoint.CompanyID)
		}

		return fmt.Sprintf("Surat Jalan '%s' diperbarui dengan status kurir '%s'", picking.Name, courierStatus), picking.ID, picking.Name, nil

	// 3. CRM Lead Ingestion (Landing page / Typeform / Meta Lead Ads)
	case ActionCreateCRMLead:
		leadTitle := "Inbound Lead Webhook"
		for _, key := range []string{"title", "form_name", "campaign", "lead_name"} {
			if val, ok := payload[key]; ok && fmt.Sprintf("%v", val) != "" {
				leadTitle = fmt.Sprintf("%v", val)
				break
			}
		}

		contactName := ""
		for _, key := range []string{"name", "full_name", "contact_name", "lead_contact"} {
			if val, ok := payload[key]; ok {
				contactName = fmt.Sprintf("%v", val)
				break
			}
		}

		email := ""
		for _, key := range []string{"email", "work_email", "contact_email"} {
			if val, ok := payload[key]; ok {
				email = fmt.Sprintf("%v", val)
				break
			}
		}

		phone := ""
		for _, key := range []string{"phone", "mobile", "whatsapp", "phone_number"} {
			if val, ok := payload[key]; ok {
				phone = fmt.Sprintf("%v", val)
				break
			}
		}

		lead := crm.CrmLead{
			ModelBase: models.ModelBase{
				CompanyID: endpoint.CompanyID,
				Active:    true,
			},
			Name:        fmt.Sprintf("%s - %s", leadTitle, contactName),
			ContactName: contactName,
			Email:       email,
			Phone:       phone,
			Stage:       "new",
			Priority:    "1",
			Description: fmt.Sprintf("Dibuat otomatis dari Inbound Webhook '%s' pada %s", endpoint.Name, time.Now().Format("2006-01-02 15:04:05")),
		}

		if err := s.db.Create(&lead).Error; err != nil {
			return "Gagal membuat CRM Lead", 0, "", err
		}

		if s.chatterSvc != nil {
			s.chatterSvc.LogTracking("crm.lead", lead.ID, 0, "Webhook Ingestion",
				fmt.Sprintf("Prospek / Lead baru masuk dari integrasi '%s' (Kontak: %s / %s)", endpoint.Name, contactName, email),
				"", endpoint.CompanyID)
		}

		return fmt.Sprintf("Lead CRM '%s' (ID #%d) berhasil dibuat otomatis", lead.Name, lead.ID), lead.ID, lead.Name, nil

	// 4. Default: Log Only
	case ActionLogOnly:
		fallthrough
	default:
		return fmt.Sprintf("Payload diterima dan dicatat dalam audit log inbound '%s'", endpoint.Name), 0, "", nil
	}
}

// ---------------------------------------------------------------------------
// Inbound Simulator (Console Testing)
// ---------------------------------------------------------------------------

func (s *Service) SimulateInboundWebhook(req SimulateInboundRequest, companyID uint) (*InboundWebhookLog, error) {
	endpoint, err := s.repo.GetEndpointByKey(req.EndpointKey)
	if err != nil {
		return nil, fmt.Errorf("endpoint dengan key '%s' tidak ditemukan: %w", req.EndpointKey, err)
	}

	rawBody, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, fmt.Errorf("gagal serialisasi payload: %w", err)
	}

	headers := req.Headers
	if headers == nil {
		headers = make(map[string]string)
	}

	// Auto compute correct signature if not explicitly provided
	if _, ok := headers["X-Signature"]; !ok {
		if endpoint.AuthType == AuthHMACSHA256 {
			headers["X-Signature"] = "sha256=" + ComputeHMACSHA256(rawBody, endpoint.SecretKey)
		} else if endpoint.AuthType == AuthBearer {
			headers["Authorization"] = "Bearer " + endpoint.SecretKey
		}
	}

	return s.ProcessInboundWebhook(req.EndpointKey, headers, rawBody, "127.0.0.1 (Simulator)")
}

// ---------------------------------------------------------------------------
// Cloudflare Workers AI Inbound Payload Analyzer & Smart Heuristic Copilot
// ---------------------------------------------------------------------------

func (s *Service) AnalyzeInboundPayloadWithAI(payloadJSON string, companyID uint) (*AIInboundPayloadReport, error) {
	var bodyMap map[string]interface{}
	_ = json.Unmarshal([]byte(payloadJSON), &bodyMap)

	// Local Smart Heuristic Engine (Baseline)
	report := &AIInboundPayloadReport{
		FieldMappings: make(map[string]string),
		AIProvider:    "Local Heuristic Engine (Inbound Schema Baseline)",
	}

	// 1. Detect Provider & Mappings
	if _, ok := bodyMap["order_id"]; ok && (bodyMap["transaction_status"] != nil || bodyMap["status_code"] != nil) {
		report.DetectedProvider = "Midtrans Payment Gateway"
		report.SuggestedModel = "account.move"
		report.SuggestedAction = ActionMarkInvoicePaid
		report.ExtractedEntityID = fmt.Sprintf("%v", bodyMap["order_id"])
		report.ExtractedStatus = fmt.Sprintf("%v", bodyMap["transaction_status"])
		if amt, ok := bodyMap["gross_amount"]; ok {
			fmt.Sscanf(fmt.Sprintf("%v", amt), "%f", &report.ExtractedAmount)
		}
		report.FieldMappings["order_id"] = "account.move.name / ref"
		report.FieldMappings["gross_amount"] = "account.move.amount_total"
		report.FieldMappings["transaction_status"] = "account.move.payment_state"
		report.SecurityAssessment = "Format standar Midtrans Snap/Core API. Dianjurkan verifikasi SHA512 signature key."
		report.ExecutiveSummary = fmt.Sprintf("Payload terdeteksi sebagai notifikasi pembayaran Midtrans untuk pesanan '%s' sebesar Rp %.2f.",
			report.ExtractedEntityID, report.ExtractedAmount)

	} else if _, ok := bodyMap["external_id"]; ok && bodyMap["payment_method"] != nil {
		report.DetectedProvider = "Xendit Invoice / Payment Callback"
		report.SuggestedModel = "account.move"
		report.SuggestedAction = ActionMarkInvoicePaid
		report.ExtractedEntityID = fmt.Sprintf("%v", bodyMap["external_id"])
		report.ExtractedStatus = fmt.Sprintf("%v", bodyMap["status"])
		if amt, ok := bodyMap["amount"]; ok {
			fmt.Sscanf(fmt.Sprintf("%v", amt), "%f", &report.ExtractedAmount)
		}
		report.FieldMappings["external_id"] = "account.move.name / ref"
		report.FieldMappings["amount"] = "account.move.amount_total"
		report.SecurityAssessment = "Format webhook Xendit Invoice. Verifikasi token callback x-callback-token."
		report.ExecutiveSummary = fmt.Sprintf("Payload terdeteksi sebagai callback invoice Xendit ID '%s'.", report.ExtractedEntityID)

	} else if _, ok := bodyMap["tracking_number"]; ok || bodyMap["waybill"] != nil || bodyMap["resi"] != nil {
		report.DetectedProvider = "Logistics 3PL Courier API (JNE / SiCepat / J&T)"
		report.SuggestedModel = "stock.picking"
		report.SuggestedAction = ActionUpdatePickingTracking
		for _, k := range []string{"tracking_number", "waybill", "resi"} {
			if v, ok := bodyMap[k]; ok {
				report.ExtractedEntityID = fmt.Sprintf("%v", v)
				break
			}
		}
		for _, k := range []string{"status", "tracking_status"} {
			if v, ok := bodyMap[k]; ok {
				report.ExtractedStatus = fmt.Sprintf("%v", v)
				break
			}
		}
		report.FieldMappings["tracking_number"] = "stock.picking.origin / note"
		report.FieldMappings["status"] = "stock.picking.state"
		report.SecurityAssessment = "Format resi logistik ekspedisi. Pastikan menggunakan HMAC-SHA256 atau IP Whitelisting."
		report.ExecutiveSummary = fmt.Sprintf("Pembaruan pelacakan kurir logistik untuk no resi '%s' status '%s'.",
			report.ExtractedEntityID, report.ExtractedStatus)

	} else if _, ok := bodyMap["email"]; ok && (bodyMap["name"] != nil || bodyMap["contact_name"] != nil) {
		report.DetectedProvider = "Web Contact Form / CRM Lead Capture"
		report.SuggestedModel = "crm.lead"
		report.SuggestedAction = ActionCreateCRMLead
		report.ExtractedEntityID = fmt.Sprintf("%v", bodyMap["name"])
		report.ExtractedStatus = "new"
		report.FieldMappings["name"] = "crm.lead.name"
		report.FieldMappings["email"] = "crm.lead.email"
		report.FieldMappings["phone"] = "crm.lead.phone"
		report.SecurityAssessment = "Form submission eksternal. Direkomendasikan Bearer Token autentikasi."
		report.ExecutiveSummary = fmt.Sprintf("Prospek penjualan baru terdeteksi dengan kontak '%v' (%v).",
			bodyMap["name"], bodyMap["email"])

	} else {
		report.DetectedProvider = "Generic Inbound Webhook"
		report.SuggestedModel = "generic"
		report.SuggestedAction = ActionLogOnly
		report.SecurityAssessment = "Skema generik tanpa pola tetap. Gunakan mode Log Only untuk inspeksi."
		report.ExecutiveSummary = "Payload webhook generik. Parameter dicatat ke log untuk audit integrasi."
	}

	// 2. If Cloudflare Workers AI is configured, invoke edge LLM for deeper insights
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := `Anda adalah AI Integration Architect untuk ERP enterprise. Analisis payload JSON webhook masuk, identifikasi penyedia layanan, petakan field ke model database ERP, dan berikan rekomendasi keamanan. Format respons WAJIB JSON persis sesuai skema.`
		userPrompt := fmt.Sprintf(`Analisis JSON inbound webhook berikut:
%s

Format respons JSON yang diharapkan:
{
  "detected_provider": "...",
  "suggested_model": "...",
  "suggested_action": "...",
  "extracted_entity_id": "...",
  "extracted_amount": 0.0,
  "extracted_status": "...",
  "field_mappings": {"field_a": "model.field"},
  "security_assessment": "...",
  "executive_summary": "..."
}`, payloadJSON)

		rawResp, err := s.aiClient.CallRaw(userPrompt, systemPrompt)
		if err == nil {
			startIdx := strings.Index(rawResp, "{")
			endIdx := strings.LastIndex(rawResp, "}")
			if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
				var aiReport AIInboundPayloadReport
				if errJson := json.Unmarshal([]byte(rawResp[startIdx:endIdx+1]), &aiReport); errJson == nil {
					aiReport.AIProvider = "Cloudflare Workers AI (@cf/meta/llama-3.1-8b-instruct)"
					return &aiReport, nil
				}
			}
		}
	}

	return report, nil
}

// ---------------------------------------------------------------------------
// Logs & Stats
// ---------------------------------------------------------------------------

func (s *Service) ListLogs(filters InboundLogFilters) ([]InboundWebhookLog, int64, error) {
	return s.repo.ListLogs(filters)
}

func (s *Service) GetLog(id uint, companyID uint) (*InboundWebhookLog, error) {
	return s.repo.GetLogByID(id, companyID)
}

func (s *Service) GetStats(companyID uint) (*InboundWebhookStats, error) {
	return s.repo.GetStats(companyID)
}
