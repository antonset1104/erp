package customer_portal

import (
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
	// 1. External Customer Portal Routes (Authenticated via X-Customer-Token or query ?token=)
	portal := r.Group("/portal/customer")
	portal.Use(h.CustomerAuthMiddleware())
	{
		portal.GET("/me", h.PortalGetProfile)
		portal.GET("/dashboard", h.PortalGetDashboard)
		portal.GET("/orders", h.PortalListOrders)
		portal.POST("/orders", h.PortalCreateOrder)
		portal.GET("/invoices", h.PortalListInvoices)
		portal.GET("/statement", h.PortalGetStatement)
		portal.POST("/rma", h.PortalSubmitRMA)
		portal.GET("/rma", h.PortalListRMAs)
		portal.POST("/payment-notification", h.PortalSubmitPaymentNotification)
	}

	// Also support /customer-portal/public for simulator & iframe
	publicSim := r.Group("/customer-portal/public")
	publicSim.Use(h.CustomerAuthMiddleware())
	{
		publicSim.GET("/me", h.PortalGetProfile)
		publicSim.GET("/account", h.PortalGetProfile)
		publicSim.GET("/dashboard", h.PortalGetDashboard)
		publicSim.GET("/orders", h.PortalListOrders)
		publicSim.POST("/orders", h.PortalCreateOrder)
		publicSim.GET("/invoices", h.PortalListInvoices)
		publicSim.GET("/statement", h.PortalGetStatement)
		publicSim.POST("/rma", h.PortalSubmitRMA)
		publicSim.GET("/rma", h.PortalListRMAs)
		publicSim.POST("/payment-notification", h.PortalSubmitPaymentNotification)
		publicSim.POST("/payment-receipt", h.PortalSubmitPaymentNotification)
	}

	// 2. Internal ERP Management Routes (Protected by ERP JWT)
	admin := r.Group("/customer-portal")
	if authMiddleware != nil {
		admin.Use(authMiddleware)
	}
	{
		admin.GET("/stats", h.GetStats)
		admin.GET("/accounts", h.ListAccounts)
		admin.POST("/accounts", h.CreateAccount)
		admin.GET("/accounts/:id", h.GetAccount)

		admin.GET("/orders", h.ListOrderRequests)
		admin.POST("/orders/:id/approve", h.ApproveOrderRequest)

		admin.GET("/rmas", h.ListRMAs)
		admin.POST("/rmas/:id/approve", h.ApproveRMA)
		admin.POST("/rmas/:id/reject", h.RejectRMA)

		admin.GET("/payment-notifications", h.ListPaymentNotifications)
		admin.GET("/statements", h.ListPaymentNotifications)
		admin.POST("/payment-notifications/:id/verify", h.VerifyPaymentNotification)

		admin.GET("/statement/:partner_id", h.GetCustomerStatement)
		admin.GET("/ai-health/:partner_id", h.GetAICustomerHealth)
		admin.POST("/ai-customer-health", h.GetAICustomerHealth)
	}
}

// CustomerAuthMiddleware validates token from X-Customer-Token, ?token=, or Authorization: Bearer
func (h *Handler) CustomerAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("X-Customer-Token")
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Akses ditolak: X-Customer-Token wajib disertakan"})
			c.Abort()
			return
		}

		acc, err := h.service.GetAccountByToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		c.Set("customer_account", acc)
		c.Next()
	}
}

func getCustomerAccountFromContext(c *gin.Context) *CustomerPortalAccount {
	if acc, exists := c.Get("customer_account"); exists {
		if v, ok := acc.(*CustomerPortalAccount); ok {
			return v
		}
	}
	return nil
}

// -------------------------------------------------------------
// Portal Customer Endpoints
// -------------------------------------------------------------

func (h *Handler) PortalGetProfile(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": acc})
}

func (h *Handler) PortalGetDashboard(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}
	dash, err := h.service.GetCustomerDashboard(acc.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dash})
}

func (h *Handler) PortalListOrders(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}
	orders, err := h.service.ListCustomerSaleOrders(acc.PartnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

func (h *Handler) PortalCreateOrder(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req CreateB2BOrderRequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.service.CreateB2BOrderRequest(acc.ID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pesanan B2B mandiri berhasil dikirim ke ERP",
		"data":    order,
	})
}

