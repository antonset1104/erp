package mrp_advanced

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := r.Group("/mrp-advanced")
	if authMiddleware != nil {
		group.Use(authMiddleware)
	}

	{
		group.GET("/stats", h.GetStats)

		// Work Centers
		group.GET("/work-centers", h.ListWorkCenters)
		group.POST("/work-centers", h.CreateWorkCenter)
		group.GET("/work-centers/:id", h.GetWorkCenter)

		// Routings
		group.GET("/routings", h.ListRoutings)
		group.POST("/routings", h.CreateRouting)
		group.GET("/routings/:id", h.GetRouting)

		// Work Orders
		group.POST("/work-orders/generate", h.GenerateWorkOrders)
		group.GET("/work-orders", h.ListWorkOrders)
		group.GET("/work-orders/:id", h.GetWorkOrder)
		group.POST("/work-orders/:id/start", h.StartWorkOrder)
		group.POST("/work-orders/:id/finish", h.FinishWorkOrder)

		// Gantt Capacity Planning & AI Advisor
		group.GET("/gantt-capacity", h.GetGanttCapacity)
		group.GET("/ai-capacity-advisor", h.AnalyzeCapacityAI)
		group.POST("/ai-capacity-advisor", h.AnalyzeCapacityAI)
	}
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ----------------------------------------------------------------------------
// WORK CENTERS
// ----------------------------------------------------------------------------

func (h *Handler) ListWorkCenters(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	list, err := h.service.ListWorkCenters(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) CreateWorkCenter(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")

	var req CreateWorkCenterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wc, err := h.service.CreateWorkCenter(&req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, wc)
}

func (h *Handler) GetWorkCenter(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID stasiun kerja tidak valid"})
		return
	}

	wc, err := h.service.GetWorkCenter(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pusat kerja tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, wc)
}

// ----------------------------------------------------------------------------
// ROUTINGS
// ----------------------------------------------------------------------------

func (h *Handler) ListRoutings(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	list, err := h.service.ListRoutings(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) CreateRouting(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")

	var req CreateRoutingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	routing, err := h.service.CreateRouting(&req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, routing)
}

func (h *Handler) GetRouting(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rute operasi tidak valid"})
		return
	}

	routing, err := h.service.GetRouting(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rute operasi tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, routing)
}

// ----------------------------------------------------------------------------
// WORK ORDERS
// ----------------------------------------------------------------------------

func (h *Handler) GenerateWorkOrders(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")

	var req GenerateWorkOrdersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	wos, err := h.service.GenerateWorkOrdersForMO(&req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, wos)
}

func (h *Handler) ListWorkOrders(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	state := c.Query("state")

	list, err := h.service.ListWorkOrders(companyID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetWorkOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID perintah kerja tidak valid"})
		return
	}

	wo, err := h.service.GetWorkOrder(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Perintah kerja tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, wo)
}

func (h *Handler) StartWorkOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID perintah kerja tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")

	wo, err := h.service.StartWorkOrder(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, wo)
}

func (h *Handler) FinishWorkOrder(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID perintah kerja tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")

	var req CompleteWorkOrderRequest
	_ = c.ShouldBindJSON(&req)

	wo, err := h.service.FinishWorkOrder(uint(id), &req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, wo)
}

// ----------------------------------------------------------------------------
// GANTT & AI
// ----------------------------------------------------------------------------

func (h *Handler) GetGanttCapacity(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	daysStr := c.DefaultQuery("days", "14")
	days, _ := strconv.Atoi(daysStr)

	res, err := h.service.GetGanttCapacitySchedule(companyID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handler) AnalyzeCapacityAI(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	report, err := h.service.AnalyzeMRPCapacityWithAI(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}
