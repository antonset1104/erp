package wms

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CrossDockMission represents a zero-putaway direct dock-to-dock transfer
type CrossDockMission struct {
	MissionID            string    `json:"mission_id"`
	FacilityID           string    `json:"facility_id"`
	PalletBarcode        string    `json:"pallet_barcode"`
	InboundDockDoor      string    `json:"inbound_dock_door"`  // e.g. "DOCK-IN-03"
	OutboundDockDoor     string    `json:"outbound_dock_door"` // e.g. "DOCK-OUT-08"
	InboundReceiptID     string    `json:"inbound_receipt_id"`
	OutboundDeliveryID   string    `json:"outbound_delivery_id"`
	WeightKG             float64   `json:"weight_kg"`
	TransferPriority     string    `json:"transfer_priority"` // "STANDARD", "URGENT_DEPARTURE_SLOT", "EXPEDITE_CRITICAL"
	AssignedRobotID      string    `json:"assigned_robot_id"`
	EstimatedDurationSec float64   `json:"estimated_duration_sec"`
	Status               string    `json:"status"` // "QUEUED", "DISPATCHED", "IN_TRANSIT", "COMPLETED_CROSSDOCK"
	CreatedAt            time.Time `json:"created_at"`
	CompletedAt          *time.Time `json:"completed_at,omitempty"`
}

// AGFRobotUnit models an autonomous industrial forklift
type AGFRobotUnit struct {
	RobotID          string    `json:"robot_id"`
	FacilityID       string    `json:"facility_id"`
	RobotModel       string    `json:"robot_model"` // e.g. "RoboFork AGF-2500 Multi-Directional"
	PayloadCapacityKG float64  `json:"payload_capacity_kg"`
	BatterySoC       float64   `json:"battery_soc_percent"` // 0.0 - 100.0%
	OperationalState string    `json:"operational_state"`   // "IDLE", "EN_ROUTE_PICKUP", "CARRYING_LOAD", "OPPORTUNITY_CHARGING", "MAINTENANCE"
	CoordX           float64   `json:"coord_x_meters"`
	CoordY           float64   `json:"coord_y_meters"`
	CruisingSpeedMPS float64   `json:"cruising_speed_mps"`
	TotalPalletsMoved int      `json:"total_pallets_moved"`
	LastTelemetryAt  time.Time `json:"last_telemetry_at"`
}

// AGFSwarmDispatcher thread-safe management
type AGFSwarmDispatcher struct {
	mu       sync.RWMutex
	missions map[string]*CrossDockMission
	robots   map[string]*AGFRobotUnit
}

// Global AGF Swarm instance
var GlobalAGFSwarmDispatcher = &AGFSwarmDispatcher{
	missions: make(map[string]*CrossDockMission),
	robots:   make(map[string]*AGFRobotUnit),
}

func init() {
	// Seed demo fleet robots
	GlobalAGFSwarmDispatcher.robots["AGF-SWARM-01"] = &AGFRobotUnit{
		RobotID:          "AGF-SWARM-01",
		FacilityID:       "DC-CIKARANG-CENTRAL",
		RobotModel:       "RoboFork AGF-2500 Multi-Directional",
		PayloadCapacityKG: 2500,
		BatterySoC:       92.5,
		OperationalState: "IDLE",
		CoordX:           10.0,
		CoordY:           15.0,
		CruisingSpeedMPS: 1.5,
		TotalPalletsMoved: 142,
		LastTelemetryAt:  time.Now(),
	}
	GlobalAGFSwarmDispatcher.robots["AGF-SWARM-02"] = &AGFRobotUnit{
		RobotID:          "AGF-SWARM-02",
		FacilityID:       "DC-CIKARANG-CENTRAL",
		RobotModel:       "RoboFork AGF-2500 Multi-Directional",
		PayloadCapacityKG: 2500,
		BatterySoC:       28.0, // Low battery, candidate for opportunity charging
		OperationalState: "OPPORTUNITY_CHARGING",
		CoordX:           5.0,
		CoordY:           5.0,
		CruisingSpeedMPS: 1.5,
		TotalPalletsMoved: 189,
		LastTelemetryAt:  time.Now(),
	}
}

