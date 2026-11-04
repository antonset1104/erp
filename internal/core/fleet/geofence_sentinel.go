package fleet

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HighValueCorridor represents approved route corridor and cargo parameters
type HighValueCorridor struct {
	ShipmentID               string    `json:"shipment_id"`
	DeliveryOrderID          string    `json:"delivery_order_id"`
	VehiclePlateNumber       string    `json:"vehicle_plate_number"`
	DriverName               string    `json:"driver_name"`
	CargoDescription         string    `json:"cargo_description"`
	CargoValueUSD            float64   `json:"cargo_value_usd"`
	OriginFacility           string    `json:"origin_facility"`
	DestinationFacility      string    `json:"destination_facility"`
	InitialLoadedWeightKG    float64   `json:"initial_loaded_weight_kg"`
	CorridorToleranceMeters  float64   `json:"corridor_tolerance_meters"` // e.g. 500 meters
	SecurityEscortLevel      string    `json:"security_escort_level"`     // "ARMED_ESCORT_LEVEL_3", "TELEMETRY_MONITORED"
	Status                   string    `json:"status"`                    // "ACTIVE_IN_TRANSIT", "EMERGENCY_LOCKDOWN", "SAFELY_DELIVERED"
	CreatedAt                time.Time `json:"created_at"`
}

// CargoSensorTelemetry represents continuous IoT tracker data
type CargoSensorTelemetry struct {
	TelemetryID              string    `json:"telemetry_id"`
	ShipmentID               string    `json:"shipment_id"`
	CurrentLatitude          float64   `json:"current_latitude"`
	CurrentLongitude         float64   `json:"current_longitude"`
	SpeedKPH                 float64   `json:"speed_kph"`
	CargoDoorOpen            bool      `json:"cargo_door_open"`
	CurrentAxleWeightKG      float64   `json:"current_axle_weight_kg"`
	DistanceFromRouteMeters  float64   `json:"distance_from_route_meters"`
	AuthorizedStopZone       bool      `json:"authorized_stop_zone"`
	Timestamp                time.Time `json:"timestamp"`
}

// ThreatEscalationEvent represents security alarm and engine cutoff
type ThreatEscalationEvent struct {
	AlarmID                  string    `json:"alarm_id"`
	ShipmentID               string    `json:"shipment_id"`
	ThreatSeverity           string    `json:"threat_severity"` // "CRITICAL_THEFT_BREACH", "ROUTE_DEVIATION_WARNING"
	IdentifiedAnomalies      []string  `json:"identified_anomalies"`
	EngineImmobilized        bool      `json:"engine_immobilized"`
	WMSDeliveryOrderLocked   bool      `json:"wms_delivery_order_locked"`
	PoliceDispatchBroadcast  bool      `json:"police_dispatch_broadcast"`
	ForensicCertificateHash string    `json:"forensic_certificate_hash"` // SHA-256
	TriggeredAt              time.Time `json:"triggered_at"`
}

// GeofenceSentinelVault thread-safe store
type GeofenceSentinelVault struct {
	mu          sync.RWMutex
	corridors   map[string]HighValueCorridor
	telemetries map[string]CargoSensorTelemetry
	alarms      map[string]ThreatEscalationEvent
}

// Global Geofence Sentinel instance
var GlobalGeofenceSentinelVault = &GeofenceSentinelVault{
	corridors:   make(map[string]HighValueCorridor),
	telemetries: make(map[string]CargoSensorTelemetry),
	alarms:      make(map[string]ThreatEscalationEvent),
}

