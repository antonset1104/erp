package customs

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// GATTTransactionValue represents the customs valuation elements under WTO GATT Article VII & Article 8
type GATTTransactionValue struct {
	ShipmentID                string    `json:"shipment_id"`
	ImporterEntity            string    `json:"importer_entity"`
	ExporterAffiliate         string    `json:"exporter_affiliate"`
	BaseInvoiceFOBUSD         float64   `json:"base_invoice_fob_usd"`
	InternationalFreightUSD   float64   `json:"international_freight_usd"`
	MarineInsuranceUSD        float64   `json:"marine_insurance_usd"`
	DutiableAssistsUSD        float64   `json:"dutiable_assists_usd"` // Tools, dies, engineering provided free
	RoyaltiesLicenseFeesUSD   float64   `json:"royalties_license_fees_usd"`
	PackingAndContainersUSD   float64   `json:"packing_and_containers_usd"`
	SellingCommissionsUSD     float64   `json:"selling_commissions_usd"`
	CalculatedDutiableCIFUSD  float64   `json:"calculated_dutiable_cif_usd"`
	ValuationMethodCited      string    `json:"valuation_method_cited"` // "METHOD_1_TRANSACTION_VALUE", "METHOD_2_IDENTICAL_GOODS"
	EvaluatedAt               time.Time `json:"evaluated_at"`
}

// TransferPricingReconciliation matches Customs CIF against OECD Arm's Length Interquartile Range
type TransferPricingReconciliation struct {
	ShipmentID               string    `json:"shipment_id"`
	CustomsCIFUnitPriceUSD   float64   `json:"customs_cif_unit_price_usd"`
	TPLowerQuartileQ1USD     float64   `json:"tp_lower_quartile_q1_usd"`
	TPMedianArmLengthUSD     float64   `json:"tp_median_arm_length_usd"`
	TPUpperQuartileQ3USD     float64   `json:"tp_upper_quartile_q3_usd"`
	AlignmentStatus          string    `json:"alignment_status"` // "HARMONIZED_IN_RANGE", "CUSTOMS_UNDERVALUATION_RISK", "TAX_PROFIT_SHIFTING_RISK"
	CustomsDutyExposureUSD   float64   `json:"customs_duty_exposure_usd"`
	CorporateIncomeTaxRiskUSD float64  `json:"corporate_income_tax_risk_usd"`
	CompensatingAdjustmentUSD float64  `json:"compensating_adjustment_usd"`
	RecommendedAction        string    `json:"recommended_action"`
	ReconciledAt             time.Time `json:"reconciled_at"`
}

// CustomsClearanceDeclaration represents the formal customs import entry document (PIB / Entry Pack)
type CustomsClearanceDeclaration struct {
	DeclarationNumber     string    `json:"declaration_number"`
	ShipmentID            string    `json:"shipment_id"`
	HSCode                string    `json:"hs_code"`
	DeclaredCIFValueUSD   float64   `json:"declared_cif_value_usd"`
	ImportDutyRatePercent float64   `json:"import_duty_rate_percent"`
	ImportDutyPayableUSD  float64   `json:"import_duty_payable_usd"`
	ImportVATRatePercent  float64   `json:"import_vat_rate_percent"`
	ImportVATPayableUSD   float64   `json:"import_vat_payable_usd"`
	ImportIncomeTax22USD  float64   `json:"import_income_tax_22_usd"`
	TotalCustomsTaxesUSD  float64   `json:"total_customs_taxes_usd"`
	PortOfEntry           string    `json:"port_of_entry"`
	CustomsStatus         string    `json:"customs_status"` // "SUBMITTED", "GREEN_LINE_CLEARED", "PHYSICAL_INSPECTION_RED"
	FilingTimestamp       time.Time `json:"filing_timestamp"`
}

// HarmonizerAuditSummary compiles end-to-end customs and tax alignment record
type HarmonizerAuditSummary struct {
	ShipmentID       string                        `json:"shipment_id"`
	GATTValuation    GATTTransactionValue          `json:"gatt_valuation"`
	TPReconciliation TransferPricingReconciliation `json:"tp_reconciliation"`
	Declaration      CustomsClearanceDeclaration   `json:"declaration"`
	AuditVerdict     string                        `json:"audit_verdict"`
	AuditedAt        time.Time                     `json:"audited_at"`
}

