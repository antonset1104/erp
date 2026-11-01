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
		"model":                 h.client.cfg.CloudflareAIModel,
		"engine": func() string {
			if h.client.IsCloudflareConfigured() {
				return "cloudflare_workers_ai_edge"
			}
			return "local_smart_heuristic_engine"
		}(),
	})
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
	Query string `json:"query" binding:"required"`
}

func (h *Handler) AskERP(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req AskERPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Gather live business context from DB
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
		// optional decode base64
		// if format data:image/png;base64,... strip header
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

func RegisterAIRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, h *Handler) {
	group := router.Group("/ai")
	group.Use(authMiddleware)
	{
		group.GET("/status", h.GetStatus)
		group.POST("/ticket-triage", h.TriageTicket)
		group.POST("/quality-rca", h.QualityRCA)
		group.POST("/expense-categorize", h.ExpenseCategorize)
		group.POST("/ask-erp", h.AskERP)
		group.POST("/approval-risk", h.ApprovalRisk)
		group.POST("/receipt-ocr", h.ReceiptOCR)
	}
}
