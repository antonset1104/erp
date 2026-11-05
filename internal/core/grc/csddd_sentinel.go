package grc

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SupplierHumanRightsProfile encapsulates social audit and modern slavery diligence metrics.
type SupplierHumanRightsProfile struct {
	SupplierID              string  `json:"supplier_id"` // e.g. "SUP-GARMENT-BD-01"
	SupplierName            string  `json:"supplier_name"`
	FactoryLocation         string  `json:"factory_location"`
	CountryCode             string  `json:"country_code"`
	TierLevel               int     `json:"tier_level"`            // 1 = Direct Tier-1, 2 = Fabric mill, 3 = Raw farm
	SocialAuditStandard     string  `json:"social_audit_standard"` // "SMETA_4_PILLAR", "SA8000", "WRAP_GOLD"
	LastAuditDate           string  `json:"last_audit_date"`
	ForcedLaborRiskScore    float64 `json:"forced_labor_risk_score"` // 0.0 - 100.0 (Higher = worse)
	ChildLaborRiskScore     float64 `json:"child_labor_risk_score"`
	FairWageScore           float64 `json:"fair_wage_score"` // 0.0 - 100.0 (Higher = better)
	CompositeCSDDDRiskScore float64 `json:"composite_csddd_risk_score"`
	ComplianceStanding      string  `json:"compliance_standing"` // "LOW_RISK_APPROVED", "ELEVATED_AUDIT_REQUIRED", "HIGH_RISK_SUSPENDED"
}

// SupplyChainGrievanceReport captures labor grievance notifications from factory workers or NGOs.
type SupplyChainGrievanceReport struct {
	GrievanceID              string `json:"grievance_id"`
	SupplierID               string `json:"supplier_id"`
	IncidentType             string `json:"incident_type"` // "FORCED_OVERTIME_WITHOUT_PAY", "UNSAFE_FIRE_EXITS", "WITHHELD_PASSPORTS", "UNDERAGE_LABOR"
	Severity                 string `json:"severity"`      // "CRITICAL_HUMAN_RIGHTS_BREACH", "SIGNIFICANT", "OBSERVATION"
	ReporterChannel          string `json:"reporter_channel"`
	Narrative                string `json:"narrative"`
	MandatoryRemediationCAPA string `json:"mandatory_remediation_capa"`
	Status                   string `json:"status"` // "INVESTIGATING", "REMEDIATION_ENFORCED", "CLOSED"
	ReportedAt               string `json:"reported_at"`
}

// CSDDDAnnualDeclaration formalizes the statutory disclosure required by EU and German LkSG authorities.
type CSDDDAnnualDeclaration struct {
	DeclarationID               string                       `json:"declaration_id"`
	ReportingYear               int                          `json:"reporting_year"`
	EnterpriseName              string                       `json:"enterprise_name"`
	TotalSuppliersAudited       int                          `json:"total_suppliers_audited"`
	HighRiskSuppliersCount      int                          `json:"high_risk_suppliers_count"`
	GrievancesInvestigatedCount int                          `json:"grievances_investigated_count"`
	RemediationSuccessRatePct   float64                      `json:"remediation_success_rate_pct"`
	StatutoryFineExposureUSD    float64                      `json:"statutory_fine_exposure_usd"`
	AuditComplianceStatus       string                       `json:"audit_compliance_status"` // "FULL_EU_CSDDD_COMPLIANCE", "ACTION_PLAN_IN_PROGRESS"
	MonitoredSuppliers          []SupplierHumanRightsProfile `json:"monitored_suppliers"`
	GeneratedAt                 string                       `json:"generated_at"`
}

// EvaluateSupplierRequest holds parameters to evaluate a vendor's social responsibility risk.
type EvaluateSupplierRequest struct {
	SupplierID          string  `json:"supplier_id"`
	SupplierName        string  `json:"supplier_name"`
	FactoryLocation     string  `json:"factory_location"`
	CountryCode         string  `json:"country_code"`
	TierLevel           int     `json:"tier_level"`
	SocialAuditStandard string  `json:"social_audit_standard"`
	ForcedLaborIndex    float64 `json:"forced_labor_index"`
	ChildLaborIndex     float64 `json:"child_labor_index"`
	FairWageRating      float64 `json:"fair_wage_rating"`
}

