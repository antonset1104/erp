package esg

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"erp-golang/internal/modules/base"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type VendorSustainabilityScorecard struct {
	VendorID                uint    `json:"vendor_id"`
	VendorName              string  `json:"vendor_name"`
	Country                 string  `json:"country"`
	CarbonEmissionIntensity float64 `json:"carbon_emission_intensity"` // kg CO2e / unit
	BenchmarkIntensity      float64 `json:"benchmark_intensity"`      // Industry avg
	ISO14001Certified       bool    `json:"iso14001_certified"`
	ISO50001Certified       bool    `json:"iso50001_certified"`
	RecycledPackagingPct    float64 `json:"recycled_packaging_pct"`
	RenewableEnergySharePct float64 `json:"renewable_energy_share_pct"`
	LogisticsGreenMode      string  `json:"logistics_green_mode"`
	CompositeGreenScore     float64 `json:"composite_green_score"` // 0 - 100
	SustainabilityTier      string  `json:"sustainability_tier"`   // "PLATINUM_LEADER", "GOLD_ADVANCED", "SILVER_COMPLIANT", "BRONZE_BASIC", "NON_COMPLIANT"
	ProcurementBonusScore   float64 `json:"procurement_bonus_score"` // 0 - 10 points in RFQ bidding
	Scope3Factor            float64 `json:"scope3_factor"`          // Multiplier for corporate Scope 3 GHG
	AssessmentDate          string  `json:"assessment_date"`
	AuditedBy               string  `json:"audited_by"`
}

type VendorAssessmentRequest struct {
	VendorID                uint    `json:"vendor_id" binding:"required"`
	VendorName              string  `json:"vendor_name"`
	Country                 string  `json:"country"`
	CarbonEmissionIntensity float64 `json:"carbon_emission_intensity" binding:"required"`
	BenchmarkIntensity      float64 `json:"benchmark_intensity"`
	ISO14001Certified       bool    `json:"iso14001_certified"`
	ISO50001Certified       bool    `json:"iso50001_certified"`
	RecycledPackagingPct    float64 `json:"recycled_packaging_pct"`
	RenewableEnergySharePct float64 `json:"renewable_energy_share_pct"`
	LogisticsGreenMode      string  `json:"logistics_green_mode"` // "ELECTRIC_EV", "RAIL_INTERMODAL", "MARITIME", "AIR_FREIGHT"
}

var esgVendorStore = struct {
	sync.RWMutex
	scorecards map[uint]*VendorSustainabilityScorecard
}{
	scorecards: make(map[uint]*VendorSustainabilityScorecard),
}

func initDefaultScorecards() {
	esgVendorStore.Lock()
	defer esgVendorStore.Unlock()

	if len(esgVendorStore.scorecards) > 0 {
		return
	}

	esgVendorStore.scorecards[1] = &VendorSustainabilityScorecard{
		VendorID:                1,
		VendorName:              "PT Pelanggan Sejahtera (Vendor Divisi Display)",
		Country:                 "Indonesia",
		CarbonEmissionIntensity: 14.2,
		BenchmarkIntensity:      28.5,
		ISO14001Certified:       true,
		ISO50001Certified:       true,
		RecycledPackagingPct:    85.0,
		RenewableEnergySharePct: 60.0,
		LogisticsGreenMode:      "ELECTRIC_EV",
		CompositeGreenScore:     91.5,
		SustainabilityTier:      "PLATINUM_LEADER",
		ProcurementBonusScore:   9.5,
		Scope3Factor:            0.78,
		AssessmentDate:          time.Now().Format("2006-01-02 15:04:05"),
		AuditedBy:               "GoERP Automated Sustainability Auditor",
	}

	esgVendorStore.scorecards[2] = &VendorSustainabilityScorecard{
		VendorID:                2,
		VendorName:              "PT Pemasok Komponen Global",
		Country:                 "Singapore",
		CarbonEmissionIntensity: 26.0,
		BenchmarkIntensity:      28.5,
		ISO14001Certified:       true,
		ISO50001Certified:       false,
		RecycledPackagingPct:    50.0,
		RenewableEnergySharePct: 25.0,
		LogisticsGreenMode:      "MARITIME",
		CompositeGreenScore:     72.0,
		SustainabilityTier:      "SILVER_COMPLIANT",
		ProcurementBonusScore:   5.0,
		Scope3Factor:            0.95,
		AssessmentDate:          time.Now().Format("2006-01-02 15:04:05"),
		AuditedBy:               "GoERP Automated Sustainability Auditor",
	}
}

