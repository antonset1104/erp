package reports

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	repGroup := r.Group("/reports")
	{
		// Catalog can be accessed freely or with auth
		repGroup.GET("/catalog", h.GetCatalog)

		// Print view is HTML output for window.open / browser print
		repGroup.GET("/print/:module/:id", h.PrintDocument)
		repGroup.GET("/preview", h.PreviewDocument)
	}
}

func (h *Handler) GetCatalog(c *gin.Context) {
	catalog := h.service.GetCatalog()
	c.JSON(http.StatusOK, catalog)
}

func (h *Handler) PrintDocument(c *gin.Context) {
	module := c.Param("module")
	idStr := c.Param("id")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.String(http.StatusBadRequest, "ID dokumen tidak valid: %s", idStr)
		return
	}

	html, err := h.service.RenderDocument(module, uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "Gagal membuat dokumen cetak: %s", err.Error())
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

func (h *Handler) PreviewDocument(c *gin.Context) {
	module := c.DefaultQuery("module", "sale")
	idStr := c.DefaultQuery("id", "1")

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		id = 1
	}

	html, err := h.service.RenderDocument(module, uint(id))
	if err != nil {
		c.String(http.StatusNotFound, "Gagal memuat pratinjau dokumen: %s", err.Error())
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}
