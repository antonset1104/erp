package currency

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

// ListCurrencies returns all currencies
func (h *Handler) ListCurrencies(c *gin.Context) {
	activeOnly := c.Query("active") == "true" || c.Query("active") == "1"
	currencies, err := h.service.ListCurrencies(activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar mata uang: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":    currencies,
		"message": "Daftar mata uang berhasil diambil",
	})
}

// GetCurrencyByID returns single currency details
func (h *Handler) GetCurrencyByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID mata uang tidak valid"})
		return
	}

	curr, err := h.service.GetCurrencyByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mata uang tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    curr,
		"message": "Detail mata uang berhasil diambil",
	})
}

// CreateCurrency creates a new currency
func (h *Handler) CreateCurrency(c *gin.Context) {
	var req CreateCurrencyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data mata uang tidak valid: " + err.Error()})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	curr, err := h.service.CreateCurrency(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    curr,
		"message": "Mata uang baru berhasil didaftarkan",
	})
}

// ListRates returns exchange rate history for a currency
func (h *Handler) ListRates(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID mata uang tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	rates, err := h.service.ListRates(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil histori kurs: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    rates,
		"message": "Histori kurs berhasil diambil",
	})
}

// CreateRate records a new exchange rate entry
func (h *Handler) CreateRate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID mata uang tidak valid"})
		return
	}

	var req CreateRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data kurs tidak valid: " + err.Error()})
		return
	}
	req.CurrencyID = uint(id)

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	rate, err := h.service.CreateRate(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    rate,
		"message": "Kurs valas berhasil dicatat",
	})
}

// Convert converts amounts between currencies
func (h *Handler) Convert(c *gin.Context) {
	var req CurrencyConversionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data konversi tidak valid: " + err.Error()})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	res, err := h.service.Convert(req.Amount, req.FromCurrencyID, req.ToCurrencyID, req.Date, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    res,
		"message": "Konversi nilai valas berhasil dihitung",
	})
}

// ComputeRevaluation calculates unrealized forex gain/loss draft
func (h *Handler) ComputeRevaluation(c *gin.Context) {
	var req ComputeRevaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter revaluasi tidak valid: " + err.Error()})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}

	rev, err := h.service.ComputeUnrealizedRevaluation(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    rev,
		"message": "Kalkulasi revaluasi valas berhasil disusun",
	})
}

// ListRevaluations returns all revaluations
func (h *Handler) ListRevaluations(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	state := c.Query("state")

	list, err := h.service.ListRevaluations(companyID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar revaluasi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    list,
		"message": "Daftar revaluasi valas berhasil diambil",
	})
}

// GetRevaluationByID returns single revaluation details
func (h *Handler) GetRevaluationByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID revaluasi tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	rev, err := h.service.GetRevaluationByID(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen revaluasi tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    rev,
		"message": "Detail revaluasi valas berhasil diambil",
	})
}

// PostRevaluation posts revaluation to General Ledger
func (h *Handler) PostRevaluation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID revaluasi tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}

	rev, err := h.service.PostRevaluation(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    rev,
		"message": "Revaluasi kurs berhasil dibukukan ke General Ledger!",
	})
}

// CancelRevaluation cancels a draft revaluation
func (h *Handler) CancelRevaluation(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID revaluasi tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}

	rev, err := h.service.CancelRevaluation(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    rev,
		"message": "Dokumen revaluasi dibatalkan",
	})
}

// CalculateRealizedGainLoss calculates realized forex on settlement
func (h *Handler) CalculateRealizedGainLoss(c *gin.Context) {
	var req RealizedGainLossCalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter kalkulasi realized forex tidak valid: " + err.Error()})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	res, err := h.service.CalculateRealizedGainLoss(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    res,
		"message": "Kalkulasi laba/rugi kurs terealisasi berhasil dihitung",
	})
}

// ForexRiskAIAudit runs Cloudflare Workers AI risk & sensitivity audit
func (h *Handler) ForexRiskAIAudit(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID mata uang tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	report, err := h.service.ForexRiskAIAudit(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    report,
		"message": "Audit risiko valas Cloudflare Workers AI selesai",
	})
}

// Stats returns executive KPI stats
func (h *Handler) Stats(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik valas: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    stats,
		"message": "Statistik mata uang & valas berhasil diambil",
	})
}
