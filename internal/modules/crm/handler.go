package crm

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

func (h *Handler) ListLeads(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stage := c.Query("stage")

	leads, err := h.svc.ListLeads(companyID, stage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, leads)
}

func (h *Handler) GetLead(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lead ID"})
		return
	}

	lead, err := h.svc.GetLead(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "CRM lead not found"})
		return
	}
	c.JSON(http.StatusOK, lead)
}

func (h *Handler) CreateLead(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var lead CrmLead
	if err := c.ShouldBindJSON(&lead); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lead.CompanyID = companyID
	lead.CreateUID = userID
	lead.WriteUID = userID
	if lead.UserID == nil || *lead.UserID == 0 {
		lead.UserID = &userID
	}

	if err := h.svc.CreateLead(&lead); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, lead)
}

type UpdateStageRequest struct {
	Stage      string `json:"stage" binding:"required"`
	LostReason string `json:"lost_reason"`
}

func (h *Handler) UpdateStage(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lead ID"})
		return
	}

	var req UpdateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lead, err := h.svc.UpdateStage(uint(id), req.Stage, req.LostReason, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Lead stage updated successfully",
		"lead":    lead,
	})
}

func (h *Handler) ConvertToQuotation(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid lead ID"})
		return
	}

	so, lead, err := h.svc.ConvertToQuotation(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Opportunity successfully converted to Sales Quotation!",
		"sales_order": so,
		"lead":        lead,
	})
}
