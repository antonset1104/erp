package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/chatter"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	chatterSvc *chatter.Service
	aiClient   *ai.Client
	httpClient *http.Client
}

func NewService(db *gorm.DB, repo *Repository, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ---------------------------------------------------------------------------
// Security & Signature Helpers
// ---------------------------------------------------------------------------

func GenerateRandomSecret() string {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("whsec_%d", time.Now().UnixNano())
	}
	return "whsec_" + hex.EncodeToString(bytes)
}

func ComputeHMACSignature(payload []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// ---------------------------------------------------------------------------
// Subscription Management
// ---------------------------------------------------------------------------

func (s *Service) CreateSubscription(req CreateWebhookRequest, userID uint, companyID uint) (*WebhookSubscription, error) {
	if companyID == 0 {
		companyID = 1
	}

	secret := strings.TrimSpace(req.SecretToken)
	if secret == "" {
		secret = GenerateRandomSecret()
	}

	maxRetries := 3
	if req.MaxRetries > 0 {
		maxRetries = req.MaxRetries
	}
	retryDelay := 60
	if req.RetryDelaySec > 0 {
		retryDelay = req.RetryDelaySec
	}
	timeoutSec := 10
	if req.TimeoutSec > 0 {
		timeoutSec = req.TimeoutSec
	}

	sub := &WebhookSubscription{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			CreateUID: userID,
			WriteUID:  userID,
			Active:    true,
		},
		Name:          req.Name,
		TargetURL:     req.TargetURL,
		SecretToken:   secret,
		EventTopics:   req.EventTopics,
		CustomHeaders: req.CustomHeaders,
		MaxRetries:    maxRetries,
		RetryDelaySec: retryDelay,
		TimeoutSec:    timeoutSec,
		IsActive:      true,
		LastStatus:    "idle",
		Description:   req.Description,
	}

	if err := s.repo.CreateSubscription(sub); err != nil {
		return nil, fmt.Errorf("failed to create webhook subscription: %w", err)
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking("webhook.subscription", sub.ID, userID, "System",
			fmt.Sprintf("Webhook endpoint '%s' didaftarkan untuk event: %s", sub.Name, sub.EventTopics), "", companyID)
	}

	return sub, nil
}

func (s *Service) GetSubscription(id uint, companyID uint) (*WebhookSubscription, error) {
	return s.repo.GetSubscriptionByID(id, companyID)
}

func (s *Service) ListSubscriptions(companyID uint) ([]WebhookSubscription, error) {
	return s.repo.ListSubscriptions(companyID)
}

func (s *Service) UpdateSubscription(id uint, req UpdateWebhookRequest, userID uint, companyID uint) (*WebhookSubscription, error) {
	sub, err := s.repo.GetSubscriptionByID(id, companyID)
	if err != nil {
		return nil, err
	}

	if req.Name != "" {
		sub.Name = req.Name
	}
	if req.TargetURL != "" {
		sub.TargetURL = req.TargetURL
	}
	if req.SecretToken != "" {
		sub.SecretToken = req.SecretToken
	}
	if req.EventTopics != "" {
		sub.EventTopics = req.EventTopics
	}
	if req.CustomHeaders != "" {
		sub.CustomHeaders = req.CustomHeaders
	}
	if req.MaxRetries > 0 {
		sub.MaxRetries = req.MaxRetries
	}
	if req.RetryDelaySec > 0 {
		sub.RetryDelaySec = req.RetryDelaySec
	}
	if req.TimeoutSec > 0 {
		sub.TimeoutSec = req.TimeoutSec
	}
	if req.IsActive != nil {
		sub.IsActive = *req.IsActive
	}
	if req.Description != "" {
		sub.Description = req.Description
	}

	sub.WriteUID = userID
	if err := s.repo.UpdateSubscription(sub); err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *Service) DeleteSubscription(id uint, companyID uint) error {
	return s.repo.DeleteSubscription(id, companyID)
}

