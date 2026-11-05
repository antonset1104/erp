package webhook

import (
	"fmt"
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

// RegisterRoutes registers all webhook REST endpoints
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	whGroup := r.Group("/webhooks", authMW)
	{
		// Metrics & AI Diagnostic
		whGroup.GET("/stats", h.GetStats)
		whGroup.GET("/metrics", h.GetStats)
		whGroup.GET("/ai-diagnostic", h.AnalyzeHealth)

		// Subscriptions CRUD & Test Ping
		whGroup.GET("", h.ListSubscriptions)
		whGroup.POST("", h.CreateSubscription)
		whGroup.GET("/:id", h.GetSubscription)
		whGroup.PUT("/:id", h.UpdateSubscription)
		whGroup.DELETE("/:id", h.DeleteSubscription)
		whGroup.POST("/:id/test", h.TestPing)

		// Deliveries History & Retry Actions
		whGroup.GET("/deliveries", h.ListDeliveries)
		whGroup.GET("/deliveries/:id", h.GetDelivery)
		whGroup.POST("/deliveries/:id/retry", h.RetryDelivery)

		// Event Emission & Queue Runner
		whGroup.POST("/publish-event", h.PublishEvent)
		whGroup.POST("/run-retries", h.RunRetryQueue)
	}
}

func (h *Handler) GetStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) ListSubscriptions(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	subs, err := h.service.ListSubscriptions(companyID)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID webhook tidak valid"})
		return
	}

	sub, err := h.service.GetSubscription(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Webhook subscription tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) CreateSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req CreateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.service.CreateSubscription(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, sub)
}

func (h *Handler) UpdateSubscription(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID webhook tidak valid"})
		return
	}

	var req UpdateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sub, err := h.service.UpdateSubscription(uint(id), req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sub)
}

func (h *Handler) DeleteSubscription(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID webhook tidak valid"})
		return
	}

	if err := h.service.DeleteSubscription(uint(id), companyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Webhook subscription berhasil dihapus"})
}

func (h *Handler) TestPing(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID webhook tidak valid"})
		return
	}

	delivery, err := h.service.TestPing(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("Ping dikirim ke %s. Status HTTP: %d", delivery.TargetURL, delivery.StatusCode),
		"delivery": delivery,
	})
}

func (h *Handler) ListDeliveries(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	filters := DeliveryFilters{
		CompanyID: companyID,
		EventName: c.Query("event_name"),
		Status:    c.Query("status"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Search:    c.Query("search"),
	}

	if subIDStr := c.Query("subscription_id"); subIDStr != "" {
		if id, err := strconv.ParseUint(subIDStr, 10, 32); err == nil {
			filters.SubscriptionID = uint(id)
		}
	}

	limit := 100
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	filters.Limit = limit

	offset := 0
	if oStr := c.Query("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}
	filters.Offset = offset

	deliveries, total, err := h.service.ListDeliveries(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   deliveries,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetDelivery(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID delivery tidak valid"})
		return
	}

	delivery, err := h.service.GetDelivery(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Log delivery webhook tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, delivery)
}

func (h *Handler) RetryDelivery(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID delivery tidak valid"})
		return
	}

	delivery, err := h.service.RetryDelivery(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":  fmt.Sprintf("Retry selesai. Status HTTP: %d (%s)", delivery.StatusCode, delivery.Status),
		"delivery": delivery,
	})
}

func (h *Handler) PublishEvent(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req PublishEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	deliveries, err := h.service.PublishEvent(req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"event":                 req.EventName,
		"deliveries_dispatched": len(deliveries),
		"deliveries":            deliveries,
	})
}

func (h *Handler) RunRetryQueue(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	count, err := h.service.RunRetryQueue(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Berhasil memproses %d delivery pending", count),
		"count":   count,
	})
}

func (h *Handler) AnalyzeHealth(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	report, err := h.service.AnalyzeWebhookHealth(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}
