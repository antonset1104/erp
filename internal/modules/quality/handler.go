package quality

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

// ----------------- Quality Points -----------------

func (h *Handler) ListPoints(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	opType := c.Query("operation_type")

	points, err := h.svc.ListPoints(companyID, opType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, points)
}

func (h *Handler) GetPoint(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid point id"})
		return
	}

	point, err := h.svc.GetPoint(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, point)
}

func (h *Handler) CreatePoint(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var qp QualityPoint
	if err := c.ShouldBindJSON(&qp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	qp.CompanyID = companyID
	qp.CreateUID = userID
	qp.WriteUID = userID
	qp.Active = true

	if err := h.svc.CreatePoint(&qp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, qp)
}

func (h *Handler) UpdatePoint(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid point id"})
		return
	}

	existing, err := h.svc.GetPoint(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var req QualityPoint
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing.Title = req.Title
	if req.OperationType != "" {
		existing.OperationType = req.OperationType
	}
	if req.TestType != "" {
		existing.TestType = req.TestType
	}
	existing.ProductID = req.ProductID
	existing.ProductCategoryID = req.ProductCategoryID
	existing.Norm = req.Norm
	existing.ToleranceMin = req.ToleranceMin
	existing.ToleranceMax = req.ToleranceMax
	existing.NormUnit = req.NormUnit
	existing.Instructions = req.Instructions
	existing.Active = req.Active
	existing.WriteUID = userID

	if err := h.svc.UpdatePoint(existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, existing)
}

func (h *Handler) DeletePoint(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid point id"})
		return
	}

	if err := h.svc.DeletePoint(uint(id), companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "quality point deleted"})
}

// ----------------- Quality Checks -----------------

func (h *Handler) ListChecks(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	pickingIDStr := c.Query("picking_id")
	prodIDStr := c.Query("production_id")
	state := c.Query("state")

	var pickingID *uint
	if pickingIDStr != "" {
		pID, _ := strconv.Atoi(pickingIDStr)
		uID := uint(pID)
		pickingID = &uID
	}

	var prodID *uint
	if prodIDStr != "" {
		pID, _ := strconv.Atoi(prodIDStr)
		uID := uint(pID)
		prodID = &uID
	}

	checks, err := h.svc.ListChecks(companyID, pickingID, prodID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, checks)
}

func (h *Handler) GetCheck(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check id"})
		return
	}

	check, err := h.svc.GetCheck(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, check)
}

func (h *Handler) CreateCheck(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var qc QualityCheck
	if err := c.ShouldBindJSON(&qc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	qc.CompanyID = companyID
	qc.CreateUID = userID
	qc.WriteUID = userID

	if err := h.svc.CreateCheck(&qc, companyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, qc)
}

type PassCheckRequest struct {
	Measure *float64 `json:"measure"`
	Notes   string   `json:"notes"`
}

func (h *Handler) ActionPassCheck(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check id"})
		return
	}

	var req PassCheckRequest
	_ = c.ShouldBindJSON(&req)

	check, err := h.svc.ActionPassCheck(uint(id), companyID, userID, req.Measure, req.Notes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, check)
}

type FailCheckRequest struct {
	Measure   *float64 `json:"measure"`
	Reason    string   `json:"reason"`
	AutoAlert bool     `json:"auto_alert"`
}

func (h *Handler) ActionFailCheck(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check id"})
		return
	}

	var req FailCheckRequest
	_ = c.ShouldBindJSON(&req)

	check, alert, err := h.svc.ActionFailCheck(uint(id), companyID, userID, req.Measure, req.Reason, req.AutoAlert)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"check": check,
		"alert": alert,
	})
}

func (h *Handler) GenerateChecksForPicking(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	pickingID, err := strconv.Atoi(c.Param("picking_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid picking id"})
		return
	}

	generated, err := h.svc.GenerateChecksForPicking(uint(pickingID), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"generated_count": len(generated),
		"checks":          generated,
	})
}

func (h *Handler) GenerateChecksForProduction(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	prodID, err := strconv.Atoi(c.Param("production_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid production id"})
		return
	}

	generated, err := h.svc.GenerateChecksForProduction(uint(prodID), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"generated_count": len(generated),
		"checks":          generated,
	})
}

// ----------------- Quality Alerts -----------------

func (h *Handler) ListAlerts(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stage := c.Query("stage")
	priority := c.Query("priority")

	alerts, err := h.svc.ListAlerts(companyID, stage, priority)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

func (h *Handler) GetAlert(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}

	alert, err := h.svc.GetAlert(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alert)
}

func (h *Handler) CreateAlert(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var qa QualityAlert
	if err := c.ShouldBindJSON(&qa); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	qa.CompanyID = companyID
	qa.CreateUID = userID
	qa.WriteUID = userID
	if qa.UserID == nil {
		qa.UserID = &userID
	}

	if err := h.svc.CreateAlert(&qa, companyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, qa)
}

type UpdateAlertStageRequest struct {
	Stage            string `json:"stage" binding:"required"`
	RootCause        string `json:"root_cause"`
	PreventiveAction string `json:"preventive_action"`
	ActionType       string `json:"action_type"`
}

func (h *Handler) UpdateAlertStage(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}

	var req UpdateAlertStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	alert, err := h.svc.UpdateAlertStage(uint(id), companyID, req.Stage, req.RootCause, req.PreventiveAction, req.ActionType, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, alert)
}

// ----------------- Stats & Dashboard -----------------

func (h *Handler) GetQualityStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stats, err := h.svc.GetQualityStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