// CSDDDStore manages thread-safe storage for human rights due diligence.
type CSDDDStore struct {
	mu          sync.RWMutex
	suppliers   map[string]*SupplierHumanRightsProfile
	grievances  []SupplyChainGrievanceReport
	declaration *CSDDDAnnualDeclaration
}

var globalCSDDDStore = &CSDDDStore{
	suppliers:  make(map[string]*SupplierHumanRightsProfile),
	grievances: make([]SupplyChainGrievanceReport, 0),
}

func init() {
	// Seed demo supplier profile
	demoSupplier := &SupplierHumanRightsProfile{
		SupplierID:              "SUP-GARMENT-BD-01",
		SupplierName:            "Apex Textile & Industrial Apparel Ltd",
		FactoryLocation:         "Dhaka Export Processing Zone",
		CountryCode:             "BD",
		TierLevel:               1,
		SocialAuditStandard:     "SMETA_4_PILLAR",
		LastAuditDate:           "2026-01-18",
		ForcedLaborRiskScore:    18.5,
		ChildLaborRiskScore:     8.0,
		FairWageScore:           88.0,
		CompositeCSDDDRiskScore: 21.5,
		ComplianceStanding:      "LOW_RISK_APPROVED",
	}
	globalCSDDDStore.suppliers[demoSupplier.SupplierID] = demoSupplier

	// Seed demo grievance
	globalCSDDDStore.grievances = append(globalCSDDDStore.grievances, SupplyChainGrievanceReport{
		GrievanceID:              "GRV-CSDDD-2026-001",
		SupplierID:               "SUP-GARMENT-BD-01",
		IncidentType:             "UNSAFE_FIRE_EXITS",
		Severity:                 "SIGNIFICANT",
		ReporterChannel:          "ANONYMOUS_MOBILE_HOTLINE",
		Narrative:                "Secondary emergency staircase on floor 3 was temporarily padlocked during maintenance.",
		MandatoryRemediationCAPA: "Remove padlocks immediately and install electronic fail-safe magnetic panic bars.",
		Status:                   "REMEDIATION_ENFORCED",
		ReportedAt:               time.Now().AddDate(0, -1, 0).Format(time.RFC3339),
	})
}

// EvaluateSupplierCSDDDLogic scores human rights parameters and determines approval standing.
func EvaluateSupplierCSDDDLogic(req EvaluateSupplierRequest) (*SupplierHumanRightsProfile, error) {
	globalCSDDDStore.mu.Lock()
	defer globalCSDDDStore.mu.Unlock()

	if req.SupplierID == "" {
		req.SupplierID = fmt.Sprintf("SUP-VND-%d", time.Now().UnixNano()%1000000)
	}
	if req.TierLevel <= 0 {
		req.TierLevel = 1
	}

	// Calculate weighted composite CSDDD risk score (0-100)
	// Forced labor (40%), Child labor (35%), Fair wage deficit (25%)
	fairWageDeficit := math.Max(0, 100.0-req.FairWageRating)
	composite := (req.ForcedLaborIndex * 0.40) + (req.ChildLaborIndex * 0.35) + (fairWageDeficit * 0.25)

	standing := "LOW_RISK_APPROVED"
	if composite >= 60.0 || req.ForcedLaborIndex >= 50.0 || req.ChildLaborIndex >= 30.0 {
		standing = "HIGH_RISK_SUSPENDED"
	} else if composite >= 35.0 {
		standing = "ELEVATED_AUDIT_REQUIRED"
	}

	profile := &SupplierHumanRightsProfile{
		SupplierID:              req.SupplierID,
		SupplierName:            req.SupplierName,
		FactoryLocation:         req.FactoryLocation,
		CountryCode:             req.CountryCode,
		TierLevel:               req.TierLevel,
		SocialAuditStandard:     req.SocialAuditStandard,
		LastAuditDate:           time.Now().Format("2006-01-02"),
		ForcedLaborRiskScore:    req.ForcedLaborIndex,
		ChildLaborRiskScore:     req.ChildLaborIndex,
		FairWageScore:           req.FairWageRating,
		CompositeCSDDDRiskScore: math.Round(composite*10) / 10,
		ComplianceStanding:      standing,
	}

	globalCSDDDStore.suppliers[req.SupplierID] = profile
	return profile, nil
}

