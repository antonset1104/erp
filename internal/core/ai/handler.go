package ai

import (
	"encoding/base64"
	"net/http"
	"strings"

	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	client *Client
	db     *gorm.DB
}

func NewHandler(client *Client, db *gorm.DB) *Handler {
	return &Handler{
		client: client,
		db:     db,
	}
}

func (h *Handler) GetStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"cloudflare_configured": h.client.IsCloudflareConfigured(),
		"gemini_configured":     h.client.IsGeminiConfigured(),
		"openai_configured":     h.client.IsOpenAIConfigured(),
		"anthropic_configured":  h.client.IsAnthropicConfigured(),
		"deepseek_configured":   h.client.IsDeepSeekConfigured(),
		"active_provider":       h.client.ActiveProviderName(),
		"model":                 h.client.modelForProvider(h.client.ActiveProviderName()),
		"engine":                h.client.ActiveProviderName(),
		"providers":             h.client.GetSupportedProviders(),
	})
}

func (h *Handler) GetProviders(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"active_provider": h.client.ActiveProviderName(),
		"data":            h.client.GetSupportedProviders(),
	})
}

func (h *Handler) GetToolsManifest(c *gin.Context) {
	c.JSON(http.StatusOK, h.client.GetAgentToolsManifest())
}

func (h *Handler) ExecuteAgent(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req AgentExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.client.ExecuteAutonomousAgent(h.db, &req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetGCPStatus(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	c.JSON(http.StatusOK, h.client.GetGCPEcosystemStatus(h.db, companyID))
}

func (h *Handler) SyncGCP(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req GCPSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res := h.client.ExecuteGCPSync(h.db, &req, companyID)
	c.JSON(http.StatusOK, res)
}

// Model Context Protocol (MCP) JSON-RPC 2.0 Handler
func (h *Handler) HandleMCP(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req MCPJSONRPCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"jsonrpc": "2.0",
			"id":      nil,
			"error": gin.H{
				"code":    -32700,
				"message": "Parse error: " + err.Error(),
			},
		})
		return
	}

	resp := h.client.HandleMCPJSONRPC(h.db, &req, companyID)
	c.JSON(http.StatusOK, resp)
}

// Google Cloud Agent-to-Agent (A2A) Discovery Card Handler
func (h *Handler) GetA2AAgentCard(c *gin.Context) {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	baseURL := scheme + "://" + c.Request.Host
	c.JSON(http.StatusOK, h.client.GetA2AAgentCard(baseURL))
}

// Google Cloud A2A Task Submission Handler
func (h *Handler) ExecuteA2ATask(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req A2ATaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, h.client.ExecuteA2ATask(h.db, &req, companyID))
}

// AI Document Intelligence (OCR & 3-Way Matching)
func (h *Handler) DocumentIntelligence(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req DocumentIntelligenceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res := h.client.ExecuteDocumentIntelligence(h.db, &req, companyID)
	c.JSON(http.StatusOK, res)
}

// Continuous AI Audit & Fraud Radar
func (h *Handler) FraudRadar(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req FraudRadarRequest
	_ = c.ShouldBindJSON(&req)

	res := h.client.RunFraudRadar(h.db, &req, companyID)
	c.JSON(http.StatusOK, res)
}

// Predictive 30/60/90-Day Cashflow Forecasting
func (h *Handler) CashflowForecast(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req CashflowForecastRequest
	_ = c.ShouldBindJSON(&req)

	res := h.client.GetCashflowForecast(h.db, &req, companyID)
	c.JSON(http.StatusOK, res)
}

// ESG & Carbon Footprint Ledger (Scope 1, 2, 3 GHG Accounting)
func (h *Handler) ESGReport(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req ESGReportRequest
	_ = c.ShouldBindJSON(&req)

	res := h.client.GetESGCarbonReport(h.db, &req, companyID)
	c.JSON(http.StatusOK, res)
}

// Omnichannel ChatOps Webhook (WhatsApp, Telegram, Slack)
func (h *Handler) ChatOpsWebhook(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req ChatOpsWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res := h.client.ExecuteChatOpsWebhook(h.db, &req, companyID)
	c.JSON(http.StatusOK, res)
}

type TicketTriageRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