// CreateCrossDockMission registers an inbound pallet direct to outbound door
func (d *AGFSwarmDispatcher) CreateCrossDockMission(
	facilityID, pallet, inDoor, outDoor, inReceipt, outDelivery string,
	weight float64, priority string,
) *CrossDockMission {
	d.mu.Lock()
	defer d.mu.Unlock()

	missionID := fmt.Sprintf("MSN-XDOCK-%d", time.Now().UnixNano()%1000000)
	mission := &CrossDockMission{
		MissionID:            missionID,
		FacilityID:           facilityID,
		PalletBarcode:        pallet,
		InboundDockDoor:      inDoor,
		OutboundDockDoor:     outDoor,
		InboundReceiptID:     inReceipt,
		OutboundDeliveryID:   outDelivery,
		WeightKG:             weight,
		TransferPriority:     priority,
		EstimatedDurationSec: 120.0,
		Status:               "QUEUED",
		CreatedAt:            time.Now(),
	}

	d.missions[missionID] = mission
	return mission
}

// DispatchSwarmRobot selects optimal AGF and assigns mission
func (d *AGFSwarmDispatcher) DispatchSwarmRobot(missionID string) (*CrossDockMission, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	mission, ok := d.missions[missionID]
	if !ok {
		return nil, fmt.Errorf("crossdock mission not found: %s", missionID)
	}

	// Find best idle robot with sufficient battery and payload capacity
	var bestRobot *AGFRobotUnit
	for _, robot := range d.robots {
		if robot.OperationalState == "IDLE" && robot.BatterySoC >= 30.0 && robot.PayloadCapacityKG >= mission.WeightKG {
			bestRobot = robot
			break
		}
	}

	// Fallback if idle robot with high battery not found, check any available
	if bestRobot == nil {
		for _, robot := range d.robots {
			if robot.OperationalState != "MAINTENANCE" && robot.BatterySoC > 15.0 && robot.PayloadCapacityKG >= mission.WeightKG {
				bestRobot = robot
				break
			}
		}
	}

	if bestRobot == nil {
		return nil, fmt.Errorf("no suitable AGF robot currently available for payload %.1f kg", mission.WeightKG)
	}

	// Calculate transit distance and duration
	// Inbound dock coord (e.g. 10, 50) to Outbound dock coord (e.g. 80, 50) = 70 meters
	dist := 70.0
	transitSec := dist / bestRobot.CruisingSpeedMPS
	if mission.TransferPriority == "EXPEDITE_CRITICAL" {
		transitSec *= 0.85 // High priority acceleration
	}

	bestRobot.OperationalState = "CARRYING_LOAD"
	bestRobot.BatterySoC -= 1.2
	bestRobot.TotalPalletsMoved++
	bestRobot.LastTelemetryAt = time.Now()

	mission.AssignedRobotID = bestRobot.RobotID
	mission.EstimatedDurationSec = math.Round(transitSec*10) / 10
	mission.Status = "IN_TRANSIT"

	return mission, nil
}

// UpdateTelemetryAndCharging updates robot status and commands opportunity charging
func (d *AGFSwarmDispatcher) UpdateTelemetryAndCharging(
	robotID string, soc float64, coordX, coordY float64, cmdCharging bool,
) (*AGFRobotUnit, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	robot, ok := d.robots[robotID]
	if !ok {
		return nil, fmt.Errorf("robot not found: %s", robotID)
	}

	robot.BatterySoC = soc
	robot.CoordX = coordX
	robot.CoordY = coordY
	robot.LastTelemetryAt = time.Now()

	if cmdCharging || soc < 25.0 {
		robot.OperationalState = "OPPORTUNITY_CHARGING"
	} else if robot.OperationalState == "OPPORTUNITY_CHARGING" && soc >= 85.0 {
		robot.OperationalState = "IDLE"
	}

	return robot, nil
}

