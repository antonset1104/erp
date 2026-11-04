package ehs

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// SensorZoneAlert represents an automated IoT or Computer Vision workplace hazard detection.
type SensorZoneAlert struct {
	AlertID                   string  `json:"alert_id"`
	WorkCenter                string  `json:"work_center"` // e.g. "Welding Bay 4", "High-Rack Aisle 12"
	AlertType                 string  `json:"alert_type"`  // "PPE_VIOLATION_NO_HELMET", "FORKLIFT_PROXIMITY_BREACH", "CONFINED_SPACE_GAS_LEAK", "HEAT_STRESS_EXCEEDED"
	ConfidenceScore           float64 `json:"confidence_score"`
	DetectedTimestamp         string  `json:"detected_timestamp"`
	ImmediateMitigationAction string  `json:"immediate_mitigation_action"`
	Status                    string  `json:"status"` // "TRIGGERED", "DISPATCHED_SUPERVISOR", "RESOLVED"
}

// OSHA300LogEntry encapsulates an official workplace injury or illness log item.
type OSHA300LogEntry struct {
	CaseNo                   string `json:"case_no"`
	EmployeeClassification   string `json:"employee_classification"`
	DateOfIncident           string `json:"date_of_incident"`
	WhereEventOccurred       string `json:"where_event_occurred"`
	DescribeInjuryOrIllness  string `json:"describe_injury_or_illness"`
	Classification           string `json:"classification"` // "DEATH", "DAYS_AWAY_FROM_WORK", "JOB_TRANSFER_RESTRICTION", "OTHER_RECORDABLE"
	DaysAwayCount            int    `json:"days_away_count"`
	DaysRestrictedCount      int    `json:"days_restricted_count"`
}

// OSHA300ASummary contains calculated occupational health and safety metrics for statutory reporting.
type OSHA300ASummary struct {
	Year                        int               `json:"year"`
	EstablishmentName           string            `json:"establishment_name"`
	TotalHoursWorked            float64           `json:"total_hours_worked"`
	AverageAnnualEmployees      int               `json:"average_annual_employees"`
	TotalDeaths                 int               `json:"total_deaths"`
	TotalCasesWithDaysAway      int               `json:"total_cases_with_days_away"`
	TotalCasesWithRestriction   int               `json:"total_cases_with_restriction"`
	TotalOtherRecordableCases   int               `json:"total_other_recordable_cases"`
	TotalRecordableCases        int               `json:"total_recordable_cases"`
	TotalDaysAway               int               `json:"total_days_away"`
	TotalDaysRestricted         int               `json:"total_days_restricted"`
	TRIRScore                   float64           `json:"trir_score"` // Total Recordable Incident Rate
	DARTScore                   float64           `json:"dart_score"` // Days Away, Restricted or Transferred Rate
	SeverityIndex               float64           `json:"severity_index"`
	ISO45001AuditStatus         string            `json:"iso45001_audit_status"` // "CERTIFIED_COMPLIANT", "OBSERVATION_WARNING", "NON_CONFORMING"
	RecentEntries               []OSHA300LogEntry `json:"recent_entries"`
	GeneratedAt                 string            `json:"generated_at"`
}

// CAPAEscalation represents a formal Corrective and Preventive Action workflow.
type CAPAEscalation struct {
	CAPAID            string `json:"capa_id"`
	LinkedCaseNo      string `json:"linked_case_no"`
	RootCauseCategory string `json:"root_cause_category"` // "MACHINE_GUARDING", "PROCEDURAL_FAILURE", "PPE_SHORTAGE", "ERGONOMIC_HAZARD"
	ActionPlan        string `json:"action_plan"`
	AssignedTo        string `json:"assigned_to"`
	Deadline          string `json:"deadline"`
	Status            string `json:"status"` // "ASSIGNED", "VERIFIED_EFFECTIVE", "CLOSED"
	CreatedAt         string `json:"created_at"`
}

// EscalateCAPARequest holds payload to trigger a formal K3/EHS CAPA.
type EscalateCAPARequest struct {
	LinkedCaseNo      string `json:"linked_case_no"`
	RootCauseCategory string `json:"root_cause_category"`
	ActionPlan        string `json:"action_plan"`
	AssignedTo        string `json:"assigned_to"`
	DeadlineDays      int    `json:"deadline_days"`
}

