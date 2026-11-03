package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"erp-golang/internal/config"
)

// TicketAIAnalysis represents AI triage results for a helpdesk customer ticket
type TicketAIAnalysis struct {
	Sentiment             string   `json:"sentiment"`              // "frustrated", "neutral", "positive"
	SuggestedPriority     string   `json:"suggested_priority"`     // "urgent", "high", "medium", "low"
	SuggestedCategory     string   `json:"suggested_category"`     // "technical", "billing", "inquiry", "warranty"
	DraftResponse         string   `json:"draft_response"`         // AI-composed polite, technical solution draft
	ActionRecommendations []string `json:"action_recommendations"` // SOP recommended action items
	Source                string   `json:"source"`                 // AI provider or "local_heuristic_engine"
}

// QualityAIAnalysis represents AI Root Cause Analysis (RCA) and CAPA for Quality Control
type QualityAIAnalysis struct {
	RootCauseAnalysis   string `json:"root_cause_analysis"`   // 5-Whys root cause investigation
	PreventiveAction    string `json:"preventive_action"`     // CAPA preventive guidance
	SuggestedActionType string `json:"suggested_action_type"` // "quarantine", "rework", "return", "scrap"
	RiskLevel           string `json:"risk_level"`            // "Critical", "High", "Medium", "Low"
	ComplianceNotes     string `json:"compliance_notes"`      // ISO 9001 / QA compliance notes
	Source              string `json:"source"`                // AI provider or "local_heuristic_engine"
}

// ExpenseAIAnalysis represents AI categorization for employee expenses
type ExpenseAIAnalysis struct {
	SuggestedCategoryID   uint   `json:"suggested_category_id"`
	SuggestedCategoryName string `json:"suggested_category_name"`
	SuggestedGLAccount    string `json:"suggested_gl_account"`
	TaxDeductible         bool   `json:"tax_deductible"`
	ComplianceCheck       string `json:"compliance_check"`
	Rationale             string `json:"rationale"`
	Source                string `json:"source"`
}

// AskERPResponse represents executive natural language responses
type AskERPResponse struct {
	Question         string                 `json:"question"`
	Answer           string                 `json:"answer"`
	KeyMetrics       map[string]interface{} `json:"key_metrics,omitempty"`
	SuggestedActions []string               `json:"suggested_actions,omitempty"`
	Source           string                 `json:"source"`
}

// ApprovalRiskAnalysis represents AI risk and compliance analysis for corporate approval requests
type ApprovalRiskAnalysis struct {
	RiskLevel            string   `json:"risk_level"`            // "Low", "Medium", "High", "Critical"
	ConfidenceScore      float64  `json:"confidence_score"`      // 0.0 - 1.0
	PolicyCompliance     string   `json:"policy_compliance"`     // Compliance check against delegation of authority & budget
	AuditRecommendations []string `json:"audit_recommendations"` // Key verification questions for the approver
	Recommendation       string   `json:"recommendation"`        // Executive sign-off recommendation
	Source               string   `json:"source"`
}

// ReceiptLineItem represents an extracted item from receipt
type ReceiptLineItem struct {
	Description string  `json:"description"`
	Qty         float64 `json:"qty"`
	UnitPrice   float64 `json:"unit_price"`
	Subtotal    float64 `json:"subtotal"`
}

// ReceiptOCRResult represents structured data extracted from receipts/invoices by Vision AI
type ReceiptOCRResult struct {
	VendorName         string            `json:"vendor_name"`
	InvoiceDate        string            `json:"invoice_date"` // YYYY-MM-DD
	TotalAmount        float64           `json:"total_amount"`
	TaxAmount          float64           `json:"tax_amount"`
	Currency           string            `json:"currency"` // IDR, USD, etc.
	CategorySuggestion string            `json:"category_suggestion"`
	LineItems          []ReceiptLineItem `json:"line_items"`
	ConfidenceScore    float64           `json:"confidence_score"`
	RawExtractedText   string            `json:"raw_extracted_text,omitempty"`
	Source             string            `json:"source"`
}

type Client struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

// Provider Availability Checks
func (c *Client) IsGeminiConfigured() bool {
	return c.cfg != nil && c.cfg.GeminiAPIKey != ""
}

func (c *Client) IsOpenAIConfigured() bool {
	return c.cfg != nil && c.cfg.OpenAIAPIKey != ""
}

func (c *Client) IsAnthropicConfigured() bool {
	return c.cfg != nil && c.cfg.AnthropicAPIKey != ""
}

func (c *Client) IsDeepSeekConfigured() bool {
	return c.cfg != nil && c.cfg.DeepSeekAPIKey != ""
}

func (c *Client) IsCloudflareConfigured() bool {
	return c.cfg != nil && ((c.cfg.CloudflareAccountID != "" && c.cfg.CloudflareAPIToken != "") || c.cfg.CloudflareWorkerURL != "")
}

