package tax_ebupot

import (
	"fmt"
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
	group := r.Group("/tax-ebupot")
	if authMiddleware != nil {
		group.Use(authMiddleware)
	}

	{
		// Stats
		group.GET("/stats", h.GetStats)

		// Periods
		group.GET("/periods", h.ListPeriods)
		group.POST("/periods", h.CreatePeriod)
		group.GET("/periods/:id", h.GetPeriod)
		group.POST("/periods/:id/lock", h.LockPeriod)

		// Withholding Slips
		group.GET("/periods/:id/slips", h.ListWithholdingSlips)
		group.POST("/periods/:id/slips", h.CreateWithholdingSlip)
		group.POST("/periods/:id/import-payroll", h.ImportFromPayroll)
		group.GET("/slips", h.ListWithholdingSlips)
		group.POST("/slips", h.CreateWithholdingSlip)
		group.POST("/import-payroll", h.ImportFromPayroll)

		// Exports & Compliance
		group.GET("/periods/:id/export-csv", h.ExportDJPCSV)
		group.GET("/periods/:id/export-xml", h.ExportDJPXML)
		group.GET("/periods/:id/reconciliation", h.GetReconciliation)
		group.GET("/periods/:id/ai-compliance", h.AnalyzeAICompliance)
		group.POST("/periods/:id/ai-compliance", h.AnalyzeAICompliance)
		group.GET("/ai-compliance", h.AnalyzeAICompliance)
		group.GET("/ai-copilot", h.AnalyzeAICompliance)
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

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) ListPeriods(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	periods, err := h.service.ListPeriods(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": periods})
}

func (h *Handler) CreatePeriod(c *gin.Context) {
	var req CreatePeriodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	period, err := h.service.CreatePeriod(&req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": period})
}

func (h *Handler) GetPeriod(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID periode tidak valid"})
		return
	}

	period, err := h.service.GetPeriod(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": period})
}

func (h *Handler) LockPeriod(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID periode tidak valid"})
		return
	}

	var req struct {
		NTPN string `json:"ntpn" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "NTPN wajib diisi"})
		return
	}

	period, err := h.service.LockPeriod(uint(id), req.NTPN)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": period, "message": "Periode e-Bupot berhasil dikunci"})
}

func (h *Handler) ListWithholdingSlips(c *gin.Context) {
	var periodID uint
	if idStr := c.Param("id"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			periodID = uint(id)
		}
	} else if pStr := c.Query("period_id"); pStr != "" {
		if id, err := strconv.ParseUint(pStr, 10, 32); err == nil {
			periodID = uint(id)
		}
	}

	if periodID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID periode tidak valid atau belum dipilih"})
		return
	}

	slipType := c.Query("type")
	if slipType == "" || slipType == "all" {
		slipType = c.Query("slip_type")
	}
	if slipType == "all" {
		slipType = ""
	}

	slips, err := h.service.ListWithholdingSlips(periodID, slipType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": slips})
}

func (h *Handler) CreateWithholdingSlip(c *gin.Context) {
	var req CreateWithholdingSlipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if idStr := c.Param("id"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			req.PeriodID = uint(id)
		}
	}

	if req.PeriodID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID periode wajib diisi"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	slip, err := h.service.CreateWithholdingSlip(&req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": slip})
}

func (h *Handler) ImportFromPayroll(c *gin.Context) {
	var req ImportFromPayrollRequest
	_ = c.ShouldBindJSON(&req)

	if idStr := c.Param("id"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			req.PeriodID = uint(id)
		}
	}

	if req.PeriodID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID periode wajib ditentukan"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	slips, err := h.service.ImportFromPayroll(&req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    slips,
		"count":   len(slips),
		"message": fmt.Sprintf("Berhasil mengimpor %d bukti potong PPh 21 dari penggajian", len(slips)),
	})
}

func (h *Handler) ExportDJPCSV(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID periode tidak valid"})
		return
	}

	csvContent, err := h.service.GenerateDJPCSV(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fileName := fmt.Sprintf("DJP_EBUPOT21_PERIOD_%d.csv", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.String(http.StatusOK, csvContent)
}

func (h *Handler) ExportDJPXML(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID periode tidak valid"})
		return
	}

	xmlContent, err := h.service.GenerateDJPXML(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	fileName := fmt.Sprintf("DJP_EBUPOT21_PERIOD_%d.xml", id)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
	c.Header("Content-Type", "application/xml; charset=utf-8")
	c.String(http.StatusOK, xmlContent)
}

func (h *Handler) GetReconciliation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID periode tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	report, err := h.service.ReconcileWithPayrollAndGL(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

func (h *Handler) AnalyzeAICompliance(c *gin.Context) {
	var periodID uint
	if idStr := c.Param("id"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			periodID = uint(id)
		}
	} else if pStr := c.Query("period_id"); pStr != "" {
		if id, err := strconv.ParseUint(pStr, 10, 32); err == nil {
			periodID = uint(id)
		}
	} else if pStr := c.Query("id"); pStr != "" {
		if id, err := strconv.ParseUint(pStr, 10, 32); err == nil {
			periodID = uint(id)
		}
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	if periodID == 0 {
		periods, err := h.service.ListPeriods(companyID)
		if err == nil && len(periods) > 0 {
			periodID = periods[0].ID
		}
	}

	if periodID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Belum ada masa pajak untuk dianalisis"})
		return
	}

	report, err := h.service.AnalyzeTaxComplianceWithAI(periodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}
