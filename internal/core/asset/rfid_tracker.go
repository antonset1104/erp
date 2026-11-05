package asset

import (
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GeofenceZone defines monitored physical warehouse and plant zones.
type GeofenceZone string

const (
	ZoneInboundDock     GeofenceZone = "ZONE_INBOUND_DOCK"
	ZoneProductionFloor GeofenceZone = "ZONE_PRODUCTION_FLOOR"
	ZoneFinishedGoodsWH GeofenceZone = "ZONE_FINISHED_GOODS_WH"
	ZoneOutboundBay     GeofenceZone = "ZONE_OUTBOUND_BAY"
	ZoneCustomerSite    GeofenceZone = "ZONE_EXTERNAL_CUSTOMER_SITE"
	ZoneMaintenanceBay  GeofenceZone = "ZONE_MAINTENANCE_BAY"
)

// TrackedAssetCategory defines the asset family.
type TrackedAssetCategory string

const (
	CategoryReturnableContainer TrackedAssetCategory = "RETURNABLE_CONTAINER" // Pallets, Totes, Kegs
	CategoryMobileMachinery     TrackedAssetCategory = "MOBILE_MACHINERY"     // Forklifts, AGVs
	CategoryPrecisionTooling    TrackedAssetCategory = "PRECISION_TOOLING"    // Molds, Dies, Calibration Tools
	CategoryFleetTrailer        TrackedAssetCategory = "FLEET_TRAILER"        // Cargo Containers
)

// RFIDPingRequest represents an incoming telemetry signal from an RFID/BLE gateway.
type RFIDPingRequest struct {
	EPCTagID       string               `json:"epc_tag_id"` // EPC Gen2 / BLE MAC
	AssetID        uint                 `json:"asset_id"`
	AssetName      string               `json:"asset_name"`
	Category       TrackedAssetCategory `json:"category"`
	ReaderID       string               `json:"reader_id"`
	CurrentZone    GeofenceZone         `json:"current_zone"`
	SignalStrength float64              `json:"signal_strength_dbm"` // RSSI
	BatteryLevel   float64              `json:"battery_level_percent"`
	PingTimestamp  string               `json:"ping_timestamp"`
}

// TrackedAssetStatus represents the real-time state of an asset on the radar.
type TrackedAssetStatus struct {
	EPCTagID          string               `json:"epc_tag_id"`
	AssetID           uint                 `json:"asset_id"`
	AssetName         string               `json:"asset_name"`
	Category          TrackedAssetCategory `json:"category"`
	AssignedHomeZone  GeofenceZone         `json:"assigned_home_zone"`
	CurrentZone       GeofenceZone         `json:"current_zone"`
	ReaderID          string               `json:"reader_id"`
	LastSeenAt        string               `json:"last_seen_at"`
	DwellDaysInZone   float64              `json:"dwell_days_in_zone"`
	IsMisplaced       bool                 `json:"is_misplaced"`
	MisplacedAlert    string               `json:"misplaced_alert,omitempty"`
	DemurrageDays     int                  `json:"demurrage_days"`
	DemurrageDailyFee float64              `json:"demurrage_daily_fee_idr"`
	AccruedDemurrage  float64              `json:"accrued_demurrage_idr"`
	BatteryStatus     string               `json:"battery_status"`
}

// RadarMapReport encapsulates all tracked assets across zones.
type RadarMapReport struct {
	GeneratedAt           string               `json:"generated_at"`
	TotalTrackedAssets    int                  `json:"total_tracked_assets"`
	MisplacedCount        int                  `json:"misplaced_count"`
	AssetsInFieldCount    int                  `json:"assets_in_field_count"`
	TotalAccruedDemurrage float64              `json:"total_accrued_demurrage_idr"`
	ZoneDistribution      map[GeofenceZone]int `json:"zone_distribution"`
	Assets                []TrackedAssetStatus `json:"assets"`
}

// RFIDTrackerService manages asset location telemetry and demurrage auditing.
type RFIDTrackerService struct {
	mu     sync.RWMutex
	assets map[string]*TrackedAssetStatus
}

// NewRFIDTrackerService creates a new RFIDTrackerService.
func NewRFIDTrackerService() *RFIDTrackerService {
	svc := &RFIDTrackerService{
		assets: make(map[string]*TrackedAssetStatus),
	}

	// Initialize seed assets for instant demonstration
	svc.assets["EPC-RTP-00101"] = &TrackedAssetStatus{
		EPCTagID:          "EPC-RTP-00101",
		AssetID:           501,
		AssetName:         "CHEP Heavy-Duty Plastic Pallet #101",
		Category:          CategoryReturnableContainer,
		AssignedHomeZone:  ZoneFinishedGoodsWH,
		CurrentZone:       ZoneCustomerSite,
		ReaderID:          "BLE-GW-EXT-CUST-04",
		LastSeenAt:        time.Now().Add(-14 * 24 * time.Hour).Format(time.RFC3339),
		DwellDaysInZone:   14.0,
		IsMisplaced:       false,
		DemurrageDays:     7, // 14 days dwell minus 7 free days
		DemurrageDailyFee: 45000,
		AccruedDemurrage:  315000,
		BatteryStatus:     "HEALTHY (92%)",
	}

	svc.assets["EPC-AGV-00204"] = &TrackedAssetStatus{
		EPCTagID:          "EPC-AGV-00204",
		AssetID:           502,
		AssetName:         "Autonomous Mobile Robot AGV #04",
		Category:          CategoryMobileMachinery,
		AssignedHomeZone:  ZoneProductionFloor,
		CurrentZone:       ZoneInboundDock,
		ReaderID:          "RFID-DOCK-READER-01",
		LastSeenAt:        time.Now().Format(time.RFC3339),
		DwellDaysInZone:   0.2,
		IsMisplaced:       false,
		DemurrageDays:     0,
		DemurrageDailyFee: 0,
		AccruedDemurrage:  0,
		BatteryStatus:     "GOOD (78%)",
	}

	svc.assets["EPC-MOLD-00309"] = &TrackedAssetStatus{
		EPCTagID:          "EPC-MOLD-00309",
		AssetID:           503,
		AssetName:         "Precision Injection Mold Core #B-12",
		Category:          CategoryPrecisionTooling,
		AssignedHomeZone:  ZoneMaintenanceBay,
		CurrentZone:       ZoneOutboundBay, // Misplaced! About to be shipped by mistake
		ReaderID:          "RFID-OUTBOUND-03",
		LastSeenAt:        time.Now().Format(time.RFC3339),
		DwellDaysInZone:   0.8,
		IsMisplaced:       true,
		MisplacedAlert:    "PERINGATAN KRITIS: Cetakan presisi terdeteksi di Outbound Bay padahal seharusnya di Maintenance Bay!",
		DemurrageDays:     0,
		DemurrageDailyFee: 0,
		AccruedDemurrage:  0,
		BatteryStatus:     "OPTIMAL (88%)",
	}

	return svc
}

// IngestPing processes an incoming RFID/BLE beacon ping.
func (s *RFIDTrackerService) IngestPing(req RFIDPingRequest) (*TrackedAssetStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.EPCTagID == "" {
		req.EPCTagID = fmt.Sprintf("EPC-TAG-%s-%04d", time.Now().Format("20060102"), req.AssetID)
	}

	asset, exists := s.assets[req.EPCTagID]
	if !exists {
		asset = &TrackedAssetStatus{
			EPCTagID:         req.EPCTagID,
			AssetID:          req.AssetID,
			AssetName:        req.AssetName,
			Category:         req.Category,
			AssignedHomeZone: req.CurrentZone,
		}
	}

	asset.CurrentZone = req.CurrentZone
	asset.ReaderID = req.ReaderID
	asset.LastSeenAt = time.Now().Format(time.RFC3339)
	asset.BatteryStatus = fmt.Sprintf("%.0f%%", req.BatteryLevel)

	// Demurrage calculation for client sites
	if req.CurrentZone == ZoneCustomerSite {
		asset.DwellDaysInZone = math.Max(asset.DwellDaysInZone, 1.0)
		freeDays := 7.0
		if asset.DwellDaysInZone > freeDays {
			asset.DemurrageDays = int(asset.DwellDaysInZone - freeDays)
			if asset.DemurrageDailyFee == 0 {
				asset.DemurrageDailyFee = 50000
			}
			asset.AccruedDemurrage = float64(asset.DemurrageDays) * asset.DemurrageDailyFee
		}
	}

	// Misplaced asset check
	if asset.Category == CategoryPrecisionTooling && req.CurrentZone == ZoneOutboundBay && asset.AssignedHomeZone == ZoneMaintenanceBay {
		asset.IsMisplaced = true
		asset.MisplacedAlert = "PERINGATAN KRITIS: Cetakan presisi terdeteksi di Outbound Bay padahal seharusnya di Maintenance Bay!"
	} else {
		asset.IsMisplaced = false
		asset.MisplacedAlert = ""
	}

	s.assets[req.EPCTagID] = asset
	return asset, nil
}

// GetRadarMap aggregates enterprise asset positioning and demurrage status.
func (s *RFIDTrackerService) GetRadarMap() (*RadarMapReport, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	dist := make(map[GeofenceZone]int)
	misplaced := 0
	inField := 0
	totalDemurrage := 0.0
	var list []TrackedAssetStatus

	for _, a := range s.assets {
		dist[a.CurrentZone]++
		if a.IsMisplaced {
			misplaced++
		}
		if a.CurrentZone == ZoneCustomerSite {
			inField++
		}
		totalDemurrage += a.AccruedDemurrage
		list = append(list, *a)
	}

	return &RadarMapReport{
		GeneratedAt:           time.Now().Format(time.RFC3339),
		TotalTrackedAssets:    len(list),
		MisplacedCount:        misplaced,
		AssetsInFieldCount:    inField,
		TotalAccruedDemurrage: totalDemurrage,
		ZoneDistribution:      dist,
		Assets:                list,
	}, nil
}

var defaultRFIDTrackerSvc = NewRFIDTrackerService()

// RegisterRFIDTrackerRoutes mounts HTTP routes for RFID/BLE Asset Tracking.
func RegisterRFIDTrackerRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	g := router.Group("/asset/tracking")
	g.Use(authMiddleware)
	{
		g.POST("/rfid-ping", func(c *gin.Context) {
			var req RFIDPingRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res, err := defaultRFIDTrackerSvc.IngestPing(req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})

		g.GET("/radar-map", func(c *gin.Context) {
			res, err := defaultRFIDTrackerSvc.GetRadarMap()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, res)
		})
	}
}
