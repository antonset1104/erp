package base

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

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := h.svc.Authenticate(req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":         user.ID,
			"name":       user.Name,
			"email":      user.Email,
			"role":       user.Role,
			"company_id": user.CompanyID,
		},
	})
}

func (h *Handler) Me(c *gin.Context) {
	userID, companyID, role := middleware.GetUserContext(c)
	email, _ := c.Get("email")
	name, _ := c.Get("name")

	c.JSON(http.StatusOK, gin.H{
		"id":         userID,
		"email":      email,
		"name":       name,
		"role":       role,
		"company_id": companyID,
	})
}

func (h *Handler) ListPartners(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	partnerType := c.Query("type") // "customer", "supplier", or all

	partners, err := h.svc.ListPartners(companyID, partnerType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, partners)
}

func (h *Handler) CreatePartner(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var p Partner
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p.CompanyID = companyID
	p.CreateUID = userID
	p.WriteUID = userID

	if err := h.svc.CreatePartner(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (h *Handler) GetPartner(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid partner ID"})
		return
	}

	p, err := h.svc.GetPartner(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Partner not found"})
		return
	}

	c.JSON(http.StatusOK, p)
}
