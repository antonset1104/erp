package integration_hub

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// FlexibleAuthMiddleware allows both Bearer JWT tokens and X-API-Key / ApiKey headers
func (h *Handler) FlexibleAuthMiddleware(jwtAuth gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKeyHeader := c.GetHeader("X-API-Key")
		authHeader := c.GetHeader("Authorization")

		// Check if X-API-Key is present
		if apiKeyHeader != "" {
			apiKey, err := h.service.AuthenticateAPIKey(apiKeyHeader, c.ClientIP(), "")
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("API Key authentication failed: %s", err.Error())})
				c.Abort()
				return
			}
			c.Set("company_id", apiKey.CompanyID)
			c.Set("user_id", apiKey.CreatedBy)
			c.Set("auth_type", "api_key")
			c.Set("api_key", apiKey)
			c.Next()
			return
		}

		// Check if Authorization header begins with "ApiKey " or is raw key
		if authHeader != "" && (strings.HasPrefix(authHeader, "ApiKey ") || strings.HasPrefix(authHeader, "Bearer ak_")) {
			rawKey := strings.TrimPrefix(authHeader, "ApiKey ")
			if strings.HasPrefix(rawKey, "Bearer ") {
				rawKey = strings.TrimPrefix(rawKey, "Bearer ")
			}
			apiKey, err := h.service.AuthenticateAPIKey(rawKey, c.ClientIP(), "")
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": fmt.Sprintf("API Key authentication failed: %s", err.Error())})
				c.Abort()
				return
			}
			c.Set("company_id", apiKey.CompanyID)
			c.Set("user_id", apiKey.CreatedBy)
			c.Set("auth_type", "api_key")
			c.Set("api_key", apiKey)
			c.Next()
			return
		}

		// Otherwise fall back to standard JWT auth
		jwtAuth(c)
	}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, jwtAuth gin.HandlerFunc) {
	group := api.Group("/integration")

	// Public Specification & Discovery
	group.GET("/openapi.json", h.GetOpenAPISpec)
	group.GET("/schema/models", h.GetAvailableModels)

	// Dual-Auth Endpoints (API Key or JWT)
	dualAuth := h.FlexibleAuthMiddleware(jwtAuth)
	dual := group.Group("")
	dual.Use(dualAuth)
	{
		// Universal Open RPC Engine
		dual.POST("/rpc/search_read", h.RPCQuery)
		dual.POST("/rpc/create", h.RPCCreate)
		dual.POST("/rpc/write", h.RPCWrite)
		dual.POST("/rpc/unlink", h.RPCUnlink)
		dual.POST("/rpc/execute_action", h.RPCExecuteAction)

		// Pre-Built Connectors
		dual.GET("/connectors", h.ListConnectors)
		dual.POST("/connectors/execute", h.ExecuteConnectorAction)
		dual.POST("/connectors/action", h.ExecuteConnectorAction)

		// Batch Data ETL Jobs
		dual.GET("/data-jobs", h.ListDataJobs)
		dual.POST("/data-jobs", h.RunBatchDataJob)

		// Stats & Telemetry
		dual.GET("/stats", h.GetStats)
		dual.GET("/logs", h.ListLogs)

		// AI Integration Architect
		dual.POST("/ai-architect", h.AnalyzeIntegrationWithAI)
		dual.POST("/ai/architect-copilot", h.AnalyzeIntegrationWithAI)
	}

	// Admin-Only Routes for Key Management (Requires User JWT)
	admin := group.Group("")
	admin.Use(jwtAuth)
	{
		admin.GET("/api-keys", h.ListAPIKeys)
		admin.POST("/api-keys", h.CreateAPIKey)
		admin.DELETE("/api-keys/:id", h.RevokeAPIKey)
		admin.POST("/api-keys/:id/revoke", h.RevokeAPIKey)
		admin.PUT("/connectors/:code", h.UpdateConnector)
	}
}

// ----------------- Handlers -----------------

func (h *Handler) GetOpenAPISpec(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	if origin == "" {
		origin = "http://" + c.Request.Host
	}
	spec := h.service.GenerateOpenAPISpec(origin)
	c.JSON(http.StatusOK, spec)
}

func (h *Handler) GetAvailableModels(c *gin.Context) {
	models := h.service.GetAvailableModels()
	c.JSON(http.StatusOK, gin.H{"data": models, "total": len(models)})
}

