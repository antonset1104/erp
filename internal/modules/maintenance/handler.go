package maintenance

import (
	"net/http"
	"strconv"

	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ----------------- Categories -----------------

func (h *Handler) ListCategories(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	cats, err := h.svc.ListCategories(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cats)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var cat MaintenanceEquipmentCategory
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cat.CompanyID = companyID
	cat.CreateUID = userID
	cat.WriteUID = userID

	if err := h.svc.CreateCategory(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cat)
}

// ----------------- Equipments -----------------

func (h *Handler) ListEquipments(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	var categoryID uint
	if catStr := c.Query("category_id"); catStr != "" {
		if id, err := strconv.ParseUint(catStr, 10, 32); err == nil {
			categoryID = uint(id)
		}
	}
	status := c.Query("status")

	list, err := h.svc.ListEquipments(companyID, categoryID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetEquipment(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid equipment ID"})
		return
	}

	eq, err := h.svc.GetEquipment(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Equipment not found"})
		return
	}
	c.JSON(http.StatusOK, eq)
}

func (h *Handler) CreateEquipment(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var eq MaintenanceEquipment
	if err := c.ShouldBindJSON(&eq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eq.CompanyID = companyID
	eq.CreateUID = userID
	eq.WriteUID = userID

	if err := h.svc.CreateEquipment(&eq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, eq)
}

// ----------------- Requests -----------------

func (h *Handler) ListRequests(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	var equipmentID uint
	if eqStr := c.Query("equipment_id"); eqStr != "" {
		if id, err := strconv.ParseUint(eqStr, 10, 32); err == nil {
			equipmentID = uint(id)
		}
	}
	stage := c.Query("stage")
	mType := c.Query("maintenance_type")
	priority := c.Query("priority")

	list, err := h.svc.ListRequests(companyID, equipmentID, stage, mType, priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetRequest(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	req, err := h.svc.GetRequest(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Maintenance request not found"})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) CreateRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req MaintenanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.CompanyID = companyID
	req.CreateUID = userID
	req.WriteUID = userID

	if err := h.svc.CreateRequest(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

type UpdateStageRequest struct {
	Stage string `json:"stage" binding:"required"`
}

func (h *Handler) UpdateRequestStage(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var body UpdateStageRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req, err := h.svc.UpdateRequestStage(uint(id), companyID, userID, body.Stage)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

type AssignTechnicianRequest struct {
	TechnicianID uint `json:"technician_id" binding:"required"`
}

func (h *Handler) AssignRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var body AssignTechnicianRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req, err := h.svc.AssignRequest(uint(id), companyID, userID, body.TechnicianID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

type CompleteRequest struct {
	Resolution    string  `json:"resolution" binding:"required"`
	DurationHours float64 `json:"duration_hours"`
}

func (h *Handler) CompleteRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request ID"})
		return
	}

	var body CompleteRequest
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req, err := h.svc.CompleteRequest(uint(id), companyID, userID, body.Resolution, body.DurationHours)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) AutoSchedulePreventive(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	created, err := h.svc.AutoSchedulePreventive(companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":          "Preventive maintenance checked & scheduled",
		"scheduled_count":  len(created),
		"scheduled_orders": created,
	})
}
