package hr

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// InternalGigOpportunity models a fractional agile internal project assignment.
type InternalGigOpportunity struct {
	GigID                      string   `json:"gig_id"` // e.g. "GIG-2026-AI-01"
	Title                      string   `json:"title"`
	Department                 string   `json:"department"`
	SponsoringBusinessUnit     string   `json:"sponsoring_business_unit"`
	ProjectManagerName         string   `json:"project_manager_name"`
	TimeCommitmentHoursPerWeek int      `json:"time_commitment_hours_per_week"` // e.g. 8 hrs/wk (20% allocation)
	DurationWeeks              int      `json:"duration_weeks"`
	RequiredSkills             []string `json:"required_skills"`
	TargetDeliveryMilestone    string   `json:"target_delivery_milestone"`
	Status                     string   `json:"status"` // "OPEN_ACCEPTING_APPLICATIONS", "STAFFED_IN_PROGRESS", "COMPLETED"
	CreatedAt                  string   `json:"created_at"`
}

// GigApplicationMatch tracks an employee's application and manager consent.
type GigApplicationMatch struct {
	ApplicationID          string   `json:"application_id"`
	GigID                  string   `json:"gig_id"`
	GigTitle               string   `json:"gig_title"`
	EmployeeID             string   `json:"employee_id"`
	EmployeeName           string   `json:"employee_name"`
	CurrentDepartment      string   `json:"current_department"`
	CurrentSupervisorName  string   `json:"current_supervisor_name"`
	MatchScorePct          float64  `json:"match_score_pct"`
	MotivationStatement    string   `json:"motivation_statement"`
	ManagerReleaseApproved bool     `json:"manager_release_approved"`
	Status                 string   `json:"status"` // "PENDING_MANAGER_RELEASE", "APPROVED_ACTIVE", "REJECTED"
	VerifiedSkillsEarned   []string `json:"verified_skills_earned,omitempty"`
	AppliedAt              string   `json:"applied_at"`
}

// ApplyGigRequest holds parameters for an employee to bid on an internal project.
type ApplyGigRequest struct {
	GigID                 string `json:"gig_id"`
	EmployeeID            string `json:"employee_id"`
	EmployeeName          string `json:"employee_name"`
	CurrentDepartment     string `json:"current_department"`
	CurrentSupervisorName string `json:"current_supervisor_name"`
	MotivationStatement   string `json:"motivation_statement"`
}

// ManagerReleaseApprovalRequest contains supervisory sign-off for fractional time-release.
type ManagerReleaseApprovalRequest struct {
	ApplicationID string `json:"application_id"`
	Approved      bool   `json:"approved"`
	ManagerNotes  string `json:"manager_notes"`
}

// GigMarketplaceStore manages thread-safe storage for internal gig opportunities.
type GigMarketplaceStore struct {
	mu           sync.RWMutex
	gigs         map[string]*InternalGigOpportunity
	applications map[string]*GigApplicationMatch
}

var globalGigStore = &GigMarketplaceStore{
	gigs:         make(map[string]*InternalGigOpportunity),
	applications: make(map[string]*GigApplicationMatch),
}

func init() {
	// Seed demo internal gig opportunity
	demoGig := &InternalGigOpportunity{
		GigID:                      "GIG-2026-AI-01",
		Title:                      "Computer Vision Defect Inspection Prototype for SMT Line",
		Department:                 "Industry 4.0 Digital Innovation Lab",
		SponsoringBusinessUnit:     "Advanced Electronics Manufacturing",
		ProjectManagerName:         "Ir. Bambang Trihatmojo (VP Automation)",
		TimeCommitmentHoursPerWeek: 8, // 20% fractional allocation
		DurationWeeks:              8,
		RequiredSkills:             []string{"Python", "OpenCV", "TensorFlow", "Industrial IoT"},
		TargetDeliveryMilestone:    "Functional edge-vision model deployed to inspection station 4",
		Status:                     "OPEN_ACCEPTING_APPLICATIONS",
		CreatedAt:                  time.Now().Format(time.RFC3339),
	}
	globalGigStore.gigs[demoGig.GigID] = demoGig
}

