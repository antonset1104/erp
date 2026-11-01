package customs_bonded

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	cb := r.Group("/customs-bonded")
	if authMiddleware != nil {
		cb.Use(authMiddleware)
	}
	{
		cb.GET("/stats", h.GetStats)
		cb.GET("/documents", h.ListDocuments)
		cb.POST("/documents", h.CreateDocument)
		cb.GET("/documents/:id", h.GetDocument)
		cb.POST("/documents/:id/sppb", h.ApproveSPPB)
		cb.POST("/documents/:id/approve-sppb", h.ApproveSPPB)
		cb.POST("/documents/:id/cancel", h.CancelDocument)

		cb.GET("/it-inventory", h.GetMutations)
		cb.GET("/it-inventory/mutations", h.GetMutations)
		cb.GET("/it-inventory/book-report", h.GetBookReport)

		cb.GET("/subcontracts", h.ListSubcontracts)
		cb.POST("/subcontracts", h.CreateSubcontract)
		cb.POST("/subcontracts/:id/return", h.ReturnSubcontract)

		cb.POST("/ai-compliance-audit", h.EvaluateAICustomsAudit)
		cb.POST("/ai-customs-audit", h.EvaluateAICustomsAudit)
	}
}

func (h *Handler) CancelDocument(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	doc, err := h.service.CancelDocument(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": doc, "message": "Dokumen pabean berhasil dibatalkan"})
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) ListDocuments(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	docType := c.Query("document_type")
	status := c.Query("status")

	docs, err := h.service.ListDocuments(companyID, docType, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": docs})
}

func (h *Handler) CreateDocument(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	var req CreateCustomsDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.service.CreateCustomsDocument(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "Dokumen Pabean berhasil didaftarkan",
		"data":    doc,
	})
}

func (h *Handler) GetDocument(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
		return
	}

	doc, err := h.service.GetDocumentByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dokumen pabean tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": doc})
}

func (h *Handler) ApproveSPPB(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID dokumen tidak valid"})
		return
	}

	var req ApproveSPPBRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	doc, err := h.service.ApproveSPPB(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "SPPB berhasil disetujui & mutasi IT Inventory dicatat otomatis",
		"data":    doc,
	})
}

func (h *Handler) GetMutations(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	book := c.Query("book_category")
	var start, end *time.Time
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			start = &t
		}
	}
	if e := c.Query("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			end = &t
		}
	}

	mutations, err := h.service.GetITInventoryMutations(companyID, book, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": mutations})
}

func (h *Handler) GetBookReport(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	book := c.Query("book_category")
	if book == "" {
		book = BookRawMaterial
	}

	var start, end *time.Time
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			start = &t
		}
	}
	if e := c.Query("end_date"); e != "" {
		if t, err := time.Parse("2006-01-02", e); err == nil {
			end = &t
		}
	}

	report, err := h.service.GetITInventoryBookReport(companyID, book, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}

func (h *Handler) ListSubcontracts(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	status := c.Query("status")
	list, err := h.service.ListSubcontracts(companyID, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

func (h *Handler) CreateSubcontract(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	var sub CustomsSubcontract
	if err := c.ShouldBindJSON(&sub); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sub.CompanyID = companyID

	res, err := h.service.CreateSubcontract(sub)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "Pekerjaan subkontrak berhasil didaftarkan",
		"data":    res,
	})
}

func (h *Handler) ReturnSubcontract(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID subkontrak tidak valid"})
		return
	}

	var req struct {
		QtyReturned float64 `json:"qty_returned" binding:"required,gt=0"`
		BC262DocNo  string  `json:"bc262_doc_no"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.service.ReturnSubcontract(uint(id), req.QtyReturned, req.BC262DocNo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Pemasukan kembali barang subkontrak (BC 2.62) berhasil dicatat",
		"data":    res,
	})
}

func (h *Handler) EvaluateAICustomsAudit(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	report, err := h.service.EvaluateAICustomsAudit(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}
