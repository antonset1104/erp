package mrp

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

func (h *Handler) ListBoms(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	boms, err := h.svc.ListBoms(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, boms)
}

func (h *Handler) CreateBom(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var bom MrpBom
	if err := c.ShouldBindJSON(&bom); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bom.CompanyID = companyID
	bom.CreateUID = userID
	bom.WriteUID = userID
	for i := range bom.Lines {
		bom.Lines[i].CompanyID = companyID
	}
	if err := h.svc.CreateBom(&bom); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, bom)
}

func (h *Handler) ListProductions(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	state := c.Query("state")
	mos, err := h.svc.ListProductions(companyID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mos)
}

func (h *Handler) CreateProduction(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var mo MrpProduction
	if err := c.ShouldBindJSON(&mo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mo.CompanyID = companyID
	mo.CreateUID = userID
	mo.WriteUID = userID
	if err := h.svc.CreateProduction(&mo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, mo)
}

func (h *Handler) ActionProduce(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid MO ID"})
		return
	}
	mo, err := h.svc.ActionProduce(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Manufacturing order produced successfully. Components consumed and finished goods added to inventory.",
		"mo":      mo,
	})
}
