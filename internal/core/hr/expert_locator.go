package hr

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SubjectMatterExpert captures deep professional credentials, patents, certifications, and availability.
type SubjectMatterExpert struct {
	EmployeeID              string   `json:"employee_id"` // e.g. "EMP-SME-2026-081"
	FullName                string   `json:"full_name"`
	CurrentJobTitle         string   `json:"current_job_title"`
	Department              string   `json:"department"`
	SubsidiaryEntity        string   `json:"subsidiary_entity"`
	LocationCity            string   `json:"location_city"`
	PrimaryDomain           string   `json:"primary_domain"` // "HYDROGEN_ENERGY", "ADVANCED_ROBOTICS", "ASME_CRYOGENICS"
	KeySkills               []string `json:"key_skills"`
	Certifications          []string `json:"certifications"`
	PatentsAuthored         int      `json:"patents_authored"`
	LanguagesSpoken         []string `json:"languages_spoken"`
	CurrentBandwidthPct     float64  `json:"current_bandwidth_pct"` // Free capacity (0 - 100%)
	InternalHourlyChargeUSD float64  `json:"internal_hourly_charge_usd"`
}

// ExpertSearchQuery holds project staffing criteria for semantic discovery.
type ExpertSearchQuery struct {
	RequiredDomain      string   `json:"required_domain"`
	MandatorySkills     []string `json:"mandatory_skills"`
	PreferredLanguages  []string `json:"preferred_languages"`
	MinimumBandwidthPct float64  `json:"minimum_bandwidth_pct"`
	TargetProjectCode   string   `json:"target_project_code"`
}

// ExpertMatchCandidate evaluates an individual expert against the inquiry criteria.
type ExpertMatchCandidate struct {
	Expert                           SubjectMatterExpert `json:"expert"`
	MatchScorePct                    float64             `json:"match_score_pct"` // 0.0 - 100.0%
	SkillOverlapList                 []string            `json:"skill_overlap_list"`
	AvailabilityStatus               string              `json:"availability_status"` // "AVAILABLE_IMMEDIATE", "PARTIAL_ALLOCATION", "COMMITTED"
	EstimatedCostSavingsVsExternalUSD float64            `json:"estimated_cost_savings_vs_external_usd"`
}

// ExpertSearchResult returns the ranked internal talent list.
type ExpertSearchResult struct {
	QueryDomain               string                 `json:"query_domain"`
	TargetProjectCode         string                 `json:"target_project_code"`
	TotalInternalExpertsFound int                    `json:"total_internal_experts_found"`
	MatchedCandidates         []ExpertMatchCandidate `json:"matched_candidates"`
	TotalPotentialSavingsUSD  float64                `json:"total_potential_savings_usd"`
	SearchDurationMs          float64                `json:"search_duration_ms"`
	RecommendedNextStep       string                 `json:"recommended_next_step"`
}

// ExpertLocatorStore manages thread-safe storage for corporate SME profiles.
type ExpertLocatorStore struct {
	mu      sync.RWMutex
	experts map[string]*SubjectMatterExpert
}

var globalExpertStore = &ExpertLocatorStore{
	experts: make(map[string]*SubjectMatterExpert),
}

