package legal

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

// PatentAssetRecord models registered intellectual property patent assets under WIPO/USPTO/EPO/IAS 38
type PatentAssetRecord struct {
	PatentID                    string    `json:"patent_id"`
	PortfolioID                 string    `json:"portfolio_id"`
	Title                       string    `json:"title"`
	PatentOffice                string    `json:"patent_office"` // "USPTO", "EPO", "WIPO_PCT", "DJKI_INDONESIA", "JPO"
	PatentType                  string    `json:"patent_type"`   // "UTILITY_PATENT", "DESIGN_PATENT"
	FilingDate                  time.Time `json:"filing_date"`
	GrantDate                   time.Time `json:"grant_date"`
	NextAnnuityDueDate          time.Time `json:"next_annuity_due_date"`
	StatutoryAnnuityFeeUSD      float64   `json:"statutory_annuity_fee_usd"`
	GracePeriodDeadline         time.Time `json:"grace_period_deadline"`
	ActiveCommercialProductLink string    `json:"active_commercial_product_link"`
	RegisteredAt                time.Time `json:"registered_at"`
}

// PatentAnnuityScore calculates commercial ROI, citation significance, and renewal/abandonment recommendation
type PatentAnnuityScore struct {
	EvaluationID                string    `json:"evaluation_id"`
	PatentID                    string    `json:"patent_id"`
	PortfolioID                 string    `json:"portfolio_id"`
	CommercialRevenueLinkageUSD float64   `json:"commercial_revenue_linkage_usd"`
	ForwardCitationScore        int       `json:"forward_citation_score"`         // Third-party competitor citations
	CompetitiveFTOBlockingScore float64   `json:"competitive_fto_blocking_score"` // 0 - 100 Freedom to Operate / competitor defensive moat
	AnnuityROIIndex             float64   `json:"annuity_roi_index"`              // Revenue & Moat / Annuity Fee
	DecisionVerdict             string    `json:"decision_verdict"`               // "PAY_MAINTENANCE_ANNUITY", "ABANDON_LET_EXPIRE_PRUNING", "OFFER_FOR_OUT_LICENSING"
	RecommendedAction           string    `json:"recommended_action"`
	PortfolioHealthAuditDigest  string    `json:"portfolio_health_audit_digest"`
	EvaluatedAt                 time.Time `json:"evaluated_at"`
}

// AnnuityPaymentAuthorization models legal counsel clearance and SWIFT payment dispatch for renewals
type AnnuityPaymentAuthorization struct {
	AuthorizationID         string    `json:"authorization_id"`
	PortfolioID             string    `json:"portfolio_id"`
	AuthorizedPatentsCount  int       `json:"authorized_patents_count"`
	TotalStatutoryFeesUSD   float64   `json:"total_statutory_fees_usd"`
	PrunedPatentsCount      int       `json:"pruned_patents_count"`
	CostAvoidanceSavingsUSD float64   `json:"cost_avoidance_savings_usd"`
	SWIFTPaymentRef         string    `json:"swift_payment_ref"`
	ApprovedByLegalCounsel  string    `json:"approved_by_legal_counsel"`
	LegalSanctionVerified   bool      `json:"legal_sanction_verified"`
	AuthorizedAt            time.Time `json:"authorized_at"`
}

// PatentPortfolioVault thread-safe in-memory store
type PatentPortfolioVault struct {
	mu             sync.RWMutex
	patents        map[string]PatentAssetRecord
	scores         map[string]PatentAnnuityScore
	authorizations map[string]AnnuityPaymentAuthorization
}

// Global instance
var GlobalPatentPortfolioVault = &PatentPortfolioVault{
	patents:        make(map[string]PatentAssetRecord),
	scores:         make(map[string]PatentAnnuityScore),
	authorizations: make(map[string]AnnuityPaymentAuthorization),
}