// LogSupplyChainGrievanceLogic captures worker disclosures and assigns mandatory corrective actions.
func LogSupplyChainGrievanceLogic(g SupplyChainGrievanceReport) (*SupplyChainGrievanceReport, error) {
	globalCSDDDStore.mu.Lock()
	defer globalCSDDDStore.mu.Unlock()

	if g.GrievanceID == "" {
		g.GrievanceID = fmt.Sprintf("GRV-CSDDD-%d", time.Now().UnixNano()%1000000)
	}
	if g.ReportedAt == "" {
		g.ReportedAt = time.Now().Format(time.RFC3339)
	}
	g.Status = "REMEDIATION_ENFORCED"

	globalCSDDDStore.grievances = append(globalCSDDDStore.grievances, g)
	return &g, nil
}

// GenerateCSDDDAnnualDeclarationLogic synthesizes official EU CSDDD and German LkSG filing declarations.
func GenerateCSDDDAnnualDeclarationLogic(year int) (*CSDDDAnnualDeclaration, error) {
	globalCSDDDStore.mu.RLock()
	defer globalCSDDDStore.mu.RUnlock()

	if year <= 0 {
		year = 2026
	}

	highRiskCount := 0
	suppList := make([]SupplierHumanRightsProfile, 0, len(globalCSDDDStore.suppliers))
	for _, s := range globalCSDDDStore.suppliers {
		suppList = append(suppList, *s)
		if s.ComplianceStanding == "HIGH_RISK_SUSPENDED" {
			highRiskCount++
		}
	}

	remediated := 0
	for _, grv := range globalCSDDDStore.grievances {
		if grv.Status == "REMEDIATION_ENFORCED" || grv.Status == "CLOSED" {
			remediated++
		}
	}

	successRate := 100.0
	if len(globalCSDDDStore.grievances) > 0 {
		successRate = (float64(remediated) / float64(len(globalCSDDDStore.grievances))) * 100.0
	}

	status := "FULL_EU_CSDDD_COMPLIANCE"
	fineExposure := 0.0
	if highRiskCount > 0 {
		status = "ACTION_PLAN_IN_PROGRESS"
		fineExposure = 250000.0 // Contingency regulatory fine risk
	}

	declID := fmt.Sprintf("DECL-CSDDD-EU-%d", year)
	decl := &CSDDDAnnualDeclaration{
		DeclarationID:               declID,
		ReportingYear:               year,
		EnterpriseName:              "GoERP Global Supply Chain Operations SE",
		TotalSuppliersAudited:       len(suppList),
		HighRiskSuppliersCount:      highRiskCount,
		GrievancesInvestigatedCount: len(globalCSDDDStore.grievances),
		RemediationSuccessRatePct:   math.Round(successRate*10) / 10,
		StatutoryFineExposureUSD:    fineExposure,
		AuditComplianceStatus:       status,
		MonitoredSuppliers:          suppList,
		GeneratedAt:                 time.Now().Format(time.RFC3339),
	}

	return decl, nil
}

// RegisterCSDDDRoutes registers supply chain human rights due diligence endpoints.
func RegisterCSDDDRoutes(r *gin.RouterGroup) {
	group := r.Group("/grc/csddd")
	{
		group.POST("/evaluate-supplier", func(c *gin.Context) {
			var req EvaluateSupplierRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid evaluate supplier payload: " + err.Error()})
				return
			}
			res, err := EvaluateSupplierCSDDDLogic(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/log-grievance", func(c *gin.Context) {
			var g SupplyChainGrievanceReport
			if err := c.ShouldBindJSON(&g); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid grievance report payload: " + err.Error()})
				return
			}
			res, err := LogSupplyChainGrievanceLogic(g)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.GET("/annual-declaration", func(c *gin.Context) {
			res, err := GenerateCSDDDAnnualDeclarationLogic(2026)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