// EHSSentinelStore provides thread-safe memory storage for EHS sentinel events.
type EHSSentinelStore struct {
	mu           sync.RWMutex
	alerts       []SensorZoneAlert
	oshaEntries  []OSHA300LogEntry
	capaList     []CAPAEscalation
	hoursWorked  float64
	numEmployees int
}

var globalSentinelStore = &EHSSentinelStore{
	alerts:       make([]SensorZoneAlert, 0),
	oshaEntries:  make([]OSHA300LogEntry, 0),
	capaList:     make([]CAPAEscalation, 0),
	hoursWorked:  1250000.0, // 1.25 million man-hours
	numEmployees: 620,
}

func init() {
	// Seed demo OSHA 300 cases
	globalSentinelStore.oshaEntries = append(globalSentinelStore.oshaEntries,
		OSHA300LogEntry{
			CaseNo:                  "OSHA-2026-001",
			EmployeeClassification:  "Machine Operator",
			DateOfIncident:          "2026-01-14",
			WhereEventOccurred:      "Press Line B",
			DescribeInjuryOrIllness: "Minor finger laceration requiring first aid and 1 day restriction",
			Classification:          "JOB_TRANSFER_RESTRICTION",
			DaysAwayCount:           0,
			DaysRestrictedCount:     3,
		},
		OSHA300LogEntry{
			CaseNo:                  "OSHA-2026-002",
			EmployeeClassification:  "Warehouse Forklift Driver",
			DateOfIncident:          "2026-02-08",
			WhereEventOccurred:      "Dock 7 Staging Area",
			DescribeInjuryOrIllness: "Lumbar back strain from manual pallet shifting",
			Classification:          "DAYS_AWAY_FROM_WORK",
			DaysAwayCount:           4,
			DaysRestrictedCount:     2,
		},
		OSHA300LogEntry{
			CaseNo:                  "OSHA-2026-003",
			EmployeeClassification:  "Assembly Technician",
			DateOfIncident:          "2026-03-02",
			WhereEventOccurred:      "Sub-Assembly Station 3",
			DescribeInjuryOrIllness: "Superficial solder burn treated on-site",
			Classification:          "OTHER_RECORDABLE",
			DaysAwayCount:           0,
			DaysRestrictedCount:     0,
		},
	)
}

// IngestZoneAlertLogic processes a live IoT camera/sensor telemetry trigger.
func IngestZoneAlertLogic(alert SensorZoneAlert) (*SensorZoneAlert, error) {
	globalSentinelStore.mu.Lock()
	defer globalSentinelStore.mu.Unlock()

	if alert.WorkCenter == "" {
		alert.WorkCenter = "General Facility Floor"
	}
	if alert.AlertID == "" {
		alert.AlertID = fmt.Sprintf("ALT-EHS-%d", time.Now().UnixNano()%1000000)
	}
	if alert.DetectedTimestamp == "" {
		alert.DetectedTimestamp = time.Now().Format(time.RFC3339)
	}

	mitigation := "Automated audio siren broadcast in zone; floor supervisor SMS dispatched."
	if alert.AlertType == "CONFINED_SPACE_GAS_LEAK" {
		mitigation = "Emergency ventilation dampers opened automatically; air-horn evacuation signaled."
	}
	alert.ImmediateMitigationAction = mitigation
	alert.Status = "DISPATCHED_SUPERVISOR"

	globalSentinelStore.alerts = append(globalSentinelStore.alerts, alert)
	return &alert, nil
}