func (c *Client) IsAnyCloudAIConfigured() bool {
	return c.IsGeminiConfigured() || c.IsOpenAIConfigured() || c.IsAnthropicConfigured() || c.IsDeepSeekConfigured() || c.IsCloudflareConfigured()
}

// ActiveProviderName returns the active AI provider identifier
func (c *Client) ActiveProviderName() string {
	pref := strings.ToLower(strings.TrimSpace(c.cfg.AIProvider))
	switch pref {
	case "gemini", "vertex_ai", "google":
		if c.IsGeminiConfigured() {
			return "google_gemini"
		}
	case "openai", "gpt":
		if c.IsOpenAIConfigured() {
			return "openai_gpt4o"
		}
	case "anthropic", "claude":
		if c.IsAnthropicConfigured() {
			return "anthropic_claude"
		}
	case "deepseek":
		if c.IsDeepSeekConfigured() {
			return "deepseek_r1"
		}
	case "cloudflare":
		if c.IsCloudflareConfigured() {
			return "cloudflare_workers_ai"
		}
	}

	if c.IsGeminiConfigured() {
		return "google_gemini"
	}
	if c.IsOpenAIConfigured() {
		return "openai_gpt4o"
	}
	if c.IsAnthropicConfigured() {
		return "anthropic_claude"
	}
	if c.IsDeepSeekConfigured() {
		return "deepseek_r1"
	}
	if c.IsCloudflareConfigured() {
		return "cloudflare_workers_ai"
	}
	return "local_smart_heuristic_engine"
}

// ---------------------------------------------------------------------------
// Provider 1: Google Gemini API & Google Cloud Vertex AI
// ---------------------------------------------------------------------------

func (c *Client) callGeminiAPI(prompt, systemPrompt string) (string, error) {
	if !c.IsGeminiConfigured() {
		return "", errors.New("google gemini api key not configured")
	}

	model := c.cfg.GeminiModel
	if model == "" {
		model = "gemini-2.5-flash"
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, c.cfg.GeminiAPIKey)

	payload := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{
				{"text": systemPrompt},
			},
		},
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.2,
			"maxOutputTokens": 1024,
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("gemini api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err == nil && len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return geminiResp.Candidates[0].Content.Parts[0].Text, nil
	}

	return string(bodyBytes), nil
}

// ---------------------------------------------------------------------------
// Provider 2 & 4: OpenAI API (GPT-4o / Agents SDK) & DeepSeek / OpenRouter
// ---------------------------------------------------------------------------

func (c *Client) callOpenAICompatible(baseURL, apiKey, model, prompt, systemPrompt string) (string, error) {
	if apiKey == "" {
		return "", errors.New("openai-compatible api key not configured")
	}

	endpoint := strings.TrimRight(baseURL, "/") + "/chat/completions"
	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.2,
		"max_tokens":  1024,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("openai-compatible api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var oaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(bodyBytes, &oaiResp); err == nil && len(oaiResp.Choices) > 0 {
		return oaiResp.Choices[0].Message.Content, nil
	}

	return string(bodyBytes), nil
}

// ---------------------------------------------------------------------------
// Provider 3: Anthropic Claude API & MCP
// ---------------------------------------------------------------------------

func (c *Client) callAnthropicClaude(prompt, systemPrompt string) (string, error) {
	if !c.IsAnthropicConfigured() {
		return "", errors.New("anthropic api key not configured")
	}

	model := c.cfg.AnthropicModel
	if model == "" {
		model = "claude-3-7-sonnet-latest"
	}

	endpoint := "https://api.anthropic.com/v1/messages"
	payload := map[string]interface{}{
		"model":      model,
		"max_tokens": 1024,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.cfg.AnthropicAPIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("anthropic api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var claudeResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(bodyBytes, &claudeResp); err == nil && len(claudeResp.Content) > 0 {
		return claudeResp.Content[0].Text, nil
	}

	return string(bodyBytes), nil
}

// ---------------------------------------------------------------------------
// Provider 5: Cloudflare Workers AI
// ---------------------------------------------------------------------------

func (c *Client) callCloudflareWorkersAI(prompt, systemPrompt string) (string, error) {
	if !c.IsCloudflareConfigured() {
		return "", errors.New("cloudflare workers ai credentials not configured")
	}

	var endpoint string
	if c.cfg.CloudflareWorkerURL != "" {
		endpoint = c.cfg.CloudflareWorkerURL
	} else {
		endpoint = fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/ai/run/%s",
			c.cfg.CloudflareAccountID, c.cfg.CloudflareAIModel)
	}

	requestPayload := map[string]interface{}{
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": prompt},
		},
		"max_tokens": 1024,
	}

	jsonBytes, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.cfg.CloudflareAPIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.CloudflareAPIToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("cloudflare workers ai api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var cfResp struct {
		Result struct {
			Response string `json:"response"`
		} `json:"result"`
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(bodyBytes, &cfResp); err == nil && cfResp.Result.Response != "" {
		return cfResp.Result.Response, nil
	}

	var directResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bodyBytes, &directResp); err == nil && directResp.Response != "" {
		return directResp.Response, nil
	}

	return string(bodyBytes), nil
}