func (s *Service) TestPing(id uint, companyID uint) (*WebhookDelivery, error) {
	sub, err := s.repo.GetSubscriptionByID(id, companyID)
	if err != nil {
		return nil, fmt.Errorf("webhook subscription #%d not found: %w", id, err)
	}

	pingData := map[string]interface{}{
		"message":   "Hello from GoERP Webhook Engine!",
		"ping_time": time.Now().Format(time.RFC3339),
		"endpoint":  sub.Name,
		"version":   "v1.27.0",
	}

	deliveries, err := s.PublishEvent(PublishEventRequest{
		EventName: EventWebhookPing,
		ModelName: "webhook.subscription",
		ResID:     sub.ID,
		Data:      pingData,
	}, companyID)
	if err != nil {
		return nil, err
	}

	for _, d := range deliveries {
		if d.SubscriptionID == sub.ID {
			return &d, nil
		}
	}

	// If no wildcard or ping topic on sub, manually dispatch single delivery
	return s.deliverToSubscription(sub, EventWebhookPing, "webhook.subscription", sub.ID, pingData, companyID)
}

// ---------------------------------------------------------------------------
// Event Publishing & Dispatch Engine
// ---------------------------------------------------------------------------

func (s *Service) PublishEvent(req PublishEventRequest, companyID uint) ([]WebhookDelivery, error) {
	if companyID == 0 {
		companyID = 1
	}

	subs, err := s.repo.GetActiveSubscriptionsForTopic(req.EventName, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active subscribers: %w", err)
	}

	var results []WebhookDelivery
	for _, sub := range subs {
		subCopy := sub
		dlv, err := s.deliverToSubscription(&subCopy, req.EventName, req.ModelName, req.ResID, req.Data, companyID)
		if err == nil && dlv != nil {
			results = append(results, *dlv)
		}
	}

	return results, nil
}

func (s *Service) deliverToSubscription(sub *WebhookSubscription, eventName string, modelName string, resID uint, data interface{}, companyID uint) (*WebhookDelivery, error) {
	code, err := s.repo.GenerateDeliveryCode(companyID)
	if err != nil {
		code = fmt.Sprintf("DLV/%d/%d", time.Now().Year(), time.Now().UnixNano()%100000)
	}

	eventID := fmt.Sprintf("evt_%d_%s", time.Now().Unix(), code[len(code)-5:])
	envelope := map[string]interface{}{
		"id":          eventID,
		"event":       eventName,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"model":       modelName,
		"resource_id": resID,
		"company_id":  companyID,
		"data":        data,
	}

	payloadBytes, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to encode event envelope: %w", err)
	}

	signature := ComputeHMACSignature(payloadBytes, sub.SecretToken)

	headersSent := map[string]string{
		"Content-Type":        "application/json",
		"User-Agent":          "GoERP-Webhook-Engine/1.27.0",
		"X-Webhook-ID":        eventID,
		"X-Webhook-Event":     eventName,
		"X-Webhook-Timestamp": time.Now().UTC().Format(time.RFC3339),
		"X-Webhook-Signature": "sha256=" + signature,
	}

	// Merge custom headers if configured
	if sub.CustomHeaders != "" {
		var custom map[string]string
		if err := json.Unmarshal([]byte(sub.CustomHeaders), &custom); err == nil {
			for k, v := range custom {
				headersSent[k] = v
			}
		}
	}
	headersBytes, _ := json.Marshal(headersSent)

	// Execute HTTP Dispatch
	timeout := time.Duration(sub.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	startTime := time.Now()
	httpReq, err := http.NewRequest("POST", sub.TargetURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("invalid webhook request: %w", err)
	}

	for k, v := range headersSent {
		httpReq.Header.Set(k, v)
	}

	resp, reqErr := client.Do(httpReq)
	duration := time.Since(startTime).Milliseconds()

	statusCode := 0
	var respBody string
	var errMsg string

	if reqErr != nil {
		errMsg = reqErr.Error()
	} else {
		defer resp.Body.Close()
		statusCode = resp.StatusCode
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		respBody = string(bodyBytes)
		if statusCode < 200 || statusCode >= 300 {
			errMsg = fmt.Sprintf("HTTP %d received from endpoint", statusCode)
		}
	}

	// Evaluate status & schedule retry if failed
	status := StatusSuccess
	var nextRetry *time.Time

	now := time.Now()
	sub.LastFiredAt = &now

	if statusCode >= 200 && statusCode < 300 {
		status = StatusSuccess
		sub.LastStatus = StatusSuccess
	} else {
		if 1 < sub.MaxRetries {
			status = StatusRetrying
			sub.LastStatus = StatusFailed
			retryDelay := sub.RetryDelaySec
			if retryDelay <= 0 {
				retryDelay = 60
			}
			t := now.Add(time.Duration(retryDelay) * time.Second)
			nextRetry = &t
		} else {
			status = StatusFailed
			sub.LastStatus = StatusFailed
		}
	}
	_ = s.repo.UpdateSubscription(sub)

	dlv := &WebhookDelivery{
		ModelBase: models.ModelBase{
			CompanyID: companyID,
			Active:    true,
		},
		DeliveryCode:   code,
		SubscriptionID: sub.ID,
		EventName:      eventName,
		ModelName:      modelName,
		ResID:          resID,
		TargetURL:      sub.TargetURL,
		RequestHeaders: string(headersBytes),
		PayloadJSON:    string(payloadBytes),
		HMACSignature:  signature,
		StatusCode:     statusCode,
		ResponseBody:   respBody,
		DurationMs:     duration,
		Status:         status,
		AttemptCount:   1,
		NextRetryAt:    nextRetry,
		ErrorMessage:   errMsg,
	}

	if err := s.repo.CreateDelivery(dlv); err != nil {
		return nil, fmt.Errorf("failed to save webhook delivery log: %w", err)
	}

	dlv.Subscription = sub
	return dlv, nil
}

