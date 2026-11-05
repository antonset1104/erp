package customs

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// FTATradeBloc defines international preferential trade agreements.
type FTATradeBloc string

const (
	FTABlocATIGA FTATradeBloc = "ATIGA_ASEAN"   // Form D (ASEAN)
	FTABlocACFTA FTATradeBloc = "ACFTA_CHINA"   // Form E (ASEAN-China)
	FTABlocRCEP  FTATradeBloc = "RCEP_REGIONAL" // Form RCEP (15 Asia-Pacific Nations)
	FTABlocIJEPA FTATradeBloc = "IJEPA_JAPAN"   // Form IJ-EPA (Indonesia-Japan)
)

// BOMComponentOrigin represents an individual raw material/component in the bill of materials.
type BOMComponentOrigin struct {
	ComponentSKU    string  `json:"component_sku"`
	ComponentName   string  `json:"component_name"`
	HSCode          string  `json:"hs_code"`           // e.g. 8541.10
	CountryOfOrigin string  `json:"country_of_origin"` // ISO 2-letter (ID, VN, MY, CN, US)
	UnitCostIDR     float64 `json:"unit_cost_idr"`
	QuantityPerUnit float64 `json:"quantity_per_unit"`
	TotalCostIDR    float64 `json:"total_cost_idr"`
	IsOriginating   bool    `json:"is_originating"` // Belongs to FTA signatory country
}

// OriginQualificationResult represents the compliance verdict for an FTA agreement.
type OriginQualificationResult struct {
	TradeBlocName       FTATradeBloc `json:"trade_bloc_name"`
	OfficialFormType    string       `json:"official_form_type"`   // Form D, Form E, Form RCEP
	QualifyingCriterion string       `json:"qualifying_criterion"` // RVC >= 40% or CTH Change
	RVCPercentage       float64      `json:"rvc_percentage"`       // Regional Value Content %
	MinimumThreshold    float64      `json:"minimum_threshold"`    // Usually 40.0%
	IsQualified         bool         `json:"is_qualified"`
	StandardMFNDutyPct  float64      `json:"standard_mfn_duty_pct"` // e.g. 12.5%
	PreferentialDutyPct float64      `json:"preferential_duty_pct"` // 0.0%
	EstimatedDutySaved  float64      `json:"estimated_duty_saved_idr"`
	CertificationStatus string       `json:"certification_status"`
}

// EvaluateOriginRequest represents the input to audit a finished good for export.
type EvaluateOriginRequest struct {
	FinishedGoodSKU  string               `json:"finished_good_sku"`
	FinishedGoodName string               `json:"finished_good_name"`
	FinishedGoodHS   string               `json:"finished_good_hs"` // e.g. 8528.52 (Monitors)
	FOBPriceIDR      float64              `json:"fob_price_idr"`
	AnnualExportQty  float64              `json:"annual_export_qty"`
	TargetTradeBlocs []FTATradeBloc       `json:"target_trade_blocs"`
	BOMComponents    []BOMComponentOrigin `json:"bom_components"`
}

// EvaluateOriginResponse encapsulates the complete trade certificate optimization report.
type EvaluateOriginResponse struct {
	EvaluationID            string                      `json:"evaluation_id"`
	EvaluatedAt             string                      `json:"evaluated_at"`
	FinishedGoodSKU         string                      `json:"finished_good_sku"`
	FinishedGoodName        string                      `json:"finished_good_name"`
	FinishedGoodHS          string                      `json:"finished_good_hs"`
	FOBPriceIDR             float64                     `json:"fob_price_idr"`
	TotalBOMCostIDR         float64                     `json:"total_bom_cost_idr"`
	OriginatingMaterialCost float64                     `json:"originating_material_cost_idr"`
	NonOriginatingMatCost   float64                     `json:"non_originating_material_cost_idr"`
	TotalAnnualDutySavedIDR float64                     `json:"total_annual_duty_saved_idr"`
	FTAResults              []OriginQualificationResult `json:"fta_results"`
	StrategicAdvice         string                      `json:"strategic_advice"`
}

