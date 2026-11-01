package intercompany

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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	group := r.Group("/intercompany")
	group.Use(authMW)
	{
		// Stats
		group.GET("/stats", h.GetStats)

		// Groups
		group.GET("/groups", h.ListGroups)
		group.POST("/groups", h.CreateGroup)

		// Relations
		group.GET("/relations", h.ListRelations)
		group.POST("/relations", h.CreateRelation)

		// Mirror Transactions
		group.GET("/transactions", h.ListTransactions)
		group.POST("/mirror/sale", h.MirrorSaleOrder)
		group.POST("/mirror/invoice", h.MirrorInvoice)

		// Consolidation Runs
		group.GET("/consolidation/runs", h.ListConsolidationRuns)
		group.POST("/consolidation/execute", h.ExecuteConsolidation)
		group.GET("/consolidation/runs/:id", h.GetConsolidationRun)
		group.GET("/consolidation/runs/:id/ai-audit", h.AnalyzeConsolidationAI)
		group.POST("/consolidation/runs/:id/ai-audit", h.AnalyzeConsolidationAI)
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

func (h *Handler) ListGroups(c *gin.Context) {
	groups, err := h.service.ListGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, groups)
}

func (h *Handler) CreateGroup(c *gin.Context) {
	var req CreateGroupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, companyID, _ := middleware.GetUserContext(c)

	group, err := h.service.CreateGroup(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, group)
}

func (h *Handler) ListRelations(c *gin.Context) {
	groupIDStr := c.Query("group_id")
	var groupID uint
	if groupIDStr != "" {
		id, _ := strconv.ParseUint(groupIDStr, 10, 32)
		groupID = uint(id)
	}
	relations, err := h.service.ListRelations(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, relations)
}

func (h *Handler) CreateRelation(c *gin.Context) {
	var req CreateRelationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, companyID, _ := middleware.GetUserContext(c)

	rel, err := h.service.CreateRelation(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rel)
}

func (h *Handler) ListTransactions(c *gin.Context) {
	groupIDStr := c.Query("group_id")
	var groupID uint
	if groupIDStr != "" {
		id, _ := strconv.ParseUint(groupIDStr, 10, 32)
		groupID = uint(id)
	}
	txs, err := h.service.ListTransactions(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, txs)
}

func (h *Handler) MirrorSaleOrder(c *gin.Context) {
	var req TriggerMirrorSaleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, companyID, _ := middleware.GetUserContext(c)

	tx, err := h.service.MirrorSaleOrderToPurchaseOrder(req.SaleOrderID, userID, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tx)
}

func (h *Handler) MirrorInvoice(c *gin.Context) {
	var req TriggerMirrorInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, companyID, _ := middleware.GetUserContext(c)

	tx, err := h.service.MirrorInvoiceToVendorBill(req.InvoiceID, userID, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tx)
}

func (h *Handler) ListConsolidationRuns(c *gin.Context) {
	groupIDStr := c.Query("group_id")
	var groupID uint
	if groupIDStr != "" {
		id, _ := strconv.ParseUint(groupIDStr, 10, 32)
		groupID = uint(id)
	}
	runs, err := h.service.ListConsolidationRuns(groupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, runs)
}

func (h *Handler) ExecuteConsolidation(c *gin.Context) {
	var req RunConsolidationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, companyID, _ := middleware.GetUserContext(c)

	run, err := h.service.ExecuteFinancialConsolidation(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, run)
}

func (h *Handler) GetConsolidationRun(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run id"})
		return
	}

	run, err := h.service.GetConsolidationRun(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "consolidation run not found"})
		return
	}
	c.JSON(http.StatusOK, run)
}

func (h *Handler) AnalyzeConsolidationAI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid run id"})
		return
	}
	_, companyID, _ := middleware.GetUserContext(c)

	report, err := h.service.AnalyzeConsolidationWithAI(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}
