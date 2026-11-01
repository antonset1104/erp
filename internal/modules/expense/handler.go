package expense

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

// ----------------- Categories -----------------

func (h *Handler) ListCategories(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	list, err := h.svc.ListCategories(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) CreateCategory(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var cat ExpenseCategory
	if err := c.ShouldBindJSON(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cat.CompanyID = companyID
	cat.CreateUID = userID
	cat.WriteUID = userID

	if err := h.svc.CreateCategory(&cat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, cat)
}

// ----------------- Expenses (Items) -----------------

func (h *Handler) ListExpenses(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	empIDStr := c.Query("employee_id")
	sheetIDStr := c.Query("sheet_id")
	state := c.Query("state")

	var empID uint
	if empIDStr != "" {
		id, _ := strconv.Atoi(empIDStr)
		empID = uint(id)
	}

	var sheetID *uint
	if sheetIDStr != "" {
		id, _ := strconv.Atoi(sheetIDStr)
		uID := uint(id)
		sheetID = &uID
	}

	list, err := h.svc.ListExpenses(companyID, empID, sheetID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetExpense(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}

	item, err := h.svc.GetExpense(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, item)
}

func (h *Handler) CreateExpense(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var exp Expense
	if err := c.ShouldBindJSON(&exp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	exp.CreateUID = userID
	exp.WriteUID = userID

	if err := h.svc.CreateExpense(&exp, companyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, exp)
}

func (h *Handler) UpdateExpense(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}

	var exp Expense
	if err := c.ShouldBindJSON(&exp); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	exp.ID = uint(id)
	exp.CompanyID = companyID
	exp.WriteUID = userID

	if err := h.svc.UpdateExpense(&exp, companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, exp)
}

func (h *Handler) DeleteExpense(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}

	if err := h.svc.DeleteExpense(uint(id), companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "expense deleted successfully"})
}

// ----------------- Expense Sheets -----------------

type CreateSheetRequest struct {
	Name           string `json:"name" binding:"required"`
	EmployeeID     uint   `json:"employee_id" binding:"required"`
	PaymentMode    string `json:"payment_mode"`
	AccountingDate string `json:"accounting_date"`
	Notes          string `json:"notes"`
	ExpenseIDs     []uint `json:"expense_ids"`
}

func (h *Handler) ListExpenseSheets(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	empIDStr := c.Query("employee_id")
	state := c.Query("state")

	var empID uint
	if empIDStr != "" {
		id, _ := strconv.Atoi(empIDStr)
		empID = uint(id)
	}

	list, err := h.svc.ListExpenseSheets(companyID, empID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

func (h *Handler) GetExpenseSheet(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sheet id"})
		return
	}

	sheet, err := h.svc.GetExpenseSheet(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sheet)
}

func (h *Handler) CreateExpenseSheet(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req CreateSheetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sheet := ExpenseSheet{
		Name:           req.Name,
		EmployeeID:     req.EmployeeID,
		PaymentMode:    req.PaymentMode,
		AccountingDate: req.AccountingDate,
		Notes:          req.Notes,
	}
	sheet.CreateUID = userID
	sheet.WriteUID = userID

	created, err := h.svc.CreateExpenseSheet(&sheet, req.ExpenseIDs, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) AttachExpenses(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sheet id"})
		return
	}

	var req struct {
		ExpenseIDs []uint `json:"expense_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.AttachExpensesToSheet(uint(id), req.ExpenseIDs, companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "expenses attached successfully"})
}

func (h *Handler) RemoveExpense(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	expenseID, err := strconv.Atoi(c.Param("expense_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expense id"})
		return
	}

	if err := h.svc.RemoveExpenseFromSheet(uint(expenseID), companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "expense removed from sheet"})
}

func (h *Handler) ActionSubmitSheet(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sheet id"})
		return
	}

	sheet, err := h.svc.ActionSubmitSheet(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sheet)
}

func (h *Handler) ActionApproveSheet(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sheet id"})
		return
	}

	sheet, err := h.svc.ActionApproveSheet(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sheet)
}

func (h *Handler) ActionPostSheet(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sheet id"})
		return
	}

	sheet, move, err := h.svc.ActionPostSheet(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"sheet": sheet,
		"move":  move,
	})
}

func (h *Handler) ActionPaySheet(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sheet id"})
		return
	}

	sheet, err := h.svc.ActionPaySheet(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sheet)
}

func (h *Handler) ActionRefuseSheet(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid sheet id"})
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "refusal reason is required"})
		return
	}

	sheet, err := h.svc.ActionRefuseSheet(uint(id), companyID, req.Reason, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, sheet)
}

// ----------------- Stats & Analytics -----------------

func (h *Handler) GetExpenseStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stats, err := h.svc.GetExpenseStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
