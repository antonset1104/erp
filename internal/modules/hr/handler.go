package hr

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

func (h *Handler) ListDepartments(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	depts, err := h.svc.ListDepartments(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, depts)
}

func (h *Handler) ListJobs(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	jobs, err := h.svc.ListJobs(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, jobs)
}

func (h *Handler) ListEmployees(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	emps, err := h.svc.ListEmployees(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, emps)
}

func (h *Handler) CreateEmployee(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var emp Employee
	if err := c.ShouldBindJSON(&emp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	emp.CompanyID = companyID
	emp.CreateUID = userID
	emp.WriteUID = userID
	if err := h.svc.CreateEmployee(&emp); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, emp)
}

func (h *Handler) ListPayslips(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	period := c.Query("period")
	slips, err := h.svc.ListPayslips(companyID, period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, slips)
}

type GeneratePayslipRequest struct {
	EmployeeID uint    `json:"employee_id" binding:"required"`
	Period     string  `json:"period" binding:"required"`
	Allowances float64 `json:"allowances"`
	Deductions float64 `json:"deductions"`
	Note       string  `json:"note"`
}

func (h *Handler) GeneratePayslip(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req GeneratePayslipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	slip, err := h.svc.GeneratePayslip(req.EmployeeID, req.Period, req.Allowances, req.Deductions, req.Note, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, slip)
}

func (h *Handler) ConfirmPayslip(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payslip ID"})
		return
	}
	slip, err := h.svc.ConfirmPayslip(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Payslip confirmed", "slip": slip})
}

func (h *Handler) PayPayslip(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payslip ID"})
		return
	}
	slip, err := h.svc.PayPayslip(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Payslip marked as paid", "slip": slip})
}
