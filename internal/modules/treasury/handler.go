package treasury

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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	group := r.Group("/treasury")
	group.Use(authMW)
	{
		// Stats
		group.GET("/stats", h.GetStats)

		// Bank Accounts
		group.GET("/accounts", h.ListBankAccounts)
		group.POST("/accounts", h.CreateBankAccount)
		group.GET("/accounts/:id", h.GetBankAccount)

		// Cash Flow Forecasts
		group.GET("/forecasts", h.ListForecasts)
		group.POST("/forecasts/generate", h.GenerateForecast)
		group.GET("/forecasts/:id", h.GetForecast)
		group.GET("/forecasts/:id/ai-advisory", h.AnalyzeTreasuryAI)
		group.POST("/forecasts/:id/ai-advisory", h.AnalyzeTreasuryAI)
		group.GET("/forecasts/:id/ai-advisor", h.AnalyzeTreasuryAI)
		group.POST("/forecasts/:id/ai-advisor", h.AnalyzeTreasuryAI)

		// Inter-Bank Transfers
		group.GET("/transfers", h.ListTransfers)
		group.POST("/transfers", h.ExecuteTransfer)
		group.GET("/transfers/:id", h.GetTransfer)
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
	c.JSON(http.StatusOK, stats)
}

// ----------------------------------------------------------------------------
// BANK ACCOUNTS
// ----------------------------------------------------------------------------

func (h *Handler) ListBankAccounts(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	accs, err := h.service.ListBankAccounts(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accs)
}

func (h *Handler) CreateBankAccount(c *gin.Context) {
	var req CreateBankAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	acc, err := h.service.CreateBankAccount(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, acc)
}

func (h *Handler) GetBankAccount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID rekening tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	acc, err := h.service.GetBankAccount(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rekening bank tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, acc)
}

// ----------------------------------------------------------------------------
// CASH FLOW FORECASTS
// ----------------------------------------------------------------------------

func (h *Handler) ListForecasts(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	forecasts, err := h.service.ListForecasts(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, forecasts)
}

func (h *Handler) GenerateForecast(c *gin.Context) {
	var req GenerateForecastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	forecast, err := h.service.GenerateCashFlowForecast(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, forecast)
}

func (h *Handler) GetForecast(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID proyeksi tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	forecast, err := h.service.GetForecast(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Proyeksi arus kas tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, forecast)
}

func (h *Handler) AnalyzeTreasuryAI(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID proyeksi tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	report, err := h.service.AnalyzeTreasuryWithAI(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// ----------------------------------------------------------------------------
// INTER-BANK TRANSFERS
// ----------------------------------------------------------------------------

func (h *Handler) ListTransfers(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	transfers, err := h.service.ListTransfers(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, transfers)
}

func (h *Handler) ExecuteTransfer(c *gin.Context) {
	var req CreateInterBankTransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	transfer, err := h.service.ExecuteInterBankTransfer(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, transfer)
}

func (h *Handler) GetTransfer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID transfer tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	transfer, err := h.service.GetTransfer(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Transfer tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, transfer)
}
