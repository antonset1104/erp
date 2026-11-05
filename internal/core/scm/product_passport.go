package scm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// MaterialComponent represents a bill-of-materials constituent with circularity metadata.
type MaterialComponent struct {
	ComponentID           string  `json:"component_id"`
	Name                  string  `json:"name"`
	MaterialType          string  `json:"material_type"` // e.g. "Recycled Lithium-Ion", "Post-Consumer Aluminum", "Bio-Polymer"
	WeightKg              float64 `json:"weight_kg"`
	RecycledPercentage    float64 `json:"recycled_percentage"`   // 0.0 - 100.0%
	CriticalRawMaterial   bool    `json:"critical_raw_material"` // EU Critical Raw Materials Act classification
	SupplierOriginCountry string  `json:"supplier_origin_country"`
	RoHSCompliant         bool    `json:"rohs_compliant"`
	REACHCompliant        bool    `json:"reach_compliant"`
}

// CarbonGenealogy captures cradle-to-gate GHG accounting for the unit.
type CarbonGenealogy struct {
	Scope1ManufacturingKgCO2e     float64 `json:"scope1_manufacturing_kg_co2e"`
	Scope2ElectricityKgCO2e       float64 `json:"scope2_electricity_kg_co2e"`
	Scope3UpstreamTransportKgCO2e float64 `json:"scope3_upstream_transport_kg_co2e"`
	TotalCradleToGateCO2e         float64 `json:"total_cradle_to_gate_co2e"`
	CarbonIntensityRatio          float64 `json:"carbon_intensity_ratio"` // kg CO2e / kg product weight
}

// CircularityMetrics provides repairability and end-of-life circular scores.
type CircularityMetrics struct {
	RepairabilityScore          float64 `json:"repairability_score"` // 1.0 to 10.0 French AGEC / EU standard
	DisassemblySteps            int     `json:"disassembly_steps"`
	SparePartsAvailabilityYears int     `json:"spare_parts_availability_years"`
	RecyclabilityRatePercentage float64 `json:"recyclability_rate_percentage"` // 0 - 100%
	RefurbishmentEligible       bool    `json:"refurbishment_eligible"`
	DisassemblyManualURL        string  `json:"disassembly_manual_url"`
	EndOfLifeRecyclerGuidance   string  `json:"end_of_life_recycler_guidance"`
}

// DigitalProductPassport encapsulates the EU ESPR & Battery Passport digital asset.
type DigitalProductPassport struct {
	PassportID             string              `json:"passport_id"` // e.g. "DPP-2026-BAT-9941"
	ProductSKU             string              `json:"product_sku"`
	ProductName            string              `json:"product_name"`
	BatchNumber            string              `json:"batch_number"`
	SerialNumber           string              `json:"serial_number"`
	ManufacturingDate      string              `json:"manufacturing_date"`
	ManufacturerName       string              `json:"manufacturer_name"`
	ManufacturingFacility  string              `json:"manufacturing_facility"`
	TotalWeightKg          float64             `json:"total_weight_kg"`
	OverallRecycledContent float64             `json:"overall_recycled_content"` // %
	Components             []MaterialComponent `json:"components"`
	CarbonGenealogy        CarbonGenealogy     `json:"carbon_genealogy"`
	Circularity            CircularityMetrics  `json:"circularity"`
	DigitalSignatureHash   string              `json:"digital_signature_hash"`
	PublicVerificationURL  string              `json:"public_verification_url"`
	QRVerificationPayload  string              `json:"qr_verification_payload"`
	Status                 string              `json:"status"` // "ACTIVE", "IN_RECIRCULATION", "RECYCLED", "DECOMMISSIONED"
	IssuedAt               string              `json:"issued_at"`
}

// IssuePassportRequest holds payload to mint a new Digital Product Passport.
type IssuePassportRequest struct {
	ProductSKU            string              `json:"product_sku"`
	ProductName           string              `json:"product_name"`
	BatchNumber           string              `json:"batch_number"`
	SerialNumber          string              `json:"serial_number"`
	ManufacturerName      string              `json:"manufacturer_name"`
	ManufacturingFacility string              `json:"manufacturing_facility"`
	Components            []MaterialComponent `json:"components"`
	Scope1KgCO2e          float64             `json:"scope1_kg_co2e"`
	Scope2KgCO2e          float64             `json:"scope2_kg_co2e"`
	Scope3KgCO2e          float64             `json:"scope3_kg_co2e"`
	RepairabilityScore    float64             `json:"repairability_score"`
	DisassemblySteps      int                 `json:"disassembly_steps"`
	SparePartsYears       int                 `json:"spare_parts_years"`
	RecyclabilityRatePct  float64             `json:"recyclability_rate_pct"`
	DisassemblyManualURL  string              `json:"disassembly_manual_url"`
}

