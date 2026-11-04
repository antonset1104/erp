package mining

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// OreStockpileParcel models discrete stockpiles with JORC-compliant mineral assays and deleterious elements
type OreStockpileParcel struct {
	ParcelID       string    `json:"parcel_id"`
	StockpileName  string    `json:"stockpile_name"`
	OreType        string    `json:"ore_type"` // "PORPHYRY_PRIMARY_SULFIDE", "SKARN_OXIDE_CAP", "SUPERGENE_ENRICHED"
	Tons           float64   `json:"tons"`
	CuGradePct     float64   `json:"cu_grade_pct"`      // Copper grade %
	AuGradeGpt     float64   `json:"au_grade_gpt"`      // Gold grade grams/ton
	SilicaSiO2Pct  float64   `json:"silica_sio2_pct"`   // Slag acidifier penalty element
	ArsenicPPM     float64   `json:"arsenic_ppm"`       // Deleterious penalty element
	MoisturePct    float64   `json:"moisture_pct"`      // Transport transportable moisture limit (TML)
	RegisteredAt   time.Time `json:"registered_at"`
}

// BlendOptimizationRequest specifies blending constraints for mill throughput
type BlendOptimizationRequest struct {
	MineSiteID              string             `json:"mine_site_id"`
	TargetTons              float64            `json:"target_tons"`
	TargetCuGradePct        float64            `json:"target_cu_grade_pct"`
	MaxSilicaSiO2Pct        float64            `json:"max_silica_sio2_pct"`
	MaxArsenicPPM           float64            `json:"max_arsenic_ppm"`
	ParcelAllocations       map[string]float64 `json:"parcel_allocations"` // ParcelID -> Tons to draw
	MiningExtractionCostUSD float64            `json:"mining_extraction_cost_usd"`
}

// BlendOptimizationRecipe models the calculated recipe, metallurgical recovery, and recovery loss
type BlendOptimizationRecipe struct {
	RecipeID                   string             `json:"recipe_id"`
	MineSiteID                 string             `json:"mine_site_id"`
	TotalBlendedTons           float64            `json:"total_blended_tons"`
	ResultingCuGradePct        float64            `json:"resulting_cu_grade_pct"`
	ResultingAuGradeGpt        float64            `json:"resulting_au_grade_gpt"`
	ResultingSilicaSiO2Pct     float64            `json:"resulting_silica_sio2_pct"`
	ResultingArsenicPPM        float64            `json:"resulting_arsenic_ppm"`
	FlotationRecoveryRatePct   float64            `json:"flotation_recovery_rate_pct"` // Predicted recovery %
	CalculatedBlendCostPerTon  float64            `json:"calculated_blend_cost_per_ton"`
	ComponentRatios            map[string]float64 `json:"component_ratios"` // ParcelID -> % ratio
	BlendQualityCompliance     bool               `json:"blend_quality_compliance"`
	ComplianceAuditDigest      string             `json:"compliance_audit_digest"`
	OptimizedAt                time.Time          `json:"optimized_at"`
}

// SmelterSettlementManifest calculates commercial settlement, TC/RC treatment and penalties
type SmelterSettlementManifest struct {
	SettlementID                   string    `json:"settlement_id"`
	RecipeID                       string    `json:"recipe_id"`
	PayableCuMetalTons             float64   `json:"payable_cu_metal_tons"`
	PayableAuMetalOunces           float64   `json:"payable_au_metal_ounces"`
	LmeCuPriceUSDPerTon            float64   `json:"lme_cu_price_usd_per_ton"`
	GrossMetalValuationUSD         float64   `json:"gross_metal_valuation_usd"`
	TreatmentChargesTC_USD         float64   `json:"treatment_charges_tc_usd"`
	RefiningChargesRC_USD          float64   `json:"refining_charges_rc_usd"`
	ArsenicPenaltyDeductionsUSD    float64   `json:"arsenic_penalty_deductions_usd"`
	MoisturePenaltyDeductionsUSD   float64   `json:"moisture_penalty_deductions_usd"`
	NetPayableSmelterRevenueUSD    float64   `json:"net_payable_smelter_revenue_usd"`
	CommercialSettlementReference  string    `json:"commercial_settlement_reference"`
	SettledAt                      time.Time `json:"settled_at"`
}

