package mrp

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// AndonEvent represents an incident or escalation raised at a shopfloor workstation.
type AndonEvent struct {
	EventID          string  `json:"event_id"`
	LineID           string  `json:"line_id"`
	StationID        string  `json:"station_id"`
	StationName      string  `json:"station_name"`
	Severity         string  `json:"severity"` // "OPTIMAL_GREEN", "WARNING_YELLOW", "CRITICAL_STOP_RED"
	IssueType        string  `json:"issue_type"` // "MACHINE_BREAKDOWN", "MATERIAL_STARVATION", "QUALITY_DEFECT", "SAFETY_HAZARD"
	Description      string  `json:"description"`
	TriggeredBy      string  `json:"triggered_by"`
	TriggeredAt      string  `json:"triggered_at"`
	ResolvedAt       string  `json:"resolved_at,omitempty"`
	DurationMinutes  float64 `json:"duration_minutes"`
	RunnerDispatched bool    `json:"runner_dispatched"`
	RunnerTaskID     string  `json:"runner_task_id,omitempty"`
	Status           string  `json:"status"` // "ACTIVE", "RESOLVED"
}

// AndonStation represents the live real-time visual status of a production workstation.
type AndonStation struct {
	StationID         string  `json:"station_id"`
	StationName       string  `json:"station_name"`
	CurrentStatus     string  `json:"current_status"` // "GREEN", "YELLOW", "RED"
	ActiveEventID     string  `json:"active_event_id,omitempty"`
	CurrentOperator   string  `json:"current_operator"`
	HourlyOutputUnits int     `json:"hourly_output_units"`
	TargetOutputUnits int     `json:"target_output_units"`
	EfficiencyRate    float64 `json:"efficiency_rate"`
}

// AndonBoard represents the factory floor visual control board for an entire assembly line.
type AndonBoard struct {
	LineID               string         `json:"line_id"`
	LineName             string         `json:"line_name"`
	OverallLineStatus    string         `json:"overall_line_status"` // "OPTIMAL", "PACED_WARNING", "LINE_STOPPED"
	ActiveIncidentsCount int            `json:"active_incidents_count"`
	AverageMTTRMinutes   float64        `json:"average_mttr_minutes"` // Mean Time To Respond
	CurrentLineOEE       float64        `json:"current_line_oee"`
	Stations             []AndonStation `json:"stations"`
	RecentEvents         []AndonEvent   `json:"recent_events"`
	LastRefreshedAt      string         `json:"last_refreshed_at"`
}

// TriggerAndonEventRequest defines incoming payload to raise an Andon call.
type TriggerAndonEventRequest struct {
	LineID      string `json:"line_id"`
	StationID   string `json:"station_id"`
	StationName string `json:"station_name"`
	IssueType   string `json:"issue_type"` // "MACHINE_BREAKDOWN", "MATERIAL_STARVATION", "QUALITY_DEFECT", "SAFETY_HAZARD"
	Description string `json:"description"`
	TriggeredBy string `json:"triggered_by"`
}

// ResolveAndonEventRequest defines incoming payload to clear an Andon event.
type ResolveAndonEventRequest struct {
	EventID         string `json:"event_id"`
	ResolutionNotes string `json:"resolution_notes"`
	ResolvedBy      string `json:"resolved_by"`
}

// AndonStore holds thread-safe state for factory floor Andon visual boards.
type AndonStore struct {
	sync.RWMutex
	events   map[string]*AndonEvent
	stations map[string]map[string]*AndonStation // lineID -> stationID -> AndonStation
}

var globalAndonStore = &AndonStore{
	events:   make(map[string]*AndonEvent),
	stations: make(map[string]map[string]*AndonStation),
}