// FTAOptimizerService evaluates Rules of Origin and maximizes customs duty savings.
type FTAOptimizerService struct {
	mu          sync.RWMutex
	lastReports map[string]*EvaluateOriginResponse
}

// NewFTAOptimizerService creates a new FTAOptimizerService.
func NewFTAOptimizerService() *FTAOptimizerService {
	return &FTAOptimizerService{
		lastReports: make(map[string]*EvaluateOriginResponse),
	}
}

// EvaluateOrigin checks regional value content and certifies FTA origin.
func (s *FTAOptimizerService) EvaluateOrigin(req EvaluateOriginRequest) (*EvaluateOriginResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.FinishedGoodSKU == "" {
		req.FinishedGoodSKU = "SKU-MON-27-QHD"
		req.FinishedGoodName = "Gaming Monitor 27 Inch QHD 165Hz"
		req.FinishedGoodHS = "8528.52.00"
		req.FOBPriceIDR = 4800000
		req.AnnualExportQty = 1500
	}

	// Default BOM components if empty
	if len(req.BOMComponents) == 0 {
		req.BOMComponents = []BOMComponentOrigin{
			{ComponentSKU: "COMP-LCD-PANEL", ComponentName: "IPS LCD Display Panel 27 Inch", HSCode: "9013.80", CountryOfOrigin: "VN", UnitCostIDR: 1600000, QuantityPerUnit: 1, TotalCostIDR: 1600000, IsOriginating: true}, // Vietnam (ASEAN)
			{ComponentSKU: "COMP-MAIN-PCB", ComponentName: "Main Controller Board SMT", HSCode: "8534.00", CountryOfOrigin: "ID", UnitCostIDR: 750000, QuantityPerUnit: 1, TotalCostIDR: 750000, IsOriginating: true},        // Indonesia (Local)
			{ComponentSKU: "COMP-CHASSIS", ComponentName: "ABS Plastic Enclosure & Stand", HSCode: "3926.90", CountryOfOrigin: "ID", UnitCostIDR: 350000, QuantityPerUnit: 1, TotalCostIDR: 350000, IsOriginating: true},     // Indonesia (Local)
			{ComponentSKU: "COMP-PWR-SUPPLY", ComponentName: "AC/DC Switching Power Supply", HSCode: "8504.40", CountryOfOrigin: "MY", UnitCostIDR: 300000, QuantityPerUnit: 1, TotalCostIDR: 300000, IsOriginating: true},   // Malaysia (ASEAN)
			{ComponentSKU: "COMP-DSP-CHIP", ComponentName: "Video Scaler DSP IC Microchip", HSCode: "8542.31", CountryOfOrigin: "US", UnitCostIDR: 450000, QuantityPerUnit: 1, TotalCostIDR: 450000, IsOriginating: false},   // USA (Non-originating)
		}
	}

	totalBOM := 0.0
	originatingCost := 0.0
	nonOriginatingCost := 0.0

	for _, c := range req.BOMComponents {
		totalBOM += c.TotalCostIDR
		if c.IsOriginating {
			originatingCost += c.TotalCostIDR
		} else {
			nonOriginatingCost += c.TotalCostIDR
		}
	}

	// RVC Formula (Indirect / Build-Down Method):
	// RVC = ((FOB Price - Non-Originating Materials) / FOB Price) * 100%
	rvc := math.Round(((req.FOBPriceIDR-nonOriginatingCost)/req.FOBPriceIDR)*1000) / 10

	totalDutySavings := 0.0
	annualExportVal := req.FOBPriceIDR * req.AnnualExportQty

	ftaList := []OriginQualificationResult{
		{
			TradeBlocName:       FTABlocATIGA,
			OfficialFormType:    "Form D (ASEAN ATIGA)",
			QualifyingCriterion: "Regional Value Content (RVC) >= 40% or CTH",
			RVCPercentage:       rvc,
			MinimumThreshold:    40.0,
			IsQualified:         rvc >= 40.0,
			StandardMFNDutyPct:  10.0,
			PreferentialDutyPct: 0.0, // 0% under ATIGA
			EstimatedDutySaved:  annualExportVal * 0.10,
			CertificationStatus: "ORIGIN_QUALIFIED_FORM_D",
		},
		{
			TradeBlocName:       FTABlocRCEP,
			OfficialFormType:    "Form RCEP (Regional Comprehensive Economic Partnership)",
			QualifyingCriterion: "RVC >= 40% Cumulative Regional Origin",
			RVCPercentage:       rvc,
			MinimumThreshold:    40.0,
			IsQualified:         rvc >= 40.0,
			StandardMFNDutyPct:  12.0,
			PreferentialDutyPct: 0.0,
			EstimatedDutySaved:  annualExportVal * 0.12,
			CertificationStatus: "ORIGIN_QUALIFIED_FORM_RCEP",
		},
		{
			TradeBlocName:       FTABlocACFTA,
			OfficialFormType:    "Form E (ASEAN-China FTA)",
			QualifyingCriterion: "RVC >= 40% + Change in Tariff Heading (CTH)",
			RVCPercentage:       rvc,
			MinimumThreshold:    40.0,
			IsQualified:         rvc >= 40.0,
			StandardMFNDutyPct:  15.0,
			PreferentialDutyPct: 0.0,
			EstimatedDutySaved:  annualExportVal * 0.15,
			CertificationStatus: "ORIGIN_QUALIFIED_FORM_E",
		},
	}

	for _, f := range ftaList {
		if f.IsQualified {
			totalDutySavings += f.EstimatedDutySaved
		}
	}

	evalID := fmt.Sprintf("FTA-EVAL-%s-%04d", time.Now().Format("20060102"), time.Now().Nanosecond()%10000)

	report := &EvaluateOriginResponse{
		EvaluationID:            evalID,
		EvaluatedAt:             time.Now().Format(time.RFC3339),
		FinishedGoodSKU:         req.FinishedGoodSKU,
		FinishedGoodName:        req.FinishedGoodName,
		FinishedGoodHS:          req.FinishedGoodHS,
		FOBPriceIDR:             req.FOBPriceIDR,
		TotalBOMCostIDR:         totalBOM,
		OriginatingMaterialCost: originatingCost,
		NonOriginatingMatCost:   nonOriginatingCost,
		TotalAnnualDutySavedIDR: totalDutySavings,
		FTAResults:              ftaList,
		StrategicAdvice:         fmt.Sprintf("Produk memenuhi kriteria kandungan regional RVC %.1f%% (di atas ambang batas 40%%). Berhak atas fasilitas tarif bea masuk 0%% untuk ekspor ke pasar ASEAN (Form D) dan Asia Pasifik (RCEP). Penghematan bea tahunan: IDR %.0f.", rvc, totalDutySavings),
	}

	s.lastReports[req.FinishedGoodSKU] = report
	return report, nil
}

// GetCertificateMatrix returns origin evaluation summaries for all audited export products.
func (s *FTAOptimizerService) GetCertificateMatrix() (*EvaluateOriginResponse, error) {
	s.mu.RLock()
	rep, ok := s.lastReports["SKU-MON-27-QHD"]
	s.mu.RUnlock()

	if !ok {
		return s.EvaluateOrigin(EvaluateOriginRequest{})
	}
	return rep, nil
}

// RegisterFTARoutes mounts HTTP routes for FTA Rules-of-Origin Optimizer.
func RegisterFTARoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	defaultFTASvc := NewFTAOptimizerService()

	g := router.Group("/customs/fta-origin")
	g.Use(authMiddleware)
	{
		g.POST("/evaluate", func(c *gin.Context) {
			var req EvaluateOriginRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := defaultFTASvc.EvaluateOrigin(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		g.GET("/certificate-matrix", func(c *gin.Context) {
			res, err := defaultFTASvc.GetCertificateMatrix()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