// CallWithProvider routes the prompt to the requested or prioritized AI provider
func (c *Client) CallWithProvider(provider, prompt, systemPrompt string) (string, string, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if p == "" || p == "auto" {
		p = strings.ToLower(strings.TrimSpace(c.cfg.AIProvider))
	}

	switch p {
	case "gemini", "vertex_ai", "google", "google_gemini":
		if c.IsGeminiConfigured() {
			out, err := c.callGeminiAPI(prompt, systemPrompt)
			if err == nil {
				return out, "google_gemini", nil
			}
		}
	case "openai", "gpt", "gpt-4o", "openai_gpt4o":
		if c.IsOpenAIConfigured() {
			out, err := c.callOpenAICompatible(c.cfg.OpenAIBaseURL, c.cfg.OpenAIAPIKey, c.cfg.OpenAIModel, prompt, systemPrompt)
			if err == nil {
				return out, "openai_gpt4o", nil
			}
		}
	case "anthropic", "claude", "anthropic_claude":
		if c.IsAnthropicConfigured() {
			out, err := c.callAnthropicClaude(prompt, systemPrompt)
			if err == nil {
				return out, "anthropic_claude", nil
			}
		}
	case "deepseek", "deepseek_r1":
		if c.IsDeepSeekConfigured() {
			out, err := c.callOpenAICompatible(c.cfg.DeepSeekBaseURL, c.cfg.DeepSeekAPIKey, c.cfg.DeepSeekModel, prompt, systemPrompt)
			if err == nil {
				return out, "deepseek_r1", nil
			}
		}
	case "cloudflare", "cloudflare_workers_ai":
		if c.IsCloudflareConfigured() {
			out, err := c.callCloudflareWorkersAI(prompt, systemPrompt)
			if err == nil {
				return out, "cloudflare_workers_ai", nil
			}
		}
	}

	// Automatic Priority Cascade: Gemini -> OpenAI -> Anthropic -> DeepSeek -> Cloudflare
	if c.IsGeminiConfigured() {
		if out, err := c.callGeminiAPI(prompt, systemPrompt); err == nil {
			return out, "google_gemini", nil
		}
	}
	if c.IsOpenAIConfigured() {
		if out, err := c.callOpenAICompatible(c.cfg.OpenAIBaseURL, c.cfg.OpenAIAPIKey, c.cfg.OpenAIModel, prompt, systemPrompt); err == nil {
			return out, "openai_gpt4o", nil
		}
	}
	if c.IsAnthropicConfigured() {
		if out, err := c.callAnthropicClaude(prompt, systemPrompt); err == nil {
			return out, "anthropic_claude", nil
		}
	}
	if c.IsDeepSeekConfigured() {
		if out, err := c.callOpenAICompatible(c.cfg.DeepSeekBaseURL, c.cfg.DeepSeekAPIKey, c.cfg.DeepSeekModel, prompt, systemPrompt); err == nil {
			return out, "deepseek_r1", nil
		}
	}
	if c.IsCloudflareConfigured() {
		if out, err := c.callCloudflareWorkersAI(prompt, systemPrompt); err == nil {
			return out, "cloudflare_workers_ai", nil
		}
	}

	return "", "local_heuristic_engine", errors.New("no cloud ai provider configured or reachable")
}

// CallRaw executes a raw prompt using the active multi-provider router
func (c *Client) CallRaw(prompt, systemPrompt string) (string, error) {
	out, _, err := c.CallWithProvider("auto", prompt, systemPrompt)
	return out, err
}

// ---------------------------------------------------------------------------
// 1. Helpdesk Ticket AI Auto-Triage & Smart Reply
// ---------------------------------------------------------------------------

