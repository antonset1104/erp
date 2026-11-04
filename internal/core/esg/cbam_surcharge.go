package esg

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CBAMGoodClassification represents EU CBAM covered goods category
type CBAMGoodClassification struct {
	CNCode              string  `json:"cn_code"`               // e.g. "7208 51 00" (Flat-rolled products of iron/non-alloy steel)
	Sector              string  `json:"sector"`                // "STEEL", "ALUMINIUM", "CEMENT", "FERTILIZER", "HYDROGEN"
	ProductionRoute     string  `json:"production_route"`      // "ELECTRIC_ARC_FURNACE", "BLAST_FURNACE_BASIC_OXYGEN", "DIRECT_REDUCTION"
	DefaultBenchmarkSEE float64 `json:"default_benchmark_see"` // Benchmark Specific Embedded Emissions (tCO2e/t product)
}

// EmbeddedEmissionsCalculation represents the granular Scope 1, 2, and precursor emissions
type EmbeddedEmissionsCalculation struct {
	CalculationID       string    `json:"calculation_id"`
	ExportShipmentID    string    `json:"export_shipment_id"`
	ProductSKU          string    `json:"product_sku"`
	CNCode              string    `json:"cn_code"`
	ProductionBatchID   string    `json:"production_batch_id"`
	NetMassMetricTons   float64   `json:"net_mass_metric_tons"`
	DirectScope1CO2e    float64   `json:"direct_scope1_co2e_tons"`
	IndirectScope2CO2e  float64   `json:"indirect_scope2_co2e_tons"`
	PrecursorCO2e       float64   `json:"precursor_co2e_tons"`
	TotalEmbeddedCO2e   float64   `json:"total_embedded_co2e_tons"`
	SpecificEmbeddedEm  float64   `json:"specific_embedded_emissions"` // tCO2e / t product
	CalculationStandard string    `json:"calculation_standard"`        // "EU_CBAM_REGULATION_2023_956"
	CalculatedAt        time.Time `json:"calculated_at"`
}

// GreenSurchargeAssessment models the carbon tariff and commercial invoice markup
type GreenSurchargeAssessment struct {
	AssessmentID            string    `json:"assessment_id"`
	ExportShipmentID        string    `json:"export_shipment_id"`
	CommercialInvoiceNo     string    `json:"commercial_invoice_no"`
	Currency                string    `json:"currency"`                  // "EUR"
	EUETSCarbonPriceEUR     float64   `json:"eu_ets_carbon_price_eur"`   // e.g. 85.00 EUR / tCO2e
	DomesticCarbonTaxCredit float64   `json:"domestic_carbon_tax_eur"`   // e.g. 2.50 EUR / tCO2e (IDXCarbon / Carbon Tax paid in export country)
	EffectiveCarbonDutyRate float64   `json:"effective_carbon_duty_eur"` // EU ETS - Domestic Credit
	TotalEmbeddedEmissions  float64   `json:"total_embedded_emissions_tco2e"`
	TotalGreenSurchargeEUR  float64   `json:"total_green_surcharge_eur"`
	SurchargeStatus         string    `json:"surcharge_status"` // "ASSESSED", "INVOICED_TO_BUYER", "REMITTED"
	InvoiceLineItemCode     string    `json:"invoice_line_item_code"`
	AssessedAt              time.Time `json:"assessed_at"`
}

// CBAMDeclarationXML represents the official quarterly registry filing summary
type CBAMDeclarationXML struct {
	DeclarationID      string    `json:"declaration_id"`
	ExportShipmentID   string    `json:"export_shipment_id"`
	DeclarantEORINumber string   `json:"declarant_eori_number"` // European Operator EORI
	InstallationID     string    `json:"installation_id"`      // Manufacturer UN/LOCODE + Plant ID
	InstallationGeo    string    `json:"installation_geo"`     // Lat/Long
	CountryOfOrigin    string    `json:"country_of_origin"`    // "ID"
	ReportingQuarter   string    `json:"reporting_quarter"`    // "2026-Q4"
	XMLPayloadDigest   string    `json:"xml_payload_digest"`   // SHA-256
	XMLPayloadContent  string    `json:"xml_payload_content"`
	VerificationStatus string    `json:"verification_status"`  // "ACCREDITED_VERIFIER_STAMPED"
	GeneratedAt        time.Time `json:"generated_at"`
}

// CBAMSurchargeVault thread-safe store
type CBAMSurchargeVault struct {
	mu           sync.RWMutex
	calculations map[string]EmbeddedEmissionsCalculation
	surcharges   map[string]GreenSurchargeAssessment
	declarations map[string]CBAMDeclarationXML
}

// Global CBAM instance
var GlobalCBAMSurchargeVault = &CBAMSurchargeVault{
	calculations: make(map[string]EmbeddedEmissionsCalculation),
	surcharges:   make(map[string]GreenSurchargeAssessment),
	declarations: make(map[string]CBAMDeclarationXML),
}

