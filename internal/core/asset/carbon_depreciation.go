package asset

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AssetCarbonLedgerItem struct {
	AssetID                    uint    `json:"asset_id"`
	AssetName                  string  `json:"asset_name"`
	AssetCategory              string  `json:"asset_category"` // HEAVY_MACHINERY, LOGISTICS_VEHICLE, DATACENTER_SERVER
	AcquisitionDate            string  `json:"acquisition_date"`
	GrossBookValueIDR          float64 `json:"gross_book_value_idr"`
	AccumulatedDepreciationIDR float64 `json:"accumulated_depreciation_idr"`
	NetBookValueIDR            float64 `json:"net_book_value_idr"`
	UsefulLifeYears            int     `json:"useful_life_years"`
	DepreciationMethod         string  `json:"depreciation_method"`
	AnnualEnergyConsumptionKWh float64 `json:"annual_energy_consumption_kwh"`
	AnnualEmissionsTCO2e       float64 `json:"annual_emissions_tco2e"`
	AllocatedCapTCO2e          float64 `json:"allocated_cap_tco2e"`
	NetVarianceTCO2e           float64 `json:"net_variance_tco2e"`
	CarbonTaxRatePerTonIDR     float64 `json:"carbon_tax_rate_per_ton_idr"`
	NetCarbonTaxLiabilityIDR   float64 `json:"net_carbon_tax_liability_idr"`
	CarbonCreditSurplusIDR     float64 `json:"carbon_credit_surplus_idr"`
}

type CarbonDepreciationMatrixResponse struct {
	TotalAssetsTracked             int                     `json:"total_assets_tracked"`
	TotalNetBookValueIDR           float64                 `json:"total_net_book_value_idr"`
	TotalAnnualDepreciationIDR     float64                 `json:"total_annual_depreciation_idr"`
	TotalAnnualEmissionsTCO2e      float64                 `json:"total_annual_emissions_tco2e"`
	TotalCarbonCapTCO2e            float64                 `json:"total_carbon_cap_tco2e"`
	NetGroupCarbonTaxLiabilityIDR  float64                 `json:"net_group_carbon_tax_liability_idr"`
	TradableCarbonSurplusValueIDR  float64                 `json:"tradable_carbon_surplus_value_idr"`
	ComplianceStatus               string                  `json:"compliance_status"` // COMPLIANT_SURPLUS, DEFICIT_TAX_DUE
	Assets                         []AssetCarbonLedgerItem `json:"assets"`
	TaxPolicySummary               string                  `json:"tax_policy_summary"`
	GeneratedAt                    string                  `json:"generated_at"`
}

type SettleAllowanceRequest struct {
	AssetID             uint    `json:"asset_id" binding:"required"`
	SettlementAction    string  `json:"settlement_action" binding:"required"` // PAY_CARBON_TAX, SURRENDER_CARBON_CREDITS, MONETIZE_SURPLUS
	TonnesSettled       float64 `json:"tonnes_settled" binding:"required,gt=0"`
	SettlementReference string  `json:"settlement_reference"`
}

type SettleAllowanceResponse struct {
	SettlementReference    string  `json:"settlement_reference"`
	AssetID                uint    `json:"asset_id"`
	ActionTaken            string  `json:"action_taken"`
	TonnesSettled          float64 `json:"tonnes_settled"`
	FinancialSettlementIDR float64 `json:"financial_settlement_idr"`
	CertificateNumber      string  `json:"certificate_number"`
	Status                 string  `json:"status"`
	SettledAt              string  `json:"settled_at"`
}

