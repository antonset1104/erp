package workflow_engine

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/workflows")
	group.Use(authMiddleware)
	{
		// Definitions
		group.GET("/definitions", h.ListDefinitions)
		group.POST("/definitions", h.CreateDefinition)
		group.GET("/definitions/:id", h.GetDefinitionByID)

		// Instances & Runtime Execution
		group.POST("/start", h.StartWorkflow)
		group.GET("/instances", h.ListInstances)
		group.GET("/instances/:id", h.GetInstanceByID)
		group.GET("/instances/:id/logs", h.GetActivityLogs)
		group.POST("/instances/:id/action", h.ExecuteAction)
		group.GET("/pending-tasks", h.ListPendingTasks)

		// KPI & Analytics
		group.GET("/stats", h.GetStats)

		// AI Copilot
		group.POST("/ai/design-copilot", h.AIAnalyzeWorkflow)
	}
}

func (h *Handler) ListDefinitions(c *gin.Context) {
	companyID := getCompanyID(c)
	list, err := h.service.ListDefinitions(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) CreateDefinition(c *gin.Context) {
	companyID := getCompanyID(c)
	var req CreateWorkflowDefinitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wf, err := h.service.CreateDefinition(companyID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, wf)
}

func (h *Handler) GetDefinitionByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	wf, err := h.service.GetDefinitionByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Definisi alur kerja tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, wf)
}

func (h *Handler) StartWorkflow(c *gin.Context) {
	companyID := getCompanyID(c)
	userID := getUserID(c)
	userName := getUserName(c)

	var req StartWorkflowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inst, err := h.service.StartWorkflow(companyID, req, userID, userName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, inst)
}

func (h *Handler) ListInstances(c *gin.Context) {
	companyID := getCompanyID(c)
	status := c.Query("status")
	list, err := h.service.ListInstances(companyID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetInstanceByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	inst, err := h.service.GetInstanceByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Instansi alur kerja tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, inst)
}

func (h *Handler) GetActivityLogs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	logs, err := h.service.GetActivityLogs(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) ExecuteAction(c *gin.Context) {
	companyID := getCompanyID(c)
	userID := getUserID(c)
	userName := getUserName(c)

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var req ExecuteWorkflowActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inst, err := h.service.ExecuteAction(companyID, uint(id), req.Action, userID, userName, req.Comments, req.AdditionalData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, inst)
}

func (h *Handler) ListPendingTasks(c *gin.Context) {
	companyID := getCompanyID(c)
	userID := getUserID(c)
	role := c.GetString("role")

	tasks, err := h.service.ListPendingTasks(companyID, role, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := getCompanyID(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) AIAnalyzeWorkflow(c *gin.Context) {
	var req AIWorkflowDesignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	report, err := h.service.AIAnalyzeWorkflow(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// Helpers
func getCompanyID(c *gin.Context) uint {
	if val, ok := c.Get("company_id"); ok {
		if id, ok := val.(uint); ok {
			return id
		}
	}
	return 1
}

func getUserID(c *gin.Context) uint {
	if val, ok := c.Get("user_id"); ok {
		if id, ok := val.(uint); ok {
			return id
		}
	}
	return 1
}

func getUserName(c *gin.Context) string {
	if val, ok := c.Get("name"); ok {
		if name, ok := val.(string); ok && name != "" {
			return name
		}
	}
	if val, ok := c.Get("user_name"); ok {
		if name, ok := val.(string); ok && name != "" {
			return name
		}
	}
	return "Administrator"
}
