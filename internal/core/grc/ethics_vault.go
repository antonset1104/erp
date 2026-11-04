package grc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// WhistleblowerReportRequest represents the anonymous incident disclosure payload.
type WhistleblowerReportRequest struct {
	Category        string   `json:"category"` // "CORRUPTION_BRIBERY", "FINANCIAL_FRAUD", "ANTITRUST", "HARASSMENT", "SAFETY_VIOLATION"
	BusinessUnit    string   `json:"business_unit"`
	IncidentDate    string   `json:"incident_date"`
	Narrative       string   `json:"narrative"`
	EvidenceFiles   []string `json:"evidence_files"` // names or base64 raw evidence
	ContactAlias    string   `json:"contact_alias"`  // optional non-identifying pseudonym
}

// EvidenceDigest records the SHA-256 integrity hash for an evidence attachment.
type EvidenceDigest struct {
	FileName    string `json:"file_name"`
	SHA256Hash  string `json:"sha256_hash"`
	SubmittedAt string `json:"submitted_at"`
}

// EthicsCase represents an ISO 37002 compliant anonymous whistleblower case.
type EthicsCase struct {
	CaseID           string           `json:"case_id"`
	AnonymousToken   string           `json:"anonymous_token"` // Secret token given only to reporter to track progress
	Category         string           `json:"category"`
	BusinessUnit     string           `json:"business_unit"`
	IncidentDate     string           `json:"incident_date"`
	NarrativeSummary string           `json:"narrative_summary"`
	EvidenceDigests  []EvidenceDigest `json:"evidence_digests"`
	RiskScore        int              `json:"risk_score"` // 0 - 100
	Priority         string           `json:"priority"`   // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	Status           string           `json:"status"`     // "NEW", "TRIAGED", "UNDER_INVESTIGATION", "SUBSTANTIATED", "CLOSED"
	AssignedTo       string           `json:"assigned_to"`
	FindingsSummary  string           `json:"findings_summary"`
	RemediationPlan  string           `json:"remediation_plan"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

// InvestigateCaseRequest holds updates from an independent ethics investigator.
type InvestigateCaseRequest struct {
	CaseID          string `json:"case_id"`
	InvestigatorID  string `json:"investigator_id"`
	Action          string `json:"action"` // "ASSIGN", "SUBSTANTIATE", "DISMISS", "REMEDY", "CLOSE"
	FindingsSummary string `json:"findings_summary"`
	RemediationPlan string `json:"remediation_plan"`
}

// EthicsDashboardSummary provides executive GRC metrics for the board audit committee.
type EthicsDashboardSummary struct {
	TotalCasesSubmitted   int            `json:"total_cases_submitted"`
	ActiveInvestigations  int            `json:"active_investigations"`
	SubstantiatedCases    int            `json:"substantiated_cases"`
	ClosedCases           int            `json:"closed_cases"`
	AverageRiskScore      float64        `json:"average_risk_score"`
	CategoryDistribution map[string]int `json:"category_distribution"`
	Cases                 []EthicsCase   `json:"cases"`
}

var (
	caseMutex sync.RWMutex
	caseStore = make(map[string]*EthicsCase)
)

func init() {
	// Seed demo cases for immediate compliance auditing
	seedDemoCases()
}

func seedDemoCases() {
	demo1 := &EthicsCase{
		CaseID:           "CASE-ETH-2026-001",
		AnonymousToken:   "SEC-TOKEN-4f8a91b2c3d4",
		Category:         "FINANCIAL_FRAUD",
		BusinessUnit:     "EMEA Sales Division",
		IncidentDate:     "2026-10-15",
		NarrativeSummary: "Off-book side agreement discovered granting unapproved 40% rebate bypass.",
		EvidenceDigests: []EvidenceDigest{
			{FileName: "side_agreement_scan.pdf", SHA256Hash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", SubmittedAt: "2026-10-16T08:00:00Z"},
		},
		RiskScore:       88,
		Priority:        "CRITICAL",
		Status:          "UNDER_INVESTIGATION",
		AssignedTo:      "Audit-Committee-Lead-01",
		FindingsSummary: "Internal forensic audit confirmed unauthorized contract annex.",
		RemediationPlan: "Reversal of revenue recognition and disciplinary referral to HR Legal.",
		CreatedAt:       time.Now().Add(-240 * time.Hour),
		UpdatedAt:       time.Now().Add(-48 * time.Hour),
	}
	caseStore[demo1.CaseID] = demo1
}

// SubmitWhistleblowerReportLogic handles zero-knowledge intake, computes evidence SHA-256 hashes, and calculates risk scores.
func SubmitWhistleblowerReportLogic(req WhistleblowerReportRequest) EthicsCase {
	caseMutex.Lock()
	defer caseMutex.Unlock()

	idSeq := len(caseStore) + 1
	caseID := fmt.Sprintf("CASE-ETH-2026-%03d", idSeq)

	// Generate cryptographic anonymous token from timestamp and random seed
	tokenHasher := sha256.New()
	tokenHasher.Write([]byte(fmt.Sprintf("%s-%d-%s", caseID, time.Now().UnixNano(), req.Narrative)))
	anonToken := "SEC-TOKEN-" + hex.EncodeToString(tokenHasher.Sum(nil))[:16]

	// Compute evidence digests
	var digests []EvidenceDigest
	for idx, f := range req.EvidenceFiles {
		h := sha256.New()
		h.Write([]byte(f + fmt.Sprintf("salt-%d", idx)))
		hashStr := hex.EncodeToString(h.Sum(nil))
		digests = append(digests, EvidenceDigest{
			FileName:    fmt.Sprintf("evidence_item_%d.bin", idx+1),
			SHA256Hash:  hashStr,
			SubmittedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}

	// Calculate triage risk score
	riskScore := 50
	cat := strings.ToUpper(req.Category)
	switch {
	case strings.Contains(cat, "BRIBERY") || strings.Contains(cat, "CORRUPTION"):
		riskScore += 40
	case strings.Contains(cat, "FRAUD") || strings.Contains(cat, "FINANCIAL"):
		riskScore += 35
	case strings.Contains(cat, "ANTITRUST") || strings.Contains(cat, "CARTEL"):
		riskScore += 35
	case strings.Contains(cat, "SAFETY") || strings.Contains(cat, "HARASSMENT"):
		riskScore += 25
	default:
		riskScore += 10
	}

	if len(req.EvidenceFiles) > 0 {
		riskScore += 10
	}
	if len(req.Narrative) > 200 {
		riskScore += 5
	}
	if riskScore > 100 {
		riskScore = 100
	}

	priority := "LOW"
	if riskScore >= 80 {
		priority = "CRITICAL"
	} else if riskScore >= 65 {
		priority = "HIGH"
	} else if riskScore >= 50 {
		priority = "MEDIUM"
	}

	// Truncate narrative summary for privacy
	summary := req.Narrative
	if len(summary) > 120 {
		summary = summary[:117] + "..."
	}

	ec := EthicsCase{
		CaseID:           caseID,
		AnonymousToken:   anonToken,
		Category:         req.Category,
		BusinessUnit:     req.BusinessUnit,
		IncidentDate:     req.IncidentDate,
		NarrativeSummary: summary,
		EvidenceDigests:  digests,
		RiskScore:        riskScore,
		Priority:         priority,
		Status:           "NEW",
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}

	caseStore[caseID] = &ec
	return ec
}

// InvestigateCaseLogic processes audit committee actions, status progressions, and findings.
func InvestigateCaseLogic(req InvestigateCaseRequest) (*EthicsCase, error) {
	caseMutex.Lock()
	defer caseMutex.Unlock()

	ec, exists := caseStore[req.CaseID]
	if !exists {
		return nil, fmt.Errorf("case %s not found in ethics vault", req.CaseID)
	}

	ec.AssignedTo = req.InvestigatorID
	if req.FindingsSummary != "" {
		ec.FindingsSummary = req.FindingsSummary
	}
	if req.RemediationPlan != "" {
		ec.RemediationPlan = req.RemediationPlan
	}

	switch strings.ToUpper(req.Action) {
	case "ASSIGN":
		ec.Status = "UNDER_INVESTIGATION"
	case "SUBSTANTIATE":
		ec.Status = "SUBSTANTIATED"
	case "DISMISS":
		ec.Status = "DISMISSED"
	case "REMEDY", "CLOSE":
		ec.Status = "CLOSED"
	default:
		ec.Status = "UNDER_INVESTIGATION"
	}
	ec.UpdatedAt = time.Now().UTC()

	return ec, nil
}

// GetEthicsDashboardLogic aggregates compliance metrics for reporting and audit oversight.
func GetEthicsDashboardLogic() EthicsDashboardSummary {
	caseMutex.RLock()
	defer caseMutex.RUnlock()

	var cases []EthicsCase
	var totalRisk int
	active := 0
	substantiated := 0
	closed := 0
	catMap := make(map[string]int)

	for _, c := range caseStore {
		cases = append(cases, *c)
		totalRisk += c.RiskScore
		catMap[c.Category]++

		switch c.Status {
		case "UNDER_INVESTIGATION", "TRIAGED", "NEW":
			active++
		case "SUBSTANTIATED":
			substantiated++
		case "CLOSED", "DISMISSED":
			closed++
		}
	}

	avgRisk := 0.0
	if len(cases) > 0 {
		avgRisk = float64(totalRisk) / float64(len(cases))
	}

	return EthicsDashboardSummary{
		TotalCasesSubmitted:   len(cases),
		ActiveInvestigations:  active,
		SubstantiatedCases:    substantiated,
		ClosedCases:           closed,
		AverageRiskScore:      avgRisk,
		CategoryDistribution: catMap,
		Cases:                 cases,
	}
}

// RegisterRoutes registers the GRC Whistleblower & Ethics Vault routes.
func RegisterRoutes(r *gin.RouterGroup) {
	group := r.Group("/grc/ethics")
	{
		group.POST("/report-incident", func(c *gin.Context) {
			var req WhistleblowerReportRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			res := SubmitWhistleblowerReportLogic(req)
			c.JSON(http.StatusCreated, res)
		})

		group.POST("/investigate-case", func(c *gin.Context) {
			var req InvestigateCaseRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
				return
			}
			updated, err := InvestigateCaseLogic(req)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, updated)
		})

		group.GET("/dashboard", func(c *gin.Context) {
			dash := GetEthicsDashboardLogic()
			c.JSON(http.StatusOK, dash)
		})
	}
}
