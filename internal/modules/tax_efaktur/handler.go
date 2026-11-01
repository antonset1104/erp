package tax_efaktur

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
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

// CreateNSFP allocates a new range of NSFP numbers
func (h *Handler) CreateNSFP(c *gin.Context) {
	var req CreateNSFPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	nsfp, err := h.service.CreateNSFP(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": nsfp, "message": "Jatah Nomor Seri Faktur Pajak (NSFP) berhasil didaftarkan"})
}

// ListNSFP retrieves all NSFP ranges
func (h *Handler) ListNSFP(c *gin.Context) {
	companyID := h.getCompanyID(c)
	list, err := h.service.ListNSFP(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GenerateOutTaxInvoice generates a Faktur Pajak Keluaran from a posted invoice
func (h *Handler) GenerateOutTaxInvoice(c *gin.Context) {
	var req GenerateTaxInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	req.TaxInvoiceType = "out_tax"

	inv, err := h.service.GenerateOutTaxInvoiceFromMove(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": inv, "message": "Faktur Pajak Keluaran berhasil diterbitkan"})
}

// RecordInTaxInvoice records an incoming Faktur Pajak Masukan from vendor
func (h *Handler) RecordInTaxInvoice(c *gin.Context) {
	var req GenerateTaxInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	req.TaxInvoiceType = "in_tax"

	inv, err := h.service.RecordInTaxInvoiceFromMove(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": inv, "message": "Faktur Pajak Masukan berhasil dicatat"})
}

// ListTaxInvoices retrieves invoices with filtering
func (h *Handler) ListTaxInvoices(c *gin.Context) {
	companyID := h.getCompanyID(c)
	taxType := c.Query("type")
	period, _ := strconv.Atoi(c.Query("period"))
	year, _ := strconv.Atoi(c.Query("year"))
	state := c.Query("state")

	list, err := h.service.ListTaxInvoices(taxType, period, year, state, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": list})
}

// GetTaxInvoiceByID fetches invoice by id
func (h *Handler) GetTaxInvoiceByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	inv, err := h.service.GetTaxInvoiceByID(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Faktur pajak tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": inv})
}

// UpdateState modifies invoice state (reported, cancelled)
func (h *Handler) UpdateState(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var req struct {
		State string `json:"state"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.State == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state wajib diisi ('reported' atau 'cancelled')"})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	inv, err := h.service.UpdateState(uint(id), req.State, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": inv, "message": "Status faktur pajak berhasil diperbarui"})
}

// Stats returns executive tax KPIs
func (h *Handler) Stats(c *gin.Context) {
	companyID := h.getCompanyID(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats, "message": "Statistik e-Faktur & PPN berhasil diambil"})
}

// SPTPeriodSummary returns SPT Masa 1111 periodic calculations
func (h *Handler) SPTPeriodSummary(c *gin.Context) {
	companyID := h.getCompanyID(c)
	period, _ := strconv.Atoi(c.Query("period"))
	year, _ := strconv.Atoi(c.Query("year"))

	if period == 0 {
		period = int(time.Now().Month())
	}
	if year == 0 {
		year = time.Now().Year()
	}

	summary, err := h.service.GetSPTPeriodSummary(period, year, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary, "message": "Rekapitulasi SPT Masa PPN 1111 berhasil dihitung"})
}

// ExportCSV exports official DJP e-Faktur format CSV
func (h *Handler) ExportCSV(c *gin.Context) {
	companyID := h.getCompanyID(c)
	taxType := c.DefaultQuery("type", "out_tax")
	period, _ := strconv.Atoi(c.Query("period"))
	year, _ := strconv.Atoi(c.Query("year"))

	if period == 0 {
		period = int(time.Now().Month())
	}
	if year == 0 {
		year = time.Now().Year()
	}

	csvContent, err := h.service.ExportEFakturCSV(taxType, period, year, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("eFaktur_%s_Masa_%02d_%d.csv", taxType, period, year)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.String(http.StatusOK, csvContent)
}

// TaxComplianceAIAudit runs Cloudflare Workers AI fiscal compliance audit
func (h *Handler) TaxComplianceAIAudit(c *gin.Context) {
	companyID := h.getCompanyID(c)
	period, _ := strconv.Atoi(c.Query("period"))
	year, _ := strconv.Atoi(c.Query("year"))

	if period == 0 {
		period = int(time.Now().Month())
	}
	if year == 0 {
		year = time.Now().Year()
	}

	report, err := h.service.TaxComplianceAIAudit(period, year, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report, "message": "Audit kepatuhan e-Faktur AI berhasil dijalankan"})
}