// ---------------------------------------------------------------------------
// Retry Queue Runner
// ---------------------------------------------------------------------------

func (s *Service) RunRetryQueue(companyID uint) (int, error) {
	if companyID == 0 {
		companyID = 1
	}

	pending, err := s.repo.GetPendingRetries(companyID, 50)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, dlv := range pending {
		dlvCopy := dlv
		if err := s.executeRetry(&dlvCopy); err == nil {
			processed++
		}
	}

	return processed, nil
}

func (s *Service) RetryDelivery(deliveryID uint, companyID uint) (*WebhookDelivery, error) {
	dlv, err := s.repo.GetDeliveryByID(deliveryID, companyID)
	if err != nil {
		return nil, fmt.Errorf("delivery #%d not found: %w", deliveryID, err)
	}

	if err := s.executeRetry(dlv); err != nil {
		return nil, err
	}

	return dlv, nil
}

func (s *Service) executeRetry(dlv *WebhookDelivery) error {
	sub := dlv.Subscription
	if sub == nil {
		var err error
		sub, err = s.repo.GetSubscriptionByID(dlv.SubscriptionID, dlv.CompanyID)
		if err != nil {
			return err
		}
		dlv.Subscription = sub
	}

	dlv.AttemptCount++

	timeout := time.Duration(sub.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	startTime := time.Now()
	httpReq, err := http.NewRequest("POST", dlv.TargetURL, bytes.NewBuffer([]byte(dlv.PayloadJSON)))
	if err != nil {
		return err
	}

	// Reapply headers
	if dlv.RequestHeaders != "" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(dlv.RequestHeaders), &headers); err == nil {
			for k, v := range headers {
				httpReq.Header.Set(k, v)
			}
		}
	}

	resp, reqErr := client.Do(httpReq)
	duration := time.Since(startTime).Milliseconds()
	dlv.DurationMs = duration

	now := time.Now()
	if reqErr != nil {
		dlv.StatusCode = 0
		dlv.ErrorMessage = reqErr.Error()
	} else {
		defer resp.Body.Close()
		dlv.StatusCode = resp.StatusCode
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		dlv.ResponseBody = string(bodyBytes)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			dlv.ErrorMessage = ""
		} else {
			dlv.ErrorMessage = fmt.Sprintf("HTTP %d received on retry", resp.StatusCode)
		}
	}

	if dlv.StatusCode >= 200 && dlv.StatusCode < 300 {
		dlv.Status = StatusSuccess
		dlv.NextRetryAt = nil
		sub.LastStatus = StatusSuccess
	} else {
		if dlv.AttemptCount < sub.MaxRetries {
			dlv.Status = StatusRetrying
			// Exponential backoff: base_delay * 2^(attempt-1)
			multiplier := 1 << (dlv.AttemptCount - 1)
			if multiplier > 16 {
				multiplier = 16
			}
			delaySec := sub.RetryDelaySec * multiplier
			if delaySec <= 0 {
				delaySec = 60
			}
			t := now.Add(time.Duration(delaySec) * time.Second)
			dlv.NextRetryAt = &t
			sub.LastStatus = StatusFailed
		} else {
			dlv.Status = StatusFailed
			dlv.NextRetryAt = nil
			sub.LastStatus = StatusFailed
		}
	}

	sub.LastFiredAt = &now
	_ = s.repo.UpdateSubscription(sub)
	return s.repo.UpdateDelivery(dlv)
}