func (h *Handler) PortalListInvoices(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}
	invoices, err := h.service.ListCustomerInvoices(acc.PartnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": invoices})
}

func (h *Handler) PortalGetStatement(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}
	stmt, err := h.service.GetCustomerStatement(acc.PartnerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stmt})
}

func (h *Handler) PortalSubmitRMA(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req SubmitRMARequestPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rma, err := h.service.SubmitRMARequest(acc.ID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Klaim Retur (RMA) berhasil diajukan dan sedang ditinjau tim kami",
		"data":    rma,
	})
}

func (h *Handler) PortalListRMAs(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}
	rmas, err := h.service.ListRMAs(acc.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rmas})
}

func (h *Handler) PortalSubmitPaymentNotification(c *gin.Context) {
	acc := getCustomerAccountFromContext(c)
	if acc == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req SubmitPaymentNotificationPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notif, err := h.service.SubmitPaymentNotification(acc.ID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Konfirmasi pembayaran berhasil dikirim untuk verifikasi tim keuangan",
		"data":    notif,
	})
}

// -------------------------------------------------------------
// ERP Internal Admin Endpoints
// -------------------------------------------------------------

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
	var req CreateCustomerAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	acc, err := h.service.CreateAccount(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Akun Customer Portal berhasil dibuat",
		"data":    acc,
	})
}

func (h *Handler) GetAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID akun tidak valid"})
		return
	}
	acc, err := h.service.GetAccountByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Akun tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": acc})
}

func (h *Handler) ListOrderRequests(c *gin.Context) {
	orders, err := h.service.ListAllOrderRequests()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": orders})
}

func (h *Handler) ApproveOrderRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pesanan tidak valid"})
		return
	}
	autoConfirm := c.Query("auto_confirm") == "true"
	order, err := h.service.ApproveB2BOrder(uint(id), autoConfirm)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order, "message": "Pesanan B2B berhasil disetujui"})
}

func (h *Handler) ListRMAs(c *gin.Context) {
	rmas, err := h.service.ListAllRMAs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rmas})
}

type ApproveRMAPayload struct {
	ResolutionAction string `json:"resolution_action"`
	AdminNotes       string `json:"admin_notes"`
}

func (h *Handler) ApproveRMA(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID RMA tidak valid"})
		return
	}
	var payload ApproveRMAPayload
	_ = c.ShouldBindJSON(&payload)

	rma, err := h.service.ApproveRMA(uint(id), payload.ResolutionAction, payload.AdminNotes)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rma, "message": "Klaim RMA disetujui"})
}

type RejectRMAPayload struct {
	Reason string `json:"reason" binding:"required"`
}

func (h *Handler) RejectRMA(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID RMA tidak valid"})
		return
	}
	var payload RejectRMAPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rma, err := h.service.RejectRMA(uint(id), payload.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rma, "message": "Klaim RMA ditolak"})
}

func (h *Handler) ListPaymentNotifications(c *gin.Context) {
	notifs, err := h.service.ListAllPaymentNotifications()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": notifs})
}

func (h *Handler) VerifyPaymentNotification(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID konfirmasi tidak valid"})
		return
	}
	autoPost := c.DefaultQuery("auto_post", "true") == "true"
	notif, err := h.service.VerifyPaymentNotification(uint(id), 1, autoPost)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": notif, "message": "Pembayaran berhasil diverifikasi dan diposting ke jurnal"})
}

func (h *Handler) GetCustomerStatement(c *gin.Context) {
	partnerID, err := strconv.ParseUint(c.Param("partner_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pelanggan tidak valid"})
		return
	}
	stmt, err := h.service.GetCustomerStatement(uint(partnerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stmt})
}

func (h *Handler) GetAICustomerHealth(c *gin.Context) {
	partnerParam := c.Param("partner_id")
	if partnerParam == "" {
		partnerParam = c.Query("partner_id")
	}
	partnerID, err := strconv.ParseUint(partnerParam, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID pelanggan tidak valid"})
		return
	}
	scorecard, err := h.service.EvaluateAICustomerHealth(uint(partnerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": scorecard})
}
