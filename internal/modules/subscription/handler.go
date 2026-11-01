package subscription

import (
	"net/http"
	"strconv"

	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all subscription REST endpoints
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	subGroup := r.Group("/subscriptions", authMW)
	{
		// SaaS Metrics & Analytics
		subGroup.GET("/metrics", h.GetMetrics)

		// Batch Billing Runner
		subGroup.POST("/run-billing", h.RunBillingBatch)

		// Templates
		subGroup.GET("/templates", h.ListTemplates)
		subGroup.POST("/templates", h.CreateTemplate)
		subGroup.GET("/templates/:id", h.GetTemplate)

		// Invoice Logs
		subGroup.GET("/invoices", h.ListInvoices)

		// Subscription Contracts CRUD & Actions
		subGroup.GET("", h.ListSubscriptions)
		subGroup.POST("", h.CreateSubscription)
		subGroup.GET("/:id", h.GetSubscription)
		subGroup.PUT("/:id", h.UpdateSubscription)

		// Lifecycle Actions
		subGroup.POST("/:id/start", h.StartSubscription)
		subGroup.POST("/:id/pause", h.PauseSubscription)
		subGroup.POST("/:id/resume", h.ResumeSubscription)
		subGroup.POST("/:id/cancel", h.CancelSubscription)
		subGroup.POST("/:id/close", h.CloseSubscription)

		// Invoicing & AI Copilot
		subGroup.POST("/:id/generate-invoice", h.GenerateSingleInvoice)
		subGroup.POST("/:id/ai-churn-analysis", h.AnalyzeChurnRisk)
	}
}

func (h *Handler) GetMetrics(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	metrics, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *Handler) ListSubscriptions(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	state := c.Query("state")
	search := c.Query("search")
	partnerID, _ := strconv.ParseUint(c.Query("partner_id"), 10, 32)
	templateID, _ := strconv.ParseUint(c.Query("template_id"), 10, 32)

	subs, err := h.service.ListSubscriptions(companyID, state, uint(partnerID), uint(templateID), search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, subs)
}

func (h *Handler) GetSubscription(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID langganan tidak valid"})
		return
	}

	sub, err := h.service.GetSubscription(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Langganan tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) CreateSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req CreateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.service.CreateSubscription(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sub)
}

func (h *Handler) UpdateSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID langganan tidak valid"})
		return
	}

	var req UpdateSubscriptionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.service.UpdateSubscription(uint(id), req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) StartSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	sub, err := h.service.StartSubscription(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) PauseSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var payload struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&payload)
	reason := payload.Reason
	if reason == "" {
		reason = "Dijeda atas permintaan pelanggan"
	}

	sub, err := h.service.PauseSubscription(uint(id), reason, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) ResumeSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	sub, err := h.service.ResumeSubscription(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) CancelSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var payload struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&payload)
	reason := payload.Reason
	if reason == "" {
		reason = "Pembatalan langganan oleh pelanggan"
	}

	sub, err := h.service.CancelSubscription(uint(id), reason, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) CloseSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var payload struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&payload)
	reason := payload.Reason
	if reason == "" {
		reason = "Kontrak selesai wajar"
	}

	sub, err := h.service.CloseSubscription(uint(id), reason, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) GenerateSingleInvoice(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	logEntry, err := h.service.GenerateSingleInvoice(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logEntry)
}

func (h *Handler) RunBillingBatch(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req BillingBatchRequest
	_ = c.ShouldBindJSON(&req)

	res, err := h.service.RunBillingBatch(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AnalyzeChurnRisk(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	report, err := h.service.AnalyzeChurnRisk(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) ListTemplates(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	tmpls, err := h.service.ListTemplates(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tmpls)
}

func (h *Handler) CreateTemplate(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tmpl, err := h.service.CreateTemplate(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, tmpl)
}

func (h *Handler) GetTemplate(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID template tidak valid"})
		return
	}

	tmpl, err := h.service.GetTemplate(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Template tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, tmpl)
}

func (h *Handler) ListInvoices(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	subID, _ := strconv.ParseUint(c.Query("subscription_id"), 10, 32)

	logs, err := h.service.ListInvoiceLogs(companyID, uint(subID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}