// ---------------------------------------------------------------------------
// Cloudflare Workers AI Diagnostic & Health Copilot
// ---------------------------------------------------------------------------

func (s *Service) AnalyzeWebhookHealth(companyID uint) (*AIWebhookDiagnosticReport, error) {
	if companyID == 0 {
		companyID = 1
	}

	deliveries, _, err := s.repo.ListDeliveries(DeliveryFilters{
		CompanyID: companyID,
		Limit:     100,
	})
	if err != nil {
		return nil, err
	}

	var failed []WebhookDelivery
	for _, d := range deliveries {
		if d.Status == StatusFailed || d.Status == StatusRetrying {
			failed = append(failed, d)
		}
	}

	totalAnalyzed := len(deliveries)
	failedCount := len(failed)

	healthStatus := "HEALTHY"
	if failedCount > 0 {
		healthStatus = "DEGRADED"
	}
	if totalAnalyzed > 0 && float64(failedCount)/float64(totalAnalyzed) > 0.4 {
		healthStatus = "CRITICAL"
	}

	// If Cloudflare Workers AI configured and there are errors, run LLM analysis
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() && failedCount > 0 {
		report, err := s.callAIWebhookDiagnostic(deliveries, failed, totalAnalyzed, healthStatus)
		if err == nil && report != nil {
			return report, nil
		}
	}

	// Fallback to local rule-based diagnostics
	var issues []string
	var actions []string

	if failedCount == 0 {
		issues = append(issues, "Seluruh endpoint webhook beroperasi normal dengan tingkat keberhasilan pengiriman 100%.")
		actions = append(actions, "Pertahankan konfigurasi shared secret HMAC-SHA256 saat ini.")
	} else {
		has401 := false
		has404 := false
		has500 := false
		hasTimeout := false

		for _, f := range failed {
			if f.StatusCode == 401 || f.StatusCode == 403 {
				has401 = true
			} else if f.StatusCode == 404 {
				has404 = true
			} else if f.StatusCode >= 500 {
				has500 = true
			} else if strings.Contains(strings.ToLower(f.ErrorMessage), "timeout") {
				hasTimeout = true
			}
		}

		if has401 {
			issues = append(issues, "Ditemukan kegagalan otentikasi (HTTP 401/403). Kemungkinan SecretToken HMAC tidak cocok dengan receiver.")
			actions = append(actions, "Sinkronkan ulang secret token pada sistem penerima dan verifikasi header 'X-Webhook-Signature'.")
		}
		if has404 {
			issues = append(issues, "Ditemukan endpoint tidak ditemukan (HTTP 404). Target URL webhook mungkin salah ketik atau telah dinonaktifkan.")
			actions = append(actions, "Periksa kembali URL webhook tujuan pada konfigurasi subscription.")
		}
		if has500 {
			issues = append(issues, "Ditemukan error server internal penerima (HTTP 500/502/503). Server target mengalami kendala memproses payload.")
			actions = append(actions, "Periksa log server penerima dan pastikan server target mampu menangani payload JSON ERP.")
		}
		if hasTimeout {
			issues = append(issues, "Ditemukan kegagalan akibat batas waktu koneksi (Request Timeout). Latensi penerima melebihi batas waktu.")
			actions = append(actions, "Tingkatkan TimeoutSec pada webhook subscription atau optimalkan pemrosesan asynchronous di sisi receiver.")
		}
	}

	execSummary := fmt.Sprintf("Kesehatan integrasi webhook berada pada status %s. Dari %d pengiriman terkini, ditemukan %d kegagalan dengan rasio keberhasilan %.1f%%.",
		healthStatus, totalAnalyzed, failedCount, 100.0-float64(failedCount*100)/float64(totalAnalyzed+1))

	return &AIWebhookDiagnosticReport{
		TotalAnalyzed:       totalAnalyzed,
		FailedDeliveries:    failedCount,
		HealthStatus:        healthStatus,
		IdentifiedIssues:    issues,
		RemediationActions:  actions,
		PayloadOptimization: "Gunakan kompresi gzip atau webhook payload filtering untuk mempercepat pengiriman transmisi tinggi.",
		ExecutiveSummary:    execSummary,
		AIProvider:          "Local Heuristic Engine (Webhook Reliability Baseline)",
		SampleErrors:        failed,
	}, nil
}