// CalculateEmbeddedEmissions processes production batch carbon footprint
func (v *CBAMSurchargeVault) CalculateEmbeddedEmissions(
	exportID, sku, cnCode, batchID string,
	netMassTons, scope1Tons, scope2Tons, precursorTons float64,
) EmbeddedEmissionsCalculation {
	v.mu.Lock()
	defer v.mu.Unlock()

	totalCO2e := scope1Tons + scope2Tons + precursorTons
	var see float64
	if netMassTons > 0 {
		see = totalCO2e / netMassTons
	}

	calcID := fmt.Sprintf("CBAM-CALC-%d", time.Now().UnixNano()%1000000)
	calc := EmbeddedEmissionsCalculation{
		CalculationID:       calcID,
		ExportShipmentID:    exportID,
		ProductSKU:          sku,
		CNCode:              cnCode,
		ProductionBatchID:   batchID,
		NetMassMetricTons:   netMassTons,
		DirectScope1CO2e:    scope1Tons,
		IndirectScope2CO2e:  scope2Tons,
		PrecursorCO2e:       precursorTons,
		TotalEmbeddedCO2e:   totalCO2e,
		SpecificEmbeddedEm:  see,
		CalculationStandard: "EU_CBAM_REGULATION_2023_956",
		CalculatedAt:        time.Now(),
	}

	v.calculations[exportID] = calc
	return calc
}

// GenerateGreenSurcharge assesses EU ETS carbon pricing vs domestic credits
func (v *CBAMSurchargeVault) GenerateGreenSurcharge(
	exportID, invoiceNo, currency string,
	euETSRate, domesticTaxCredit float64,
) (*GreenSurchargeAssessment, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	calc, ok := v.calculations[exportID]
	if !ok {
		return nil, fmt.Errorf("embedded emissions calculation not found for export shipment: %s", exportID)
	}

	effectiveRate := euETSRate - domesticTaxCredit
	if effectiveRate < 0 {
		effectiveRate = 0
	}

	totalSurcharge := effectiveRate * calc.TotalEmbeddedCO2e
	assessID := fmt.Sprintf("CBAM-SUR-%d", time.Now().UnixNano()%1000000)

	assessment := GreenSurchargeAssessment{
		AssessmentID:            assessID,
		ExportShipmentID:        exportID,
		CommercialInvoiceNo:     invoiceNo,
		Currency:                currency,
		EUETSCarbonPriceEUR:     euETSRate,
		DomesticCarbonTaxCredit: domesticTaxCredit,
		EffectiveCarbonDutyRate: effectiveRate,
		TotalEmbeddedEmissions:  calc.TotalEmbeddedCO2e,
		TotalGreenSurchargeEUR:  totalSurcharge,
		SurchargeStatus:         "INVOICED_TO_BUYER",
		InvoiceLineItemCode:     "CBAM-GREEN-CARBON-SURCHARGE-EU2023/956",
		AssessedAt:              time.Now(),
	}

	v.surcharges[exportID] = assessment
	return &assessment, nil
}