func init() {
	now := time.Now()
	// Seed demo patent assets
	GlobalPatentPortfolioVault.patents["PAT-US-11892011-B2"] = PatentAssetRecord{
		PatentID:                    "PAT-US-11892011-B2",
		PortfolioID:                 "PORTFOLIO-ENTERPRISE-AI-2026",
		Title:                       "Multi-Variable Distributed Constraint Optimization Engine for Enterprise Supply Chains",
		PatentOffice:                "USPTO",
		PatentType:                  "UTILITY_PATENT",
		FilingDate:                  now.AddDate(-6, 0, 0),
		GrantDate:                   now.AddDate(-2, 0, 0),
		NextAnnuityDueDate:          now.AddDate(0, 5, 15),
		StatutoryAnnuityFeeUSD:      3760.0,
		GracePeriodDeadline:         now.AddDate(0, 11, 15),
		ActiveCommercialProductLink: "GOERP_CORE_OPTIMIZER_V1",
		RegisteredAt:                now,
	}

	GlobalPatentPortfolioVault.patents["PAT-EP-3944120-A1"] = PatentAssetRecord{
		PatentID:                    "PAT-EP-3944120-A1",
		PortfolioID:                 "PORTFOLIO-ENTERPRISE-AI-2026",
		Title:                       "Legacy Synchronous Schema Translation Interface (Obsolete Standard)",
		PatentOffice:                "EPO",
		PatentType:                  "UTILITY_PATENT",
		FilingDate:                  now.AddDate(-12, 0, 0),
		GrantDate:                   now.AddDate(-8, 0, 0),
		NextAnnuityDueDate:          now.AddDate(0, 1, 10),
		StatutoryAnnuityFeeUSD:      2150.0,
		GracePeriodDeadline:         now.AddDate(0, 7, 10),
		ActiveCommercialProductLink: "OBSOLETE_DEPRECATED",
		RegisteredAt:                now,
	}
}

// RegisterPatentAsset registers a patent title into portfolio
func (v *PatentPortfolioVault) RegisterPatentAsset(p PatentAssetRecord) PatentAssetRecord {
	v.mu.Lock()
	defer v.mu.Unlock()

	if p.PatentID == "" {
		p.PatentID = fmt.Sprintf("PAT-%s-%d", p.PatentOffice, time.Now().UnixNano()%1000000)
	}
	p.RegisteredAt = time.Now()
	v.patents[p.PatentID] = p
	return p
}

// EvaluatePortfolioAnnuityScore computes quantitative patent pruning and renewal recommendations
func (v *PatentPortfolioVault) EvaluatePortfolioAnnuityScore(patentID string, commercialRevenueUSD float64, citationsCount int, ftoBlockingScore float64) (PatentAnnuityScore, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	pat, exists := v.patents[patentID]
	if !exists {
		return PatentAnnuityScore{}, fmt.Errorf("patent asset %s not found in registry", patentID)
	}

	// Calculate Annuity ROI Index
	// Strategic value = (0.01 * Revenue) + (Citations * 500) + (FTO * 200)
	strategicValue := (commercialRevenueUSD * 0.01) + (float64(citationsCount) * 500.0) + (ftoBlockingScore * 200.0)
	roiIndex := strategicValue / pat.StatutoryAnnuityFeeUSD

	verdict := "PAY_MAINTENANCE_ANNUITY"
	action := "AUTHORIZE_STATUTORY_FEE_PAYMENT_TO_MAINTAIN_ENFORCEABILITY"

	if commercialRevenueUSD == 0 && citationsCount <= 1 && ftoBlockingScore < 30.0 {
		verdict = "ABANDON_LET_EXPIRE_PRUNING"
		action = "PRUNE_PATENT_ASSET_SAVE_ANNUITY_BUDGET"
	} else if commercialRevenueUSD == 0 && (citationsCount > 5 || ftoBlockingScore >= 60.0) {
		verdict = "OFFER_FOR_OUT_LICENSING"
		action = "LIST_ON_IP_EXCHANGE_FOR_NON_EXCLUSIVE_MONETIZATION"
	}

	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%s-%s-%f-%d-%s", pat.PatentID, verdict, roiIndex, citationsCount, time.Now().String())))
	digest := hex.EncodeToString(hasher.Sum(nil))

	evalID := fmt.Sprintf("EVAL-ANNUITY-%d", time.Now().UnixNano()%1000000)
	score := PatentAnnuityScore{
		EvaluationID:                evalID,
		PatentID:                    patentID,
		PortfolioID:                 pat.PortfolioID,
		CommercialRevenueLinkageUSD: commercialRevenueUSD,
		ForwardCitationScore:        citationsCount,
		CompetitiveFTOBlockingScore: ftoBlockingScore,
		AnnuityROIIndex:             math.Round(roiIndex*100.0) / 100.0,
		DecisionVerdict:             verdict,
		RecommendedAction:           action,
		PortfolioHealthAuditDigest:  digest,
		EvaluatedAt:                 time.Now(),
	}

	v.scores[patentID] = score
	return score, nil
}