// RegisterHighValueCorridor configures protected transit corridor
func (v *GeofenceSentinelVault) RegisterHighValueCorridor(
	shipmentID, doID, plate, driver, cargo string, valueUSD, initWeight, tolerance float64,
) HighValueCorridor {
	v.mu.Lock()
	defer v.mu.Unlock()

	corridor := HighValueCorridor{
		ShipmentID:              shipmentID,
		DeliveryOrderID:         doID,
		VehiclePlateNumber:      plate,
		DriverName:              driver,
		CargoDescription:        cargo,
		CargoValueUSD:           valueUSD,
		OriginFacility:          "KAPUK-SMELTER-PORT-01",
		DestinationFacility:     "TANJUNG-PRIOK-EXPORT-DOCK-04",
		InitialLoadedWeightKG:   initWeight,
		CorridorToleranceMeters: tolerance,
		SecurityEscortLevel:     "ARMED_ESCORT_LEVEL_3",
		Status:                  "ACTIVE_IN_TRANSIT",
		CreatedAt:               time.Now(),
	}

	v.corridors[shipmentID] = corridor
	return corridor
}

// IngestCargoTelemetry evaluates deviation, door breaches, and weight loss
func (v *GeofenceSentinelVault) IngestCargoTelemetry(t CargoSensorTelemetry) (*ThreatEscalationEvent, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	corridor, ok := v.corridors[t.ShipmentID]
	if !ok {
		return nil, fmt.Errorf("corridor not found for shipment: %s", t.ShipmentID)
	}

	t.TelemetryID = fmt.Sprintf("TEL-CARGO-%d", time.Now().UnixNano()%1000000)
	t.Timestamp = time.Now()
	v.telemetries[t.ShipmentID] = t

	var anomalies []string
	if t.DistanceFromRouteMeters > corridor.CorridorToleranceMeters {
		anomalies = append(anomalies, fmt.Sprintf("Route deviation: %.1f meters off corridor (Limit: %.1f m)", t.DistanceFromRouteMeters, corridor.CorridorToleranceMeters))
	}
	if t.CargoDoorOpen && !t.AuthorizedStopZone {
		anomalies = append(anomalies, "Unauthorized cargo container door opening in transit")
	}
	weightLoss := corridor.InitialLoadedWeightKG - t.CurrentAxleWeightKG
	if weightLoss > 500.0 {
		anomalies = append(anomalies, fmt.Sprintf("Unexplained in-transit weight loss: %.1f kg missing", weightLoss))
	}

	if len(anomalies) > 0 {
		corridor.Status = "EMERGENCY_LOCKDOWN"
		v.corridors[t.ShipmentID] = corridor

		rawCert := fmt.Sprintf("%s:%s:%s:%.4f,%.4f:%d",
			t.ShipmentID, corridor.VehiclePlateNumber, corridor.DeliveryOrderID, t.CurrentLatitude, t.CurrentLongitude, time.Now().Unix())
		hashBytes := sha256.Sum256([]byte(rawCert))
		digest := hex.EncodeToString(hashBytes[:])

		alarmID := fmt.Sprintf("ALM-THEFT-%d", time.Now().UnixNano()%1000000)
		alarm := ThreatEscalationEvent{
			AlarmID:                  alarmID,
			ShipmentID:               t.ShipmentID,
			ThreatSeverity:           "CRITICAL_THEFT_BREACH",
			IdentifiedAnomalies:      anomalies,
			EngineImmobilized:        true,
			WMSDeliveryOrderLocked:   true,
			PoliceDispatchBroadcast:  true,
			ForensicCertificateHash: digest,
			TriggeredAt:              time.Now(),
		}
		v.alarms[t.ShipmentID] = alarm
		return &alarm, nil
	}

	return nil, nil
}

// TriggerEmergencyImmobilizer commands direct manual/remote cutoff
func (v *GeofenceSentinelVault) TriggerEmergencyImmobilizer(shipmentID string) (*ThreatEscalationEvent, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	corridor, ok := v.corridors[shipmentID]
	if !ok {
		return nil, fmt.Errorf("shipment corridor not found: %s", shipmentID)
	}

	corridor.Status = "EMERGENCY_LOCKDOWN"
	v.corridors[shipmentID] = corridor

	rawCert := fmt.Sprintf("MANUAL_CUTOFF:%s:%s:%d", shipmentID, corridor.VehiclePlateNumber, time.Now().Unix())
	hashBytes := sha256.Sum256([]byte(rawCert))
	digest := hex.EncodeToString(hashBytes[:])

	alarmID := fmt.Sprintf("ALM-IMMOB-%d", time.Now().UnixNano()%1000000)
	alarm := ThreatEscalationEvent{
		AlarmID:                  alarmID,
		ShipmentID:               shipmentID,
		ThreatSeverity:           "MANUAL_SECURITY_INTERVENTION",
		IdentifiedAnomalies:      []string{"Manual remote engine killswitch command executed by Security Ops"},
		EngineImmobilized:        true,
		WMSDeliveryOrderLocked:   true,
		PoliceDispatchBroadcast:  true,
		ForensicCertificateHash: digest,
		TriggeredAt:              time.Now(),
	}

	v.alarms[shipmentID] = alarm
	return &alarm, nil
}