// GATTHarmonizerEngine manages GATT customs valuation and TP reconciliation
type GATTHarmonizerEngine struct {
	mu           sync.RWMutex
	valuations   map[string]GATTTransactionValue
	reconciles   map[string]TransferPricingReconciliation
	declarations map[string]CustomsClearanceDeclaration
}

var (
	globalGATTEngine *GATTHarmonizerEngine
	gattOnce         sync.Once
)

// GetGATTHarmonizerEngine returns singleton instance
func GetGATTHarmonizerEngine() *GATTHarmonizerEngine {
	gattOnce.Do(func() {
		globalGATTEngine = &GATTHarmonizerEngine{
			valuations:   make(map[string]GATTTransactionValue),
			reconciles:   make(map[string]TransferPricingReconciliation),
			declarations: make(map[string]CustomsClearanceDeclaration),
		}
	})
	return globalGATTEngine
}

// EvaluateTransactionValue calculates dutiable customs CIF value under GATT Article VII
func (e *GATTHarmonizerEngine) EvaluateTransactionValue(req GATTTransactionValue) GATTTransactionValue {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Sum of invoice price + additions under WTO Article 8
	cif := req.BaseInvoiceFOBUSD +
		req.InternationalFreightUSD +
		req.MarineInsuranceUSD +
		req.DutiableAssistsUSD +
		req.RoyaltiesLicenseFeesUSD +
		req.PackingAndContainersUSD +
		req.SellingCommissionsUSD

	req.CalculatedDutiableCIFUSD = cif
	req.ValuationMethodCited = "METHOD_1_TRANSACTION_VALUE_GATT_ART_VII"
	req.EvaluatedAt = time.Now()

	e.valuations[req.ShipmentID] = req
	return req
}

// ReconcileTransferPricing evaluates customs price against OECD arm's length interquartile range
func (e *GATTHarmonizerEngine) ReconcileTransferPricing(shipmentID string, unitCIF float64, q1 float64, median float64, q3 float64) TransferPricingReconciliation {
	e.mu.Lock()
	defer e.mu.Unlock()

	status := "HARMONIZED_IN_RANGE"
	var dutyExposure float64 = 0.0
	var citRisk float64 = 0.0
	var compAdj float64 = 0.0
	action := "Customs declared value fully aligns with Transfer Pricing Local File. No accrual required."

	if unitCIF < q1 {
		status = "CUSTOMS_UNDERVALUATION_RISK"
		diff := q1 - unitCIF
		dutyExposure = diff * 0.175 // Estimated 7.5% Duty + 10% VAT delta
		action = fmt.Sprintf("CIF unit price $%.2f is below Q1 ($%.2f). Customs audit penalty risk. File voluntary customs disclosure.", unitCIF, q1)
	} else if unitCIF > q3 {
		status = "TAX_PROFIT_SHIFTING_RISK"
		diff := unitCIF - q3
		citRisk = diff * 0.22 // 22% Corporate Income Tax disallowance risk
		compAdj = diff
		action = fmt.Sprintf("CIF unit price $%.2f exceeds Q3 ($%.2f). Tax authority may disallow deduction. Book year-end downward TP adjustment.", unitCIF, q3)
	}

	recon := TransferPricingReconciliation{
		ShipmentID:                shipmentID,
		CustomsCIFUnitPriceUSD:   unitCIF,
		TPLowerQuartileQ1USD:     q1,
		TPMedianArmLengthUSD:     median,
		TPUpperQuartileQ3USD:     q3,
		AlignmentStatus:          status,
		CustomsDutyExposureUSD:   dutyExposure,
		CorporateIncomeTaxRiskUSD: citRisk,
		CompensatingAdjustmentUSD: compAdj,
		RecommendedAction:        action,
		ReconciledAt:             time.Now(),
	}

	e.reconciles[shipmentID] = recon
	return recon
}

