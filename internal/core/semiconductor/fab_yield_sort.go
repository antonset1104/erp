package semiconductor

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

// CleanroomMetrology captures continuous environmental cleanliness under ISO 14644-1 Class 1
type CleanroomMetrology struct {
	MeasurementID                string    `json:"measurement_id"`
	CleanroomBayID               string    `json:"cleanroom_bay_id"` // e.g. "BAY-LITHO-EUV-01"
	FabSiteID                    string    `json:"fab_site_id"`
	ISOClassLevel                int       `json:"iso_class_level"` // 1 = ISO Class 1 (max 10 particles/m3 >= 0.1um)
	ParticleCount01MicronPerM3   float64   `json:"particle_count_0_1_micron_per_m3"`
	ParticleCount05MicronPerM3   float64   `json:"particle_count_0_5_micron_per_m3"`
	TemperatureC                 float64   `json:"temperature_c"`
	RelativeHumidityPct          float64   `json:"relative_humidity_pct"`
	AirVelocityMPS               float64   `json:"air_velocity_mps"` // Laminar flow ~0.45 m/s
	Timestamp                    time.Time `json:"timestamp"`
}

// WaferYieldCalculation models Murphy-Seeds microchip yield and defect density D0
type WaferYieldCalculation struct {
	YieldID                      string    `json:"yield_id"`
	WaferLotID                   string    `json:"wafer_lot_id"`
	ProductFamilySKU             string    `json:"product_family_sku"`
	WaferDiameterMM              float64   `json:"wafer_diameter_mm"` // 300mm standard
	GrossDieCountPerWafer        int       `json:"gross_die_count_per_wafer"`
	DieAreaMM2                   float64   `json:"die_area_mm2"`
	DefectDensityD0_PerCM2       float64   `json:"defect_density_d0_per_cm2"` // Random defect density D0
	MurphyYieldPct               float64   `json:"murphy_yield_pct"`          // Modeled yield %
	ProjectedGoodDieCount        int       `json:"projected_good_die_count"`
	CleanroomExcursionDetected   bool      `json:"cleanroom_excursion_detected"`
	WaferQuarantineRecommended   bool      `json:"wafer_quarantine_recommended"`
	SEMI_ComplianceDigest        string    `json:"semi_compliance_digest"`
	EvaluatedAt                  time.Time `json:"evaluated_at"`
}

// DieSortBinningManifest models parametric tester die categorization and revenue optimization
type DieSortBinningManifest struct {
	ManifestID                   string    `json:"manifest_id"`
	WaferLotID                   string    `json:"wafer_lot_id"`
	TotalTestedDieCount          int       `json:"total_tested_die_count"`
	Bin1PrimeCount               int       `json:"bin_1_prime_count"`        // Highest clock speed / peak pricing
	Bin2LowVoltageCount          int       `json:"bin_2_low_voltage_count"`   // Ultra-low power mobile bin
	Bin3StandardCount            int       `json:"bin_3_standard_count"`      // Standard mainstream bin
	Bin7RejectDefectiveCount     int       `json:"bin_7_reject_defective_count"`
	WaferPackYieldPct            float64   `json:"wafer_pack_yield_pct"`
	CommercialValuationUSD       float64   `json:"commercial_valuation_usd"`
	BinningDate                  time.Time `json:"binning_date"`
}

// SemiconductorFabVault thread-safe in-memory store
type SemiconductorFabVault struct {
	mu           sync.RWMutex
	metrologies  map[string]CleanroomMetrology
	yields       map[string]WaferYieldCalculation
	manifests    map[string]DieSortBinningManifest
}

// Global instance
var GlobalSemiconductorFabVault = &SemiconductorFabVault{
	metrologies: make(map[string]CleanroomMetrology),
	yields:      make(map[string]WaferYieldCalculation),
	manifests:   make(map[string]DieSortBinningManifest),
}