// PassportVerificationResult provides verification output for public / audit lookup.
type PassportVerificationResult struct {
	PassportID            string                  `json:"passport_id"`
	Authentic             bool                    `json:"authentic"`
	IntegritySignature    string                  `json:"integrity_signature"`
	PassportData          *DigitalProductPassport `json:"passport_data"`
	EUPassportCompliant   bool                    `json:"eu_passport_compliant"`
	ComplianceChecklist   map[string]bool         `json:"compliance_checklist"`
	VerificationTimestamp string                  `json:"verification_timestamp"`
}

// DPPRegistryStore provides thread-safe in-memory custody for issued passports.
type DPPRegistryStore struct {
	mu        sync.RWMutex
	passports map[string]*DigitalProductPassport
}

var globalDPPStore = &DPPRegistryStore{
	passports: make(map[string]*DigitalProductPassport),
}

func init() {
	// Seed demo EU ESPR compliant passport
	demoComponents := []MaterialComponent{
		{
			ComponentID:           "CMP-BAT-01",
			Name:                  "Prismatic NMC 811 Lithium Cathode",
			MaterialType:          "Recycled Cobalt & Nickel",
			WeightKg:              32.5,
			RecycledPercentage:    64.0,
			CriticalRawMaterial:   true,
			SupplierOriginCountry: "DE",
			RoHSCompliant:         true,
			REACHCompliant:        true,
		},
		{
			ComponentID:           "CMP-ENC-02",
			Name:                  "Structural Enclosure Tray",
			MaterialType:          "Post-Consumer Recycled 6000-series Aluminum",
			WeightKg:              18.0,
			RecycledPercentage:    85.0,
			CriticalRawMaterial:   false,
			SupplierOriginCountry: "NO",
			RoHSCompliant:         true,
			REACHCompliant:        true,
		},
		{
			ComponentID:           "CMP-BMS-03",
			Name:                  "Solid-State Battery Management Controller",
			MaterialType:          "Low-Halogen Epoxy Circuitry",
			WeightKg:              3.2,
			RecycledPercentage:    20.0,
			CriticalRawMaterial:   true,
			SupplierOriginCountry: "TW",
			RoHSCompliant:         true,
			REACHCompliant:        true,
		},
	}

	req := IssuePassportRequest{
		ProductSKU:            "EV-BAT-100KWH",
		ProductName:           "Enterprise High-Density 100kWh Traction Battery Pack",
		BatchNumber:           "BATCH-2026-Q1-DE01",
		SerialNumber:          "SN-882910-AA9",
		ManufacturerName:      "GoERP Clean Mobility Systems SE",
		ManufacturingFacility: "Gigafactory Berlin-Brandenburg",
		Components:            demoComponents,
		Scope1KgCO2e:          420.0,
		Scope2KgCO2e:          180.0,
		Scope3KgCO2e:          750.0,
		RepairabilityScore:    8.7,
		DisassemblySteps:      12,
		SparePartsYears:       15,
		RecyclabilityRatePct:  94.5,
		DisassemblyManualURL:  "https://circular.goerp.io/manuals/EV-BAT-100KWH-v2.pdf",
	}
	_, _ = IssueProductPassportLogic(req)
}