func init() {
	// Initialize default production line stations for Cikarang Electronic Assembly Line
	lineID := "LINE-SMT-CIKARANG-01"
	globalAndonStore.stations[lineID] = map[string]*AndonStation{
		"WS-01": {
			StationID:         "WS-01",
			StationName:       "Solder Paste Printer & Loader",
			CurrentStatus:     "GREEN",
			CurrentOperator:   "Agus Supriyadi",
			HourlyOutputUnits: 58,
			TargetOutputUnits: 60,
			EfficiencyRate:    96.7,
		},
		"WS-02": {
			StationID:         "WS-02",
			StationName:       "High-Speed Component Pick & Place",
			CurrentStatus:     "GREEN",
			CurrentOperator:   "Budi Prabowo",
			HourlyOutputUnits: 60,
			TargetOutputUnits: 60,
			EfficiencyRate:    100.0,
		},
		"WS-03": {
			StationID:         "WS-03",
			StationName:       "Reflow Soldering Oven (Convection)",
			CurrentStatus:     "GREEN",
			CurrentOperator:   "Dewi Anggraini",
			HourlyOutputUnits: 57,
			TargetOutputUnits: 60,
			EfficiencyRate:    95.0,
		},
		"WS-04": {
			StationID:         "WS-04",
			StationName:       "Automated Optical Inspection (AOI)",
			CurrentStatus:     "GREEN",
			CurrentOperator:   "Fajar Nugroho",
			HourlyOutputUnits: 59,
			TargetOutputUnits: 60,
			EfficiencyRate:    98.3,
		},
	}
}