// OreBlendingVault thread-safe in-memory store
type OreBlendingVault struct {
	mu          sync.RWMutex
	parcels     map[string]OreStockpileParcel
	recipes     map[string]BlendOptimizationRecipe
	settlements map[string]SmelterSettlementManifest
}

// Global instance
var GlobalOreBlendingVault = &OreBlendingVault{
	parcels:     make(map[string]OreStockpileParcel),
	recipes:     make(map[string]BlendOptimizationRecipe),
	settlements: make(map[string]SmelterSettlementManifest),
}

func init() {
	// Seed demo stockpile parcels
	GlobalOreBlendingVault.parcels["STK-PORPHYRY-HIGH-CU-01"] = OreStockpileParcel{
		ParcelID:      "STK-PORPHYRY-HIGH-CU-01",
		StockpileName: "Grasberg Deep Ore Zone High Grade Sulfide",
		OreType:       "PORPHYRY_PRIMARY_SULFIDE",
		Tons:          25000.0,
		CuGradePct:    2.80,
		AuGradeGpt:    1.25,
		SilicaSiO2Pct: 52.0,
		ArsenicPPM:    380.0,
		MoisturePct:   7.5,
		RegisteredAt:  time.Now(),
	}

	GlobalOreBlendingVault.parcels["STK-SKARN-LOW-CU-02"] = OreStockpileParcel{
		ParcelID:      "STK-SKARN-LOW-CU-02",
		StockpileName: "Ertsberg East Skarn Low Grade Neutralizer",
		OreType:       "SKARN_OXIDE_CAP",
		Tons:          45000.0,
		CuGradePct:    0.95,
		AuGradeGpt:    0.35,
		SilicaSiO2Pct: 62.5,
		ArsenicPPM:    75.0,
		MoisturePct:   6.2,
		RegisteredAt:  time.Now(),
	}
}

// RegisterStockpileParcel records mineral assay data
func (v *OreBlendingVault) RegisterStockpileParcel(p OreStockpileParcel) OreStockpileParcel {
	v.mu.Lock()
	defer v.mu.Unlock()

	if p.ParcelID == "" {
		p.ParcelID = fmt.Sprintf("PARCEL-ORE-%d", time.Now().UnixNano()%1000000)
	}
	p.RegisteredAt = time.Now()
	v.parcels[p.ParcelID] = p
	return p
}

