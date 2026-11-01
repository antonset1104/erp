package leave

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

// ----------------- Leave Types -----------------

func (h *Handler) ListLeaveTypes(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	list, err := h.svc.ListLeaveTypes(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) CreateLeaveType(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var lt LeaveType
	if err := c.ShouldBindJSON(&lt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lt.CompanyID = companyID
	lt.CreateUID = userID
	lt.WriteUID = userID

	if err := h.svc.CreateLeaveType(&lt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, lt)
}

// ----------------- Allocations -----------------

func (h *Handler) ListAllocations(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	empIDStr := c.Query("employee_id")
	yearStr := c.Query("year")

	var empID uint
	if empIDStr != "" {
		id, _ := strconv.Atoi(empIDStr)
		empID = uint(id)
	}
	var year int
	if yearStr != "" {
		year, _ = strconv.Atoi(yearStr)
	}

	list, err := h.svc.ListAllocations(companyID, empID, year)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) CreateAllocation(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var alloc LeaveAllocation
	if err := c.ShouldBindJSON(&alloc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.CreateAllocation(&alloc, companyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, alloc)
}

func (h *Handler) ApproveAllocation(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid allocation id"})
		return
	}

	alloc, err := h.svc.ApproveAllocation(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alloc)
}

// ----------------- Leave Requests -----------------

func (h *Handler) ListLeaveRequests(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	empIDStr := c.Query("employee_id")
	state := c.Query("state")
	typeCode := c.Query("type")

	var empID uint
	if empIDStr != "" {
		id, _ := strconv.Atoi(empIDStr)
		empID = uint(id)
	}

	list, err := h.svc.ListLeaveRequests(companyID, empID, state, typeCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetLeaveRequest(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid leave request id"})
		return
	}

	req, err := h.svc.GetLeaveRequest(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "leave request not found"})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) CreateLeaveRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req LeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.CreateLeaveRequest(&req, companyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

func (h *Handler) ApproveLeaveRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid leave request id"})
		return
	}

	req, err := h.svc.ApproveLeaveRequest(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) RefuseLeaveRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid leave request id"})
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&body)

	req, err := h.svc.RefuseLeaveRequest(uint(id), companyID, userID, body.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) GetLeaveSummary(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	empIDStr := c.Query("employee_id")
	var empID uint
	if empIDStr != "" {
		id, _ := strconv.Atoi(empIDStr)
		empID = uint(id)
	}

	summary, err := h.svc.GetLeaveSummary(companyID, empID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}