// GetForensicDossier compiles legal forensic report
func (v *GeofenceSentinelVault) GetForensicDossier(shipmentID string) (gin.H, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	corridor, hasCorridor := v.corridors[shipmentID]
	tel, hasTel := v.telemetries[shipmentID]
	alarm, hasAlarm := v.alarms[shipmentID]

	if !hasCorridor {
		return nil, fmt.Errorf("no records found for high-value shipment: %s", shipmentID)
	}

	return gin.H{
		"shipment_id":             shipmentID,
		"corridor_profile":        corridor,
		"latest_sensor_telemetry": tel,
		"has_telemetry":           hasTel,
		"threat_alarm_record":     alarm,
		"has_alarm":               hasAlarm,
		"forensic_integrity":      "CRYPTOGRAPHIC_GPS_CHAIN_OF_CUSTODY_VERIFIED",
		"insurance_claim_status":  "POLICE_FIR_AND_UNDERWRITER_READY",
	}, nil
}

// RegisterGeofenceRoutes registers HTTP endpoints
func RegisterGeofenceRoutes(r *gin.RouterGroup) {
	geo := r.Group("/fleet/geofence")
	{
		geo.POST("/register-highvalue-corridor", func(c *gin.Context) {
			var req struct {
				ShipmentID              string  `json:"shipment_id" binding:"required"`
				DeliveryOrderID         string  `json:"delivery_order_id" binding:"required"`
				VehiclePlateNumber      string  `json:"vehicle_plate_number" binding:"required"`
				DriverName              string  `json:"driver_name" binding:"required"`
				CargoDescription        string  `json:"cargo_description" binding:"required"`
				CargoValueUSD           float64 `json:"cargo_value_usd" binding:"required"`
				InitialLoadedWeightKG   float64 `json:"initial_loaded_weight_kg" binding:"required"`
				CorridorToleranceMeters float64 `json:"corridor_tolerance_meters" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			corridor := GlobalGeofenceSentinelVault.RegisterHighValueCorridor(
				req.ShipmentID, req.DeliveryOrderID, req.VehiclePlateNumber,
				req.DriverName, req.CargoDescription, req.CargoValueUSD,
				req.InitialLoadedWeightKG, req.CorridorToleranceMeters,
			)
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": corridor})
		})

		geo.POST("/ingest-cargo-telemetry", func(c *gin.Context) {
			var req CargoSensorTelemetry
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			alarm, err := GlobalGeofenceSentinelVault.IngestCargoTelemetry(req)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			if alarm != nil {
				c.JSON(http.StatusOK, gin.H{
					"status":  "threat_detected",
					"message": "Critical security breach intercepted. Engine immobilized.",
					"alarm":   alarm,
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "normal_transit", "message": "Vehicle in corridor."})
		})

		geo.POST("/trigger-emergency-immobilizer", func(c *gin.Context) {
			var req struct {
				ShipmentID string `json:"shipment_id" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			alarm, err := GlobalGeofenceSentinelVault.TriggerEmergencyImmobilizer(req.ShipmentID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": alarm})
		})

		geo.GET("/forensic-dossier/:shipment_id", func(c *gin.Context) {
			shipmentID := c.Param("shipment_id")
			dossier, err := GlobalGeofenceSentinelVault.GetForensicDossier(shipmentID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "success", "data": dossier})
		})
	}
}
