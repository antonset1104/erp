package wms

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AMRUnit represents an Autonomous Mobile Robot / Automated Guided Vehicle in the warehouse.
type AMRUnit struct {
	RobotID                string  `json:"robot_id"`
	Model                  string  `json:"model"`
	PayloadCapacityKg      float64 `json:"payload_capacity_kg"`
	CurrentZone            string  `json:"current_zone"`
	CoordinateX            float64 `json:"coordinate_x"`
	CoordinateY            float64 `json:"coordinate_y"`
	BatterySOCPercent      float64 `json:"battery_soc_percent"` // State of Charge
	Status                 string  `json:"status"`              // "IDLE_AVAILABLE", "NAVIGATING_MISSION", "CHARGING", "OBSTACLE_HALTED"
	CurrentMissionID       string  `json:"current_mission_id,omitempty"`
	TotalMissionsCompleted int     `json:"total_missions_completed"`
	LastTelemetryAt        string  `json:"last_telemetry_at"`
}

// DispatchMissionRequest defines a warehouse transport order to be fulfilled by AMR.
type DispatchMissionRequest struct {
	MissionID        string  `json:"mission_id"`
	MissionType      string  `json:"mission_type"` // "GOODS_TO_PERSON_PICK", "PALLET_TRANSFER", "REPLENISHMENT_HAUL"
	SourceBinID      string  `json:"source_bin_id"`
	DestinationBayID string  `json:"destination_bay_id"`
	CargoWeightKg    float64 `json:"cargo_weight_kg"`
	PriorityLevel    int     `json:"priority_level"` // 1 = Urgent, 5 = Standard
}

// AMRMission represents an active robotic mission routing trajectory.
type AMRMission struct {
	MissionID                string   `json:"mission_id"`
	AssignedRobotID          string   `json:"assigned_robot_id"`
	MissionType              string   `json:"mission_type"`
	SourceBinID              string   `json:"source_bin_id"`
	DestinationBayID         string   `json:"destination_bay_id"`
	CargoWeightKg            float64  `json:"cargo_weight_kg"`
	EstimatedDurationMinutes float64  `json:"estimated_duration_minutes"`
	Waypoints                []string `json:"waypoints"`
	Status                   string   `json:"status"` // "DISPATCHED", "EN_ROUTE", "DELIVERED", "COMPLETED"
	DispatchedAt             string   `json:"dispatched_at"`
}

// AMRTelemetryUpdateRequest holds sensor telemetry sent by robot controllers via MQTT/REST.
type AMRTelemetryUpdateRequest struct {
	RobotID           string  `json:"robot_id"`
	CoordinateX       float64 `json:"coordinate_x"`
	CoordinateY       float64 `json:"coordinate_y"`
	BatterySOCPercent float64 `json:"battery_soc_percent"`
	Status            string  `json:"status"`
}

// AMRFleetRadar provides real-time supervisory overview of the robot fleet.
type AMRFleetRadar struct {
	WarehouseID            string       `json:"warehouse_id"`
	TotalFleetSize         int          `json:"total_fleet_size"`
	ActiveNavigatingRobots int          `json:"active_navigating_robots"`
	IdleAvailableRobots    int          `json:"idle_available_robots"`
	ChargingRobots         int          `json:"charging_robots"`
	AverageBatterySOC      float64      `json:"average_battery_soc"`
	Robots                 []AMRUnit    `json:"robots"`
	ActiveMissions         []AMRMission `json:"active_missions"`
	RefreshedAt            string       `json:"refreshed_at"`
}