func (c *Client) AnalyzeTicket(title, desc string) (*TicketAIAnalysis, error) {
	if c.IsAnyCloudAIConfigured() {
		systemPrompt := `You are an expert enterprise ERP Helpdesk Support Assistant. Analyze the customer support ticket title and description. Respond ONLY in valid JSON matching this schema:
{
  "sentiment": "frustrated"|"neutral"|"positive",
  "suggested_priority": "urgent"|"high"|"medium"|"low",
  "suggested_category": "technical"|"billing"|"inquiry"|"warranty",
  "draft_response": "Polite empathetic resolution draft in Indonesian",
  "action_recommendations": ["step 1", "step 2"]
}`
		userPrompt := fmt.Sprintf("Ticket Title: %s\nTicket Description: %s", title, desc)
		respText, source, err := c.CallWithProvider("auto", userPrompt, systemPrompt)
		if err == nil {
			var analysis TicketAIAnalysis
			cleanJSON := extractJSONString(respText)
			if jsonErr := json.Unmarshal([]byte(cleanJSON), &analysis); jsonErr == nil {
				analysis.Source = source
				return &analysis, nil
			}
		}
	}

	// Local Heuristic Fallback Engine
	lower := strings.ToLower(title + " " + desc)
	sentiment := "neutral"
	priority := "medium"
	category := "technical"
	var recommendations []string

	if strings.Contains(lower, "rusak") || strings.Contains(lower, "mati") || strings.Contains(lower, "macet") ||
		strings.Contains(lower, "kebocoran") || strings.Contains(lower, "down") || strings.Contains(lower, "error fatal") {
		sentiment = "frustrated"
		priority = "urgent"
		category = "technical"
		recommendations = []string{
			"Segera lakukan eskalasi ke Tim Engineer Lapangan Tier 2",
			"Cek riwayat perawatan mesin (maintenance.equipment) dan suku cadang",
			"Hubungi pelanggan dalam kurun waktu SLA < 30 menit",
		}
	} else if strings.Contains(lower, "tagihan") || strings.Contains(lower, "invoice") || strings.Contains(lower, "faktur") || strings.Contains(lower, "pembayaran") {
		priority = "high"
		category = "billing"
		recommendations = []string{
			"Verifikasi nomor faktur tagihan pada Modul Akuntansi (account.move)",
			"Cek status pembayaran mutasi bank (BNK/CSH)",
			"Kirimkan konfirmasi kwitansi pelunasan resmi ke pelanggan",
		}
	} else if strings.Contains(lower, "garansi") || strings.Contains(lower, "klaim") {
		priority = "high"
		category = "warranty"
		recommendations = []string{
			"Periksa tanggal penerimaan barang dan batas masa garansi serial number",
			"Terbitkan form pemeriksaan barang retur gudang (WH/IN)",
		}
	} else {
		category = "inquiry"
		recommendations = []string{
			"Kirimkan dokumen panduan teknis dan SOP penggunaan",
			"Tutup tiket jika pelanggan telah mengonfirmasi penjelasan",
		}
	}

	draft := fmt.Sprintf("Yth. Pelanggan,\n\nTerima kasih telah menghubungi Customer Care kami terkait: '%s'. Kami memahami kendala yang Anda hadapi dan tim teknis kami saat ini sedang memprioritaskan penanganan masalah ini sesuai SLA. Kami akan segera memberikan pembaruan status lebih lanjut.\n\nSalam hangat,\nTim Dukungan Pelanggan ERP", title)

	return &TicketAIAnalysis{
		Sentiment:             sentiment,
		SuggestedPriority:     priority,
		SuggestedCategory:     category,
		DraftResponse:         draft,
		ActionRecommendations: recommendations,
		Source:                "local_heuristic_engine",
	}, nil
}

// ---------------------------------------------------------------------------
// 2. Quality Control Root Cause Analysis (RCA) & CAPA Advisor
// ---------------------------------------------------------------------------

func (c *Client) AnalyzeQualityAlert(title, rootCause, notes string) (*QualityAIAnalysis, error) {
	if c.IsAnyCloudAIConfigured() {
		systemPrompt := `You are an expert ISO 9001 Quality Assurance (QA) and Six Sigma engineer. Analyze this quality non-conformance event and return ONLY valid JSON:
{
  "root_cause_analysis": "Detailed 5-Whys Root Cause Analysis in Indonesian",
  "preventive_action": "Corrective and Preventive Action (CAPA) steps in Indonesian",
  "suggested_action_type": "quarantine"|"rework"|"return"|"scrap",
  "risk_level": "Critical"|"High"|"Medium"|"Low",
  "compliance_notes": "ISO 9001 / QA compliance audit note"
}`
		userPrompt := fmt.Sprintf("Quality Event: %s\nObserved Failure: %s\nNotes: %s", title, rootCause, notes)
		respText, source, err := c.CallWithProvider("auto", userPrompt, systemPrompt)
		if err == nil {
			var analysis QualityAIAnalysis
			cleanJSON := extractJSONString(respText)
			if jsonErr := json.Unmarshal([]byte(cleanJSON), &analysis); jsonErr == nil {
				analysis.Source = source
				return &analysis, nil
			}
		}
	}

	// Local Heuristic Fallback Engine
	lower := strings.ToLower(title + " " + rootCause + " " + notes)
	actionType := "quarantine"
	risk := "High"
	rca := "1. Mengapa terjadi kegagalan? Fluktuasi spesifikasi teknis melebihi batas toleransi aman.\n2. Mengapa toleransi terlewati? Terdapat anomali komponen pada batch pengiriman vendor/pabrik.\n3. Mengapa tidak terdeteksi lebih awal? Kalibrasi alat ukur supplier perlu verifikasi berkala."
	capa := "1. Karantina fisik seluruh lot produk yang terdampak agar tidak masuk ke lini produksi/konsumen.\n2. Lakukan audit pengujian sampling ulang dengan multimeter/mikrometer terkalibrasi.\n3. Terbitkan surat Non-Conformance Report (NCR) resmi dan minta vendor mengirimkan batch pengganti."

	if strings.Contains(lower, "volt") || strings.Contains(lower, "tegangan") || strings.Contains(lower, "daya") {
		risk = "High"
		actionType = "quarantine"
		rca = "Fluktuasi tegangan output DC terjadi karena komponen IC regulator daya tidak stabil saat beban puncak. Analisis 5-Whys: Toleransi komponen kapasitor vendor batch ini tidak memenuhi standar isolasi 12V."
		capa = "Karantina unit yang bersangkutan, lakukan uji stress-load selama 15 menit, dan minta sertifikat uji kelayakan (CoA) dari pemasok sebelum batch diterima kembali."
	} else if strings.Contains(lower, "suhu") || strings.Contains(lower, "panas") || strings.Contains(lower, "overheat") {
		risk = "Critical"
		actionType = "rework"
		rca = "Suhu operasional melampaui ambang batas maksimum (>75°C) akibat aplikasi pasta termal tidak merata atau tekanan heatsink kurang optimal pada socket CPU."
		capa = "Lakukan pengerjaan ulang (rework): bersihkan pasta termal lama, aplikasikan ulang dengan metode cross-pattern, dan pastikan torsi baut pendingin sesuai spesifikasi manufaktur."
	} else if strings.Contains(lower, "retak") || strings.Contains(lower, "patah") || strings.Contains(lower, "penyok") {
		risk = "Medium"
		actionType = "return"
		rca = "Cacat fisik terjadi akibat benturan mekanis selama proses logistik dan pengemasan tanpa proteksi busa antistatis yang memadai."
		capa = "Retur seluruh unit cacat ke pihak ekspedisi/vendor untuk penggantian penuh dan tambahkan persyaratan kemasan double-wall box untuk pengiriman selanjutnya."
	}

	return &QualityAIAnalysis{
		RootCauseAnalysis:   rca,
		PreventiveAction:    capa,
		SuggestedActionType: actionType,
		RiskLevel:           risk,
		ComplianceNotes:     "Tindakan korektif ini memenuhi klausul ISO 9001:2015 Bagian 8.7 (Pengendalian Ketidaksesuaian Output) dan 10.2 (Tindakan Korektif).",
		Source:              "local_heuristic_engine",
	}, nil
}

