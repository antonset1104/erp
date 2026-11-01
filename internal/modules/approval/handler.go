package approval

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

// Categories
func (h *Handler) ListCategories(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	cats, err := h.svc.ListCategories(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cats)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var cat ApprovalCategory
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat.CompanyID = companyID
	cat.CreateUID = userID
	cat.WriteUID = userID

	if err := h.svc.CreateCategory(&cat); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cat)
}

// Requests
func (h *Handler) ListRequests(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	categoryID, _ := strconv.ParseUint(c.Query("category_id"), 10, 64)
	stage := c.Query("stage")
	userID, _ := strconv.ParseUint(c.Query("user_id"), 10, 64)

	reqs, err := h.svc.ListRequests(companyID, uint(categoryID), stage, uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, reqs)
}

func (h *Handler) CreateRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var input CreateRequestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.CompanyID = companyID

	req, err := h.svc.CreateRequest(input, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

func (h *Handler) GetRequest(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	req, err := h.svc.GetRequestByID(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permohonan persetujuan tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) SubmitRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	req, err := h.svc.ActionSubmit(uint(id), userID, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

type ApproveBody struct {
	Comment string `json:"comment"`
}

func (h *Handler) ApproveRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var body ApproveBody
	_ = c.ShouldBindJSON(&body)

	req, err := h.svc.ActionApprove(uint(id), userID, companyID, body.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

type RefuseBody struct {
	RefusalReason string `json:"refusal_reason" binding:"required"`
}

func (h *Handler) RefuseRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	var body RefuseBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Alasan penolakan wajib diisi"})
		return
	}

	req, err := h.svc.ActionRefuse(uint(id), userID, companyID, body.RefusalReason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) CancelRequest(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	req, err := h.svc.ActionCancel(uint(id), userID, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

func (h *Handler) GetStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stats, err := h.svc.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
