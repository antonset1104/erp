package audit_trail

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers all audit trail REST endpoints
func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	auditGroup := r.Group("/audit-trail", authMW)
	{
		// Dashboard Metrics & Stats
		auditGroup.GET("/stats", h.GetStats)

		// Audit Log Entries & CDC
		auditGroup.GET("/logs", h.ListLogs)
		auditGroup.GET("/logs/:id", h.GetLog)
		auditGroup.POST("/record", h.RecordChange)
		auditGroup.POST("/logs/:id/rollback", h.RollbackLog)

		// Audit Rules Configuration
		auditGroup.GET("/rules", h.ListRules)
		auditGroup.POST("/rules", h.CreateRule)
		auditGroup.GET("/rules/:id", h.GetRule)
		auditGroup.PUT("/rules/:id", h.UpdateRule)
		auditGroup.DELETE("/rules/:id", h.DeleteRule)

		// AI Security & Anomaly Audit Copilot
		auditGroup.GET("/anomaly-audit", h.AnalyzeAnomalies)

		// CSV Export for External Regulators / SOX / ISO 27001
		auditGroup.GET("/export", h.ExportCSV)
	}
}

func (h *Handler) GetStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) ListLogs(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	filters := AuditLogFilters{
		CompanyID: companyID,
		ModelName: c.Query("model_name"),
		Action:    c.Query("action"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Search:    c.Query("search"),
	}

	if resIDStr := c.Query("res_id"); resIDStr != "" {
		if id, err := strconv.ParseUint(resIDStr, 10, 32); err == nil {
			filters.ResID = uint(id)
		}
	}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if id, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			filters.UserID = uint(id)
		}
	}
	if suspStr := c.Query("is_suspicious"); suspStr != "" {
		b := suspStr == "true" || suspStr == "1"
		filters.IsSuspicious = &b
	}

	limit := 100
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	filters.Limit = limit

	offset := 0
	if oStr := c.Query("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}
	filters.Offset = offset

	entries, total, err := h.service.ListEntries(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   entries,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handler) GetLog(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID log tidak valid"})
		return
	}

	entry, err := h.service.GetEntryByID(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Log audit tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, entry)
}

func (h *Handler) RecordChange(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	userName := c.GetString("name")
	if userName == "" {
		userName = "System Administrator"
	}
	userEmail := c.GetString("email")

	var req RecordChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IPAddress == "" {
		req.IPAddress = c.ClientIP()
	}
	if req.UserAgent == "" {
		req.UserAgent = c.GetHeader("User-Agent")
	}

	entry, err := h.service.RecordChange(req, userID, userName, userEmail, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if entry == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Perubahan diabaikan berdasarkan aturan audit (rule bypass)"})
		return
	}

	c.JSON(http.StatusCreated, entry)
}

func (h *Handler) RollbackLog(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	userName := c.GetString("name")
	if userName == "" {
		userName = "System Administrator"
	}
	userEmail := c.GetString("email")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID log tidak valid"})
		return
	}

	var req RollbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Alasan rollback (reason) wajib diisi"})
		return
	}

	compEntry, err := h.service.RollbackRecord(uint(id), req.Reason, userID, userName, userEmail, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     fmt.Sprintf("Berhasil melakukan rollback untuk log %s", compEntry.LogCode),
		"rollback_log": compEntry,
	})
}

func (h *Handler) ListRules(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	rules, err := h.service.ListRules(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rules)
}

func (h *Handler) CreateRule(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.service.CreateRule(req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rule)
}

func (h *Handler) GetRule(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID aturan tidak valid"})
		return
	}

	rule, err := h.service.GetRuleByID(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Aturan audit tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func (h *Handler) UpdateRule(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID aturan tidak valid"})
		return
	}

	var req CreateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule, err := h.service.UpdateRule(uint(id), req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rule)
}

func (h *Handler) DeleteRule(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID aturan tidak valid"})
		return
	}

	if err := h.service.DeleteRule(uint(id), companyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Aturan audit berhasil dihapus"})
}

func (h *Handler) AnalyzeAnomalies(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	report, err := h.service.AnalyzeAnomalies(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) ExportCSV(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	filters := AuditLogFilters{
		CompanyID: companyID,
		ModelName: c.Query("model_name"),
		Action:    c.Query("action"),
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
		Search:    c.Query("search"),
	}

	if resIDStr := c.Query("res_id"); resIDStr != "" {
		if id, err := strconv.ParseUint(resIDStr, 10, 32); err == nil {
			filters.ResID = uint(id)
		}
	}
	if suspStr := c.Query("is_suspicious"); suspStr != "" {
		b := suspStr == "true" || suspStr == "1"
		filters.IsSuspicious = &b
	}

	data, err := h.service.ExportCSV(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	filename := fmt.Sprintf("audit_trail_export_%s.csv", time.Now().Format("20060102_150405"))
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv", data)
}