// GetSwarmStatus retrieves comprehensive fleet health and crossdock throughput
func (d *AGFSwarmDispatcher) GetSwarmStatus(facilityID string) gin.H {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var fleetList []AGFRobotUnit
	var activeMissions []CrossDockMission
	var totalPallets int
	var avgBattery float64

	for _, r := range d.robots {
		if r.FacilityID == facilityID || facilityID == "" {
			fleetList = append(fleetList, *r)
			totalPallets += r.TotalPalletsMoved
			avgBattery += r.BatterySoC
		}
	}
	if len(fleetList) > 0 {
		avgBattery /= float64(len(fleetList))
	}

	for _, m := range d.missions {
		if m.FacilityID == facilityID || facilityID == "" {
			activeMissions = append(activeMissions, *m)
		}
	}

	return gin.H{
		"facility_id":            facilityID,
		"fleet_size":             len(fleetList),
		"average_battery_soc":    math.Round(avgBattery*10) / 10,
		"total_pallets_handled":  totalPallets,
		"active_missions_count":  len(activeMissions),
		"robots":                 fleetList,
		"recent_missions":        activeMissions,
		"swarm_control_system":   "GoERP Industrial AGF Swarm Dispatcher v1.62.0",
		"zero_putaway_optimized": true,
	}
}

// RegisterAGFSwarmRoutes registers HTTP endpoints
func RegisterAGFSwarmRoutes(r *gin.RouterGroup) {
	agf := r.Group("/wms/agf")
	{
		agf.POST("/create-crossdock-mission", func(c *gin.Context) {
			var req struct {
				FacilityID         string  `json:"facility_id" binding:"required"`
				PalletBarcode      string  `json:"pallet_barcode" binding:"required"`
				InboundDockDoor    string  `json:"inbound_dock_door" binding:"required"`
				OutboundDockDoor   string  `json:"outbound_dock_door" binding:"required"`
				InboundReceiptID   string  `json:"inbound_receipt_id" binding:"required"`
				OutboundDeliveryID string  `json:"outbound_delivery_id" binding:"required"`
				WeightKG           float64 `json:"weight_kg" binding:"required"`
				TransferPriority   string  `json:"transfer_priority"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if req.TransferPriority == "" {
				req.TransferPriority = "STANDARD"
			}

			mission := GlobalAGFSwarmDispatcher.CreateCrossDockMission(
				req.FacilityID, req.PalletBarcode, req.InboundDockDoor, req.OutboundDockDoor,
				req.InboundReceiptID, req.OutboundDeliveryID, req.WeightKG, req.TransferPriority,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": mission})
		})

		agf.POST("/dispatch-swarm-robot", func(c *gin.Context) {
			var req struct {
				MissionID string `json:"mission_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			mission, err := GlobalAGFSwarmDispatcher.DispatchSwarmRobot(req.MissionID)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": mission})
		})

		agf.POST("/update-telemetry-charging", func(c *gin.Context) {
			var req struct {
				RobotID             string  `json:"robot_id" binding:"required"`
				BatterySoC          float64 `json:"battery_soc_percent" binding:"required"`
				CoordX              float64 `json:"coord_x_meters"`
				CoordY              float64 `json:"coord_y_meters"`
				CommandFastCharging bool    `json:"command_fast_charging"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			robot, err := GlobalAGFSwarmDispatcher.UpdateTelemetryAndCharging(
				req.RobotID, req.BatterySoC, req.CoordX, req.CoordY, req.CommandFastCharging,
			)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": robot})
		})

		agf.GET("/swarm-status/:facility_id", func(c *gin.Context) {
			facilityID := c.Param("facility_id")
			status := GlobalAGFSwarmDispatcher.GetSwarmStatus(facilityID)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": status})
		})
	}
}