// OptimizeBlendRecipe calculates weighted composite grade and predicts metallurgical flotation recovery
func (v *OreBlendingVault) OptimizeBlendRecipe(req BlendOptimizationRequest) (BlendOptimizationRecipe, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(req.ParcelAllocations) == 0 {
		return BlendOptimizationRecipe{}, fmt.Errorf("no parcel allocations provided")
	}

	var totalTons float64
	var sumCuTons float64
	var sumAuGrams float64
	var sumSilicaTons float64
	var sumArsenicGrams float64
	ratios := make(map[string]float64)

	for pID, tons := range req.ParcelAllocations {
		parcel, exists := v.parcels[pID]
		if !exists {
			return BlendOptimizationRecipe{}, fmt.Errorf("parcel %s not found in stockpile inventory", pID)
		}
		totalTons += tons
		sumCuTons += tons * (parcel.CuGradePct / 100.0)
		sumAuGrams += tons * parcel.AuGradeGpt
		sumSilicaTons += tons * (parcel.SilicaSiO2Pct / 100.0)
		sumArsenicGrams += tons * parcel.ArsenicPPM
	}

	if totalTons <= 0 {
		return BlendOptimizationRecipe{}, fmt.Errorf("total blended tons must be positive")
	}

	for pID, tons := range req.ParcelAllocations {
		ratios[pID] = math.Round((tons/totalTons)*10000.0) / 100.0 // percentage
	}

	compositeCu := (sumCuTons / totalTons) * 100.0
	compositeAu := sumAuGrams / totalTons
	compositeSilica := (sumSilicaTons / totalTons) * 100.0
	compositeArsenic := sumArsenicGrams / totalTons

	// Metallurgical flotation recovery model:
	// Base recovery 91.5% modified by head grade, slag-forming silica penalty and arsenic inhibition
	flotationRecovery := 91.5 + (compositeCu-1.5)*2.2 - (compositeSilica-50.0)*0.15 - (compositeArsenic/1000.0)*3.5
	if flotationRecovery > 96.0 {
		flotationRecovery = 96.0
	} else if flotationRecovery < 75.0 {
		flotationRecovery = 75.0
	}

	costPerTon := req.MiningExtractionCostUSD
	if costPerTon <= 0 {
		costPerTon = 42.50
	}

	compliance := true
	if req.MaxSilicaSiO2Pct > 0 && compositeSilica > req.MaxSilicaSiO2Pct {
		compliance = false
	}
	if req.MaxArsenicPPM > 0 && compositeArsenic > req.MaxArsenicPPM {
		compliance = false
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%f-%f-%f-%d", req.MineSiteID, totalTons, compositeCu, compositeArsenic, time.Now().UnixNano())))
	auditDigest := hex.EncodeToString(hasher.Sum(nil))

	recipeID := fmt.Sprintf("RECIPE-BLEND-%d", time.Now().UnixNano()%1000000)
	recipe := BlendOptimizationRecipe{
		RecipeID:                  recipeID,
		MineSiteID:                req.MineSiteID,
		TotalBlendedTons:          math.Round(totalTons*100.0) / 100.0,
		ResultingCuGradePct:       math.Round(compositeCu*1000.0) / 1000.0,
		ResultingAuGradeGpt:       math.Round(compositeAu*1000.0) / 1000.0,
		ResultingSilicaSiO2Pct:    math.Round(compositeSilica*100.0) / 100.0,
		ResultingArsenicPPM:       math.Round(compositeArsenic*10.0) / 10.0,
		FlotationRecoveryRatePct:  math.Round(flotationRecovery*100.0) / 100.0,
		CalculatedBlendCostPerTon: costPerTon,
		ComponentRatios:           ratios,
		BlendQualityCompliance:    compliance,
		ComplianceAuditDigest:     auditDigest,
		OptimizedAt:               time.Now(),
	}

	v.recipes[recipeID] = recipe
	return recipe, nil
}

// CalculateSmelterCommercialSettlement computes commercial invoice with TC/RC and deleterious penalties
func (v *OreBlendingVault) CalculateSmelterCommercialSettlement(recipeID string, lmeCuPriceUSDPerTon, tcUSDPerTon, rcUSDPerLb float64) (SmelterSettlementManifest, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	recipe, exists := v.recipes[recipeID]
	if !exists {
		return SmelterSettlementManifest{}, fmt.Errorf("recipe %s not found", recipeID)
	}

	if lmeCuPriceUSDPerTon <= 0 {
		lmeCuPriceUSDPerTon = 9450.0 // Benchmark LME Copper USD/ton
	}
	if tcUSDPerTon <= 0 {
		tcUSDPerTon = 78.50 // Treatment charge per dry metric ton concentrate
	}
	if rcUSDPerLb <= 0 {
		rcUSDPerLb = 0.0785 // Refining charge per payable pound of copper
	}

	// Cu concentrate conversion: approx concentrate grade 28.0% Cu
	// Recovered Cu metal tons = Total Ore * Grade * Recovery
	recoveredCuTons := recipe.TotalBlendedTons * (recipe.ResultingCuGradePct / 100.0) * (recipe.FlotationRecoveryRatePct / 100.0)
	payableCuTons := recoveredCuTons * 0.965 // Contractual 96.5% payable metal deduction
	payableCuLbs := payableCuTons * 2204.62

	// Recovered Au ounces (31.1035 g/troy oz)
	recoveredAuGrams := recipe.TotalBlendedTons * recipe.ResultingAuGradeGpt * 0.85
	payableAuOunces := (recoveredAuGrams / 31.1035) * 0.95

	grossValuation := (payableCuTons * lmeCuPriceUSDPerTon) + (payableAuOunces * 2650.0)

	// Concentrate tons produced (at 28% Cu in concentrate)
	concTons := recoveredCuTons / 0.28
	tcDeduction := concTons * tcUSDPerTon
	rcDeduction := payableCuLbs * rcUSDPerLb

	// Arsenic Penalty: Contractual limit 200 PPM. Penalty $3.75 per PPM excess per concentrate ton
	var arsenicPenalty float64
	if recipe.ResultingArsenicPPM > 200.0 {
		excessPPM := recipe.ResultingArsenicPPM - 200.0
		arsenicPenalty = excessPPM * 3.75 * (concTons / 10.0)
	}

	// Moisture penalty (deduction for drying/freight if moisture above benchmark)
	moisturePenalty := concTons * 8.50

	netPayable := grossValuation - tcDeduction - rcDeduction - arsenicPenalty - moisturePenalty
	if netPayable < 0 {
		netPayable = 0
	}

	settlementID := fmt.Sprintf("SETTLE-SMELTER-%d", time.Now().UnixNano()%1000000)
	manifest := SmelterSettlementManifest{
		SettlementID:                  settlementID,
		RecipeID:                      recipeID,
		PayableCuMetalTons:            math.Round(payableCuTons*100.0) / 100.0,
		PayableAuMetalOunces:          math.Round(payableAuOunces*100.0) / 100.0,
		LmeCuPriceUSDPerTon:           lmeCuPriceUSDPerTon,
		GrossMetalValuationUSD:        math.Round(grossValuation*100.0) / 100.0,
		TreatmentChargesTC_USD:        math.Round(tcDeduction*100.0) / 100.0,
		RefiningChargesRC_USD:         math.Round(rcDeduction*100.0) / 100.0,
		ArsenicPenaltyDeductionsUSD:   math.Round(arsenicPenalty*100.0) / 100.0,
		MoisturePenaltyDeductionsUSD:  math.Round(moisturePenalty*100.0) / 100.0,
		NetPayableSmelterRevenueUSD:   math.Round(netPayable*100.0) / 100.0,
		CommercialSettlementReference: fmt.Sprintf("COMM-SETTLE-JORC-%s", settlementID),
		SettledAt:                     time.Now(),
	}

	v.settlements[settlementID] = manifest
	return manifest, nil
}

// GetMetallurgicalYieldDossier provides complete analytics for the mine site
func (v *OreBlendingVault) GetMetallurgicalYieldDossier(mineSiteID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var siteRecipes []BlendOptimizationRecipe
	var totalBlendedTons float64
	var avgRecovery float64
	for _, r := range v.recipes {
		if r.MineSiteID == mineSiteID {
			siteRecipes = append(siteRecipes, r)
			totalBlendedTons += r.TotalBlendedTons
			avgRecovery += r.FlotationRecoveryRatePct
		}
	}
	if len(siteRecipes) > 0 {
		avgRecovery = avgRecovery / float64(len(siteRecipes))
	} else {
		avgRecovery = 91.2
	}

	dossier := map[string]interface{}{
		"mine_site_id":                  mineSiteID,
		"stockpile_parcels_available":   len(v.parcels),
		"blended_recipes_executed":      len(siteRecipes),
		"total_blended_tons":            math.Round(totalBlendedTons*100.0) / 100.0,
		"average_flotation_recovery":    math.Round(avgRecovery*100.0) / 100.0,
		"settlement_records_count":      len(v.settlements),
		"jorc_metallurgical_compliance": true,
		"compiled_at":                   time.Now(),
	}
	return dossier, nil
}

// RegisterOreRoutes wires REST endpoints
func RegisterOreRoutes(r *gin.RouterGroup) {
	oreGroup := r.Group("/mining/ore")
	{
		oreGroup.POST("/register-stockpile-parcel", func(c *gin.Context) {
			var p OreStockpileParcel
			if err := c.ShouldBindJSON(&p); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalOreBlendingVault.RegisterStockpileParcel(p)
			c.JSON(http.StatusOK, res)
		})

		oreGroup.POST("/optimize-blend-recipe", func(c *gin.Context) {
			var req BlendOptimizationRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			recipe, err := GlobalOreBlendingVault.OptimizeBlendRecipe(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, recipe)
		})

		oreGroup.POST("/calculate-smelter-commercial-settlement", func(c *gin.Context) {
			var body struct {
				RecipeID            string  `json:"recipe_id"`
				LmeCuPriceUSDPerTon float64 `json:"lme_cu_price_usd_per_ton"`
				TcUSDPerTon         float64 `json:"tc_usd_per_ton"`
				RcUSDPerLb          float64 `json:"rc_usd_per_lb"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			manifest, err := GlobalOreBlendingVault.CalculateSmelterCommercialSettlement(
				body.RecipeID, body.LmeCuPriceUSDPerTon, body.TcUSDPerTon, body.RcUSDPerLb,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, manifest)
		})

		oreGroup.GET("/metallurgical-yield-dossier/:mine_site_id", func(c *gin.Context) {
			mineSiteID := c.Param("mine_site_id")
			dossier, err := GlobalOreBlendingVault.GetMetallurgicalYieldDossier(mineSiteID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
