package budget

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// --- Analytic Accounts ---

func (h *Handler) ListAnalyticAccounts(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	items, err := h.svc.ListAnalyticAccounts(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) CreateAnalyticAccount(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	var req CreateAnalyticAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	acc, err := h.svc.CreateAnalyticAccount(req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": acc})
}

func (h *Handler) GetAnalyticAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	acc, err := h.svc.GetAnalyticAccount(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Akun analitik tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": acc})
}

// --- Budgetary Positions ---

func (h *Handler) ListBudgetaryPositions(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	items, err := h.svc.ListBudgetaryPositions(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) CreateBudgetaryPosition(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	var req CreateBudgetaryPositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pos, err := h.svc.CreateBudgetaryPosition(req, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": pos})
}

func (h *Handler) GetBudgetaryPosition(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	pos, err := h.svc.GetBudgetaryPosition(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pos anggaran tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pos})
}

// --- Budgets ---

func (h *Handler) ListBudgets(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	state := c.Query("state")
	search := c.Query("search")

	items, err := h.svc.ListBudgets(companyID, state, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *Handler) CreateBudget(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	userID := c.GetUint("user_id")
	if userID == 0 {
		userID = 1
	}

	var req CreateBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	b, err := h.svc.CreateBudget(req, userID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": b})
}

func (h *Handler) GetBudget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	b, err := h.svc.GetBudget(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen anggaran tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": b})
}

func (h *Handler) ComputeBudget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	if err := h.svc.ComputeBudget(uint(id), companyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	b, _ := h.svc.GetBudget(uint(id))
	c.JSON(http.StatusOK, gin.H{"data": b, "message": "Kalkulasi realisasi anggaran selesai diperbarui"})
}

func (h *Handler) ConfirmBudget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
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
	b, err := h.svc.ActionConfirm(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": b, "message": "Anggaran berhasil dikonfirmasi"})
}

func (h *Handler) ValidateBudget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
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
	b, err := h.svc.ActionValidate(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": b, "message": "Anggaran resmi divalidasi dan aktif"})
}

func (h *Handler) DoneBudget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
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
	b, err := h.svc.ActionDone(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": b, "message": "Periode anggaran ditutup & selesai"})
}

func (h *Handler) CancelBudget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
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
	b, err := h.svc.ActionCancel(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": b, "message": "Anggaran berhasil dibatalkan"})
}

func (h *Handler) ResetDraftBudget(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
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
	b, err := h.svc.ActionResetDraft(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": b, "message": "Anggaran dikembalikan ke status draf"})
}

func (h *Handler) AuditWithAI(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
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
	result, err := h.svc.AuditWithAI(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}
	stats, err := h.svc.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}