// AssessVendorSustainability scores vendor green credentials
func AssessVendorSustainability(db *gorm.DB, req *VendorAssessmentRequest) *VendorSustainabilityScorecard {
	esgVendorStore.Lock()
	defer esgVendorStore.Unlock()

	vName := req.VendorName
	if vName == "" {
		var p base.Partner
		if err := db.Where("id = ?", req.VendorID).First(&p).Error; err == nil && p.Name != "" {
			vName = p.Name
		} else {
			vName = fmt.Sprintf("Vendor #%d", req.VendorID)
		}
	}

	benchmark := req.BenchmarkIntensity
	if benchmark <= 0 {
		benchmark = 28.5 // Industry default standard
	}

	// 1. Carbon intensity score (35% weight)
	carbonScore := 50.0
	if req.CarbonEmissionIntensity < benchmark {
		carbonScore = math.Min(100.0, 50.0+(1.0-(req.CarbonEmissionIntensity/benchmark))*50.0)
	} else {
		carbonScore = math.Max(10.0, 50.0-((req.CarbonEmissionIntensity-benchmark)/benchmark)*40.0)
	}

	// 2. ISO Certifications (25% weight)
	certScore := 0.0
	if req.ISO14001Certified {
		certScore += 50.0
	}
	if req.ISO50001Certified {
		certScore += 50.0
	}

	// 3. Circular packaging & renewable energy (25% weight)
	pkgScore := (math.Min(100.0, req.RecycledPackagingPct) * 0.5) + (math.Min(100.0, req.RenewableEnergySharePct) * 0.5)

	// 4. Logistics Green Mode (15% weight)
	logisticsScore := 50.0
	switch req.LogisticsGreenMode {
	case "ELECTRIC_EV":
		logisticsScore = 100.0
	case "RAIL_INTERMODAL":
		logisticsScore = 85.0
	case "MARITIME":
		logisticsScore = 70.0
	case "AIR_FREIGHT":
		logisticsScore = 20.0
	}

	compScore := (carbonScore * 0.35) + (certScore * 0.25) + (pkgScore * 0.25) + (logisticsScore * 0.15)
	compScore = math.Round(compScore*10) / 10

	var tier string
	var bonus float64
	var scope3Factor float64

	if compScore >= 85.0 {
		tier = "PLATINUM_LEADER"
		bonus = 10.0
		scope3Factor = 0.75
	} else if compScore >= 70.0 {
		tier = "GOLD_ADVANCED"
		bonus = 7.0
		scope3Factor = 0.88
	} else if compScore >= 55.0 {
		tier = "SILVER_COMPLIANT"
		bonus = 4.0
		scope3Factor = 1.00
	} else if compScore >= 40.0 {
		tier = "BRONZE_BASIC"
		bonus = 2.0
		scope3Factor = 1.15
	} else {
		tier = "NON_COMPLIANT"
		bonus = 0.0
		scope3Factor = 1.35
	}

	card := &VendorSustainabilityScorecard{
		VendorID:                req.VendorID,
		VendorName:              vName,
		Country:                 req.Country,
		CarbonEmissionIntensity: req.CarbonEmissionIntensity,
		BenchmarkIntensity:      benchmark,
		ISO14001Certified:       req.ISO14001Certified,
		ISO50001Certified:       req.ISO50001Certified,
		RecycledPackagingPct:    req.RecycledPackagingPct,
		RenewableEnergySharePct: req.RenewableEnergySharePct,
		LogisticsGreenMode:      req.LogisticsGreenMode,
		CompositeGreenScore:     compScore,
		SustainabilityTier:      tier,
		ProcurementBonusScore:   bonus,
		Scope3Factor:            scope3Factor,
		AssessmentDate:          time.Now().Format("2006-01-02 15:04:05"),
		AuditedBy:               "GoERP Automated Sustainability Auditor",
	}

	esgVendorStore.scorecards[req.VendorID] = card
	return card
}

// Handler functions for HTTP routes
type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	initDefaultScorecards()
	return &Handler{db: db}
}

func (h *Handler) ListVendorScorecards(c *gin.Context) {
	esgVendorStore.RLock()
	defer esgVendorStore.RUnlock()

	var list []VendorSustainabilityScorecard
	for _, card := range esgVendorStore.scorecards {
		list = append(list, *card)
	}

	c.JSON(http.StatusOK, gin.H{
		"total_vendors_assessed": len(list),
		"scorecards":             list,
	})
}

func (h *Handler) SubmitAssessment(c *gin.Context) {
	var req VendorAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	card := AssessVendorSustainability(h.db, &req)
	c.JSON(http.StatusCreated, card)
}

// RegisterESGVendorRoutes registers supplier sustainability endpoints
func RegisterESGVendorRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewHandler(db)
	group := router.Group("/esg")
	group.Use(authMiddleware)
	{
		group.GET("/vendor-scorecards", h.ListVendorScorecards)
		group.POST("/vendor-assessments", h.SubmitAssessment)
	}
}