func init() {
	// Pre-seed demo global SMEs
	demoExperts := []SubjectMatterExpert{
		{
			EmployeeID:              "EMP-SME-2026-081",
			FullName:                "Dr. Aris Wibowo, Ph.D.",
			CurrentJobTitle:         "Principal Cryogenic Fuel Systems Architect",
			Department:              "Clean Mobility & Propulsion R&D",
			SubsidiaryEntity:        "GoERP Hydrogen Powertrain GmbH (Munich)",
			LocationCity:            "Munich, Germany",
			PrimaryDomain:           "HYDROGEN_ENERGY",
			KeySkills:               []string{"Liquid Hydrogen LH2", "Cryogenic Pressure Vessels", "CFD Thermal Modeling", "ASME Section VIII"},
			Certifications:          []string{"ASME Section VIII Div 2", "Six Sigma Black Belt", "TUV Rheinland Functional Safety"},
			PatentsAuthored:         6,
			LanguagesSpoken:         []string{"Indonesian", "English", "German"},
			CurrentBandwidthPct:     40.0,
			InternalHourlyChargeUSD: 145.0,
		},
		{
			EmployeeID:              "EMP-SME-2026-094",
			FullName:                "Eng. Siti Rahmawati, M.Eng.",
			CurrentJobTitle:         "Senior Industrial Robotics & Vision Specialist",
			Department:              "Automation & Smart Factory Center",
			SubsidiaryEntity:        "PT GoERP Solusi Manufaktur (Jakarta)",
			LocationCity:            "Cikarang, Indonesia",
			PrimaryDomain:           "ADVANCED_ROBOTICS",
			KeySkills:               []string{"ROS2 Navigation", "Stereo Depth Camera AI", "PLC Automation", "FANUC/KUKA Scripting"},
			Certifications:          []string{"Certified Automation Professional (ISA)", "NVIDIA Deep Learning Institute"},
			PatentsAuthored:         2,
			LanguagesSpoken:         []string{"Indonesian", "English", "Japanese"},
			CurrentBandwidthPct:     30.0,
			InternalHourlyChargeUSD: 95.0,
		},
		{
			EmployeeID:              "EMP-SME-2026-112",
			FullName:                "Marcus Vance, PMP",
			CurrentJobTitle:         "Lead Electrochemical Materials Scientist",
			Department:              "Battery Chemistry & Solid-State R&D",
			SubsidiaryEntity:        "GoERP Technologies Inc (Silicon Valley)",
			LocationCity:            "Fremont, USA",
			PrimaryDomain:           "BATTERY_CHEMISTRY",
			KeySkills:               []string{"Solid-State Electrolytes", "NMC Cathode Sintering", "Cell Degradation EIS", "Dry Electrode Coating"},
			Certifications:          []string{"PMP", "Electrochemical Society Senior Member"},
			PatentsAuthored:         8,
			LanguagesSpoken:         []string{"English", "French"},
			CurrentBandwidthPct:     15.0,
			InternalHourlyChargeUSD: 180.0,
		},
	}

	for _, e := range demoExperts {
		cp := e
		globalExpertStore.experts[e.EmployeeID] = &cp
	}
}

// IndexExpertProfileLogic registers or updates an employee's professional profile.
func IndexExpertProfileLogic(expert SubjectMatterExpert) (*SubjectMatterExpert, error) {
	globalExpertStore.mu.Lock()
	defer globalExpertStore.mu.Unlock()

	if expert.EmployeeID == "" {
		expert.EmployeeID = fmt.Sprintf("EMP-SME-2026-%03d", len(globalExpertStore.experts)+120)
	}
	if expert.InternalHourlyChargeUSD <= 0 {
		expert.InternalHourlyChargeUSD = 120.0
	}

	globalExpertStore.experts[expert.EmployeeID] = &expert
	return &expert, nil
}