func (s *Service) callAIWebhookDiagnostic(all []WebhookDelivery, failed []WebhookDelivery, total int, baseHealth string) (*AIWebhookDiagnosticReport, error) {
	type errSample struct {
		Event  string `json:"event"`
		URL    string `json:"url"`
		Status int    `json:"status"`
		Error  string `json:"error"`
	}

	var samples []errSample
	for i, f := range failed {
		if i >= 6 {
			break
		}
		samples = append(samples, errSample{
			Event:  f.EventName,
			URL:    f.TargetURL,
			Status: f.StatusCode,
			Error:  f.ErrorMessage,
		})
	}
	sampleBytes, _ := json.Marshal(samples)

	systemPrompt := "Anda adalah Webhook & API Integration Reliability Specialist. Analisis log error webhook outbound berikut dan berikan evaluasi JSON murni tanpa markdown."
	userPrompt := fmt.Sprintf(`Total pengiriman: %d, Kegagalan: %d
Contoh kegagalan: %s

Berikan JSON persis seperti berikut:
{
  "health_status": "HEALTHY|DEGRADED|CRITICAL",
  "identified_issues": ["isu 1", "isu 2"],
  "remediation_actions": ["aksi 1", "aksi 2"],
  "payload_optimization": "rekomendasi optimasi",
  "executive_summary": "ringkasan eksekutif"
}`, total, len(failed), string(sampleBytes))

	respStr, err := s.aiClient.CallRaw(userPrompt, systemPrompt)
	if err != nil {
		return nil, err
	}

	cleanJSON := strings.TrimSpace(respStr)
	if idx := strings.Index(cleanJSON, "{"); idx != -1 {
		if last := strings.LastIndex(cleanJSON, "}"); last != -1 && last > idx {
			cleanJSON = cleanJSON[idx : last+1]
		}
	}

	var aiResp struct {
		HealthStatus        string   `json:"health_status"`
		IdentifiedIssues    []string `json:"identified_issues"`
		RemediationActions  []string `json:"remediation_actions"`
		PayloadOptimization string   `json:"payload_optimization"`
		ExecutiveSummary    string   `json:"executive_summary"`
	}

	if err := json.Unmarshal([]byte(cleanJSON), &aiResp); err != nil {
		return nil, err
	}

	status := aiResp.HealthStatus
	if status == "" {
		status = baseHealth
	}

	return &AIWebhookDiagnosticReport{
		TotalAnalyzed:       total,
		FailedDeliveries:    len(failed),
		HealthStatus:        status,
		IdentifiedIssues:    aiResp.IdentifiedIssues,
		RemediationActions:  aiResp.RemediationActions,
		PayloadOptimization: aiResp.PayloadOptimization,
		ExecutiveSummary:    aiResp.ExecutiveSummary,
		AIProvider:          "Cloudflare Workers AI (@cf/meta/llama-3-8b-instruct)",
		SampleErrors:        failed,
	}, nil
}

// ---------------------------------------------------------------------------
// Query Helpers
// ---------------------------------------------------------------------------

func (s *Service) ListDeliveries(filters DeliveryFilters) ([]WebhookDelivery, int64, error) {
	return s.repo.ListDeliveries(filters)
}

func (s *Service) GetDelivery(id uint, companyID uint) (*WebhookDelivery, error) {
	return s.repo.GetDeliveryByID(id, companyID)
}

func (s *Service) GetStats(companyID uint) (*WebhookStats, error) {
	return s.repo.GetStats(companyID)
}