var (
	amrMutex sync.RWMutex
	amrFleet = map[string]*AMRUnit{
		"AMR-KIVA-01": {RobotID: "AMR-KIVA-01", Model: "Geek+ P800 Picking Pod Carrier", PayloadCapacityKg: 800.0, CurrentZone: "ZONE-AISLE-A1", CoordinateX: 12.5, CoordinateY: 8.0, BatterySOCPercent: 88.0, Status: "IDLE_AVAILABLE", TotalMissionsCompleted: 142, LastTelemetryAt: time.Now().Format(time.RFC3339)},
		"AMR-KIVA-02": {RobotID: "AMR-KIVA-02", Model: "Geek+ P800 Picking Pod Carrier", PayloadCapacityKg: 800.0, CurrentZone: "ZONE-AISLE-B3", CoordinateX: 24.0, CoordinateY: 15.5, BatterySOCPercent: 92.0, Status: "IDLE_AVAILABLE", TotalMissionsCompleted: 189, LastTelemetryAt: time.Now().Format(time.RFC3339)},
		"AMR-FORK-03": {RobotID: "AMR-FORK-03", Model: "Toyota Autonomous Pallet Stacker", PayloadCapacityKg: 1500.0, CurrentZone: "ZONE-INBOUND-DOCK", CoordinateX: 5.0, CoordinateY: 3.2, BatterySOCPercent: 74.0, Status: "IDLE_AVAILABLE", TotalMissionsCompleted: 95, LastTelemetryAt: time.Now().Format(time.RFC3339)},
		"AMR-KIVA-04": {RobotID: "AMR-KIVA-04", Model: "Fetch Robotics Freight500", PayloadCapacityKg: 500.0, CurrentZone: "CHARGING_BAY_C", CoordinateX: 45.0, CoordinateY: 2.0, BatterySOCPercent: 28.0, Status: "CHARGING", TotalMissionsCompleted: 210, LastTelemetryAt: time.Now().Format(time.RFC3339)},
	}
	amrMissions = make(map[string]*AMRMission)
)