// SearchExpertsLogic performs weighted semantic attribute matching against the SME graph.
func SearchExpertsLogic(query ExpertSearchQuery) (*ExpertSearchResult, error) {
	start := time.Now()
	globalExpertStore.mu.RLock()
	defer globalExpertStore.mu.RUnlock()

	if query.RequiredDomain == "" {
		query.RequiredDomain = "HYDROGEN_ENERGY"
	}
	if query.TargetProjectCode == "" {
		query.TargetProjectCode = "PRJ-CLEAN-PROPULSION-01"
	}

	const externalConsultantRateUSD = 350.0 // Benchmark top-tier engineering firm hourly rate

	var candidates []ExpertMatchCandidate
	totalSavings := 0.0

	for _, expert := range globalExpertStore.experts {
		score := 0.0

		// Domain match (35 pts)
		if strings.EqualFold(expert.PrimaryDomain, query.RequiredDomain) {
			score += 35.0
		} else if strings.Contains(strings.ToLower(expert.PrimaryDomain), strings.ToLower(query.RequiredDomain)) {
			score += 20.0
		}

		// Skill overlap (up to 35 pts)
		var matchedSkills []string
		for _, reqSkill := range query.MandatorySkills {
			for _, expSkill := range expert.KeySkills {
				if strings.EqualFold(reqSkill, expSkill) || strings.Contains(strings.ToLower(expSkill), strings.ToLower(reqSkill)) {
					matchedSkills = append(matchedSkills, expSkill)
					score += 10.0
					break
				}
			}
		}
		if score > 70.0 {
			score = 70.0 // cap skill + domain at 70
		}

		// Language match (15 pts)
		for _, prefLang := range query.PreferredLanguages {
			for _, expLang := range expert.LanguagesSpoken {
				if strings.EqualFold(prefLang, expLang) {
					score += 7.5
					break
				}
			}
		}

		// Bandwidth capacity (15 pts)
		if expert.CurrentBandwidthPct >= query.MinimumBandwidthPct {
			score += 15.0
		} else {
			score += (expert.CurrentBandwidthPct / 100.0) * 10.0
		}

		// Add bonus for patents
		score += math.Min(float64(expert.PatentsAuthored)*1.5, 5.0)

		if score > 100.0 {
			score = 100.0
		}

		if score >= 35.0 {
			availStatus := "AVAILABLE_IMMEDIATE"
			if expert.CurrentBandwidthPct < 20.0 {
				availStatus = "COMMITTED"
			} else if expert.CurrentBandwidthPct < 40.0 {
				availStatus = "PARTIAL_ALLOCATION"
			}

			// Assume 80 project hours
			costSavings := (externalConsultantRateUSD - expert.InternalHourlyChargeUSD) * 80.0
			totalSavings += costSavings

			candidates = append(candidates, ExpertMatchCandidate{
				Expert:                           *expert,
				MatchScorePct:                    math.Round(score*10) / 10,
				SkillOverlapList:                 matchedSkills,
				AvailabilityStatus:               availStatus,
				EstimatedCostSavingsVsExternalUSD: math.Round(costSavings),
			})
		}
	}

	// Sort candidates descending by match score
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].MatchScorePct > candidates[j].MatchScorePct
	})

	elapsed := float64(time.Since(start).Microseconds()) / 1000.0

	return &ExpertSearchResult{
		QueryDomain:               query.RequiredDomain,
		TargetProjectCode:         query.TargetProjectCode,
		TotalInternalExpertsFound: len(candidates),
		MatchedCandidates:         candidates,
		TotalPotentialSavingsUSD:  math.Round(totalSavings),
		SearchDurationMs:          elapsed,
		RecommendedNextStep:       "Issue internal matrix allocation request to subsidiary business unit head.",
	}, nil
}

// RegisterExpertLocatorRoutes registers HR SME discovery API endpoints.
func RegisterExpertLocatorRoutes(r *gin.RouterGroup) {
	group := r.Group("/hr/experts")
	{
		group.POST("/search", func(c *gin.Context) {
			var query ExpertSearchQuery
			if err := c.ShouldBindJSON(&query); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid expert search payload: " + err.Error()})
				return
			}
			res, err := SearchExpertsLogic(query)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/index-profile", func(c *gin.Context) {
			var expert SubjectMatterExpert
			if err := c.ShouldBindJSON(&expert); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid index profile payload: " + err.Error()})
				return
			}
			res, err := IndexExpertProfileLogic(expert)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.GET("/competency-radar/:skill_domain", func(c *gin.Context) {
			domain := c.Param("skill_domain")
			res, err := SearchExpertsLogic(ExpertSearchQuery{
				RequiredDomain: domain,
			})
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