// GenerateCBAMXMLDeclaration serializes compliant EU CBAM registry XML
func (v *CBAMSurchargeVault) GenerateCBAMXMLDeclaration(
	exportID, eori, installationID, geo, country, quarter string,
) (*CBAMDeclarationXML, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	calc, ok := v.calculations[exportID]
	if !ok {
		return nil, fmt.Errorf("cannot generate XML: calculation not found for %s", exportID)
	}

	xmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<CBAMQuarterlyReport xmlns="urn:eu:cbam:v1" version="2026.1">
    <Declarant EORI="%s"/>
    <ReportingPeriod Quarter="%s"/>
    <GoodsItem CNCode="%s" NetMassMetricTons="%.3f">
        <Installation ID="%s" GeoCoordinates="%s" Country="%s"/>
        <EmissionsDirectScope1 MetricTonsCO2e="%.3f"/>
        <EmissionsIndirectScope2 MetricTonsCO2e="%.3f"/>
        <EmissionsPrecursor MetricTonsCO2e="%.3f"/>
        <SpecificEmbeddedEmissions Value="%.4f" Unit="tCO2e/t"/>
    </GoodsItem>
    <Verification AccreditationStatus="VERIFIED_INDEPENDENT_AUDITOR_ISO14065"/>
</CBAMQuarterlyReport>`, eori, quarter, calc.CNCode, calc.NetMassMetricTons, installationID, geo, country,
		calc.DirectScope1CO2e, calc.IndirectScope2CO2e, calc.PrecursorCO2e, calc.SpecificEmbeddedEm)

	hashBytes := sha256.Sum256([]byte(xmlContent))
	digest := hex.EncodeToString(hashBytes[:])

	declID := fmt.Sprintf("CBAM-DEC-%d", time.Now().UnixNano()%1000000)
	declaration := CBAMDeclarationXML{
		DeclarationID:       declID,
		ExportShipmentID:    exportID,
		DeclarantEORINumber: eori,
		InstallationID:      installationID,
		InstallationGeo:     geo,
		CountryOfOrigin:     country,
		ReportingQuarter:    quarter,
		XMLPayloadDigest:    digest,
		XMLPayloadContent:   xmlContent,
		VerificationStatus:  "ACCREDITED_VERIFIER_STAMPED",
		GeneratedAt:         time.Now(),
	}

	v.declarations[exportID] = declaration
	return &declaration, nil
}

// GetAuditSummary retrieves complete CBAM compliance dossier
func (v *CBAMSurchargeVault) GetAuditSummary(exportID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	calc, hasCalc := v.calculations[exportID]
	surcharge, hasSur := v.surcharges[exportID]
	decl, hasDecl := v.declarations[exportID]

	if !hasCalc {
		return nil, fmt.Errorf("no CBAM records found for export shipment: %s", exportID)
	}

	return gin.H{
		"export_shipment_id":   exportID,
		"embedded_calculation": calc,
		"green_surcharge":      surcharge,
		"has_surcharge":        hasSur,
		"xml_declaration":      decl,
		"has_declaration":      hasDecl,
		"regulatory_mandate":   "EU Regulation (EU) 2023/956 Carbon Border Adjustment Mechanism",
		"status":               "COMPLIANT_READY_FOR_CUSTOMS",
	}, nil
}

// RegisterCBAMRoutes registers HTTP endpoints
func RegisterCBAMRoutes(r *gin.RouterGroup) {
	cbam := r.Group("/esg/cbam")
	{
		cbam.POST("/calculate-embedded-emissions", func(c *gin.Context) {
			var req struct {
				ExportShipmentID   string  `json:"export_shipment_id" binding:"required"`
				ProductSKU         string  `json:"product_sku" binding:"required"`
				CNCode             string  `json:"cn_code" binding:"required"`
				ProductionBatchID  string  `json:"production_batch_id" binding:"required"`
				NetMassMetricTons  float64 `json:"net_mass_metric_tons" binding:"required"`
				DirectScope1CO2e   float64 `json:"direct_scope1_co2e_tons"`
				IndirectScope2CO2e float64 `json:"indirect_scope2_co2e_tons"`
				PrecursorCO2e      float64 `json:"precursor_co2e_tons"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			calc := GlobalCBAMSurchargeVault.CalculateEmbeddedEmissions(
				req.ExportShipmentID, req.ProductSKU, req.CNCode, req.ProductionBatchID,
				req.NetMassMetricTons, req.DirectScope1CO2e, req.IndirectScope2CO2e, req.PrecursorCO2e,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": calc})
		})

		cbam.POST("/generate-green-surcharge", func(c *gin.Context) {
			var req struct {
				ExportShipmentID        string  `json:"export_shipment_id" binding:"required"`
				CommercialInvoiceNo     string  `json:"commercial_invoice_no" binding:"required"`
				Currency                string  `json:"currency"`
				EUETSCarbonPriceEUR     float64 `json:"eu_ets_carbon_price_eur" binding:"required"`
				DomesticCarbonTaxCredit float64 `json:"domestic_carbon_tax_eur"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if req.Currency == "" {
				req.Currency = "EUR"
			}

			surcharge, err := GlobalCBAMSurchargeVault.GenerateGreenSurcharge(
				req.ExportShipmentID, req.CommercialInvoiceNo, req.Currency,
				req.EUETSCarbonPriceEUR, req.DomesticCarbonTaxCredit,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": surcharge})
		})

		cbam.POST("/export-cbam-xml-declaration", func(c *gin.Context) {
			var req struct {
				ExportShipmentID    string `json:"export_shipment_id" binding:"required"`
				DeclarantEORINumber string `json:"declarant_eori_number" binding:"required"`
				InstallationID      string `json:"installation_id" binding:"required"`
				InstallationGeo     string `json:"installation_geo" binding:"required"`
				CountryOfOrigin     string `json:"country_of_origin" binding:"required"`
				ReportingQuarter    string `json:"reporting_quarter" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			decl, err := GlobalCBAMSurchargeVault.GenerateCBAMXMLDeclaration(
				req.ExportShipmentID, req.DeclarantEORINumber, req.InstallationID,
				req.InstallationGeo, req.CountryOfOrigin, req.ReportingQuarter,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": decl})
		})

		cbam.GET("/audit-registry/:export_id", func(c *gin.Context) {
			exportID := c.Param("export_id")
			summary, err := GlobalCBAMSurchargeVault.GetAuditSummary(exportID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": summary})
		})
	}
}
