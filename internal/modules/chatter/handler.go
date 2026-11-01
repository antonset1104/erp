package chatter

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

func (h *Handler) ListMessages(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	modelName := c.Param("model")
	resID, err := strconv.ParseUint(c.Param("res_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	messages, err := h.svc.ListMessages(modelName, uint(resID), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, messages)
}

type PostMessageRequest struct {
	Body string `json:"body" binding:"required"`
}

func (h *Handler) PostMessage(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	modelName := c.Param("model")
	resID, err := strconv.ParseUint(c.Param("res_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid resource ID"})
		return
	}

	var req PostMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	authorName := "User"
	if email, exists := c.Get("user_email"); exists {
		authorName = email.(string)
	}

	msg := MailMessage{
		ModelName:   modelName,
		ResID:       uint(resID),
		AuthorID:    userID,
		AuthorName:  authorName,
		Body:        req.Body,
		MessageType: "comment",
	}
	msg.CompanyID = companyID
	msg.CreateUID = userID
	msg.WriteUID = userID

	if err := h.svc.PostMessage(&msg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, msg)
}
