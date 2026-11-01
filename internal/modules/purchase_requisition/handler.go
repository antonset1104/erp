package purchase_requisition

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

// List handles listing requisitions
func (h *Handler) List(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	state := c.Query("state")
	reqType := c.Query("type")
	search := c.Query("search")

	list, err := h.service.List(companyID, state, reqType, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar purchase requisition: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    list,
		"message": "Daftar purchase requisition berhasil diambil",
	})
}

// Create handles creating new purchase requisition or blanket order
func (h *Handler) Create(c *gin.Context) {
	var req CreateRequisitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data permohonan tidak valid: " + err.Error()})
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

	doc, err := h.service.CreateRequisition(req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat purchase requisition: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    doc,
		"message": "Purchase requisition berhasil didaftarkan",
	})
}

// GetByID handles retrieving requisition details
func (h *Handler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
		return
	}

	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	doc, err := h.service.GetByID(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen purchase requisition tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    doc,
		"message": "Detail purchase requisition berhasil diambil",
	})
}

// Confirm approves requisition from draft to confirmed
func (h *Handler) Confirm(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
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

	doc, err := h.service.ActionConfirm(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    doc,
		"message": "Permintaan pembelian disetujui",
	})
}

// OpenTender opens bidding
func (h *Handler) OpenTender(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
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

	doc, err := h.service.ActionOpenTender(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    doc,
		"message": "Tender pengadaan dibuka untuk rekanan",
	})
}

// GenerateRFQ generates an RFQ for vendor
func (h *Handler) GenerateRFQ(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
		return
	}

	var req GenerateRFQRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Vendor ID wajib disertakan: " + err.Error()})
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

	po, err := h.service.GenerateRFQ(uint(id), req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    po,
		"message": "RFQ berhasil diterbitkan untuk vendor",
	})
}

// SelectBid chooses winning vendor PO
func (h *Handler) SelectBid(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
		return
	}

	var body struct {
		WinningPOID uint `json:"winning_po_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Winning PO ID wajib disertakan: " + err.Error()})
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

	doc, err := h.service.SelectWinningBid(uint(id), body.WinningPOID, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    doc,
		"message": "Pemenang tender berhasil dipilih dan disetujui",
	})
}

// ReleaseBlanket releases partial PO against blanket order
func (h *Handler) ReleaseBlanket(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
		return
	}

	var req ReleaseBlanketPORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Rincian rilis PO tidak valid: " + err.Error()})
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

	po, err := h.service.ReleaseBlanketPO(uint(id), req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data":    po,
		"message": "Rilis Blanket PO berhasil diterbitkan",
	})
}

// Cancel cancels requisition
func (h *Handler) Cancel(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
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

	doc, err := h.service.ActionCancel(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    doc,
		"message": "Dokumen purchase requisition dibatalkan",
	})
}

// ResetDraft resets to draft
func (h *Handler) ResetDraft(c *gin.Context) {
	idParam := c.Param("id")
	id, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
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

	doc, err := h.service.ActionResetDraft(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    doc,
		"message": "Status dokumen dikembalikan ke Draf",
	})
}

// Stats returns KPI metrics
func (h *Handler) Stats(c *gin.Context) {
	companyID := c.GetUint("company_id")
	if companyID == 0 {
		companyID = 1
	}

	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat statistik: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    stats,
		"message": "Statistik purchase requisition berhasil diambil",
	})
}
