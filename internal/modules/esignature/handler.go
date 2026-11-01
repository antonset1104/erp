package esignature

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Protected ERP Routes (/api/v1/esignature)
	group := r.Group("/esignature")
	if authMiddleware != nil {
		group.Use(authMiddleware)
	}
	{
		group.GET("/stats", h.GetStats)
		group.GET("/requests", h.ListRequests)
		group.POST("/requests", h.CreateRequest)
		group.GET("/requests/:id", h.GetRequestByID)
		group.POST("/requests/:id/cancel", h.CancelRequest)
		group.GET("/requests/:id/ai-audit", h.AnalyzeAIAudit)
		group.GET("/documents/:type/:id", h.GetRequestsForDocument)
	}

	// Public / External Signing & Verification Portal (no JWT required)
	publicPortal := r.Group("/public/esign")
	{
		publicPortal.GET("/:token", h.VerifyByToken)
		publicPortal.GET("/verify/:token", h.VerifyByToken)
		publicPortal.POST("/:token/sign", h.SubmitSignature)
		publicPortal.POST("/:token/reject", h.RejectSignature)
	}
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) ListRequests(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	status := c.Query("status")
	docType := c.Query("doc_type")
	if docType == "" {
		docType = c.Query("document_type")
	}

	requests, err := h.service.ListSignRequests(companyID, status, docType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": requests})
}

func (h *Handler) CreateRequest(c *gin.Context) {
	var req CreateSignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	signReq, err := h.service.CreateSignRequest(&req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": signReq})
}

func (h *Handler) GetRequestByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID permohonan tidak valid"})
		return
	}

	signReq, err := h.service.GetSignRequestByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Permohonan tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": signReq})
}

func (h *Handler) CancelRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID permohonan tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	if err := h.service.CancelSignRequest(uint(id), companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permohonan tanda tangan berhasil dibatalkan"})
}

func (h *Handler) GetRequestsForDocument(c *gin.Context) {
	docType := c.Param("type")
	docID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	requests, err := h.service.GetSignRequestsForDocument(companyID, docType, uint(docID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": requests})
}

func (h *Handler) AnalyzeAIAudit(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID permohonan tidak valid"})
		return
	}

	report, err := h.service.AnalyzeSignatureComplianceWithAI(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// Public Portal Handlers

func (h *Handler) VerifyByToken(c *gin.Context) {
	token := c.Param("token")
	signReq, err := h.service.GetSignRequestByToken(token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tautan tanda tangan tidak ditemukan atau tidak valid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": signReq})
}

func (h *Handler) SubmitSignature(c *gin.Context) {
	token := c.Param("token")
	var req ExecuteSignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	signedReq, err := h.service.ExecuteSigning(token, &req, clientIP, userAgent)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    signedReq,
		"message": fmt.Sprintf("Dokumen %s berhasil ditandatangani secara digital", signedReq.DocumentNumber),
	})
}

func (h *Handler) RejectSignature(c *gin.Context) {
	token := c.Param("token")
	var req RejectSignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	rejectedReq, err := h.service.RejectSignRequest(token, req.Reason, clientIP, userAgent)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    rejectedReq,
		"message": "Penolakan tanda tangan berhasil dicatat",
	})
}
