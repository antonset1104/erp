package inbound_webhook

import (
	"io"
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

// RegisterRoutes registers all public ingestion and protected management endpoints
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	// 1. Public Ingestion Gateway for External Partners
	publicGroup := r.Group("/inbound-webhooks")
	{
		publicGroup.POST("/gateway/:key", h.HandlePublicInbound)
	}

	// 2. Protected Management Endpoints
	mgmtGroup := r.Group("/inbound-webhooks", authMW)
	{
		mgmtGroup.GET("/stats", h.GetStats)
		mgmtGroup.GET("/metrics", h.GetStats)

		// Endpoints Management
		mgmtGroup.GET("/endpoints", h.ListEndpoints)
		mgmtGroup.POST("/endpoints", h.CreateEndpoint)
		mgmtGroup.GET("/endpoints/:id", h.GetEndpoint)
		mgmtGroup.PUT("/endpoints/:id", h.UpdateEndpoint)
		mgmtGroup.DELETE("/endpoints/:id", h.DeleteEndpoint)

		// Inbound Logs & Detail
		mgmtGroup.GET("/logs", h.ListLogs)
		mgmtGroup.GET("/logs/:id", h.GetLog)

		// Simulator & AI Payload Mapper
		mgmtGroup.POST("/simulate", h.SimulateInbound)
		mgmtGroup.POST("/ai-analyze", h.AnalyzePayloadAI)
	}
}

// ---------------------------------------------------------------------------
// Public Ingestion Handler
// ---------------------------------------------------------------------------

func (h *Handler) HandlePublicInbound(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing endpoint key in url"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	clientIP := c.ClientIP()

	log, err := h.service.ProcessInboundWebhook(key, headers, rawBody, clientIP)
	if err != nil {
		status := http.StatusBadRequest
		if log != nil && log.HTTPStatusCode > 0 {
			status = log.HTTPStatusCode
		}
		c.JSON(status, gin.H{
			"status": "error",
			"log_code": func() string {
				if log != nil {
					return log.LogCode
				}
				return ""
			}(),
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "acknowledged",
		"log_code":     log.LogCode,
		"action_taken": log.ActionTaken,
		"duration_ms":  log.DurationMs,
	})
}

// ---------------------------------------------------------------------------
// Protected Endpoints Management Handlers
// ---------------------------------------------------------------------------

func (h *Handler) GetStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) ListEndpoints(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	endpoints, err := h.service.ListEndpoints(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, endpoints)
}

func (h *Handler) CreateEndpoint(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req CreateInboundEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endpoint, err := h.service.CreateEndpoint(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, endpoint)
}

func (h *Handler) GetEndpoint(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint id"})
		return
	}

	endpoint, err := h.service.GetEndpoint(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "endpoint not found"})
		return
	}
	c.JSON(http.StatusOK, endpoint)
}

func (h *Handler) UpdateEndpoint(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint id"})
		return
	}

	var req UpdateInboundEndpointRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	endpoint, err := h.service.UpdateEndpoint(uint(id), req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, endpoint)
}

func (h *Handler) DeleteEndpoint(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endpoint id"})
		return
	}

	if err := h.service.DeleteEndpoint(uint(id), companyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "endpoint successfully deleted"})
}

func (h *Handler) ListLogs(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	var filters InboundLogFilters
	if err := c.ShouldBindQuery(&filters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filters.CompanyID = companyID

	logs, total, err := h.service.ListLogs(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
	})
}

func (h *Handler) GetLog(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid log id"})
		return
	}

	log, err := h.service.GetLog(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "log not found"})
		return
	}
	c.JSON(http.StatusOK, log)
}

func (h *Handler) SimulateInbound(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req SimulateInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log, err := h.service.SimulateInboundWebhook(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "log": log})
		return
	}
	c.JSON(http.StatusOK, log)
}

func (h *Handler) AnalyzePayloadAI(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var body struct {
		PayloadJSON string `json:"payload_json" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.service.AnalyzeInboundPayloadWithAI(body.PayloadJSON, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}
