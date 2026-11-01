package attendance

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

// CheckIn handles POST /api/v1/attendance/check-in
func (h *Handler) CheckIn(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var req CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload check-in tidak valid: " + err.Error()})
		return
	}

	// Extract Cloudflare Edge Telemetry
	ip := c.GetHeader("CF-Connecting-IP")
	if ip == "" {
		ip = c.ClientIP()
	}
	country := c.GetHeader("CF-IPCountry")
	if country == "" {
		country = "ID"
	}
	cfRay := c.GetHeader("CF-Ray")

	if req.Device == "" {
		req.Device = c.GetHeader("User-Agent")
		if len(req.Device) > 120 {
			req.Device = req.Device[:120]
		}
	}

	att, err := h.svc.CheckIn(req, companyID, userID, ip, country, cfRay)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Check-in berhasil dicatat",
		"attendance": att,
	})
}

// CheckOut handles POST /api/v1/attendance/check-out
func (h *Handler) CheckOut(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var req CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload check-out tidak valid: " + err.Error()})
		return
	}

	att, err := h.svc.CheckOut(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "Check-out berhasil dicatat",
		"attendance": att,
	})
}

// GetActiveAttendance handles GET /api/v1/attendance/active/:employee_id
func (h *Handler) GetActiveAttendance(c *gin.Context) {
	empIDStr := c.Param("employee_id")
	empID, err := strconv.ParseUint(empIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID karyawan tidak valid"})
		return
	}

	att, err := h.svc.Repo().GetActiveAttendanceByEmployee(uint(empID))
	if err != nil || att == nil {
		c.JSON(http.StatusOK, gin.H{
			"active":     false,
			"attendance": nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"active":     true,
		"attendance": att,
	})
}

// ListAttendances handles GET /api/v1/attendance
func (h *Handler) ListAttendances(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	var filter AttendanceFilter
	if empStr := c.Query("employee_id"); empStr != "" {
		if id, err := strconv.ParseUint(empStr, 10, 64); err == nil {
			filter.EmployeeID = uint(id)
		}
	}
	if deptStr := c.Query("department_id"); deptStr != "" {
		if id, err := strconv.ParseUint(deptStr, 10, 64); err == nil {
			filter.DepartmentID = uint(id)
		}
	}
	filter.DateFrom = c.Query("date_from")
	filter.DateTo = c.Query("date_to")
	filter.Status = c.Query("status")
	filter.AnomalyOnly = c.Query("anomaly_only") == "true"
	filter.Search = c.Query("search")

	if limStr := c.Query("limit"); limStr != "" {
		if l, err := strconv.Atoi(limStr); err == nil {
			filter.Limit = l
		}
	}
	if offStr := c.Query("offset"); offStr != "" {
		if o, err := strconv.Atoi(offStr); err == nil {
			filter.Offset = o
		}
	}

	list, total, err := h.svc.ListAttendances(companyID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"data":  list,
	})
}

// GetAttendance handles GET /api/v1/attendance/:id
func (h *Handler) GetAttendance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	att, err := h.svc.GetAttendance(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data presensi tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, att)
}

// DeleteAttendance handles DELETE /api/v1/attendance/:id
func (h *Handler) DeleteAttendance(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	if err := h.svc.DeleteAttendance(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Presensi berhasil dihapus"})
}

// ListShifts handles GET /api/v1/attendance/shifts
func (h *Handler) ListShifts(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	shifts, err := h.svc.ListShifts(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, shifts)
}

// CreateShift handles POST /api/v1/attendance/shifts
func (h *Handler) CreateShift(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var shift Shift
	if err := c.ShouldBindJSON(&shift); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	shift.CompanyID = companyID
	shift.CreateUID = userID
	shift.WriteUID = userID

	if err := h.svc.CreateShift(&shift); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, shift)
}

// GetStats handles GET /api/v1/attendance/stats
func (h *Handler) GetStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stats, err := h.svc.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// RunAIAnomalyCheck handles POST /api/v1/attendance/ai-anomaly-check
func (h *Handler) RunAIAnomalyCheck(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	result, err := h.svc.RunAIAnomalyCheck(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// SyncPayroll handles POST /api/v1/attendance/sync-payroll
func (h *Handler) SyncPayroll(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req SyncPayrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payslip, err := h.svc.SyncToPayslip(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sinkronisasi presensi ke slip gaji berhasil",
		"payslip": payslip,
	})
}