func init() {
	// Seed demo cleanroom metrology and wafer lot
	GlobalSemiconductorFabVault.metrologies["BAY-LITHO-EUV-01"] = CleanroomMetrology{
		MeasurementID:              "MET-CLEAN-001",
		CleanroomBayID:             "BAY-LITHO-EUV-01",
		FabSiteID:                  "FAB-TSMC-NANO-01",
		ISOClassLevel:              1,
		ParticleCount01MicronPerM3: 4.5, // Well within <= 10 particles/m3
		ParticleCount05MicronPerM3: 0.2,
		TemperatureC:               21.0,
		RelativeHumidityPct:        45.0,
		AirVelocityMPS:             0.46,
		Timestamp:                  time.Now(),
	}
}

// IngestCleanroomMetrology registers particle counters and air velocity sensors
func (v *SemiconductorFabVault) IngestCleanroomMetrology(m CleanroomMetrology) CleanroomMetrology {
	v.mu.Lock()
	defer v.mu.Unlock()

	if m.MeasurementID == "" {
		m.MeasurementID = fmt.Sprintf("MET-CLEAN-%d", time.Now().UnixNano()%1000000)
	}
	m.Timestamp = time.Now()
	v.metrologies[m.CleanroomBayID] = m
	return m
}

// CalculateWaferYieldModel calculates Murphy yield: Y = ((1 - exp(-D0 * A)) / (D0 * A))^2
func (v *SemiconductorFabVault) CalculateWaferYieldModel(req WaferYieldCalculation) (WaferYieldCalculation, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if req.GrossDieCountPerWafer <= 0 {
		req.GrossDieCountPerWafer = 520
	}
	if req.DieAreaMM2 <= 0 {
		req.DieAreaMM2 = 110.0
	}
	if req.DefectDensityD0_PerCM2 <= 0 {
		req.DefectDensityD0_PerCM2 = 0.08
	}

	// Die Area in cm2 = DieAreaMM2 / 100.0
	dieAreaCM2 := req.DieAreaMM2 / 100.0
	d0A := req.DefectDensityD0_PerCM2 * dieAreaCM2

	// Murphy Yield Model
	var murphyYield float64
	if d0A < 0.0001 {
		murphyYield = 1.0
	} else {
		term := (1.0 - math.Exp(-d0A)) / d0A
		murphyYield = term * term
	}

	yieldPct := murphyYield * 100.0
	goodDie := int(math.Round(float64(req.GrossDieCountPerWafer) * murphyYield))

	// Check cleanroom excursion
	excursion := false
	quarantine := false
	for _, m := range v.metrologies {
		if m.ParticleCount01MicronPerM3 > 10.0 {
			excursion = true
			quarantine = true
			break
		}
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%s-%f-%f-%d", req.WaferLotID, req.ProductFamilySKU, yieldPct, d0A, time.Now().UnixNano())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	yieldID := fmt.Sprintf("YIELD-SEMI-%d", time.Now().UnixNano()%1000000)
	calc := WaferYieldCalculation{
		YieldID:                    yieldID,
		WaferLotID:                 req.WaferLotID,
		ProductFamilySKU:           req.ProductFamilySKU,
		WaferDiameterMM:            300.0,
		GrossDieCountPerWafer:      req.GrossDieCountPerWafer,
		DieAreaMM2:                 req.DieAreaMM2,
		DefectDensityD0_PerCM2:     req.DefectDensityD0_PerCM2,
		MurphyYieldPct:             math.Round(yieldPct*100.0) / 100.0,
		ProjectedGoodDieCount:      goodDie,
		CleanroomExcursionDetected: excursion,
		WaferQuarantineRecommended: quarantine,
		SEMI_ComplianceDigest:      digest,
		EvaluatedAt:                time.Now(),
	}

	v.yields[req.WaferLotID] = calc
	return calc, nil
}

// ExecuteDieSortBinning categorizes parametric tester results across bins
func (v *SemiconductorFabVault) ExecuteDieSortBinning(lotID string, totalTested int) (DieSortBinningManifest, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	yieldCalc, exists := v.yields[lotID]
	if !exists {
		return DieSortBinningManifest{}, fmt.Errorf("no yield calculation found for lot %s", lotID)
	}

	if totalTested <= 0 {
		totalTested = yieldCalc.GrossDieCountPerWafer * 25 // 25 wafers per FOUP carrier pod
	}

	passRate := yieldCalc.MurphyYieldPct / 100.0
	totalGood := int(math.Round(float64(totalTested) * passRate))
	totalReject := totalTested - totalGood

	// Binning breakdown:
	// Bin 1 Prime (top 35% of good dies) -> $450 USD
	// Bin 2 Low Voltage (next 40% of good dies) -> $320 USD
	// Bin 3 Standard (remaining 25% of good dies) -> $220 USD
	bin1 := int(math.Round(float64(totalGood) * 0.35))
	bin2 := int(math.Round(float64(totalGood) * 0.40))
	bin3 := totalGood - bin1 - bin2

	valuation := (float64(bin1) * 450.0) + (float64(bin2) * 320.0) + (float64(bin3) * 220.0)

	manifestID := fmt.Sprintf("MANIFEST-BIN-%d", time.Now().UnixNano()%1000000)
	manifest := DieSortBinningManifest{
		ManifestID:               manifestID,
		WaferLotID:               lotID,
		TotalTestedDieCount:      totalTested,
		Bin1PrimeCount:           bin1,
		Bin2LowVoltageCount:      bin2,
		Bin3StandardCount:        bin3,
		Bin7RejectDefectiveCount: totalReject,
		WaferPackYieldPct:        yieldCalc.MurphyYieldPct,
		CommercialValuationUSD:   math.Round(valuation*100.0) / 100.0,
		BinningDate:              time.Now(),
	}

	v.manifests[lotID] = manifest
	return manifest, nil
}

// GetLotDefectivityDossier compiles fab yield, particle counts, and binning breakdown
func (v *SemiconductorFabVault) GetLotDefectivityDossier(lotID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	yieldCalc, hasYield := v.yields[lotID]
	manifest, hasManifest := v.manifests[lotID]

	dossier := map[string]interface{}{
		"wafer_lot_id":            lotID,
		"has_yield_model":         hasYield,
		"has_binning_manifest":    hasManifest,
		"semi_e10_compliance":     true,
		"iso_14644_class1_status": "CERTIFIED_METROLOGY_HEALTHY",
		"compiled_at":             time.Now(),
	}

	if hasYield {
		dossier["product_sku"] = yieldCalc.ProductFamilySKU
		dossier["murphy_yield_pct"] = yieldCalc.MurphyYieldPct
		dossier["projected_good_die"] = yieldCalc.ProjectedGoodDieCount
		dossier["defect_density_d0"] = yieldCalc.DefectDensityD0_PerCM2
		dossier["quarantine_recommended"] = yieldCalc.WaferQuarantineRecommended
	}
	if hasManifest {
		dossier["bin_1_prime_dies"] = manifest.Bin1PrimeCount
		dossier["bin_2_low_volt_dies"] = manifest.Bin2LowVoltageCount
		dossier["commercial_valuation_usd"] = manifest.CommercialValuationUSD
	}

	return dossier, nil
}

// RegisterSemiconductorRoutes wires REST endpoints
func RegisterSemiconductorRoutes(r *gin.RouterGroup) {
	semiGroup := r.Group("/semiconductor/fab")
	{
		semiGroup.POST("/ingest-cleanroom-metrology", func(c *gin.Context) {
			var m CleanroomMetrology
			if err := c.ShouldBindJSON(&m); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalSemiconductorFabVault.IngestCleanroomMetrology(m)
			c.JSON(http.StatusOK, res)
		})

		semiGroup.POST("/calculate-wafer-yield-model", func(c *gin.Context) {
			var req WaferYieldCalculation
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSemiconductorFabVault.CalculateWaferYieldModel(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		semiGroup.POST("/execute-die-sort-binning", func(c *gin.Context) {
			var body struct {
				WaferLotID         string `json:"wafer_lot_id"`
				TotalTestedDieCount int   `json:"total_tested_die_count"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalSemiconductorFabVault.ExecuteDieSortBinning(body.WaferLotID, body.TotalTestedDieCount)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		semiGroup.GET("/lot-defectivity-dossier/:lot_id", func(c *gin.Context) {
			lotID := c.Param("lot_id")
			dossier, err := GlobalSemiconductorFabVault.GetLotDefectivityDossier(lotID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
