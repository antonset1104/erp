package pos

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

func (h *Handler) ListConfigs(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	configs, err := h.svc.ListConfigs(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, configs)
}

func (h *Handler) ListSessions(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	sessions, err := h.svc.ListSessions(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

type OpenSessionRequest struct {
	ConfigID    uint    `json:"config_id" binding:"required"`
	OpeningCash float64 `json:"opening_cash"`
}

func (h *Handler) OpenSession(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req OpenSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	session, err := h.svc.OpenSession(req.ConfigID, userID, req.OpeningCash, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, session)
}

type CloseSessionRequest struct {
	ClosingCash float64 `json:"closing_cash"`
}

func (h *Handler) CloseSession(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session ID"})
		return
	}
	var req CloseSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	session, err := h.svc.CloseSession(uint(id), req.ClosingCash, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *Handler) ListOrders(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	orders, err := h.svc.ListOrders(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, orders)
}

func (h *Handler) CreateOrder(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var order PosOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	order.CreateUID = userID
	order.WriteUID = userID
	res, err := h.svc.CreateOrder(&order, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}
