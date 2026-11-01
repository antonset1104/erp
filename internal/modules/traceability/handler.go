package traceability

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) getCompanyID(c *gin.Context) uint {
	if cid, exists := c.Get("company_id"); exists {
		if id, ok := cid.(uint); ok {
			return id
		}
	}
	return 1
}

func (h *Handler) getUserID(c *gin.Context) uint {
	if uid, exists := c.Get("user_id"); exists {
		if id, ok := uid.(uint); ok {
			return id
		}
	}
	return 1
}

// GetStats handles GET /api/v1/traceability/stats
func (h *Handler) GetStats(c *gin.Context) {
	companyID := h.getCompanyID(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// ListTraces handles GET /api/v1/traceability/traces
func (h *Handler) ListTraces(c *gin.Context) {
	companyID := h.getCompanyID(c)
	var lotID uint
	if lidStr := c.Query("lot_id"); lidStr != "" {
		if lid, err := strconv.ParseUint(lidStr, 10, 32); err == nil {
			lotID = uint(lid)
		}
	}
	eventType := c.Query("event_type")
	search := c.Query("search")

	traces, err := h.service.ListTraceRecords(lotID, companyID, eventType, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": traces})
}

// RecordMovement handles POST /api/v1/traceability/record
func (h *Handler) RecordMovement(c *gin.Context) {
	var req RecordLotMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)

	record, err := h.service.RecordMovement(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    record,
		"message": "Pergerakan lot berhasil dicatat dalam rekam jejak",
	})
}

// GetTraceabilityTree handles GET /api/v1/traceability/tree/:lot_id
func (h *Handler) GetTraceabilityTree(c *gin.Context) {
	companyID := h.getCompanyID(c)
	lotID, err := strconv.ParseUint(c.Param("lot_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID lot tidak valid"})
		return
	}

	direction := c.DefaultQuery("direction", "full")

	report, err := h.service.BuildTraceabilityTree(uint(lotID), direction, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// AllocateFEFO handles POST /api/v1/traceability/fefo
func (h *Handler) AllocateFEFO(c *gin.Context) {
	var req FEFOAllocationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)

	result, err := h.service.AllocateFEFO(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    result,
		"message": "Alokasi picking FEFO berhasil dihitung",
	})
}

// AuditExpiryRisk handles GET /api/v1/traceability/expiry-audit
func (h *Handler) AuditExpiryRisk(c *gin.Context) {
	companyID := h.getCompanyID(c)

	report, err := h.service.AuditExpiryRisk(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    report,
		"message": "Audit risiko kadaluwarsa berhasil dianalisis",
	})
}

// InitiateRecall handles POST /api/v1/traceability/recalls
func (h *Handler) InitiateRecall(c *gin.Context) {
	var req ProductRecallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)

	recall, err := h.service.InitiateRecall(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    recall,
		"message": "Penarikan produk (recall) darurat berhasil diterbitkan",
	})
}

// ListRecalls handles GET /api/v1/traceability/recalls
func (h *Handler) ListRecalls(c *gin.Context) {
	companyID := h.getCompanyID(c)
	state := c.Query("state")

	recalls, err := h.service.ListRecalls(companyID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": recalls})
}

// GetRecall handles GET /api/v1/traceability/recalls/:id
func (h *Handler) GetRecall(c *gin.Context) {
	companyID := h.getCompanyID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID recall tidak valid"})
		return
	}

	recall, err := h.service.GetRecall(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": recall})
}

// CompleteRecall handles POST /api/v1/traceability/recalls/:id/complete
func (h *Handler) CompleteRecall(c *gin.Context) {
	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID recall tidak valid"})
		return
	}

	var body struct {
		ResolutionNotes string `json:"resolution_notes"`
	}
	_ = c.ShouldBindJSON(&body)

	recall, err := h.service.CompleteRecall(uint(id), companyID, body.ResolutionNotes, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    recall,
		"message": "Penarikan produk telah diselesaikan",
	})
}