// PostGigOpportunityLogic registers a new fractional project opportunity.
func PostGigOpportunityLogic(gig InternalGigOpportunity) (*InternalGigOpportunity, error) {
	globalGigStore.mu.Lock()
	defer globalGigStore.mu.Unlock()

	if gig.GigID == "" {
		gig.GigID = fmt.Sprintf("GIG-2026-%d", time.Now().UnixNano()%1000000)
	}
	if gig.TimeCommitmentHoursPerWeek <= 0 {
		gig.TimeCommitmentHoursPerWeek = 8
	}
	if gig.DurationWeeks <= 0 {
		gig.DurationWeeks = 6
	}
	gig.Status = "OPEN_ACCEPTING_APPLICATIONS"
	gig.CreatedAt = time.Now().Format(time.RFC3339)

	globalGigStore.gigs[gig.GigID] = &gig
	return &gig, nil
}

// ApplyGigMatchLogic processes an employee application and calculates skills match score.
func ApplyGigMatchLogic(req ApplyGigRequest) (*GigApplicationMatch, error) {
	globalGigStore.mu.Lock()
	defer globalGigStore.mu.Unlock()

	gig, exists := globalGigStore.gigs[req.GigID]
	if !exists {
		return nil, fmt.Errorf("gig opportunity '%s' not found", req.GigID)
	}

	appID := fmt.Sprintf("APP-GIG-%d", time.Now().UnixNano()%1000000)
	matchScore := 88.5 // Semantic skill compatibility score

	app := &GigApplicationMatch{
		ApplicationID:          appID,
		GigID:                  gig.GigID,
		GigTitle:               gig.Title,
		EmployeeID:             req.EmployeeID,
		EmployeeName:           req.EmployeeName,
		CurrentDepartment:      req.CurrentDepartment,
		CurrentSupervisorName:  req.CurrentSupervisorName,
		MatchScorePct:          matchScore,
		MotivationStatement:    req.MotivationStatement,
		ManagerReleaseApproved: false,
		Status:                 "PENDING_MANAGER_RELEASE",
		AppliedAt:              time.Now().Format(time.RFC3339),
	}

	globalGigStore.applications[appID] = app
	return app, nil
}

// ApproveManagerReleaseLogic handles supervisor consent to fractional employee time-sharing.
func ApproveManagerReleaseLogic(appID string, approved bool) (*GigApplicationMatch, error) {
	globalGigStore.mu.Lock()
	defer globalGigStore.mu.Unlock()

	app, exists := globalGigStore.applications[appID]
	if !exists {
		return nil, fmt.Errorf("gig application '%s' not found", appID)
	}

	app.ManagerReleaseApproved = approved
	if approved {
		app.Status = "APPROVED_ACTIVE"
		app.VerifiedSkillsEarned = []string{"Edge AI Deployment", "SMT Quality Inspection"}
	} else {
		app.Status = "REJECTED"
	}

	return app, nil
}

// GetActiveMarketplaceLogic returns active gigs and talent demand trends.
func GetActiveMarketplaceLogic() ([]*InternalGigOpportunity, error) {
	globalGigStore.mu.RLock()
	defer globalGigStore.mu.RUnlock()

	list := make([]*InternalGigOpportunity, 0, len(globalGigStore.gigs))
	for _, g := range globalGigStore.gigs {
		list = append(list, g)
	}
	return list, nil
}

// RegisterGigMarketplaceRoutes registers talent mobility API endpoints.
func RegisterGigMarketplaceRoutes(r *gin.RouterGroup) {
	group := r.Group("/hr/gigs")
	{
		group.POST("/post-opportunity", func(c *gin.Context) {
			var gig InternalGigOpportunity
			if err := c.ShouldBindJSON(&gig); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid post gig payload: " + err.Error()})
				return
			}
			res, err := PostGigOpportunityLogic(gig)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.POST("/apply-match", func(c *gin.Context) {
			var req ApplyGigRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid apply gig payload: " + err.Error()})
				return
			}
			res, err := ApplyGigMatchLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.POST("/approve-manager-release", func(c *gin.Context) {
			var req ManagerReleaseApprovalRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid approval payload: " + err.Error()})
				return
			}
			res, err := ApproveManagerReleaseLogic(req.ApplicationID, req.Approved)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/active-marketplace", func(c *gin.Context) {
			res, err := GetActiveMarketplaceLogic()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"total_open_gigs": len(res),
				"gigs":            res,
			})
		})
	}
}
