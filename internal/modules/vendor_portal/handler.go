package vendor_portal

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// 1. External Vendor Portal Routes (Authenticated via X-Vendor-Token or query ?token=)
	portal := r.Group("/portal/vendor")
	portal.Use(h.VendorAuthMiddleware())
	{
		portal.GET("/me", h.PortalGetProfile)
		portal.GET("/rfqs", h.PortalListOpenRFQs)
		portal.POST("/bids", h.PortalSubmitBid)
		portal.GET("/bids", h.PortalListMyBids)
		portal.GET("/purchase-orders", h.PortalListPOs)
		portal.POST("/asn", h.PortalSubmitASN)
		portal.GET("/asn", h.PortalListMyASNs)
		portal.POST("/invoices", h.PortalSubmitInvoice)
		portal.GET("/invoices", h.PortalListMyInvoices)
	}

	// Also support /vendor-portal/public for simulator
	publicSim := r.Group("/vendor-portal/public")
	publicSim.Use(h.VendorAuthMiddleware())
	{
		publicSim.GET("/rfqs", h.PortalListOpenRFQs)
		publicSim.POST("/bids", h.PortalSubmitBid)
		publicSim.POST("/asn", h.PortalSubmitASN)
		publicSim.POST("/invoices", h.PortalSubmitInvoice)
	}

	// 2. Internal ERP Management Routes (Protected by ERP JWT)
	admin := r.Group("/vendor-portal")
	if authMiddleware != nil {
		admin.Use(authMiddleware)
	}
	{
		admin.GET("/stats", h.GetStats)
		admin.GET("/accounts", h.ListAccounts)
		admin.POST("/accounts", h.CreateAccount)
		admin.GET("/accounts/:id", h.GetAccount)

		admin.GET("/bids", h.ListBids)
		admin.GET("/bids/:id", h.GetBid)
		admin.POST("/bids/:id/accept", h.AcceptBid)

		admin.GET("/asns", h.ListASNs)
		admin.GET("/asns/:id", h.GetASN)
		admin.POST("/asns/:id/receive", h.ReceiveASN)

		admin.GET("/invoices", h.ListInvoices)
		admin.GET("/invoices/:id", h.GetInvoice)
		admin.POST("/invoices/:id/3way-match", h.Execute3WayMatch)

		admin.GET("/ai-rating/:partner_id", h.GetAIVendorRating)
		admin.POST("/ai-vendor-rating", h.GetAIVendorRating)
	}
}

// VendorAuthMiddleware authenticates vendor requests using X-Vendor-Token
func (h *Handler) VendorAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Vendor-Token")
		if token == "" {
			token = c.Query("token")
		}
		if token == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Akses ditolak: X-Vendor-Token wajib disertakan"})
			c.Abort()
			return
		}

		acc, err := h.service.GetAccountByToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("vendor_account", acc)
		c.Next()
	}
}

func getVendorAccountFromContext(c *gin.Context) *VendorPortalAccount {
	if acc, exists := c.Get("vendor_account"); exists {
		if v, ok := acc.(*VendorPortalAccount); ok {
			return v
		}
	}
	return nil
}

// Portal Handlers
func (h *Handler) PortalGetProfile(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": acc})
}

func (h *Handler) PortalListOpenRFQs(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	rfqs, err := h.service.ListOpenRFQs(acc.PartnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rfqs})
}

func (h *Handler) PortalSubmitBid(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	var req SubmitBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bid, err := h.service.SubmitQuotationBid(acc.PartnerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": bid, "message": fmt.Sprintf("Penawaran harga %s berhasil diajukan", bid.BidNumber)})
}

func (h *Handler) PortalListMyBids(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	bids, err := h.service.ListBidsByPartnerID(acc.PartnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bids})
}

func (h *Handler) PortalListPOs(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	pos, err := h.service.repo.ListConfirmedPOsForVendor(acc.PartnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pos})
}

func (h *Handler) PortalSubmitASN(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	var req SubmitASNRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asn, err := h.service.SubmitASN(acc.PartnerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": asn, "message": fmt.Sprintf("Surat jalan ASN %s berhasil dibuat", asn.ASNNumber)})
}

func (h *Handler) PortalListMyASNs(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	asns, err := h.service.ListASNsByPartnerID(acc.PartnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": asns})
}

func (h *Handler) PortalSubmitInvoice(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	var req SubmitVendorInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	inv, matchRes, err := h.service.SubmitVendorInvoice(acc.PartnerID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"data":         inv,
		"match_result": matchRes,
		"message":      fmt.Sprintf("Tagihan %s berhasil dikirim dan diverifikasi 3-Way Matching", inv.InvoiceNumberSupplier),
	})
}

func (h *Handler) PortalListMyInvoices(c *gin.Context) {
	acc := getVendorAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	invoices, err := h.service.ListInvoiceSubmissionsByPartnerID(acc.PartnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": invoices})
}

// Internal ERP Admin Handlers
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

func (h *Handler) ListAccounts(c *gin.Context) {
	accounts, err := h.service.ListAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

func (h *Handler) CreateAccount(c *gin.Context) {
	var req CreateVendorAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	account, err := h.service.CreateAccount(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": account, "message": "Akun portal vendor berhasil dibuat"})
}

func (h *Handler) GetAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID akun tidak valid"})
		return
	}
	account, err := h.service.GetAccountByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Akun portal vendor tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": account})
}

func (h *Handler) ListBids(c *gin.Context) {
	bids, err := h.service.ListBids()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bids})
}

func (h *Handler) GetBid(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID bid tidak valid"})
		return
	}
	bid, err := h.service.GetBidByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bid tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bid})
}

func (h *Handler) AcceptBid(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID bid tidak valid"})
		return
	}
	bid, err := h.service.AcceptQuotationBid(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bid, "message": fmt.Sprintf("Penawaran %s diterima", bid.BidNumber)})
}

func (h *Handler) ListASNs(c *gin.Context) {
	asns, err := h.service.ListASNs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": asns})
}

func (h *Handler) GetASN(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID ASN tidak valid"})
		return
	}
	asn, err := h.service.GetASNByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "ASN tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": asn})
}

func (h *Handler) ReceiveASN(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID ASN tidak valid"})
		return
	}

	var body struct {
		ReceivedLines map[uint]float64 `json:"received_lines"`
	}
	_ = c.ShouldBindJSON(&body)

	asn, err := h.service.ReceiveASN(uint(id), body.ReceivedLines)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": asn, "message": fmt.Sprintf("Barang ASN %s telah diterima di gudang", asn.ASNNumber)})
}

func (h *Handler) ListInvoices(c *gin.Context) {
	invoices, err := h.service.ListInvoiceSubmissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": invoices})
}

func (h *Handler) GetInvoice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tagihan tidak valid"})
		return
	}
	inv, err := h.service.GetInvoiceSubmissionByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tagihan vendor tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": inv})
}

func (h *Handler) Execute3WayMatch(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tagihan tidak valid"})
		return
	}
	result, err := h.service.Execute3WayMatch(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": result, "message": "Rekonsiliasi 3-Way Matching selesai diproses"})
}

func (h *Handler) GetAIVendorRating(c *gin.Context) {
	partnerParam := c.Param("partner_id")
	if partnerParam == "" {
		partnerParam = c.Query("partner_id")
	}
	partnerID, err := strconv.ParseUint(partnerParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID partner tidak valid"})
		return
	}
	rating, err := h.service.EvaluateAIVendorPerformance(uint(partnerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rating})
}
