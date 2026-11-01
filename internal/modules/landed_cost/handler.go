package landed_cost

import (
	"net/http"
	"strconv"

	"erp-golang/internal/modules/stock"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service  *Service
	stockSvc *stock.Service
}

func NewHandler(service *Service, stockSvc *stock.Service) *Handler {
	return &Handler{
		service:  service,
		stockSvc: stockSvc,
	}
}

func (h *Handler) getCompanyID(c *gin.Context) uint {
	if cid, exists := c.Get("company_id"); exists {
		if id, ok := cid.(uint); ok {
			return id
		}
	}
	return 1
}

func (h *Handler) getUserID(c *gin.Context) uint {
	if uid, exists := c.Get("user_id"); exists {
		if id, ok := uid.(uint); ok {
			return id
		}
	}
	return 1
}

// CreateLandedCost handles POST /api/v1/landed-costs
func (h *Handler) CreateLandedCost(c *gin.Context) {
	var req CreateLandedCostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)

	cost, err := h.service.CreateLandedCost(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    cost,
		"message": "Dokumen Landed Cost berhasil dibuat",
	})
}

// ListLandedCosts handles GET /api/v1/landed-costs
func (h *Handler) ListLandedCosts(c *gin.Context) {
	companyID := h.getCompanyID(c)
	state := c.Query("state")

	costs, err := h.service.ListLandedCosts(companyID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": costs})
}

// GetLandedCost handles GET /api/v1/landed-costs/:id
func (h *Handler) GetLandedCost(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID landed cost tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	cost, err := h.service.GetLandedCost(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": cost})
}

// ComputeValuationLines handles POST /api/v1/landed-costs/:id/compute
func (h *Handler) ComputeValuationLines(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID landed cost tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)

	cost, err := h.service.ComputeValuationLines(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    cost,
		"message": "Alokasi biaya landed cost berhasil dihitung",
	})
}

// ActionValidate handles POST /api/v1/landed-costs/:id/validate
func (h *Handler) ActionValidate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID landed cost tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)

	cost, err := h.service.ActionValidate(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    cost,
		"message": "Landed cost berhasil divalidasi dan jurnal penyesuaian nilai persediaan telah diposting",
	})
}

// ActionCancel handles POST /api/v1/landed-costs/:id/cancel
func (h *Handler) ActionCancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID landed cost tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)

	cost, err := h.service.ActionCancel(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    cost,
		"message": "Dokumen landed cost berhasil dibatalkan",
	})
}

// GetStats handles GET /api/v1/landed-costs/stats
func (h *Handler) GetStats(c *gin.Context) {
	companyID := h.getCompanyID(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// AILandedCostCopilot handles GET /api/v1/landed-costs/:id/ai-audit
func (h *Handler) AILandedCostCopilot(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID landed cost tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	report, err := h.service.AILandedCostCopilot(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// ListEligiblePickings handles GET /api/v1/landed-costs/pickings
func (h *Handler) ListEligiblePickings(c *gin.Context) {
	companyID := h.getCompanyID(c)
	if h.stockSvc != nil {
		pickings, err := h.stockSvc.ListPickings(companyID, "incoming")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": pickings})
		return
	}

	var pickings []stock.StockPicking
	err := h.service.db.Where("company_id = ? AND picking_type = ?", companyID, "incoming").
		Order("id DESC").Find(&pickings).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pickings})
}
