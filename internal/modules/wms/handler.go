package wms

import (
	"fmt"
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

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMW gin.HandlerFunc) {
	group := r.Group("/wms")
	if authMW != nil {
		group.Use(authMW)
	}
	{
		group.GET("/stats", h.GetStats)

		// Zones
		group.GET("/zones", h.ListZones)
		group.POST("/zones", h.CreateZone)

		// Bins
		group.GET("/bins", h.ListBins)
		group.POST("/bins", h.CreateBin)
		group.GET("/bins/:id", h.GetBin)

		// Putaway Rules & Suggestions
		group.GET("/putaway-rules", h.ListPutawayRules)
		group.POST("/putaway-rules", h.CreatePutawayRule)
		group.POST("/putaway/suggest", h.SuggestPutaway)
		group.POST("/putaway/assign", h.AssignToBin)

		// Wave Pickings
		group.GET("/waves", h.ListWaves)
		group.POST("/waves", h.CreateWave)
		group.GET("/waves/:id", h.GetWave)
		group.POST("/waves/:id/start", h.StartWave)
		group.POST("/waves/:id/lines/:line_id/pick", h.PickLine)
		group.POST("/waves/:id/pick/:line_id", h.PickLine)
		group.POST("/waves/:id/complete", h.CompleteWave)

		// AI Slotting Advisor
		group.GET("/ai-slotting-advisor", h.GetAISlottingAdvisor)
		group.GET("/ai/slotting-advisor", h.GetAISlottingAdvisor)
	}
}

func (h *Handler) GetStats(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.Query("warehouse_id"), 10, 64)
	stats, err := h.service.GetStats(uint(warehouseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// ---------------- Zones ----------------

func (h *Handler) ListZones(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.Query("warehouse_id"), 10, 64)
	zones, err := h.service.ListZones(uint(warehouseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": zones})
}

func (h *Handler) CreateZone(c *gin.Context) {
	var req CreateZoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	zone, err := h.service.CreateZone(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": zone, "message": fmt.Sprintf("Zona %s berhasil dibuat", zone.Code)})
}

// ---------------- Bins ----------------

func (h *Handler) ListBins(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.Query("warehouse_id"), 10, 64)
	zoneID, _ := strconv.ParseUint(c.Query("zone_id"), 10, 64)
	status := c.Query("status")

	bins, err := h.service.ListBins(uint(warehouseID), uint(zoneID), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bins})
}

func (h *Handler) CreateBin(c *gin.Context) {
	var req CreateBinRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	bin, err := h.service.CreateBin(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": bin, "message": fmt.Sprintf("Bin Location %s berhasil didaftarkan", bin.Code)})
}

func (h *Handler) GetBin(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID bin tidak valid"})
		return
	}
	bin, err := h.service.GetBin(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bin location tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": bin})
}

// ---------------- Putaway ----------------

func (h *Handler) ListPutawayRules(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.Query("warehouse_id"), 10, 64)
	rules, err := h.service.ListPutawayRules(uint(warehouseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

func (h *Handler) CreatePutawayRule(c *gin.Context) {
	var req CreatePutawayRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rule, err := h.service.CreatePutawayRule(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": rule, "message": "Aturan putaway otomatis berhasil disimpan"})
}

func (h *Handler) SuggestPutaway(c *gin.Context) {
	var req SuggestPutawayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sug, err := h.service.SuggestPutawayLocation(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sug})
}

func (h *Handler) AssignToBin(c *gin.Context) {
	var body struct {
		BinID     uint    `json:"bin_id" binding:"required"`
		ProductID uint    `json:"product_id" binding:"required"`
		Quantity  float64 `json:"quantity" binding:"required"`
		WeightKg  float64 `json:"weight_kg"`
		VolumeCbm float64 `json:"volume_cbm"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.AssignInventoryToBin(body.BinID, body.ProductID, body.Quantity, body.WeightKg, body.VolumeCbm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Barang berhasil dialokasikan ke bin lokasi"})
}

// ---------------- Waves ----------------

func (h *Handler) ListWaves(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.Query("warehouse_id"), 10, 64)
	status := c.Query("status")
	waves, err := h.service.ListWavePickings(uint(warehouseID), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": waves})
}

func (h *Handler) CreateWave(c *gin.Context) {
	var req CreateWavePickingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	wave, err := h.service.CreateWavePicking(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": wave, "message": fmt.Sprintf("Wave picking %s berhasil dibuat", wave.WaveNumber)})
}

func (h *Handler) GetWave(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID wave tidak valid"})
		return
	}
	wave, err := h.service.GetWavePicking(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Wave picking tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": wave})
}

func (h *Handler) StartWave(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID wave tidak valid"})
		return
	}
	wave, err := h.service.StartWavePicking(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": wave, "message": fmt.Sprintf("Wave picking %s dimulai", wave.WaveNumber)})
}

func (h *Handler) PickLine(c *gin.Context) {
	waveID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID wave tidak valid"})
		return
	}
	lineID, err := strconv.ParseUint(c.Param("line_id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID line tidak valid"})
		return
	}

	var req PickLineRequest
	_ = c.ShouldBindJSON(&req)

	line, err := h.service.ConfirmPickLine(uint(waveID), uint(lineID), req.QuantityPicked)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": line, "message": "Konfirmasi pengambilan item berhasil dicatat"})
}

func (h *Handler) CompleteWave(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID wave tidak valid"})
		return
	}
	wave, err := h.service.CompleteWavePicking(uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": wave, "message": fmt.Sprintf("Wave picking %s berhasil diselesaikan", wave.WaveNumber)})
}

func (h *Handler) GetAISlottingAdvisor(c *gin.Context) {
	warehouseID, _ := strconv.ParseUint(c.Query("warehouse_id"), 10, 64)
	report, err := h.service.AnalyzeWarehouseSlottingWithAI(uint(warehouseID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": report})
}