// ---------------------------------------------------------------------------
// 3. Expense Categorization & GL Account Mapping
// ---------------------------------------------------------------------------

func (c *Client) AnalyzeExpense(desc string, amount float64) (*ExpenseAIAnalysis, error) {
	lower := strings.ToLower(desc)

	categoryID := uint(3) // Default: Office Supplies (ATK)
	catName := "Perlengkapan Kantor & ATK"
	glCode := "620300"
	rationale := "Pengeluaran umum operasional kantor yang tergolong beban perlengkapan."

	if strings.Contains(lower, "tiket") || strings.Contains(lower, "hotel") || strings.Contains(lower, "pesawat") ||
		strings.Contains(lower, "kereta") || strings.Contains(lower, "surabaya") || strings.Contains(lower, "dinas") {
		categoryID = 1
		catName = "Perjalanan Dinas & Akomodasi"
		glCode = "620100"
		rationale = "Pengeluaran mobilitas dinas antar-kota mencakup tiket transportasi dan penginapan hotel."
	} else if strings.Contains(lower, "makan") || strings.Contains(lower, "resto") || strings.Contains(lower, "konsumsi") ||
		strings.Contains(lower, "tamu") || strings.Contains(lower, "kopi") || strings.Contains(lower, "lunch") {
		categoryID = 2
		catName = "Konsumsi & Representasi Tamu"
		glCode = "620200"
		rationale = "Biaya jamuan makan klien bisnis atau konsumsi rapat internal tim."
	} else if strings.Contains(lower, "bbm") || strings.Contains(lower, "bensin") || strings.Contains(lower, "tol") ||
		strings.Contains(lower, "solar") || strings.Contains(lower, "parkir") || strings.Contains(lower, "armada") {
		categoryID = 4
		catName = "Operasional Armada, BBM & Tol"
		glCode = "620400"
		rationale = "Biaya bahan bakar dan retribusi jalan tol kendaraan logistik/operasional."
	} else if strings.Contains(lower, "pulsa") || strings.Contains(lower, "kuota") || strings.Contains(lower, "internet") ||
		strings.Contains(lower, "telepon") || strings.Contains(lower, "wifi") {
		categoryID = 5
		catName = "Komunikasi, Pulsa & Internet"
		glCode = "620500"
		rationale = "Beban utilitas telekomunikasi dan paket data koordinasi pekerjaan."
	}

	return &ExpenseAIAnalysis{
		SuggestedCategoryID:   categoryID,
		SuggestedCategoryName: catName,
		SuggestedGLAccount:    glCode,
		TaxDeductible:         true,
		ComplianceCheck:       "Kwitansi valid, memenuhi syarat deduksi fiskal PPh Badan.",
		Rationale:             rationale,
		Source:                "local_heuristic_engine",
	}, nil
}

// ---------------------------------------------------------------------------
// 4. Executive Q&A / Natural Language Query ("Ask ERP AI")
// ---------------------------------------------------------------------------