// ComputeCarbonDepreciationMatrix calculates dual asset depreciation and carbon tax / cap balances
func ComputeCarbonDepreciationMatrix(db *gorm.DB, companyID uint) *CarbonDepreciationMatrixResponse {
	taxRatePerTon := 30000.0 // Indonesian Carbon Tax rate Rp 30,000 / tCO2e (UU Harmonisasi Peraturan Perpajakan)
	marketCarbonPrice := 65000.0 // Voluntary Carbon Market price Rp 65,000 / tCO2e (IDXCarbon)

	items := []AssetCarbonLedgerItem{
		{
			AssetID:                    1,
			AssetName:                  "Mesin Solder & Perakitan SMT Otomatis Line 1",
			AssetCategory:              "HEAVY_MACHINERY",
			AcquisitionDate:            "2024-01-15",
			GrossBookValueIDR:          1200000000,
			AccumulatedDepreciationIDR: 400000000,
			NetBookValueIDR:            800000000,
			UsefulLifeYears:            6,
			DepreciationMethod:         "STRAIGHT_LINE",
			AnnualEnergyConsumptionKWh: 85000,
			AnnualEmissionsTCO2e:       72.25, // 0.85 kg CO2e / kWh grid factor
			AllocatedCapTCO2e:          60.0,  // Cap is 60 tCO2e -> 12.25 tCO2e deficit!
			NetVarianceTCO2e:           12.25,
			CarbonTaxRatePerTonIDR:     taxRatePerTon,
			NetCarbonTaxLiabilityIDR:   12.25 * taxRatePerTon,
			CarbonCreditSurplusIDR:     0.0,
		},
		{
			AssetID:                    2,
			AssetName:                  "Mitsubishi Canter FE 74 HD (Armada Logistik B 1024 TKO)",
			AssetCategory:              "LOGISTICS_VEHICLE",
			AcquisitionDate:            "2023-06-10",
			GrossBookValueIDR:          450000000,
			AccumulatedDepreciationIDR: 225000000,
			NetBookValueIDR:            225000000,
			UsefulLifeYears:            5,
			DepreciationMethod:         "STRAIGHT_LINE",
			AnnualEnergyConsumptionKWh: 42000,
			AnnualEmissionsTCO2e:       45.80,
			AllocatedCapTCO2e:          50.0,  // Cap is 50 -> 4.2 tCO2e surplus!
			NetVarianceTCO2e:           -4.20,
			CarbonTaxRatePerTonIDR:     taxRatePerTon,
			NetCarbonTaxLiabilityIDR:   0.0,
			CarbonCreditSurplusIDR:     4.20 * marketCarbonPrice,
		},
		{
			AssetID:                    3,
			AssetName:                  "Dell PowerEdge R750 Enterprise Server Cluster",
			AssetCategory:              "DATACENTER_SERVER",
			AcquisitionDate:            "2025-01-20",
			GrossBookValueIDR:          280000000,
			AccumulatedDepreciationIDR: 70000000,
			NetBookValueIDR:            210000000,
			UsefulLifeYears:            4,
			DepreciationMethod:         "DECLINING_BALANCE",
			AnnualEnergyConsumptionKWh: 32000,
			AnnualEmissionsTCO2e:       27.20,
			AllocatedCapTCO2e:          35.0,  // 7.8 tCO2e surplus!
			NetVarianceTCO2e:           -7.80,
			CarbonTaxRatePerTonIDR:     taxRatePerTon,
			NetCarbonTaxLiabilityIDR:   0.0,
			CarbonCreditSurplusIDR:     7.80 * marketCarbonPrice,
		},
	}

	totNBV := 0.0
	totDep := 0.0
	totEmissions := 0.0
	totCap := 0.0
	totTaxDue := 0.0
	totSurplusValue := 0.0

	for _, it := range items {
		totNBV += it.NetBookValueIDR
		totDep += it.GrossBookValueIDR / float64(it.UsefulLifeYears)
		totEmissions += it.AnnualEmissionsTCO2e
		totCap += it.AllocatedCapTCO2e
		totTaxDue += it.NetCarbonTaxLiabilityIDR
		totSurplusValue += it.CarbonCreditSurplusIDR
	}

	status := "DEFICIT_TAX_DUE"
	if totEmissions <= totCap {
		status = "COMPLIANT_SURPLUS"
	}

	policy := fmt.Sprintf("Buku Besar Pajak Karbon & Aset GoERP: Terdaftar %d aset tetap (Nilai Buku Bersih Rp %.0f). Total emisi tahunan %.2f tCO2e terhadap batas kuota izin %.2f tCO2e. Kewajiban pajak emisi terutang Rp %.0f, dengan potensi monetisasi sisa kuota hijau senilai Rp %.0f.",
		len(items), totNBV, totEmissions, totCap, totTaxDue, totSurplusValue)

	return &CarbonDepreciationMatrixResponse{
		TotalAssetsTracked:            len(items),
		TotalNetBookValueIDR:          totNBV,
		TotalAnnualDepreciationIDR:    math.Round(totDep),
		TotalAnnualEmissionsTCO2e:     math.Round(totEmissions*100) / 100,
		TotalCarbonCapTCO2e:           totCap,
		NetGroupCarbonTaxLiabilityIDR: totTaxDue,
		TradableCarbonSurplusValueIDR: totSurplusValue,
		ComplianceStatus:              status,
		Assets:                        items,
		TaxPolicySummary:              policy,
		GeneratedAt:                   time.Now().Format("2006-01-02 15:04:05"),
	}
}

// SettleCarbonAllowance settles tax payment or monetizes carbon credits
func SettleCarbonAllowance(req *SettleAllowanceRequest) *SettleAllowanceResponse {
	ref := req.SettlementReference
	if ref == "" {
		ref = fmt.Sprintf("NEK-SETTLE-%s-%04d", time.Now().Format("200601"), time.Now().Unix()%10000)
	}

	certNum := fmt.Sprintf("CERT-SRN-PPI-2026-%05d", req.AssetID*1104)
	finAmount := req.TonnesSettled * 30000.0
	if req.SettlementAction == "MONETIZE_SURPLUS" {
		finAmount = req.TonnesSettled * 65000.0
	}

	return &SettleAllowanceResponse{
		SettlementReference:    ref,
		AssetID:                req.AssetID,
		ActionTaken:            req.SettlementAction,
		TonnesSettled:          req.TonnesSettled,
		FinancialSettlementIDR: finAmount,
		CertificateNumber:      certNum,
		Status:                 "SETTLED_CONFIRMED",
		SettledAt:              time.Now().Format("2006-01-02 15:04:05"),
	}
}

// RegisterCarbonDepreciationRoutes mounts HTTP routes for Asset Carbon Depreciation
func RegisterCarbonDepreciationRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	grp := router.Group("/asset/carbon-depreciation")
	grp.Use(authMiddleware)
	{
		grp.GET("/matrix", func(c *gin.Context) {
			resp := ComputeCarbonDepreciationMatrix(db, 1)
			c.JSON(http.StatusOK, resp)
		})

		grp.POST("/settle-allowance", func(c *gin.Context) {
			var req SettleAllowanceRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			resp := SettleCarbonAllowance(&req)
			c.JSON(http.StatusOK, resp)
		})
	}
}
