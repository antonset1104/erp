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

// ColdChainTelemetryReading records real-time sensor measurements from maritime reefer containers.
type ColdChainTelemetryReading struct {
	ReadingID           string  `json:"reading_id"`
	ShipmentID          string  `json:"shipment_id"`  // e.g. "SHP-REEFER-2026-904"
	ContainerNo         string  `json:"container_no"` // e.g. "MSCU-884210-9"
	TemperatureCelsius  float64 `json:"temperature_celsius"`
	RelativeHumidityPct float64 `json:"relative_humidity_pct"`
	TiltAngleDegrees    float64 `json:"tilt_angle_degrees"`
	CompressorStatus    string  `json:"compressor_status"` // "RUNNING_NORMAL", "POWER_DISCONNECTED", "DEFROST"
	GPSLatitude         float64 `json:"gps_latitude"`
	GPSLongitude        float64 `json:"gps_longitude"`
	Timestamp           string  `json:"timestamp"`
}

// ColdChainShipmentPolicy holds cargo insurance parameters and SLA thermal boundaries.
type ColdChainShipmentPolicy struct {
	ShipmentID                 string  `json:"shipment_id"`
	CargoDescription           string  `json:"cargo_description"`
	DeclaredInsuredValueUSD    float64 `json:"declared_insured_value_usd"`
	MinTempLimitC              float64 `json:"min_temp_limit_c"`
	MaxTempLimitC              float64 `json:"max_temp_limit_c"`
	MaxAllowableExcursionMins  int     `json:"max_allowable_excursion_mins"`
	UnderwriterName            string  `json:"underwriter_name"`
	PolicyNumber               string  `json:"policy_number"`
	CumulativeExcursionMinutes int     `json:"cumulative_excursion_minutes"`
	PeakExcursionTempC         float64 `json:"peak_excursion_temp_c"`
	Status                     string  `json:"status"` // "IN_TRANSIT_COMPLIANT", "EXCURSION_BREACH_ALERT", "CLAIM_PAID"
	CreatedAt                  string  `json:"created_at"`
}

// ParametricClaimSettlement models the instant digital payout upon verified sensor excursion.
type ParametricClaimSettlement struct {
	ClaimID                  string  `json:"claim_id"`
	ShipmentID               string  `json:"shipment_id"`
	PolicyNumber             string  `json:"policy_number"`
	ExcursionDurationMinutes int     `json:"excursion_duration_minutes"`
	PeakRecordedTempC        float64 `json:"peak_recorded_temp_c"`
	CargoDamageVerdict       string  `json:"cargo_damage_verdict"` // "TOTAL_LOSS_BIOLOGICAL_DEGRADATION", "PARTIAL_RETEST_REQUIRED"
	ClaimPayoutUSD           float64 `json:"claim_payout_usd"`
	TamperProofAuditHash     string  `json:"tamper_proof_audit_hash"`
	DirectDisbursementRef    string  `json:"direct_disbursement_ref"`
	SettledAt                string  `json:"settled_at"`
	Status                   string  `json:"status"` // "AUTO_DISBURSED", "PENDING_UNDERWRITER_CONFIRMATION"
}

// SettleClaimRequest holds parameters to trigger parametric cargo insurance payment.
type SettleClaimRequest struct {
	ShipmentID string `json:"shipment_id"`
	Inspector  string `json:"inspector"`
}

// ColdChainStore manages thread-safe state for cargo temperature and claims.
type ColdChainStore struct {
	mu        sync.RWMutex
	shipments map[string]*ColdChainShipmentPolicy
	readings  map[string][]ColdChainTelemetryReading
	claims    map[string]*ParametricClaimSettlement
}

var globalColdChainStore = &ColdChainStore{
	shipments: make(map[string]*ColdChainShipmentPolicy),
	readings:  make(map[string][]ColdChainTelemetryReading),
	claims:    make(map[string]*ParametricClaimSettlement),
}

func init() {
	// Seed demo high-value pharma biologics cargo
	demoShipment := &ColdChainShipmentPolicy{
		ShipmentID:                 "SHP-REEFER-2026-904",
		CargoDescription:           "Monoclonal Antibody Oncology Biologics (10,000 Pre-Filled Syringes)",
		DeclaredInsuredValueUSD:    1250000.0,
		MinTempLimitC:              2.0,
		MaxTempLimitC:              8.0,
		MaxAllowableExcursionMins:  120, // 2 hours max
		UnderwriterName:            "Munich Re Marine Parametric Syndicate",
		PolicyNumber:               "POL-PARAMETRIC-REEFER-8819",
		CumulativeExcursionMinutes: 145,  // Breached! 145 mins
		PeakExcursionTempC:         14.2, // Rose to 14.2 C
		Status:                     "EXCURSION_BREACH_ALERT",
		CreatedAt:                  time.Now().Format(time.RFC3339),
	}
	globalColdChainStore.shipments[demoShipment.ShipmentID] = demoShipment

	// Seed telemetry reading
	globalColdChainStore.readings[demoShipment.ShipmentID] = []ColdChainTelemetryReading{
		{
			ReadingID:           "READ-TEL-001",
			ShipmentID:          demoShipment.ShipmentID,
			ContainerNo:         "MSCU-884210-9",
			TemperatureCelsius:  14.2,
			RelativeHumidityPct: 82.5,
			TiltAngleDegrees:    1.2,
			CompressorStatus:    "POWER_DISCONNECTED",
			GPSLatitude:         -6.102,
			GPSLongitude:        106.882,
			Timestamp:           time.Now().Format(time.RFC3339),
		},
	}
}