// GenerateClearanceDeclaration creates statutory import declaration pack
func (e *GATTHarmonizerEngine) GenerateClearanceDeclaration(shipmentID string, hsCode string, dutyRate float64, port string) CustomsClearanceDeclaration {
	e.mu.Lock()
	defer e.mu.Unlock()

	val, exists := e.valuations[shipmentID]
	cif := 150000.0
	if exists {
		cif = val.CalculatedDutiableCIFUSD
	}

	importDuty := (dutyRate / 100.0) * cif
	vatBase := cif + importDuty
	importVAT := (11.0 / 100.0) * vatBase // 11% standard import VAT
	incomeTax22 := (2.5 / 100.0) * vatBase // 2.5% PPh 22 with API-U license
	totalTaxes := importDuty + importVAT + incomeTax22

	decl := CustomsClearanceDeclaration{
		DeclarationNumber:     fmt.Sprintf("PIB-%s-%d", port, time.Now().UnixNano()%1000000),
		ShipmentID:            shipmentID,
		HSCode:                hsCode,
		DeclaredCIFValueUSD:   cif,
		ImportDutyRatePercent: dutyRate,
		ImportDutyPayableUSD:  importDuty,
		ImportVATRatePercent:  11.0,
		ImportVATPayableUSD:   importVAT,
		ImportIncomeTax22USD:  incomeTax22,
		TotalCustomsTaxesUSD:  totalTaxes,
		PortOfEntry:           port,
		CustomsStatus:         "GREEN_LINE_CLEARED",
		FilingTimestamp:       time.Now(),
	}

	e.declarations[shipmentID] = decl
	return decl
}

// GetAuditSummary builds complete customs and tax file
func (e *GATTHarmonizerEngine) GetAuditSummary(shipmentID string) HarmonizerAuditSummary {
	e.mu.RLock()
	defer e.mu.RUnlock()

	val := e.valuations[shipmentID]
	recon := e.reconciles[shipmentID]
	decl := e.declarations[shipmentID]

	verdict := "FULL_GATT_AND_OECD_HARMONIZED"
	if recon.AlignmentStatus != "HARMONIZED_IN_RANGE" && recon.AlignmentStatus != "" {
		verdict = "REQUIRES_CUSTOMS_TP_RECONCILIATION_ACCRUAL"
	}

	return HarmonizerAuditSummary{
		ShipmentID:       shipmentID,
		GATTValuation:    val,
		TPReconciliation: recon,
		Declaration:      decl,
		AuditVerdict:     verdict,
		AuditedAt:        time.Now(),
	}
}

// RegisterGATTRoutes registers REST API endpoints for GATT Valuation & TP Harmonization
func RegisterGATTRoutes(r *gin.RouterGroup) {
	group := r.Group("/customs/gatt-valuation")
	{
		group.POST("/evaluate-transaction-value", func(c *gin.Context) {
			var req GATTTransactionValue
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetGATTHarmonizerEngine().EvaluateTransactionValue(req)
			c.JSON(http.StatusOK, gin.H{
				"status":    "VALUATION_COMPLETED",
				"valuation": res,
			})
		})

		group.POST("/reconcile-transfer-pricing", func(c *gin.Context) {
			var req struct {
				ShipmentID string  `json:"shipment_id" binding:"required"`
				UnitCIF    float64 `json:"unit_cif" binding:"required"`
				Q1         float64 `json:"q1" binding:"required"`
				Median     float64 `json:"median" binding:"required"`
				Q3         float64 `json:"q3" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetGATTHarmonizerEngine().ReconcileTransferPricing(req.ShipmentID, req.UnitCIF, req.Q1, req.Median, req.Q3)
			c.JSON(http.StatusOK, gin.H{
				"status":         "RECONCILIATION_COMPLETED",
				"reconciliation": res,
			})
		})

		group.POST("/generate-clearance-declaration", func(c *gin.Context) {
			var req struct {
				ShipmentID string  `json:"shipment_id" binding:"required"`
				HSCode     string  `json:"hs_code" binding:"required"`
				DutyRate   float64 `json:"duty_rate" binding:"required"`
				Port       string  `json:"port" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			decl := GetGATTHarmonizerEngine().GenerateClearanceDeclaration(req.ShipmentID, req.HSCode, req.DutyRate, req.Port)
			c.JSON(http.StatusOK, gin.H{
				"status":      "DECLARATION_GENERATED",
				"declaration": decl,
			})
		})

		group.GET("/audit-summary/:shipment_id", func(c *gin.Context) {
			shipmentID := c.Param("shipment_id")
			audit := GetGATTHarmonizerEngine().GetAuditSummary(shipmentID)
			c.JSON(http.StatusOK, gin.H{
				"status": "AUDIT_SUMMARY_COMPILED",
				"audit":  audit,
			})
		})
	}
}
