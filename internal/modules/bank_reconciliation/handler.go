package bank_reconciliation

import (
	"fmt"
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

// CreateStatement creates a new statement header
func (h *Handler) CreateStatement(c *gin.Context) {
	var req CreateStatementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	stmt, err := h.service.CreateStatement(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": stmt, "message": "Rekening koran berhasil dibuat"})
}

// ListStatements lists all statements
func (h *Handler) ListStatements(c *gin.Context) {
	companyID := h.getCompanyID(c)
	state := c.Query("state")
	journalID, _ := strconv.ParseUint(c.Query("journal_id"), 10, 64)

	stmts, err := h.service.ListStatements(companyID, state, uint(journalID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stmts})
}

// GetStatement retrieves a statement by ID
func (h *Handler) GetStatement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening koran tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	stmt, err := h.service.GetStatement(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stmt})
}

// ConfirmStatement closes and locks the statement
func (h *Handler) ConfirmStatement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening koran tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	stmt, err := h.service.ConfirmStatement(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stmt, "message": "Rekening koran berhasil dikonfirmasi dan ditutup"})
}

// DeleteStatement removes a draft statement
func (h *Handler) DeleteStatement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening koran tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	if err := h.service.repo.DeleteStatement(uint(id), companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rekening koran berhasil dihapus"})
}

// AddStatementLine adds a manual transaction line
func (h *Handler) AddStatementLine(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening koran tidak valid"})
		return
	}

	var req CreateStatementLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	line, err := h.service.AddStatementLine(uint(id), req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": line, "message": "Baris mutasi berhasil ditambahkan"})
}

// DeleteStatementLine deletes a transaction line
func (h *Handler) DeleteStatementLine(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID baris mutasi tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	if err := h.service.DeleteStatementLine(uint(id), companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Baris mutasi berhasil dihapus"})
}

// ParseAndImportCSV imports statement lines from CSV
func (h *Handler) ParseAndImportCSV(c *gin.Context) {
	var req ImportStatementCSVRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	stmt, count, err := h.service.ParseAndImportCSV(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":        stmt,
		"lines_count": count,
		"message":     fmt.Sprintf("Berhasil mengimpor %d transaksi mutasi bank", count),
	})
}

// GetAutoMatchSuggestions computes smart matching suggestions
func (h *Handler) GetAutoMatchSuggestions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening koran tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	suggestions, err := h.service.GetAutoMatchSuggestions(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": suggestions})
}

// ReconcileLine reconciles a line
func (h *Handler) ReconcileLine(c *gin.Context) {
	var req ReconcileLineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	if err := h.service.ReconcileLine(req, companyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transaksi berhasil direkonsiliasi"})
}

// BatchReconcile reconciles multiple items
func (h *Handler) BatchReconcile(c *gin.Context) {
	var req BatchReconcileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	count, err := h.service.BatchReconcile(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reconciled_count": count,
		"message":          fmt.Sprintf("Berhasil merekonsiliasi %d transaksi", count),
	})
}

// AutoReconcileHighConfidence auto-reconciles high confidence suggestions
func (h *Handler) AutoReconcileHighConfidence(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening koran tidak valid"})
		return
	}

	minConf, _ := strconv.ParseFloat(c.DefaultQuery("min_confidence", "0.90"), 64)

	companyID := h.getCompanyID(c)
	userID := h.getUserID(c)
	count, err := h.service.AutoReconcileHighConfidence(uint(id), minConf, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reconciled_count": count,
		"message":          fmt.Sprintf("Auto-Reconcile sukses: %d transaksi cocok otomatis telah diproses", count),
	})
}

// AIBankCopilot runs AI audit report on statement
func (h *Handler) AIBankCopilot(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening koran tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	report, err := h.service.AIBankCopilot(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": report})
}

// ExportStatementCSV exports statement lines as CSV
func (h *Handler) ExportStatementCSV(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening koran tidak valid"})
		return
	}

	companyID := h.getCompanyID(c)
	csvData, filename, err := h.service.ExportStatementCSV(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", csvData)
}

// ListRules retrieves reconciliation rules
func (h *Handler) ListRules(c *gin.Context) {
	companyID := h.getCompanyID(c)
	rules, err := h.service.ListReconcileModels(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

// CreateRule creates a new reconciliation rule
func (h *Handler) CreateRule(c *gin.Context) {
	var model AccountReconcileModel
	if err := c.ShouldBindJSON(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := h.getCompanyID(c)
	model.CompanyID = companyID
	if err := h.service.CreateReconcileModel(&model); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": model, "message": "Aturan rekonsiliasi berhasil disimpan"})
}

// Stats returns reconciliation KPI metrics
func (h *Handler) Stats(c *gin.Context) {
	companyID := h.getCompanyID(c)
	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}
