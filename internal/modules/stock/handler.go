package stock

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

func (h *Handler) ListPickings(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	pickingType := c.Query("type")

	pickings, err := h.svc.ListPickings(companyID, pickingType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pickings)
}

func (h *Handler) GetPicking(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid picking ID"})
		return
	}

	p, err := h.svc.GetPicking(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Picking not found"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *Handler) CreatePicking(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var p StockPicking
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p.CompanyID = companyID
	p.CreateUID = userID
	p.WriteUID = userID

	for i := range p.MoveLines {
		p.MoveLines[i].CompanyID = companyID
		p.MoveLines[i].LocationID = p.LocationID
		p.MoveLines[i].LocationDestID = p.LocationDestID
		p.MoveLines[i].State = "draft"
	}

	if err := h.svc.CreatePicking(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (h *Handler) ValidatePicking(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid picking ID"})
		return
	}

	picking, err := h.svc.ValidatePicking(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Stock picking successfully validated and executed",
		"picking": picking,
	})
}

func (h *Handler) ListQuants(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	quants, err := h.svc.ListQuants(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, quants)
}

func (h *Handler) ListLocations(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	locs, err := h.svc.ListLocations(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, locs)
}

func (h *Handler) ListOrderpoints(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	ops, err := h.svc.ListOrderpoints(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ops)
}

func (h *Handler) CreateOrderpoint(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var op StockOrderpoint
	if err := c.ShouldBindJSON(&op); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	op.CompanyID = companyID
	op.CreateUID = userID
	op.WriteUID = userID

	if err := h.svc.CreateOrderpoint(&op); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, op)
}

func (h *Handler) ListWarehouses(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	warehouses, err := h.svc.ListWarehouses(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, warehouses)
}

func (h *Handler) GetWarehouse(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid warehouse ID"})
		return
	}

	w, err := h.svc.GetWarehouse(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Warehouse not found"})
		return
	}
	c.JSON(http.StatusOK, w)
}

func (h *Handler) CreateWarehouse(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var w StockWarehouse
	if err := c.ShouldBindJSON(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	w.CompanyID = companyID
	w.CreateUID = userID
	w.WriteUID = userID

	if err := h.svc.CreateWarehouse(&w); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, w)
}

type InternalTransferInput struct {
	SourceLocationID uint        `json:"source_location_id" binding:"required"`
	DestLocationID   uint        `json:"dest_location_id" binding:"required"`
	Origin           string      `json:"origin"`
	Note             string      `json:"note"`
	Lines            []StockMove `json:"lines" binding:"required"`
}

func (h *Handler) CreateInternalTransfer(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var input InternalTransferInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	picking, err := h.svc.CreateInternalTransfer(companyID, userID, input.SourceLocationID, input.DestLocationID, input.Lines, input.Origin, input.Note)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, picking)
}

func (h *Handler) PrintPicking(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid picking ID")
		return
	}

	companyID := uint(1)
	if ctxCompanyID, exists := c.Get("company_id"); exists {
		companyID = ctxCompanyID.(uint)
	}

	p, err := h.svc.GetPicking(uint(id), companyID)
	if err != nil {
		c.String(http.StatusNotFound, "Picking not found: %s", err.Error())
		return
	}

	html := RenderPickingHTML(p)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