func (h *Handler) TriageTicket(c *gin.Context) {
	var req TicketTriageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.client.AnalyzeTicket(req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

type QualityRCARequest struct {
	Title     string `json:"title" binding:"required"`
	RootCause string `json:"root_cause"`
	Notes     string `json:"notes"`
}

func (h *Handler) QualityRCA(c *gin.Context) {
	var req QualityRCARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.client.AnalyzeQualityAlert(req.Title, req.RootCause, req.Notes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

type ExpenseCategorizeRequest struct {
	Description string  `json:"description" binding:"required"`
	Amount      float64 `json:"amount"`
}

func (h *Handler) ExpenseCategorize(c *gin.Context) {
	var req ExpenseCategorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.client.AnalyzeExpense(req.Description, req.Amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

type AskERPRequest struct {
	Query    string `json:"query" binding:"required"`
	Provider string `json:"provider"`
}

func (h *Handler) AskERP(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req AskERPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var saleCount int64
	var totalSales float64
	var purchaseCount int64
	var ticketCount int64
	var qualityAlertCount int64

	h.db.Table("sale_orders").Where("company_id = ?", companyID).Count(&saleCount)
	h.db.Table("sale_orders").Where("company_id = ? AND state IN ?", companyID, []string{"sale", "done"}).Select("COALESCE(SUM(amount_total), 0)").Scan(&totalSales)
	h.db.Table("purchase_orders").Where("company_id = ?", companyID).Count(&purchaseCount)
	h.db.Table("helpdesk_tickets").Where("company_id = ? AND stage NOT IN ('solved', 'cancelled')", companyID).Count(&ticketCount)
	h.db.Table("quality_alerts").Where("company_id = ? AND stage != 'closed'", companyID).Count(&qualityAlertCount)

	statsContext := map[string]interface{}{
		"total_sales":     totalSales,
		"sales_orders":    saleCount,
		"purchase_orders": purchaseCount,
		"tickets":         ticketCount,
		"quality_alerts":  qualityAlertCount,
	}

	resp, err := h.client.AskERP(req.Query, statsContext)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

type ApprovalRiskRequest struct {
	Title        string  `json:"title" binding:"required"`
	CategoryName string  `json:"category_name"`
	Amount       float64 `json:"amount"`
	Reason       string  `json:"reason"`
	Reference    string  `json:"reference"`
}

func (h *Handler) ApprovalRisk(c *gin.Context) {
	var req ApprovalRiskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.client.AnalyzeApprovalRisk(req.Title, req.CategoryName, req.Amount, req.Reason, req.Reference)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}

type ReceiptOCRRequest struct {
	ImageBase64 string `json:"image_base64"`
	TextHint    string `json:"text_hint"`
	MimeType    string `json:"mime_type"`
}

func (h *Handler) ReceiptOCR(c *gin.Context) {
	var req ReceiptOCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var imgBytes []byte
	if req.ImageBase64 != "" {
		parts := strings.Split(req.ImageBase64, ",")
		rawB64 := req.ImageBase64
		if len(parts) > 1 {
			rawB64 = parts[1]
		}
		imgBytes, _ = base64.StdEncoding.DecodeString(rawB64)
	}

	analysis, err := h.client.AnalyzeReceiptVision(imgBytes, req.MimeType, req.TextHint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, analysis)
}


// Demand Sensing & Smart Inventory Replenishment
func (h *Handler) DemandSensing(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req DemandSensingRequest
	_ = c.ShouldBindJSON(&req)
	c.JSON(http.StatusOK, h.client.AnalyzeDemandSensing(h.db, &req, companyID))
}

func (h *Handler) AutoReplenish(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req AutoReplenishRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.client.ExecuteAutoReplenishmentPO(h.db, &req, companyID))
}

// Hands-Free Voice Ops Assistant
func (h *Handler) VoiceCommand(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req VoiceCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.client.ExecuteVoiceCommand(h.db, &req, companyID))
}

// AI Autonomous Supplier Negotiation & RFQ Bidding Wars
func (h *Handler) CreateRFQBidTender(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req CreateTenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tender, err := h.client.CreateBiddingTender(h.db, &req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tender)
}

func (h *Handler) SubmitVendorBid(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req VendorBidSubmission
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	eval, err := h.client.SubmitVendorBid(h.db, &req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, eval)
}

func (h *Handler) EvaluateRFQBidTender(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	tenderID := c.Param("tender_id")
	card, err := h.client.EvaluateBiddingTender(h.db, tenderID, companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, card)
}

func (h *Handler) AwardRFQBidTender(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	tenderID := c.Param("tender_id")
	var req AwardTenderRequest
	_ = c.ShouldBindJSON(&req)
	po, err := h.client.AwardBiddingTender(h.db, tenderID, req.VendorID, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, po)
}

// Supply Chain Digital Twin & What-If Scenario Simulator
func (h *Handler) DigitalTwinSimulate(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req DigitalTwinSimulateRequest
	_ = c.ShouldBindJSON(&req)
	resp := h.client.ExecuteDigitalTwinSimulation(h.db, &req, companyID)
	c.JSON(http.StatusOK, resp)
}

// AI Autonomous Dynamic Pricing Optimizer
func (h *Handler) RecommendPricing(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req PricingRecommendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp, err := h.client.OptimizeProductPricing(h.db, &req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ApplyPricingRule(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req ApplyPricingRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	prod, err := h.client.ApplyPricingRule(h.db, &req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, prod)
}

// AI Knowledge Base Semantic Search & Customer Support Copilot (RAG)
func (h *Handler) QueryKnowledge(c *gin.Context) {
	var req KnowledgeQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.client.QueryKnowledgeRAG(&req))
}

func (h *Handler) SupportCopilotDraft(c *gin.Context) {
	var req CopilotDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.client.DraftSupportCopilotReply(&req))
}

// AI Contract Lifecycle Management (CLM) & Clause Risk Analysis
func (h *Handler) AnalyzeContract(c *gin.Context) {
	var req ContractAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.client.AnalyzeContractCLM(&req))
}

func (h *Handler) RedlineContract(c *gin.Context) {
	var req ContractRedlineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, h.client.RedlineContractClause(&req))
}

// Multi-Tier Supply Chain Bullwhip Guard & Lead-Time Anomaly Detector
func (h *Handler) GetBullwhipIndex(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	c.JSON(http.StatusOK, h.client.ComputeBullwhipIndex(h.db, companyID))
}

func (h *Handler) DetectLeadTimeAnomalies(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req LeadTimeAnomalyRequest
	_ = c.ShouldBindJSON(&req)
	c.JSON(http.StatusOK, h.client.DetectLeadTimeAnomalies(h.db, &req, companyID))
}

// RegisterRootProtocolRoutes registers public protocol discovery endpoints like /.well-known/agent.json and /mcp
func RegisterRootProtocolRoutes(r *gin.Engine, h *Handler) {
	r.GET("/.well-known/agent.json", h.GetA2AAgentCard)
	r.POST("/mcp", h.HandleMCP)
	r.GET("/mcp", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "GoERP Native Model Context Protocol (MCP) Server. Send POST JSON-RPC 2.0 requests to this endpoint.",
			"version": "2025-03-26",
		})
	})
}

func RegisterAIRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, h *Handler) {
	group := router.Group("/ai")
	{
		// Public protocol & discovery endpoints (optional token)
		group.GET("/a2a/agent.json", h.GetA2AAgentCard)
		group.POST("/mcp", h.HandleMCP)
		group.POST("/chatops/webhook", h.ChatOpsWebhook) // Webhooks from WhatsApp/Slack can use signature verification or direct POST
		group.POST("/rfq/submit-bid", h.SubmitVendorBid) // Public tokenized vendor bid intake portal

		// Authenticated AI routes
		authed := group.Group("")
		authed.Use(authMiddleware)
		{
			authed.GET("/status", h.GetStatus)
			authed.GET("/providers", h.GetProviders)
			authed.GET("/tools", h.GetToolsManifest)
			authed.POST("/agent/execute", h.ExecuteAgent)
			authed.GET("/gcp/status", h.GetGCPStatus)
			authed.POST("/gcp/sync", h.SyncGCP)

			// Enterprise AI Intelligence Endpoints
			authed.POST("/a2a/tasks/send", h.ExecuteA2ATask)
			authed.POST("/doc-intelligence", h.DocumentIntelligence)
			authed.GET("/fraud-radar", h.FraudRadar)
			authed.POST("/fraud-radar", h.FraudRadar)
			authed.GET("/cashflow-forecast", h.CashflowForecast)
			authed.POST("/cashflow-forecast", h.CashflowForecast)
			authed.GET("/esg-report", h.ESGReport)
			authed.POST("/esg-report", h.ESGReport)
			authed.GET("/demand-sensing", h.DemandSensing)
			authed.POST("/demand-sensing", h.DemandSensing)
			authed.POST("/replenish/auto-po", h.AutoReplenish)
			authed.POST("/voice/command", h.VoiceCommand)

			// AI Autonomous RFQ Bidding & Digital Twin
			authed.POST("/rfq/tenders", h.CreateRFQBidTender)
			authed.GET("/rfq/evaluate/:tender_id", h.EvaluateRFQBidTender)
			authed.POST("/rfq/award/:tender_id", h.AwardRFQBidTender)
			authed.POST("/digital-twin/simulate", h.DigitalTwinSimulate)

			// AI Dynamic Pricing & RAG Copilot
			authed.POST("/pricing/recommend", h.RecommendPricing)
			authed.POST("/pricing/apply-rule", h.ApplyPricingRule)
			authed.POST("/knowledge/query", h.QueryKnowledge)
			authed.POST("/helpdesk/copilot-draft", h.SupportCopilotDraft)

			// AI Contract Lifecycle Management (CLM)
			authed.POST("/contracts/analyze", h.AnalyzeContract)
			authed.POST("/contracts/redline", h.RedlineContract)

			// Supply Chain Bullwhip Guard & Lead-Time Anomaly Detector
			authed.GET("/supply-chain/bullwhip-index", h.GetBullwhipIndex)
			authed.POST("/supply-chain/lead-time-anomalies", h.DetectLeadTimeAnomalies)

			// Domain AI Endpoints
			authed.POST("/ticket-triage", h.TriageTicket)
			authed.POST("/quality-rca", h.QualityRCA)
			authed.POST("/expense-categorize", h.ExpenseCategorize)
			authed.POST("/ask-erp", h.AskERP)
			authed.POST("/ask", h.AskERP)
			authed.POST("/approval-risk", h.ApprovalRisk)
			authed.POST("/receipt-ocr", h.ReceiptOCR)
		}
	}
}
