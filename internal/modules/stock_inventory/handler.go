package stock_inventory

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListInventories(c *gin.Context) {
	companyID := c.GetUint("company_id")
	state := c.Query("state")
	search := c.Query("search")

	var warehouseID, locationID uint
	if wStr := c.Query("warehouse_id"); wStr != "" {
		if val, err := strconv.ParseUint(wStr, 10, 32); err == nil {
			warehouseID = uint(val)
		}
	}
	if lStr := c.Query("location_id"); lStr != "" {
		if val, err := strconv.ParseUint(lStr, 10, 32); err == nil {
			locationID = uint(val)
		}
	}

	list, err := h.svc.ListInventories(companyID, state, warehouseID, locationID, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetInventory(c *gin.Context) {
	companyID := c.GetUint("company_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	inv, err := h.svc.GetInventory(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *Handler) CreateInventory(c *gin.Context) {
	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")

	var req CreateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inv, err := h.svc.CreateInventory(&req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, inv)
}

func (h *Handler) StartInventory(c *gin.Context) {
	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	inv, err := h.svc.StartInventory(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *Handler) UpdateLine(c *gin.Context) {
	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")
	invID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID opname tidak valid"})
		return
	}
	lineID, err := strconv.ParseUint(c.Param("line_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID baris tidak valid"})
		return
	}

	var req UpdateInventoryLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inv, err := h.svc.UpdateLine(uint(invID), uint(lineID), &req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *Handler) AddLine(c *gin.Context) {
	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")
	invID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID opname tidak valid"})
		return
	}

	var req AddInventoryLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inv, err := h.svc.AddLine(uint(invID), &req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *Handler) AuditWithAI(c *gin.Context) {
	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	result, err := h.svc.AuditWithAI(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ValidateInventory(c *gin.Context) {
	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	inv, err := h.svc.ValidateInventory(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *Handler) CancelInventory(c *gin.Context) {
	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	inv, err := h.svc.CancelInventory(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, inv)
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := c.GetUint("company_id")
	stats, err := h.svc.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