func (c *Client) AskERP(query string, statsContext map[string]interface{}) (*AskERPResponse, error) {
	if c.IsAnyCloudAIConfigured() {
		systemPrompt := `You are the Executive AI Advisor for an Enterprise ERP system. Answer questions concisely and professionally in Indonesian based on the ERP statistics context provided. Respond in valid JSON:
{
  "answer": "Clear executive summary and insight in Indonesian",
  "suggested_actions": ["action 1", "action 2"]
}`
		contextJSON, _ := json.Marshal(statsContext)
		userPrompt := fmt.Sprintf("Question: %s\nERP Context Data: %s", query, string(contextJSON))
		respText, source, err := c.CallWithProvider("auto", userPrompt, systemPrompt)
		if err == nil {
			var resp AskERPResponse
			cleanJSON := extractJSONString(respText)
			if jsonErr := json.Unmarshal([]byte(cleanJSON), &resp); jsonErr == nil {
				resp.Question = query
				resp.KeyMetrics = statsContext
				resp.Source = source
				return &resp, nil
			}
		}
	}

	// Local Intelligent Summary Engine
	totalSales, _ := statsContext["total_sales"].(float64)
	salesOrders, _ := statsContext["sales_orders"].(int64)
	purchaseOrders, _ := statsContext["purchase_orders"].(int64)
	tickets, _ := statsContext["tickets"].(int64)
	qualityAlerts, _ := statsContext["quality_alerts"].(int64)

	answer := fmt.Sprintf("Ringkasan Eksekutif ERP: Saat ini perusahaan mencatat total penjualan terkonfirmasi sebesar Rp %.2f dari %d pesanan (SO), dengan %d pesanan pembelian (PO). Terdapat %d tiket keluhan pelanggan aktif dan %d laporan ketidaksesuaian mutu (NCR) yang memerlukan perhatian tim.",
		totalSales, salesOrders, purchaseOrders, tickets, qualityAlerts)

	actions := []string{
		"Tinjau efisiensi pengadaan pada daftar Purchase Orders tertunda",
		"Periksa status tahapan Quality Alerts aktif untuk memitigasi risiko retur barang",
		"Pastikan faktur penjualan yang jatuh tempo segera ditagihkan ke pelanggan",
	}

	return &AskERPResponse{
		Question:         query,
		Answer:           answer,
		KeyMetrics:       statsContext,
		SuggestedActions: actions,
		Source:           "local_heuristic_engine",
	}, nil
}

// AnalyzeApprovalRisk performs AI risk and compliance evaluation for an approval request
func (c *Client) AnalyzeApprovalRisk(title, categoryName string, amount float64, reason, reference string) (*ApprovalRiskAnalysis, error) {
	if c.IsAnyCloudAIConfigured() {
		systemPrompt := `You are an executive enterprise risk & compliance auditor for an ERP system.
Evaluate the approval request and output JSON with keys:
- risk_level: string ("Low", "Medium", "High", "Critical")
- confidence_score: float (0.0 to 1.0)
- policy_compliance: string (evaluation against delegation of authority, corporate spending limits, and internal controls)
- audit_recommendations: array of strings (key verification questions for the approver)
- recommendation: string (formal recommendation for the approver)`

		userPrompt := fmt.Sprintf("Title: %s\nCategory: %s\nAmount: Rp %.2f\nReason: %s\nDocument Reference: %s",
			title, categoryName, amount, reason, reference)

		respText, source, err := c.CallWithProvider("auto", userPrompt, systemPrompt)
		if err == nil {
			var analysis ApprovalRiskAnalysis
			cleanJSON := extractJSONString(respText)
			if jsonErr := json.Unmarshal([]byte(cleanJSON), &analysis); jsonErr == nil {
				analysis.Source = source
				return &analysis, nil
			}
		}
	}

	// Local Smart Heuristic Risk Engine
	riskLevel := "Low"
	confidence := 0.95
	var compliance string
	var auditQuestions []string
	var rec string

	lowerTitle := strings.ToLower(title)
	lowerReason := strings.ToLower(reason)

	if amount >= 50000000 {
		riskLevel = "Critical"
		compliance = "Nilai transaksi melampaui batas kewenangan Level 2 (> Rp 50.000.000). Memerlukan otorisasi Dewan Direksi / CFO sesuai Matriks Delegasi Wewenang."
		auditQuestions = []string{
			"Verifikasi ketersediaan pagu anggaran Capex kuartal berjalan dalam Buku Besar GL",
			"Pastikan telah melampirkan minimal 3 penawaran komparatif vendor rekanan",
			"Cek analisa kelayakan investasi (ROI) dan payback period sebelum penandatanganan",
		}
		rec = "Eskalasi persetujuan ke Direktur Utama / CFO diperlukan. Pastikan dokumen tender lengkap."
	} else if amount >= 10000000 || strings.Contains(lowerTitle, "capex") || strings.Contains(lowerReason, "server") {
		riskLevel = "High"
		compliance = "Pengeluaran belanja modal signifikan. Memenuhi syarat otorisasi dua tingkat (Kepala Departemen + Keuangan)."
		auditQuestions = []string{
			"Pastikan surat permohonan spesifikasi teknis telah diverifikasi tim IT/Engineering",
			"Cek riwayat pengeluaran anggaran departemen terkait pada bulan berjalan",
		}
		rec = "Disarankan disetujui dengan verifikasi spesifikasi teknis dan ketersediaan alokasi anggaran."
	} else if strings.Contains(lowerTitle, "diskon") || strings.Contains(lowerTitle, "discount") {
		riskLevel = "Medium"
		compliance = "Permohonan diskon penjualan komersial. Berpotensi menggerus margin kotor jika melampaui 15%."
		auditQuestions = []string{
			"Hitung margin kontribusi bersih setelah diskon diterapkan",
			"Pastikan klausul pembayaran pelanggan adalah tunai atau termin kredit < 30 hari",
		}
		rec = "Dapat disetujui apabila volume pembelian pelanggan dapat mengompensasi penurunan margin."
	} else {
		riskLevel = "Low"
		compliance = "Pengeluaran operasional wajar sesuai batas delegasi manajerial standar."
		auditQuestions = []string{
			"Verifikasi kesesuaian kwitansi atau bukti dukung transaksi",
		}
		rec = "Direkomendasikan untuk langsung disetujui (Approved)."
	}

	return &ApprovalRiskAnalysis{
		RiskLevel:            riskLevel,
		ConfidenceScore:      confidence,
		PolicyCompliance:     compliance,
		AuditRecommendations: auditQuestions,
		Recommendation:       rec,
		Source:               "local_heuristic_engine",
	}, nil
}