// GenerateOSHA300SummaryLogic computes official OSHA 300A and ISO 45001 indicators.
func GenerateOSHA300SummaryLogic(year int) (*OSHA300ASummary, error) {
	globalSentinelStore.mu.RLock()
	defer globalSentinelStore.mu.RUnlock()

	if year <= 0 {
		year = 2026
	}

	var deaths, daysAwayCases, restrictionCases, otherCases int
	var totalDaysAway, totalDaysRestricted int

	for _, entry := range globalSentinelStore.oshaEntries {
		switch entry.Classification {
		case "DEATH":
			deaths++
		case "DAYS_AWAY_FROM_WORK":
			daysAwayCases++
		case "JOB_TRANSFER_RESTRICTION":
			restrictionCases++
		case "OTHER_RECORDABLE":
			otherCases++
		}
		totalDaysAway += entry.DaysAwayCount
		totalDaysRestricted += entry.DaysRestrictedCount
	}

	totalRecordable := deaths + daysAwayCases + restrictionCases + otherCases

	// Statutory OSHA Formulas based on 200,000 baseline hours (100 full-time workers @ 40 hrs/wk * 50 wks)
	trir := 0.0
	dart := 0.0
	if globalSentinelStore.hoursWorked > 0 {
		trir = (float64(totalRecordable) * 200000.0) / globalSentinelStore.hoursWorked
		dart = (float64(daysAwayCases+restrictionCases) * 200000.0) / globalSentinelStore.hoursWorked
	}

	severity := 0.0
	if daysAwayCases > 0 {
		severity = float64(totalDaysAway) / float64(daysAwayCases)
	}

	auditStatus := "CERTIFIED_COMPLIANT"
	if trir > 2.5 || dart > 1.5 {
		auditStatus = "NON_CONFORMING"
	} else if trir > 1.0 {
		auditStatus = "OBSERVATION_WARNING"
	}

	return &OSHA300ASummary{
		Year:                      year,
		EstablishmentName:         "GoERP Global Manufacturing Complex - Cikarang Plant",
		TotalHoursWorked:          globalSentinelStore.hoursWorked,
		AverageAnnualEmployees:    globalSentinelStore.numEmployees,
		TotalDeaths:               deaths,
		TotalCasesWithDaysAway:    daysAwayCases,
		TotalCasesWithRestriction: restrictionCases,
		TotalOtherRecordableCases: otherCases,
		TotalRecordableCases:      totalRecordable,
		TotalDaysAway:             totalDaysAway,
		TotalDaysRestricted:       totalDaysRestricted,
		TRIRScore:                 math.Round(trir*100) / 100,
		DARTScore:                 math.Round(dart*100) / 100,
		SeverityIndex:             math.Round(severity*100) / 100,
		ISO45001AuditStatus:       auditStatus,
		RecentEntries:             globalSentinelStore.oshaEntries,
		GeneratedAt:               time.Now().Format(time.RFC3339),
	}, nil
}

// EscalateCAPALogic creates an enforceable Corrective and Preventive Action workflow.
func EscalateCAPALogic(req EscalateCAPARequest) (*CAPAEscalation, error) {
	globalSentinelStore.mu.Lock()
	defer globalSentinelStore.mu.Unlock()

	if req.LinkedCaseNo == "" {
		return nil, fmt.Errorf("linked OSHA case number is mandatory")
	}
	if req.ActionPlan == "" {
		return nil, fmt.Errorf("CAPA action plan description cannot be empty")
	}
	if req.DeadlineDays <= 0 {
		req.DeadlineDays = 14
	}

	deadline := time.Now().AddDate(0, 0, req.DeadlineDays).Format("2006-01-02")
	capaID := fmt.Sprintf("CAPA-2026-%04d", len(globalSentinelStore.capaList)+1)

	capa := CAPAEscalation{
		CAPAID:            capaID,
		LinkedCaseNo:      req.LinkedCaseNo,
		RootCauseCategory: req.RootCauseCategory,
		ActionPlan:        req.ActionPlan,
		AssignedTo:        req.AssignedTo,
		Deadline:          deadline,
		Status:            "ASSIGNED",
		CreatedAt:         time.Now().Format(time.RFC3339),
	}

	globalSentinelStore.capaList = append(globalSentinelStore.capaList, capa)
	return &capa, nil
}

// RegisterEHSSentinelRoutes registers the EHS Sentinel and OSHA compliance routes.
func RegisterEHSSentinelRoutes(r *gin.RouterGroup) {
	group := r.Group("/ehs/sentinel")
	{
		group.POST("/telemetry-alert", func(c *gin.Context) {
			var alert SensorZoneAlert
			if err := c.ShouldBindJSON(&alert); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid zone alert payload: " + err.Error()})
				return
			}
			res, err := IngestZoneAlertLogic(alert)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/osha-300-summary", func(c *gin.Context) {
			res, err := GenerateOSHA300SummaryLogic(2026)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/capa-escalate", func(c *gin.Context) {
			var req EscalateCAPARequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid CAPA escalation payload: " + err.Error()})
				return
			}
			res, err := EscalateCAPALogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})
	}
}