func (h *Handler) RPCQuery(c *gin.Context) {
	start := time.Now()
	companyID := getCompanyID(c)

	var req RPCQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ExecuteRPCQuery(&req, companyID)
	duration := time.Since(start).Milliseconds()
	apiKey, _ := c.Get("api_key")
	var keyPtr *APIKey
	if k, ok := apiKey.(*APIKey); ok {
		keyPtr = k
	}

	if err != nil {
		h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/search_read", req.Model, "search_read", http.StatusInternalServerError, duration, "", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "success": false})
		return
	}

	h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/search_read", req.Model, "search_read", http.StatusOK, duration, fmt.Sprintf("Fetched %d records", resp.Count), "")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RPCCreate(c *gin.Context) {
	start := time.Now()
	companyID := getCompanyID(c)

	var req RPCCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ExecuteRPCCreate(&req, companyID)
	duration := time.Since(start).Milliseconds()
	keyPtr := getAPIKeyPtr(c)

	if err != nil {
		h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/create", req.Model, "create", http.StatusInternalServerError, duration, "", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "success": false})
		return
	}

	h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/create", req.Model, "create", http.StatusCreated, duration, "Record created successfully", "")
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) RPCWrite(c *gin.Context) {
	start := time.Now()
	companyID := getCompanyID(c)

	var req RPCWriteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ExecuteRPCWrite(&req, companyID)
	duration := time.Since(start).Milliseconds()
	keyPtr := getAPIKeyPtr(c)

	if err != nil {
		h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/write", req.Model, "write", http.StatusInternalServerError, duration, "", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "success": false})
		return
	}

	h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/write", req.Model, "write", http.StatusOK, duration, fmt.Sprintf("Updated record ID %d", req.ID), "")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RPCUnlink(c *gin.Context) {
	start := time.Now()
	companyID := getCompanyID(c)

	var req RPCUnlinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ExecuteRPCUnlink(&req, companyID)
	duration := time.Since(start).Milliseconds()
	keyPtr := getAPIKeyPtr(c)

	if err != nil {
		h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/unlink", req.Model, "unlink", http.StatusInternalServerError, duration, "", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "success": false})
		return
	}

	h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/unlink", req.Model, "unlink", http.StatusOK, duration, fmt.Sprintf("Deleted record ID %d", req.ID), "")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) RPCExecuteAction(c *gin.Context) {
	start := time.Now()
	companyID := getCompanyID(c)

	var req RPCExecuteActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ExecuteRPCAction(&req, companyID)
	duration := time.Since(start).Milliseconds()
	keyPtr := getAPIKeyPtr(c)

	if err != nil {
		h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/execute_action", req.Model, req.Action, http.StatusInternalServerError, duration, "", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "success": false})
		return
	}

	h.service.LogIntegrationCall(companyID, keyPtr, c.ClientIP(), getAuthType(c), "POST", "/integration/rpc/execute_action", req.Model, req.Action, http.StatusOK, duration, fmt.Sprintf("Action %s on ID %d", req.Action, req.ID), "")
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) ListConnectors(c *gin.Context) {
	companyID := getCompanyID(c)
	conns, err := h.service.ListConnectors(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": conns})
}

func (h *Handler) ExecuteConnectorAction(c *gin.Context) {
	companyID := getCompanyID(c)
	var req ExecuteConnectorActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.ExecuteConnectorAction(&req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *Handler) UpdateConnector(c *gin.Context) {
	var conn IntegrationConnector
	if err := c.ShouldBindJSON(&conn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateConnector(&conn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Connector updated", "data": conn})
}

func (h *Handler) ListAPIKeys(c *gin.Context) {
	companyID := getCompanyID(c)
	keys, err := h.service.ListAPIKeys(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": keys})
}

func (h *Handler) CreateAPIKey(c *gin.Context) {
	companyID := getCompanyID(c)
	userID := getUserID(c)

	var req CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.CreateAPIKey(&req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *Handler) RevokeAPIKey(c *gin.Context) {
	companyID := getCompanyID(c)
	userID := getUserID(c)
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 32)

	if err := h.service.RevokeAPIKey(uint(id), companyID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "API Key revoked successfully"})
}

func (h *Handler) RunBatchDataJob(c *gin.Context) {
	companyID := getCompanyID(c)
	var req RunBatchDataJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.service.RunBatchDataJob(&req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Data Job executed", "data": job})
}

func (h *Handler) ListDataJobs(c *gin.Context) {
	companyID := getCompanyID(c)
	jobs, err := h.service.ListDataJobs(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": jobs})
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := getCompanyID(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) ListLogs(c *gin.Context) {
	companyID := getCompanyID(c)
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}
	logs, err := h.service.ListLogs(companyID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

func (h *Handler) AnalyzeIntegrationWithAI(c *gin.Context) {
	var req AIIntegrationArchitectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.service.AnalyzeIntegrationWithAI(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}

// ----------------- Helpers -----------------

func getCompanyID(c *gin.Context) uint {
	if val, exists := c.Get("company_id"); exists {
		if id, ok := val.(uint); ok && id > 0 {
			return id
		}
	}
	return 1
}

func getUserID(c *gin.Context) uint {
	if val, exists := c.Get("user_id"); exists {
		if id, ok := val.(uint); ok {
			return id
		}
	}
	return 1
}

func getAuthType(c *gin.Context) string {
	if val, exists := c.Get("auth_type"); exists {
		if at, ok := val.(string); ok {
			return at
		}
	}
	return "jwt"
}

func getAPIKeyPtr(c *gin.Context) *APIKey {
	apiKey, exists := c.Get("api_key")
	if exists {
		if k, ok := apiKey.(*APIKey); ok {
			return k
		}
	}
	return nil
}
