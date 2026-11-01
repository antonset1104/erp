package product

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

func (h *Handler) ListProducts(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	products, err := h.svc.ListProducts(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, products)
}

func (h *Handler) GetProduct(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID"})
		return
	}

	p, err := h.svc.GetProduct(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	c.JSON(http.StatusOK, p)
}

func (h *Handler) CreateProduct(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)

	var p Product
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p.CompanyID = companyID
	p.CreateUID = userID
	p.WriteUID = userID

	if err := h.svc.CreateProduct(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, p)
}

func (h *Handler) ListCategories(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	cats, err := h.svc.ListCategories(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cats)
}

func (h *Handler) ListUoM(c *gin.Context) {
	uoms, err := h.svc.ListUoM()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, uoms)
}

func (h *Handler) ExportProductsCSV(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	products, err := h.svc.ListProducts(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	headers := []string{"ID", "SKU / Code", "Name", "Type", "List Price", "Cost Price", "Qty On Hand"}
	var rows [][]string
	for _, p := range products {
		rows = append(rows, []string{
			strconv.FormatUint(uint64(p.ID), 10),
			p.DefaultCode,
			p.Name,
			p.Type,
			strconv.FormatFloat(p.ListPrice, 'f', 2, 64),
			strconv.FormatFloat(p.StandardPrice, 'f', 2, 64),
			strconv.FormatFloat(p.QtyAvailable, 'f', 2, 64),
		})
	}

	export.StreamCSV(c, "products_catalog.csv", headers, rows)
}