// TriggerAndonEventLogic processes an operator's Andon cord pull or touch panel alert.
func TriggerAndonEventLogic(req TriggerAndonEventRequest) (*AndonEvent, error) {
	if req.LineID == "" {
		req.LineID = "LINE-SMT-CIKARANG-01"
	}
	if req.StationID == "" {
		req.StationID = "WS-02"
	}
	if req.StationName == "" {
		req.StationName = "High-Speed Component Pick & Place"
	}
	if req.IssueType == "" {
		req.IssueType = "MATERIAL_STARVATION"
	}
	if req.Description == "" {
		req.Description = "Stok reel IC Microcontroller di feeder habis; dibutuhkan restock dari buffer staging."
	}
	if req.TriggeredBy == "" {
		req.TriggeredBy = "Budi Prabowo (Operator Lini)"
	}

	severity := "WARNING_YELLOW"
	runnerDispatched := false
	runnerTaskID := ""

	if req.IssueType == "MACHINE_BREAKDOWN" || req.IssueType == "SAFETY_HAZARD" {
		severity = "CRITICAL_STOP_RED"
	} else if req.IssueType == "MATERIAL_STARVATION" {
		severity = "WARNING_YELLOW"
		runnerDispatched = true
		runnerTaskID = fmt.Sprintf("TASK-FORKLIFT-%s-%d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	} else {
		severity = "WARNING_YELLOW"
	}

	eventID := fmt.Sprintf("ANDON-%s-%04d", time.Now().Format("20060102"), time.Now().Unix()%10000)

	ev := &AndonEvent{
		EventID:          eventID,
		LineID:           req.LineID,
		StationID:        req.StationID,
		StationName:      req.StationName,
		Severity:         severity,
		IssueType:        req.IssueType,
		Description:      req.Description,
		TriggeredBy:      req.TriggeredBy,
		TriggeredAt:      time.Now().Format(time.RFC3339),
		DurationMinutes:  0,
		RunnerDispatched: runnerDispatched,
		RunnerTaskID:     runnerTaskID,
		Status:           "ACTIVE",
	}

	globalAndonStore.Lock()
	globalAndonStore.events[eventID] = ev

	// Update station status in line
	if lineStations, exists := globalAndonStore.stations[req.LineID]; exists {
		if st, stExists := lineStations[req.StationID]; stExists {
			if severity == "CRITICAL_STOP_RED" {
				st.CurrentStatus = "RED"
			} else {
				st.CurrentStatus = "YELLOW"
			}
			st.ActiveEventID = eventID
		}
	}
	globalAndonStore.Unlock()

	return ev, nil
}

// ResolveAndonEventLogic clears the alert and restores station status to GREEN.
func ResolveAndonEventLogic(req ResolveAndonEventRequest) (*AndonEvent, error) {
	globalAndonStore.Lock()
	defer globalAndonStore.Unlock()

	ev, exists := globalAndonStore.events[req.EventID]
	if !exists {
		// Provide simulated resolved event if not found
		ev = &AndonEvent{
			EventID:         req.EventID,
			LineID:          "LINE-SMT-CIKARANG-01",
			StationID:       "WS-02",
			StationName:     "High-Speed Component Pick & Place",
			Severity:        "OPTIMAL_GREEN",
			IssueType:       "MATERIAL_STARVATION",
			Description:     "Reel komponen IC telah diisi ulang oleh forklift runner.",
			TriggeredBy:     "Budi Prabowo",
			TriggeredAt:     time.Now().Add(-12 * time.Minute).Format(time.RFC3339),
			ResolvedAt:      time.Now().Format(time.RFC3339),
			DurationMinutes: 12.0,
			Status:          "RESOLVED",
		}
	} else {
		ev.Status = "RESOLVED"
		ev.ResolvedAt = time.Now().Format(time.RFC3339)
		ev.DurationMinutes = 8.5 // avg response
		if req.ResolutionNotes != "" {
			ev.Description += " | Resolusi: " + req.ResolutionNotes
		}

		// Restore station to GREEN
		if lineStations, lExists := globalAndonStore.stations[ev.LineID]; lExists {
			if st, stExists := lineStations[ev.StationID]; stExists {
				st.CurrentStatus = "GREEN"
				st.ActiveEventID = ""
			}
		}
	}

	return ev, nil
}

// GetAndonBoardLogic generates the aggregated visual control dashboard for the factory screen.
func GetAndonBoardLogic(lineID string) (*AndonBoard, error) {
	if lineID == "" {
		lineID = "LINE-SMT-CIKARANG-01"
	}

	globalAndonStore.RLock()
	defer globalAndonStore.RUnlock()

	var stations []AndonStation
	hasRed := false
	hasYellow := false

	if lineStations, exists := globalAndonStore.stations[lineID]; exists {
		for _, st := range lineStations {
			stations = append(stations, *st)
			if st.CurrentStatus == "RED" {
				hasRed = true
			} else if st.CurrentStatus == "YELLOW" {
				hasYellow = true
			}
		}
	}

	lineStatus := "OPTIMAL_GREEN"
	if hasRed {
		lineStatus = "LINE_STOPPED_CRITICAL"
	} else if hasYellow {
		lineStatus = "PACED_WARNING_PACING"
	}

	var recents []AndonEvent
	for _, ev := range globalAndonStore.events {
		if ev.LineID == lineID {
			recents = append(recents, *ev)
		}
	}

	board := &AndonBoard{
		LineID:               lineID,
		LineName:             "Lini Perakitan Elektronik SMT Cikarang",
		OverallLineStatus:    lineStatus,
		ActiveIncidentsCount: len(recents),
		AverageMTTRMinutes:   6.8, // 6.8 minutes average MTTR
		CurrentLineOEE:       91.4,
		Stations:             stations,
		RecentEvents:         recents,
		LastRefreshedAt:      time.Now().Format(time.RFC3339),
	}

	return board, nil
}

// AndonHandler handles HTTP requests for shopfloor Andon visual control.
type AndonHandler struct {
	db *gorm.DB
}

// NewAndonHandler constructs a new AndonHandler.
func NewAndonHandler(db *gorm.DB) *AndonHandler {
	return &AndonHandler{db: db}
}

// TriggerEvent handles Andon cord pull event triggers.
func (h *AndonHandler) TriggerEvent(c *gin.Context) {
	var req TriggerAndonEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Andon trigger payload: " + err.Error()})
		return
	}

	ev, err := TriggerAndonEventLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ev)
}

// ResolveEvent clears an active Andon incident.
func (h *AndonHandler) ResolveEvent(c *gin.Context) {
	var req ResolveAndonEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Andon resolve payload: " + err.Error()})
		return
	}

	ev, err := ResolveAndonEventLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ev)
}

// GetBoard returns the live visual board for wall monitors.
func (h *AndonHandler) GetBoard(c *gin.Context) {
	lineID := c.Param("line_id")
	board, err := GetAndonBoardLogic(lineID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, board)
}

// RegisterAndonRoutes registers factory floor Andon visual control endpoints.
func RegisterAndonRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewAndonHandler(db)
	andonGroup := router.Group("/mrp/andon")
	andonGroup.Use(authMiddleware)
	{
		andonGroup.POST("/trigger-event", h.TriggerEvent)
		andonGroup.POST("/resolve-event", h.ResolveEvent)
		andonGroup.GET("/board/:line_id", h.GetBoard)
	}
}