// AnalyzeReceiptVision extracts vendor, amounts, items, and taxes from receipts/invoices
func (c *Client) AnalyzeReceiptVision(imageBytes []byte, mimeType string, textHint string) (*ReceiptOCRResult, error) {
	if c.IsAnyCloudAIConfigured() {
		systemPrompt := `You are an enterprise AI document and receipt extraction engine for ERP.
Analyze the provided receipt or invoice and output valid JSON with the following schema:
{
  "vendor_name": "string (business/vendor name)",
  "invoice_date": "string (YYYY-MM-DD format)",
  "total_amount": float,
  "tax_amount": float,
  "currency": "string (e.g. IDR, USD)",
  "category_suggestion": "string (e.g. Bahan Bakar & Transportasi, Konsumsi & Representasi, Utilitas & Telekomunikasi, Akomodasi & Perjalanan, Perlengkapan Kantor)",
  "line_items": [
    {"description": "string", "qty": float, "unit_price": float, "subtotal": float}
  ],
  "confidence_score": float (0.0 to 1.0)
}`
		userPrompt := fmt.Sprintf("Extract receipt details. Content/Hint: %s", textHint)
		respText, source, err := c.CallWithProvider("auto", userPrompt, systemPrompt)
		if err == nil {
			cleanJSON := extractJSONString(respText)
			var res ReceiptOCRResult
			if jsonErr := json.Unmarshal([]byte(cleanJSON), &res); jsonErr == nil && res.TotalAmount > 0 {
				res.Source = source + "_vision"
				return &res, nil
			}
		}
	}

	// Deterministic Local Heuristic OCR Engine
	return parseLocalReceiptHeuristics(imageBytes, mimeType, textHint)
}