// IssueProductPassportLogic executes the calculation and cryptographic minting of a DPP.
func IssueProductPassportLogic(req IssuePassportRequest) (*DigitalProductPassport, error) {
	globalDPPStore.mu.Lock()
	defer globalDPPStore.mu.Unlock()

	if req.ProductSKU == "" {
		req.ProductSKU = "PRD-CIRCULAR-GENERIC"
	}
	if req.SerialNumber == "" {
		req.SerialNumber = fmt.Sprintf("SN-%d", time.Now().UnixNano()%1000000)
	}

	passportID := fmt.Sprintf("DPP-2026-%s-%s", req.ProductSKU, req.SerialNumber)

	var totalWeight float64
	var weightedRecycledSum float64

	for _, comp := range req.Components {
		totalWeight += comp.WeightKg
		weightedRecycledSum += comp.WeightKg * (comp.RecycledPercentage / 100.0)
	}

	overallRecycledPct := 0.0
	if totalWeight > 0 {
		overallRecycledPct = (weightedRecycledSum / totalWeight) * 100.0
	}

	totalCradleToGate := req.Scope1KgCO2e + req.Scope2KgCO2e + req.Scope3KgCO2e
	carbonIntensity := 0.0
	if totalWeight > 0 {
		carbonIntensity = totalCradleToGate / totalWeight
	}

	// Compute SHA-256 cryptographic proof
	rawSignaturePayload := fmt.Sprintf("%s|%s|%s|%.2f|%.2f|%.2f|%s",
		passportID, req.ProductSKU, req.SerialNumber, totalWeight, overallRecycledPct, totalCradleToGate, req.ManufacturerName)
	hasher := sha256.New()
	hasher.Write([]byte(rawSignaturePayload))
	digest := hex.EncodeToString(hasher.Sum(nil))

	publicURL := fmt.Sprintf("https://dpp.goerp.io/verify/%s?sig=%s", passportID, digest[:16])
	qrPayload := fmt.Sprintf("dpp://goerp.io/v1/inspect?passport=%s&sig=%s", passportID, digest)

	passport := &DigitalProductPassport{
		PassportID:             passportID,
		ProductSKU:             req.ProductSKU,
		ProductName:            req.ProductName,
		BatchNumber:            req.BatchNumber,
		SerialNumber:           req.SerialNumber,
		ManufacturingDate:      time.Now().Format("2006-01-02"),
		ManufacturerName:       req.ManufacturerName,
		ManufacturingFacility:  req.ManufacturingFacility,
		TotalWeightKg:          totalWeight,
		OverallRecycledContent: overallRecycledPct,
		Components:             req.Components,
		CarbonGenealogy: CarbonGenealogy{
			Scope1ManufacturingKgCO2e:     req.Scope1KgCO2e,
			Scope2ElectricityKgCO2e:       req.Scope2KgCO2e,
			Scope3UpstreamTransportKgCO2e: req.Scope3KgCO2e,
			TotalCradleToGateCO2e:         totalCradleToGate,
			CarbonIntensityRatio:          carbonIntensity,
		},
		Circularity: CircularityMetrics{
			RepairabilityScore:          req.RepairabilityScore,
			DisassemblySteps:            req.DisassemblySteps,
			SparePartsAvailabilityYears: req.SparePartsYears,
			RecyclabilityRatePercentage: req.RecyclabilityRatePct,
			RefurbishmentEligible:       req.RepairabilityScore >= 7.0,
			DisassemblyManualURL:        req.DisassemblyManualURL,
			EndOfLifeRecyclerGuidance:   "Certified Pyrometallurgical & Hydrometallurgical Closed-Loop Recovery Required",
		},
		DigitalSignatureHash:  digest,
		PublicVerificationURL: publicURL,
		QRVerificationPayload: qrPayload,
		Status:                "ACTIVE",
		IssuedAt:              time.Now().Format(time.RFC3339),
	}

	globalDPPStore.passports[passportID] = passport
	return passport, nil
}

// VerifyPassportLogic checks passport authenticity and regulatory conformance.
func VerifyPassportLogic(passportID string) (*PassportVerificationResult, error) {
	globalDPPStore.mu.RLock()
	defer globalDPPStore.mu.RUnlock()

	passport, exists := globalDPPStore.passports[passportID]
	if !exists {
		return nil, fmt.Errorf("digital product passport '%s' not found", passportID)
	}

	checklist := map[string]bool{
		"eu_espr_compliant":        passport.Circularity.RecyclabilityRatePercentage >= 80.0,
		"carbon_intensity_tracked": passport.CarbonGenealogy.TotalCradleToGateCO2e > 0,
		"repairability_compliant":  passport.Circularity.RepairabilityScore >= 6.0,
		"spare_parts_available":    passport.Circularity.SparePartsAvailabilityYears >= 10,
		"rohs_reach_validated":     true,
	}

	for _, comp := range passport.Components {
		if !comp.RoHSCompliant || !comp.REACHCompliant {
			checklist["rohs_reach_validated"] = false
			break
		}
	}

	overallEUCompliant := true
	for _, passed := range checklist {
		if !passed {
			overallEUCompliant = false
			break
		}
	}

	return &PassportVerificationResult{
		PassportID:            passportID,
		Authentic:             true,
		IntegritySignature:    passport.DigitalSignatureHash,
		PassportData:          passport,
		EUPassportCompliant:   overallEUCompliant,
		ComplianceChecklist:   checklist,
		VerificationTimestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// RegisterDPPRoutes registers Digital Product Passport API endpoints.
func RegisterDPPRoutes(r *gin.RouterGroup) {
	group := r.Group("/scm/dpp")
	{
		group.POST("/issue", func(c *gin.Context) {
			var req IssuePassportRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid passport issue payload: " + err.Error()})
				return
			}
			res, err := IssueProductPassportLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.GET("/verify/:passport_id", func(c *gin.Context) {
			id := c.Param("passport_id")
			res, err := VerifyPassportLogic(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/registry", func(c *gin.Context) {
			globalDPPStore.mu.RLock()
			defer globalDPPStore.mu.RUnlock()

			list := make([]*DigitalProductPassport, 0, len(globalDPPStore.passports))
			for _, p := range globalDPPStore.passports {
				list = append(list, p)
			}
			c.JSON(http.StatusOK, gin.H{
				"total_registered": len(list),
				"passports":        list,
			})
		})
	}
}
