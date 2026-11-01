package account

import (
	"net/http"
	"strconv"

	"erp-golang/internal/core/export"
	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListMoves(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	moveType := c.Query("type")
	state := c.Query("state")

	moves, err := h.svc.ListMoves(companyID, moveType, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, moves)
}

func (h *Handler) GetMove(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid move ID"})
		return
	}

	move, err := h.svc.GetMove(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invoice not found"})
		return
	}

	c.JSON(http.StatusOK, move)
}

func (h *Handler) CreateMove(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var move AccountMove
	if err := c.ShouldBindJSON(&move); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	move.CompanyID = companyID
	move.CreateUID = userID
	move.WriteUID = userID

	for i := range move.LineIDs {
		move.LineIDs[i].CompanyID = companyID
		move.LineIDs[i].CreateUID = userID
		move.LineIDs[i].WriteUID = userID
	}

	if err := h.svc.CreateMove(&move); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, move)
}

func (h *Handler) ActionPost(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid move ID"})
		return
	}

	move, err := h.svc.ActionPost(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Invoice successfully posted to ledger",
		"move":    move,
	})
}

type PaymentRequest struct {
	Amount    float64 `json:"amount" binding:"required"`
	JournalID uint    `json:"journal_id" binding:"required"`
	Memo      string  `json:"memo"`
}

func (h *Handler) RegisterPayment(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid move ID"})
		return
	}

	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, move, err := h.svc.RegisterPayment(uint(id), req.Amount, req.JournalID, req.Memo, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Payment registered successfully",
		"payment": payment,
		"move":    move,
	})
}

func (h *Handler) ListAccounts(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	accounts, err := h.svc.ListAccounts(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, accounts)
}

func (h *Handler) ListJournals(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	journals, err := h.svc.ListJournals(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, journals)
}

func (h *Handler) GetProfitAndLoss(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	report, err := h.svc.GetProfitAndLoss(companyID, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) GetBalanceSheet(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	asOfDate := c.Query("as_of_date")

	report, err := h.svc.GetBalanceSheet(companyID, asOfDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) GetTrialBalance(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	report, err := h.svc.GetTrialBalance(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *Handler) PrintMove(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid invoice ID")
		return
	}

	companyID := uint(1)
	if ctxCompanyID, exists := c.Get("company_id"); exists {
		companyID = ctxCompanyID.(uint)
	}

	move, err := h.svc.GetMove(uint(id), companyID)
	if err != nil {
		c.String(http.StatusNotFound, "Invoice not found: %s", err.Error())
		return
	}

	html := RenderInvoiceHTML(move)
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (h *Handler) ExportTrialBalanceCSV(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	report, err := h.svc.GetTrialBalance(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	headers := []string{"Account Code", "Account Name", "Type", "Total Debit (Rp)", "Total Credit (Rp)", "Balance (Rp)"}
	var rows [][]string
	for _, l := range report.Lines {
		rows = append(rows, []string{
			l.Code,
			l.Name,
			l.Type,
			strconv.FormatFloat(l.Debit, 'f', 2, 64),
			strconv.FormatFloat(l.Credit, 'f', 2, 64),
			strconv.FormatFloat(l.Balance, 'f', 2, 64),
		})
	}
	// Add total row
	rows = append(rows, []string{
		"TOTAL",
		"KESELURUHAN (BALANCED)",
		"-",
		strconv.FormatFloat(report.TotalDebit, 'f', 2, 64),
		strconv.FormatFloat(report.TotalCredit, 'f', 2, 64),
		strconv.FormatFloat(report.Difference, 'f', 2, 64),
	})

	export.StreamCSV(c, "trial_balance_export.csv", headers, rows)
}


