package payroll

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

func (h *Handler) ListStructures(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	items, err := h.svc.ListStructures(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) CreateStructure(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	var req CreatePayrollStructureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.CreateStructure(req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}

func (h *Handler) GetStructure(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	item, err := h.svc.GetStructure(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Struktur gaji tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *Handler) CreateSalaryRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID struktur tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	var req CreateSalaryRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rule, err := h.svc.CreateRule(uint(id), req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": rule})
}

func (h *Handler) ListPayslipRuns(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	items, err := h.svc.ListPayslipRuns(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) CreatePayslipRun(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	var req CreatePayslipRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	run, err := h.svc.CreatePayslipRun(req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": run})
}

func (h *Handler) GetPayslipRun(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	run, err := h.svc.GetPayslipRun(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Batch payroll tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": run})
}

func (h *Handler) GenerateRunPayslips(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	run, err := h.svc.GenerateRunPayslips(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": run, "message": "Batch payslip berhasil digenerate"})
}

func (h *Handler) ListPayslips(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	empID, _ := strconv.ParseUint(c.Query("employee_id"), 10, 32)
	runID, _ := strconv.ParseUint(c.Query("payslip_run_id"), 10, 32)
	period := c.Query("period")
	state := c.Query("state")

	items, err := h.svc.ListPayslips(companyID, uint(empID), uint(runID), period, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) GetPayslip(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	slip, err := h.svc.GetPayslip(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Slip gaji tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": slip})
}

func (h *Handler) CreatePayslip(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	var req CreatePayslipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	slip, err := h.svc.CreatePayslip(req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": slip})
}

func (h *Handler) ComputePayslip(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	var req ComputePayslipRequest
	_ = c.ShouldBindJSON(&req) // optional body

	slip, err := h.svc.ComputePayslip(uint(id), req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": slip, "message": "Kalkulasi slip gaji berhasil diperbarui"})
}

func (h *Handler) AuditPayslipWithAI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	result, err := h.svc.AuditWithAI(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) ConfirmPayslip(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	slip, err := h.svc.ConfirmPayslip(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": slip, "message": "Slip gaji disetujui & diposting ke GL Buku Besar"})
}

func (h *Handler) PayPayslip(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	slip, err := h.svc.PayPayslip(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": slip, "message": "Pembayaran gaji berhasil dicatat"})
}

func (h *Handler) CancelPayslip(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	slip, err := h.svc.CancelPayslip(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": slip, "message": "Slip gaji berhasil dibatalkan"})
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	stats, err := h.svc.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}