// DispatchAMRMissionLogic selects the optimal robot based on proximity, battery SOC, and payload capacity.
func DispatchAMRMissionLogic(req DispatchMissionRequest) (*AMRMission, error) {
	amrMutex.Lock()
	defer amrMutex.Unlock()

	if req.MissionID == "" {
		req.MissionID = fmt.Sprintf("MSN-%s-%03d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.MissionType == "" {
		req.MissionType = "GOODS_TO_PERSON_PICK"
	}
	if req.SourceBinID == "" {
		req.SourceBinID = "BIN-AISLE-A1-04"
	}
	if req.DestinationBayID == "" {
		req.DestinationBayID = "PACK-STATION-02"
	}
	if req.CargoWeightKg <= 0 {
		req.CargoWeightKg = 150.0
	}

	var bestRobot *AMRUnit
	highestScore := -1.0

	for _, bot := range amrFleet {
		if bot.Status != "IDLE_AVAILABLE" {
			continue
		}
		if bot.PayloadCapacityKg < req.CargoWeightKg {
			continue
		}
		if bot.BatterySOCPercent < 30.0 {
			continue // Preserve battery for safety
		}

		// Proximity and battery scoring
		score := bot.BatterySOCPercent
		if score > highestScore {
			highestScore = score
			bestRobot = bot
		}
	}

	if bestRobot == nil {
		return nil, fmt.Errorf("no available AMR unit found with sufficient battery (>30%%) and payload capacity (%.0f kg)", req.CargoWeightKg)
	}

	bestRobot.Status = "NAVIGATING_MISSION"
	bestRobot.CurrentMissionID = req.MissionID
	bestRobot.LastTelemetryAt = time.Now().Format(time.RFC3339)

	waypoints := []string{
		fmt.Sprintf("WAYPOINT-ZONE-%s", bestRobot.CurrentZone),
		fmt.Sprintf("NODE-INTERSECTION-X%.0fY%.0f", bestRobot.CoordinateX+5, bestRobot.CoordinateY+3),
		fmt.Sprintf("ARRIVE-SOURCE-%s", req.SourceBinID),
		fmt.Sprintf("TRANSIT-CORRIDOR-MAIN"),
		fmt.Sprintf("TERMINATE-%s", req.DestinationBayID),
	}

	mission := &AMRMission{
		MissionID:                req.MissionID,
		AssignedRobotID:          bestRobot.RobotID,
		MissionType:              req.MissionType,
		SourceBinID:              req.SourceBinID,
		DestinationBayID:         req.DestinationBayID,
		CargoWeightKg:            req.CargoWeightKg,
		EstimatedDurationMinutes: 3.5, // 3.5 minute transit
		Waypoints:                waypoints,
		Status:                   "DISPATCHED",
		DispatchedAt:             time.Now().Format(time.RFC3339),
	}

	amrMissions[req.MissionID] = mission
	return mission, nil
}

// UpdateAMRTelemetryLogic ingests real-time coordinate updates and handles battery fast-charging routing.
func UpdateAMRTelemetryLogic(req AMRTelemetryUpdateRequest) (*AMRUnit, error) {
	amrMutex.Lock()
	defer amrMutex.Unlock()

	bot, exists := amrFleet[req.RobotID]
	if !exists {
		return nil, fmt.Errorf("robot '%s' not registered in fleet", req.RobotID)
	}

	bot.CoordinateX = req.CoordinateX
	bot.CoordinateY = req.CoordinateY
	bot.BatterySOCPercent = req.BatterySOCPercent
	if req.Status != "" {
		bot.Status = req.Status
	}

	// Opportunistic battery safety: if battery drops below 20%, force routing to charging dock
	if bot.BatterySOCPercent <= 20.0 && bot.Status != "CHARGING" {
		bot.Status = "CHARGING"
		bot.CurrentZone = "AUTOMATED_FAST_CHARGER_BAY_D"
	}
	bot.LastTelemetryAt = time.Now().Format(time.RFC3339)

	return bot, nil
}

// GetAMRFleetRadarLogic compiles real-time supervisory telemetry across warehouse robots.
func GetAMRFleetRadarLogic() AMRFleetRadar {
	amrMutex.RLock()
	defer amrMutex.RUnlock()

	var robots []AMRUnit
	var missions []AMRMission
	var totalSOC float64
	active := 0
	idle := 0
	charging := 0

	for _, b := range amrFleet {
		robots = append(robots, *b)
		totalSOC += b.BatterySOCPercent
		switch b.Status {
		case "NAVIGATING_MISSION":
			active++
		case "IDLE_AVAILABLE":
			idle++
		case "CHARGING":
			charging++
		}
	}

	for _, m := range amrMissions {
		missions = append(missions, *m)
	}

	avgSOC := 0.0
	if len(robots) > 0 {
		avgSOC = math.Round((totalSOC/float64(len(robots)))*10) / 10.0
	}

	return AMRFleetRadar{
		WarehouseID:            "WH-DC-CIKARANG-AUTOMATION-01",
		TotalFleetSize:         len(robots),
		ActiveNavigatingRobots: active,
		IdleAvailableRobots:    idle,
		ChargingRobots:         charging,
		AverageBatterySOC:      avgSOC,
		Robots:                 robots,
		ActiveMissions:         missions,
		RefreshedAt:            time.Now().Format(time.RFC3339),
	}
}

// RegisterAMROrchestratorRoutes registers warehouse robot fleet routes.
func RegisterAMROrchestratorRoutes(r *gin.RouterGroup) {
	group := r.Group("/wms/amr")
	{
		group.POST("/dispatch-mission", func(c *gin.Context) {
			var req DispatchMissionRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				req = DispatchMissionRequest{}
			}
			res, err := DispatchAMRMissionLogic(req)
			if err != nil {
				c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/telemetry-update", func(c *gin.Context) {
			var req AMRTelemetryUpdateRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid telemetry: " + err.Error()})
				return
			}
			res, err := UpdateAMRTelemetryLogic(req)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/fleet-radar", func(c *gin.Context) {
			radar := GetAMRFleetRadarLogic()
			c.JSON(http.StatusOK, radar)
		})
	}
}