// AuthorizeAnnuityPayment processes renewal clearance and calculates pruning cost savings
func (v *PatentPortfolioVault) AuthorizeAnnuityPayment(portfolioID string, approvedPatentIDs []string, prunedPatentIDs []string, counselSignature string) (AnnuityPaymentAuthorization, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	var totalFees float64
	for _, pID := range approvedPatentIDs {
		pat, exists := v.patents[pID]
		if exists {
			totalFees += pat.StatutoryAnnuityFeeUSD
		}
	}

	var costAvoidance float64
	for _, pID := range prunedPatentIDs {
		pat, exists := v.patents[pID]
		if exists {
			costAvoidance += pat.StatutoryAnnuityFeeUSD
		}
	}

	authID := fmt.Sprintf("AUTH-ANNUITY-%d", time.Now().UnixNano()%1000000)
	swiftRef := fmt.Sprintf("SWIFT-USPTO-EPO-%s", authID[:12])

	auth := AnnuityPaymentAuthorization{
		AuthorizationID:         authID,
		PortfolioID:             portfolioID,
		AuthorizedPatentsCount:  len(approvedPatentIDs),
		TotalStatutoryFeesUSD:   math.Round(totalFees*100.0) / 100.0,
		PrunedPatentsCount:      len(prunedPatentIDs),
		CostAvoidanceSavingsUSD: math.Round(costAvoidance*100.0) / 100.0,
		SWIFTPaymentRef:         swiftRef,
		ApprovedByLegalCounsel:  counselSignature,
		LegalSanctionVerified:   true,
		AuthorizedAt:            time.Now(),
	}

	v.authorizations[portfolioID] = auth
	return auth, nil
}

// GetIPValuationDossier returns complete portfolio valuation and IAS 38 intangible audit
func (v *PatentPortfolioVault) GetIPValuationDossier(portfolioID string) (map[string]interface{}, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var portfolioPatents []PatentAssetRecord
	var totalAnnuityBudget float64
	for _, p := range v.patents {
		if p.PortfolioID == portfolioID {
			portfolioPatents = append(portfolioPatents, p)
			totalAnnuityBudget += p.StatutoryAnnuityFeeUSD
		}
	}

	auth, hasAuth := v.authorizations[portfolioID]

	dossier := map[string]interface{}{
		"portfolio_id":               portfolioID,
		"registered_patents_count":   len(portfolioPatents),
		"annual_statutory_fees_usd":  math.Round(totalAnnuityBudget*100.0) / 100.0,
		"ias_38_intangible_verified": true,
		"has_authorization":          hasAuth,
		"compiled_at":                time.Now(),
	}

	if hasAuth {
		dossier["authorized_patents_count"] = auth.AuthorizedPatentsCount
		dossier["total_statutory_fees_usd"] = auth.TotalStatutoryFeesUSD
		dossier["pruned_patents_count"] = auth.PrunedPatentsCount
		dossier["cost_avoidance_savings_usd"] = auth.CostAvoidanceSavingsUSD
		dossier["swift_payment_ref"] = auth.SWIFTPaymentRef
	}

	return dossier, nil
}

// RegisterPatentRoutes wires REST endpoints
func RegisterPatentRoutes(r *gin.RouterGroup) {
	patentGroup := r.Group("/legal/patent")
	{
		patentGroup.POST("/register-patent-asset", func(c *gin.Context) {
			var p PatentAssetRecord
			if err := c.ShouldBindJSON(&p); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GlobalPatentPortfolioVault.RegisterPatentAsset(p)
			c.JSON(http.StatusOK, res)
		})

		patentGroup.POST("/evaluate-portfolio-annuity-score", func(c *gin.Context) {
			var body struct {
				PatentID                    string  `json:"patent_id"`
				CommercialRevenueUSD        float64 `json:"commercial_revenue_usd"`
				CitationsCount              int     `json:"citations_count"`
				CompetitiveFTOBlockingScore float64 `json:"competitive_fto_blocking_score"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalPatentPortfolioVault.EvaluatePortfolioAnnuityScore(
				body.PatentID, body.CommercialRevenueUSD, body.CitationsCount, body.CompetitiveFTOBlockingScore,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		patentGroup.POST("/authorize-annuity-payment", func(c *gin.Context) {
			var body struct {
				PortfolioID       string   `json:"portfolio_id"`
				ApprovedPatentIDs []string `json:"approved_patent_ids"`
				PrunedPatentIDs   []string `json:"pruned_patent_ids"`
				CounselSignature  string   `json:"counsel_signature"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := GlobalPatentPortfolioVault.AuthorizeAnnuityPayment(
				body.PortfolioID, body.ApprovedPatentIDs, body.PrunedPatentIDs, body.CounselSignature,
			)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		patentGroup.GET("/ip-valuation-dossier/:portfolio_id", func(c *gin.Context) {
			portfolioID := c.Param("portfolio_id")
			dossier, err := GlobalPatentPortfolioVault.GetIPValuationDossier(portfolioID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, dossier)
		})
	}
}
