package barcode

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

// Scan processes a barcode scan request
func (h *Handler) Scan(c *gin.Context) {
	var req BarcodeScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format permintaan scan barcode tidak valid: " + err.Error()})
		return
	}

	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")

	res, err := h.service.Scan(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses scan barcode: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    res,
		"message": res.Message,
	})
}

// Lookup handles quick GET barcode search
func (h *Handler) Lookup(c *gin.Context) {
	code := c.Param("code")
	companyID := c.GetUint("company_id")
	userID := c.GetUint("user_id")

	res, err := h.service.Scan(BarcodeScanRequest{Barcode: code, Context: "general"}, companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencari barcode: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    res,
		"message": res.Message,
	})
}

// GenerateLabel returns printable vector SVG barcode label
func (h *Handler) GenerateLabel(c *gin.Context) {
	entityType := c.Param("type")
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	svg, err := h.service.GenerateLabelSVG(entityType, uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	// If client specifically asks for JSON
	format := c.Query("format")
	if format == "json" {
		c.JSON(http.StatusOK, gin.H{
			"data":    gin.H{"svg": svg, "type": entityType, "id": id},
			"message": "Berhasil generate barcode label",
		})
		return
	}

	c.Header("Content-Type", "image/svg+xml")
	c.Header("Cache-Control", "no-cache")
	c.String(http.StatusOK, svg)
}

// ListLots lists lot and serial numbers
func (h *Handler) ListLots(c *gin.Context) {
	productID, _ := strconv.ParseUint(c.Query("product_id"), 10, 32)
	search := c.Query("search")
	companyID := c.GetUint("company_id")

	lots, err := h.service.ListLots(uint(productID), companyID, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar lot/serial number: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    lots,
		"message": "Daftar lot/serial number berhasil diambil",
	})
}

// CreateLot creates a new lot/serial number
func (h *Handler) CreateLot(c *gin.Context) {
	var req CreateLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data permohonan lot tidak valid: " + err.Error()})
		return
	}

	companyID := c.GetUint("company_id")
	lot, err := h.service.CreateLot(req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat lot/serial number: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    lot,
		"message": "Nomor Lot/Serial berhasil didaftarkan",
	})
}

// GetLot retrieves lot by ID
func (h *Handler) GetLot(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID lot tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	lot, err := h.service.GetLot(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data lot/serial number tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    lot,
		"message": "Detail lot/serial number berhasil diambil",
	})
}

// GetStats returns barcode & lot KPI metrics
func (h *Handler) GetStats(c *gin.Context) {
	companyID := c.GetUint("company_id")
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat statistik barcode: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    stats,
		"message": "Statistik barcode berhasil diambil",
	})
}