// IngestColdChainTelemetryLogic registers a live sensor ping and updates thermal breach trackers.
func IngestColdChainTelemetryLogic(reading ColdChainTelemetryReading) (*ColdChainTelemetryReading, error) {
	globalColdChainStore.mu.Lock()
	defer globalColdChainStore.mu.Unlock()

	if reading.ShipmentID == "" {
		reading.ShipmentID = "SHP-REEFER-GENERIC"
	}
	if reading.ReadingID == "" {
		reading.ReadingID = fmt.Sprintf("READ-TEL-%d", time.Now().UnixNano()%1000000)
	}
	if reading.Timestamp == "" {
		reading.Timestamp = time.Now().Format(time.RFC3339)
	}

	globalColdChainStore.readings[reading.ShipmentID] = append(globalColdChainStore.readings[reading.ShipmentID], reading)

	if policy, exists := globalColdChainStore.shipments[reading.ShipmentID]; exists {
		if reading.TemperatureCelsius > policy.MaxTempLimitC || reading.TemperatureCelsius < policy.MinTempLimitC {
			policy.CumulativeExcursionMinutes += 15 // assume 15 min ping interval
			if reading.TemperatureCelsius > policy.PeakExcursionTempC {
				policy.PeakExcursionTempC = reading.TemperatureCelsius
			}
			if policy.CumulativeExcursionMinutes >= policy.MaxAllowableExcursionMins {
				policy.Status = "EXCURSION_BREACH_ALERT"
			}
		}
	}

	return &reading, nil
}

// EvaluateExcursionLogic checks if excursion threshold warrants automatic claim filing.
func EvaluateExcursionLogic(shipmentID string) (*ColdChainShipmentPolicy, error) {
	globalColdChainStore.mu.RLock()
	defer globalColdChainStore.mu.RUnlock()

	policy, exists := globalColdChainStore.shipments[shipmentID]
	if !exists {
		return nil, fmt.Errorf("shipment policy '%s' not found", shipmentID)
	}
	return policy, nil
}

// SettleSmartClaimLogic triggers instant parametric compensation disbursement.
func SettleSmartClaimLogic(req SettleClaimRequest) (*ParametricClaimSettlement, error) {
	globalColdChainStore.mu.Lock()
	defer globalColdChainStore.mu.Unlock()

	policy, exists := globalColdChainStore.shipments[req.ShipmentID]
	if !exists {
		return nil, fmt.Errorf("shipment policy '%s' not found", req.ShipmentID)
	}
	if policy.CumulativeExcursionMinutes < policy.MaxAllowableExcursionMins {
		return nil, fmt.Errorf("excursion duration (%d mins) does not exceed parametric claim trigger threshold (%d mins)",
			policy.CumulativeExcursionMinutes, policy.MaxAllowableExcursionMins)
	}

	claimID := fmt.Sprintf("CLM-PARAMETRIC-%d", time.Now().UnixNano()%1000000)
	payout := policy.DeclaredInsuredValueUSD // Total loss payout for spoiled biologics

	// Generate SHA-256 cryptographic proof of sensor evidence
	rawEvidence := fmt.Sprintf("%s|%s|%.2f|%d|%.2f|%s",
		claimID, policy.ShipmentID, payout, policy.CumulativeExcursionMinutes, policy.PeakExcursionTempC, policy.UnderwriterName)
	hasher := sha256.New()
	hasher.Write([]byte(rawEvidence))
	digest := hex.EncodeToString(hasher.Sum(nil))

	disbRef := fmt.Sprintf("WIRE-INS-DISB-%d", time.Now().UnixNano()%10000000)

	claim := &ParametricClaimSettlement{
		ClaimID:                  claimID,
		ShipmentID:               policy.ShipmentID,
		PolicyNumber:             policy.PolicyNumber,
		ExcursionDurationMinutes: policy.CumulativeExcursionMinutes,
		PeakRecordedTempC:        policy.PeakExcursionTempC,
		CargoDamageVerdict:       "TOTAL_LOSS_BIOLOGICAL_DEGRADATION",
		ClaimPayoutUSD:           payout,
		TamperProofAuditHash:     digest,
		DirectDisbursementRef:    disbRef,
		SettledAt:                time.Now().Format(time.RFC3339),
		Status:                   "AUTO_DISBURSED",
	}

	policy.Status = "CLAIM_PAID"
	globalColdChainStore.claims[claimID] = claim
	return claim, nil
}

// GetCargoStatusLogic returns full thermal track-and-trace audit data.
func GetCargoStatusLogic(shipmentID string) (*ColdChainShipmentPolicy, error) {
	return EvaluateExcursionLogic(shipmentID)
}

// RegisterColdChainInsuranceRoutes mounts cold chain insurance API endpoints.
func RegisterColdChainInsuranceRoutes(r *gin.RouterGroup) {
	group := r.Group("/fleet/coldchain")
	{
		group.POST("/telemetry-reading", func(c *gin.Context) {
			var reading ColdChainTelemetryReading
			if err := c.ShouldBindJSON(&reading); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid coldchain telemetry payload: " + err.Error()})
				return
			}
			res, err := IngestColdChainTelemetryLogic(reading)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, res)
		})

		group.POST("/evaluate-excursion", func(c *gin.Context) {
			shipmentID := c.DefaultQuery("shipment_id", "SHP-REEFER-2026-904")
			res, err := EvaluateExcursionLogic(shipmentID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.POST("/settle-smart-claim", func(c *gin.Context) {
			var req SettleClaimRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid settle claim payload: " + err.Error()})
				return
			}
			res, err := SettleSmartClaimLogic(req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		group.GET("/cargo-status/:shipment_id", func(c *gin.Context) {
			id := c.Param("shipment_id")
			res, err := GetCargoStatusLogic(id)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