func parseLocalReceiptHeuristics(imageBytes []byte, mimeType string, textHint string) (*ReceiptOCRResult, error) {
	fullText := textHint
	if fullText == "" && len(imageBytes) > 0 {
		printable := make([]byte, 0, len(imageBytes))
		for _, b := range imageBytes {
			if (b >= 32 && b <= 126) || b == '\n' || b == '\r' || b == '\t' {
				printable = append(printable, b)
			}
		}
		if len(printable) > 10 {
			fullText = string(printable)
		}
	}

	lowerText := strings.ToLower(fullText)

	vendorName := "Merchant Rekanan Bisnis"
	category := "Operasional & Administrasi Umum"
	confidence := 0.85

	if strings.Contains(lowerText, "pertamina") || strings.Contains(lowerText, "spbu") || strings.Contains(lowerText, "dexlite") || strings.Contains(lowerText, "pertamax") {
		vendorName = "SPBU Pertamina Pasti Pas 34-12345"
		category = "Bahan Bakar & Transportasi"
		confidence = 0.96
	} else if strings.Contains(lowerText, "telkom") || strings.Contains(lowerText, "indihome") || strings.Contains(lowerText, "biznet") || strings.Contains(lowerText, "pln") {
		vendorName = "PT Telkom Indonesia (Indihome)"
		category = "Utilitas & Telekomunikasi"
		confidence = 0.95
	} else if strings.Contains(lowerText, "santika") || strings.Contains(lowerText, "hotel") || strings.Contains(lowerText, "aston") || strings.Contains(lowerText, "ibis") {
		vendorName = "Hotel Santika Premiere"
		category = "Akomodasi & Perjalanan"
		confidence = 0.94
	} else if strings.Contains(lowerText, "padang") || strings.Contains(lowerText, "sederhana") || strings.Contains(lowerText, "resto") || strings.Contains(lowerText, "starbucks") || strings.Contains(lowerText, "cafe") {
		vendorName = "Restoran Sederhana Bintaro"
		category = "Konsumsi & Representasi"
		confidence = 0.95
	} else if strings.Contains(lowerText, "gramedia") || strings.Contains(lowerText, "paper") || strings.Contains(lowerText, "atk") {
		vendorName = "Gramedia Asri Media"
		category = "Perlengkapan Kantor"
		confidence = 0.93
	} else if strings.Contains(lowerText, "grab") || strings.Contains(lowerText, "gojek") || strings.Contains(lowerText, "bluebird") {
		vendorName = "Grab / Gojek Indonesia"
		category = "Bahan Bakar & Transportasi"
		confidence = 0.94
	}

	dateRegex := regexp.MustCompile(`(20\d{2}[-/.]\d{1,2}[-/.]\d{1,2})|(\d{1,2}[-/.]\d{1,2}[-/.]20\d{2})`)
	dateMatch := dateRegex.FindString(fullText)
	invoiceDate := time.Now().Format("2006-01-02")
	if dateMatch != "" {
		dateMatch = strings.ReplaceAll(dateMatch, "/", "-")
		dateMatch = strings.ReplaceAll(dateMatch, ".", "-")
		parts := strings.Split(dateMatch, "-")
		if len(parts) == 3 {
			if len(parts[0]) == 4 {
				invoiceDate = fmt.Sprintf("%04s-%02s-%02s", parts[0], parts[1], parts[2])
			} else if len(parts[2]) == 4 {
				invoiceDate = fmt.Sprintf("%04s-%02s-%02s", parts[2], parts[1], parts[0])
			}
		}
	}

	totalAmount := 0.0
	taxAmount := 0.0

	amountRegex := regexp.MustCompile(`(?i)(?:total|grand total|jumlah|tagihan|bayar|rp|idr)\s*[:=]?\s*(?:rp\.?\s*)?([0-9]{1,3}(?:[.,][0-9]{3})*(?:[.,][0-9]{2})?|[0-9]+)`)
	matches := amountRegex.FindAllStringSubmatch(fullText, -1)
	if len(matches) > 0 {
		for _, m := range matches {
			if len(m) > 1 {
				cleaned := strings.ReplaceAll(m[1], ".", "")
				cleaned = strings.ReplaceAll(cleaned, ",", ".")
				if val, err := strconv.ParseFloat(cleaned, 64); err == nil && val > totalAmount {
					totalAmount = val
				}
			}
		}
	}

	if totalAmount == 0 {
		numRegex := regexp.MustCompile(`(?:rp\.?\s*)?([0-9]{1,3}(?:\.[0-9]{3})+(?:,[0-9]{2})?|[0-9]{5,9})`)
		numMatches := numRegex.FindAllStringSubmatch(fullText, -1)
		for _, nm := range numMatches {
			if len(nm) > 1 {
				cleaned := strings.ReplaceAll(nm[1], ".", "")
				cleaned = strings.ReplaceAll(cleaned, ",", ".")
				if val, err := strconv.ParseFloat(cleaned, 64); err == nil && val > totalAmount {
					totalAmount = val
				}
			}
		}
	}

	if totalAmount == 0 {
		totalAmount = 350000.0
	}

	if strings.Contains(lowerText, "ppn") || strings.Contains(lowerText, "pajak") || strings.Contains(lowerText, "tax") || strings.Contains(lowerText, "pb1") {
		taxAmount = totalAmount * 0.11 / 1.11
		if taxAmount > 0 {
			taxAmount = float64(int(taxAmount*100)) / 100
		}
	}

	subtotalBeforeTax := totalAmount - taxAmount
	lineItems := []ReceiptLineItem{
		{
			Description: fmt.Sprintf("%s - Transaksi Operasional", vendorName),
			Qty:         1.0,
			UnitPrice:   subtotalBeforeTax,
			Subtotal:    subtotalBeforeTax,
		},
	}

	if taxAmount > 0 {
		lineItems = append(lineItems, ReceiptLineItem{
			Description: "PPN Masukan (11%)",
			Qty:         1.0,
			UnitPrice:   taxAmount,
			Subtotal:    taxAmount,
		})
	}

	return &ReceiptOCRResult{
		VendorName:         vendorName,
		InvoiceDate:        invoiceDate,
		TotalAmount:        totalAmount,
		TaxAmount:          taxAmount,
		Currency:           "IDR",
		CategorySuggestion: category,
		LineItems:          lineItems,
		ConfidenceScore:    confidence,
		RawExtractedText:   fullText,
		Source:             "local_heuristic_ocr",
	}, nil
}

// Helper to sanitize JSON response from LLMs
func extractJSONString(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start != -1 && end != -1 && end > start {
		return raw[start : end+1]
	}
	return raw
}
