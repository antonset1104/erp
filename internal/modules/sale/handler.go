package sale

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

func (h *Handler) ListOrders(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	state := c.Query("state")

	orders, err := h.svc.ListOrders(companyID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}

func (h *Handler) GetOrder(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := h.svc.GetOrder(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sales order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *Handler) CreateOrder(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var order SaleOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order.CompanyID = companyID
	order.CreateUID = userID
	order.WriteUID = userID

	for i := range order.OrderLines {
		order.OrderLines[i].CompanyID = companyID
		order.OrderLines[i].CreateUID = userID
		order.OrderLines[i].WriteUID = userID
	}

	if err := h.svc.CreateOrder(&order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *Handler) ActionConfirm(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, delivery, err := h.svc.ActionConfirm(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Sales order confirmed successfully",
		"order":    order,
		"delivery": delivery,
	})
}

func (h *Handler) ExportSalesCSV(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	orders, err := h.svc.ListOrders(companyID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	headers := []string{"Order Number", "Date", "Customer", "Subtotal (Rp)", "Tax (Rp)", "Total Amount (Rp)", "Status"}
	var rows [][]string
	for _, o := range orders {
		rows = append(rows, []string{
			o.Name,
			o.DateOrder,
			o.Partner.Name,
			strconv.FormatFloat(o.AmountUntaxed, 'f', 2, 64),
			strconv.FormatFloat(o.AmountTax, 'f', 2, 64),
			strconv.FormatFloat(o.AmountTotal, 'f', 2, 64),
			o.State,
		})
	}

	export.StreamCSV(c, "sales_orders_export.csv", headers, rows)
}
