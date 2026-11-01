package asset

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

func (h *Handler) ListAssets(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	state := c.Query("state")

	assets, err := h.svc.ListAssets(companyID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, assets)
}

func (h *Handler) GetAsset(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid asset ID"})
		return
	}

	asset, err := h.svc.GetAsset(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Asset not found"})
		return
	}
	c.JSON(http.StatusOK, asset)
}

func (h *Handler) CreateAsset(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var a Asset
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a.CompanyID = companyID
	a.CreateUID = userID
	a.WriteUID = userID

	if err := h.svc.CreateAsset(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if fullAsset, err := h.svc.GetAsset(a.ID, companyID); err == nil {
		c.JSON(http.StatusCreated, fullAsset)
		return
	}

	c.JSON(http.StatusCreated, a)
}

func (h *Handler) ComputeDepreciation(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid asset ID"})
		return
	}

	asset, err := h.svc.ComputeDepreciationBoard(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Depreciation board recomputed successfully",
		"asset":   asset,
	})
}

func (h *Handler) ActionConfirm(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid asset ID"})
		return
	}

	asset, err := h.svc.ActionConfirm(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Asset confirmed and set to Running state",
		"asset":   asset,
	})
}

func (h *Handler) PostDepreciationLine(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	lineID, err := strconv.ParseUint(c.Param("line_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid line ID"})
		return
	}

	line, err := h.svc.PostDepreciationLine(uint(lineID), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Depreciation entry successfully posted to General Ledger!",
		"line":    line,
	})
}

type AutoDepreciateRequest struct {
	AsOfDate string `json:"as_of_date"`
}

func (h *Handler) AutoPostDueDepreciations(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var req AutoDepreciateRequest
	_ = c.ShouldBindJSON(&req)

	count, err := h.svc.AutoPostDueDepreciations(companyID, userID, req.AsOfDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Auto depreciation completed",
		"posted_count": count,
	})
}
