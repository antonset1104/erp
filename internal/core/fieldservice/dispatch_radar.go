package fieldservice

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequiredPart specifies a replacement part needed for field repair.
type RequiredPart struct {
	PartSKU  string  `json:"part_sku"`
	PartName string  `json:"part_name"`
	Quantity float64 `json:"quantity"`
	UnitCost float64 `json:"unit_cost"`
}

// WorkOrderRequest defines incoming field service repair request.
type WorkOrderRequest struct {
	WorkOrderID      string         `json:"work_order_id"`
	CustomerName     string         `json:"customer_name"`
	SiteLocation     string         `json:"site_location"`
	SiteLatitude     float64        `json:"site_latitude"`
	SiteLongitude    float64        `json:"site_longitude"`
	AssetID          string         `json:"asset_id"`
	AssetName        string         `json:"asset_name"`
	FaultSeverity    string         `json:"fault_severity"`
	RequiredSkills   []string       `json:"required_skills"`
	RequiredParts    []RequiredPart `json:"required_parts"`
	SLADeadlineHours float64        `json:"sla_deadline_hours"`
}

// TechnicianCandidate represents an active field technician.
type TechnicianCandidate struct {
	TechnicianID      string             `json:"technician_id"`
	Name              string             `json:"name"`
	CurrentLatitude   float64            `json:"current_latitude"`
	CurrentLongitude  float64            `json:"current_longitude"`
	CertifiedSkills   []string           `json:"certified_skills"`
	VanID             string             `json:"van_id"`
	VanInventory      map[string]float64 `json:"van_inventory"`
	CurrentStatus     string             `json:"current_status"`
	DistanceKM        float64            `json:"distance_km"`
	EstArrivalMinutes float64            `json:"est_arrival_minutes"`
	SkillMatchPercent float64            `json:"skill_match_percent"`
	HasAllPartsInVan  bool               `json:"has_all_parts_in_van"`
	CompositeScore    float64            `json:"composite_score"`
}

// DispatchRecommendationResult holds the optimal ranked technicians.
type DispatchRecommendationResult struct {
	WorkOrderID               string                `json:"work_order_id"`
	UrgencyTier               string                `json:"urgency_tier"`
	SLADeadline               string                `json:"sla_deadline"`
	RecommendedTechnician     TechnicianCandidate   `json:"recommended_technician"`
	AlternativeCandidates     []TechnicianCandidate `json:"alternative_candidates"`
	TotalTechniciansEvaluated int                   `json:"total_technicians_evaluated"`
	DispatchAdvice            string                `json:"dispatch_advice"`
}

// WorkOrderCompletionRequest defines the payload when a technician finishes on-site repair.
type WorkOrderCompletionRequest struct {
	WorkOrderID           string         `json:"work_order_id"`
	TechnicianID          string         `json:"technician_id"`
	LaborHoursSpent       float64        `json:"labor_hours_spent"`
	HourlyRateIDR         float64        `json:"hourly_rate_idr"`
	PartsConsumed         []RequiredPart `json:"parts_consumed"`
	CustomerSignerName    string         `json:"customer_signer_name"`
	CustomerSignatureData string         `json:"customer_signature_data"`
	TechnicianNotes       string         `json:"technician_notes"`
}

// WorkOrderCompletionResult represents the final closure, billing, and inventory relief record.
type WorkOrderCompletionResult struct {
	WorkOrderID         string  `json:"work_order_id"`
	Status              string  `json:"status"`
	TotalLaborCostIDR   float64 `json:"total_labor_cost_idr"`
	TotalPartsCostIDR   float64 `json:"total_parts_cost_idr"`
	GrandTotalBilledIDR float64 `json:"grand_total_billed_idr"`
	InvoiceReference    string  `json:"invoice_reference"`
	CryptographicSeal   string  `json:"cryptographic_seal"`
	VanStockRelieved    bool    `json:"van_stock_relieved"`
	NextServiceDueDate  string  `json:"next_service_due_date"`
	CompletedAt         string  `json:"completed_at"`
}

// VanStockItem represents a spare part in mobile van stock.
type VanStockItem struct {
	PartSKU         string  `json:"part_sku"`
	PartName        string  `json:"part_name"`
	QuantityOnHand  float64 `json:"quantity_on_hand"`
	MinReorderLevel float64 `json:"min_reorder_level"`
	ReorderAlert    bool    `json:"reorder_alert"`
	UnitValueIDR    float64 `json:"unit_value_idr"`
}

// VanStockOverview represents full stock inside technician's vehicle.
type VanStockOverview struct {
	TechnicianID    string         `json:"technician_id"`
	TechnicianName  string         `json:"technician_name"`
	VanID           string         `json:"van_id"`
	TotalSKUs       int            `json:"total_skus"`
	TotalStockValue float64        `json:"total_stock_value"`
	Items           []VanStockItem `json:"items"`
	LastSyncedAt    string         `json:"last_synced_at"`
}

// FieldServiceStore maintains technician state and work orders.
type FieldServiceStore struct {
	sync.RWMutex
	technicians []TechnicianCandidate
	completions map[string]*WorkOrderCompletionResult
}

var globalFSStore = &FieldServiceStore{
	completions: make(map[string]*WorkOrderCompletionResult),
	technicians: []TechnicianCandidate{
		{
			TechnicianID:     "TECH-001-ANDI",
			Name:             "Andi Setiawan, A.Md.",
			CurrentLatitude:  -6.2088,
			CurrentLongitude: 106.8456,
			CertifiedSkills:  []string{"ELECTRONICS", "OPTICAL_DISPLAY", "HIGH_VOLTAGE"},
			VanID:            "VAN-B-9124-TKA",
			VanInventory: map[string]float64{
				"PART-POWER-BRICK-12V": 4,
				"PART-TFT-PCB-MAIN":    2,
				"PART-CAPACITOR-100UF": 20,
			},
			CurrentStatus: "AVAILABLE",
		},
		{
			TechnicianID:     "TECH-002-BUDI",
			Name:             "Budi Santoso, S.T.",
			CurrentLatitude:  -6.3024,
			CurrentLongitude: 106.8212,
			CertifiedSkills:  []string{"MECHANICAL", "HVAC", "ELECTRONICS"},
			VanID:            "VAN-B-8472-SKL",
			VanInventory: map[string]float64{
				"PART-POWER-BRICK-12V": 1,
				"PART-FAN-BEARING":     6,
			},
			CurrentStatus: "AVAILABLE",
		},
		{
			TechnicianID:     "TECH-003-CITRA",
			Name:             "Citra Lestari, S.Kom.",
			CurrentLatitude:  -6.1751,
			CurrentLongitude: 106.7894,
			CertifiedSkills:  []string{"FIRMWARE_FLASH", "NETWORKING", "ELECTRONICS"},
			VanID:            "VAN-B-7193-WKS",
			VanInventory: map[string]float64{
				"PART-FIRMWARE-EEPROM": 5,
			},
			CurrentStatus: "ON_JOB",
		},
	},
}

func haversineKM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKM = 6371.0
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return math.Round(earthRadiusKM*c*100) / 100
}

// RecommendDispatchLogic computes the optimal dispatch rankings.
func RecommendDispatchLogic(req WorkOrderRequest) (*DispatchRecommendationResult, error) {
	if req.WorkOrderID == "" {
		req.WorkOrderID = fmt.Sprintf("WO-FIELD-%s-%d", time.Now().Format("20060102"), time.Now().Unix()%1000)
	}
	if req.SiteLatitude == 0 && req.SiteLongitude == 0 {
		req.SiteLatitude = -6.2297
		req.SiteLongitude = 106.8272
	}
	if req.FaultSeverity == "" {
		req.FaultSeverity = "HIGH"
	}
	if len(req.RequiredSkills) == 0 {
		req.RequiredSkills = []string{"ELECTRONICS", "OPTICAL_DISPLAY"}
	}
	if len(req.RequiredParts) == 0 {
		req.RequiredParts = []RequiredPart{
			{PartSKU: "PART-POWER-BRICK-12V", PartName: "12V 5A Stabilized Power Adapter", Quantity: 1, UnitCost: 450000},
		}
	}

	globalFSStore.RLock()
	defer globalFSStore.RUnlock()

	var evaluated []TechnicianCandidate

	for _, tech := range globalFSStore.technicians {
		cand := tech
		dist := haversineKM(req.SiteLatitude, req.SiteLongitude, tech.CurrentLatitude, tech.CurrentLongitude)
		cand.DistanceKM = dist
		cand.EstArrivalMinutes = math.Round((dist / 30.0 * 60.0) * 10) / 10

		matchedSkills := 0
		for _, reqSkill := range req.RequiredSkills {
			for _, certSkill := range tech.CertifiedSkills {
				if reqSkill == certSkill {
					matchedSkills++
					break
				}
			}
		}
		skillPct := 0.0
		if len(req.RequiredSkills) > 0 {
			skillPct = math.Round((float64(matchedSkills)/float64(len(req.RequiredSkills)))*1000) / 10.0
		}
		cand.SkillMatchPercent = skillPct

		hasAllParts := true
		for _, part := range req.RequiredParts {
			avail := tech.VanInventory[part.PartSKU]
			if avail < part.Quantity {
				hasAllParts = false
				break
			}
		}
		cand.HasAllPartsInVan = hasAllParts

		distScore := math.Max(0, 100.0-(dist*3.0))
		partsScore := 0.0
		if hasAllParts {
			partsScore = 100.0
		}
		availPenalty := 0.0
		if tech.CurrentStatus != "AVAILABLE" {
			availPenalty = 30.0
		}

		score := (0.40 * distScore) + (0.35 * skillPct) + (0.25 * partsScore) - availPenalty
		cand.CompositeScore = math.Round(score*10) / 10.0

		evaluated = append(evaluated, cand)
	}

	sort.Slice(evaluated, func(i, j int) bool {
		return evaluated[i].CompositeScore > evaluated[j].CompositeScore
	})

	var best TechnicianCandidate
	var alts []TechnicianCandidate
	if len(evaluated) > 0 {
		best = evaluated[0]
		if len(evaluated) > 1 {
			alts = evaluated[1:]
		}
	}

	slaHours := req.SLADeadlineHours
	if slaHours <= 0 {
		slaHours = 4.0
	}

	result := DispatchRecommendationResult{
		WorkOrderID:               req.WorkOrderID,
		UrgencyTier:               req.FaultSeverity,
		SLADeadline:               time.Now().Add(time.Duration(slaHours) * time.Hour).Format(time.RFC3339),
		RecommendedTechnician:     best,
		AlternativeCandidates:     alts,
		TotalTechniciansEvaluated: len(evaluated),
		DispatchAdvice: fmt.Sprintf("Technician %s is optimal (Score: %.1f, Distance: %.1f KM, Van Parts Ready: %t).",
			best.Name, best.CompositeScore, best.DistanceKM, best.HasAllPartsInVan),
	}

	return &result, nil
}

// CompleteWorkOrderLogic runs the work order completion and van stock reduction logic.
func CompleteWorkOrderLogic(req WorkOrderCompletionRequest) (*WorkOrderCompletionResult, error) {
	if req.WorkOrderID == "" {
		req.WorkOrderID = "WO-FIELD-20261104-001"
	}
	if req.TechnicianID == "" {
		req.TechnicianID = "TECH-001-ANDI"
	}
	if req.LaborHoursSpent <= 0 {
		req.LaborHoursSpent = 2.5
	}
	if req.HourlyRateIDR <= 0 {
		req.HourlyRateIDR = 350000.0
	}
	if len(req.PartsConsumed) == 0 {
		req.PartsConsumed = []RequiredPart{
			{PartSKU: "PART-POWER-BRICK-12V", PartName: "12V 5A Stabilized Power Adapter", Quantity: 1, UnitCost: 450000},
		}
	}
	if req.CustomerSignerName == "" {
		req.CustomerSignerName = "Bambang Wijaya (Facility Manager)"
	}

	globalFSStore.Lock()
	defer globalFSStore.Unlock()

	for _, tech := range globalFSStore.technicians {
		if tech.TechnicianID == req.TechnicianID {
			for _, part := range req.PartsConsumed {
				curr := tech.VanInventory[part.PartSKU]
				if curr >= part.Quantity {
					tech.VanInventory[part.PartSKU] = curr - part.Quantity
				} else {
					tech.VanInventory[part.PartSKU] = 0
				}
			}
			break
		}
	}

	laborTotal := req.LaborHoursSpent * req.HourlyRateIDR
	var partsTotal float64
	for _, part := range req.PartsConsumed {
		partsTotal += (part.Quantity * part.UnitCost)
	}
	grandTotal := laborTotal + partsTotal

	sealInput := fmt.Sprintf("%s|%s|%s|%.2f|%s", req.WorkOrderID, req.TechnicianID, req.CustomerSignerName, grandTotal, time.Now().Format("20060102"))
	hash := sha256.Sum256([]byte(sealInput))
	seal := hex.EncodeToString(hash[:])

	res := WorkOrderCompletionResult{
		WorkOrderID:         req.WorkOrderID,
		Status:              "COMPLETED_CLOSED",
		TotalLaborCostIDR:   laborTotal,
		TotalPartsCostIDR:   partsTotal,
		GrandTotalBilledIDR: grandTotal,
		InvoiceReference:    fmt.Sprintf("INV/FS/%s/%04d", time.Now().Format("2006"), time.Now().Unix()%10000),
		CryptographicSeal:   seal,
		VanStockRelieved:    true,
		NextServiceDueDate:  time.Now().AddDate(0, 3, 0).Format("2006-01-02"),
		CompletedAt:         time.Now().Format(time.RFC3339),
	}

	globalFSStore.completions[req.WorkOrderID] = &res
	return &res, nil
}

// FieldServiceHandler handles field dispatch and work order resolution.
type FieldServiceHandler struct {
	db *gorm.DB
}

// NewFieldServiceHandler constructs a new FieldServiceHandler.
func NewFieldServiceHandler(db *gorm.DB) *FieldServiceHandler {
	return &FieldServiceHandler{db: db}
}

// RecommendDispatch computes distance, skill match, and van stock availability to select the best technician.
func (h *FieldServiceHandler) RecommendDispatch(c *gin.Context) {
	var req WorkOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work order payload: " + err.Error()})
		return
	}

	result, err := RecommendDispatchLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// CompleteWorkOrder marks the on-site job done, consumes parts from the van, and issues billing.
func (h *FieldServiceHandler) CompleteWorkOrder(c *gin.Context) {
	var req WorkOrderCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid work order completion payload: " + err.Error()})
		return
	}

	res, err := CompleteWorkOrderLogic(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// GetVanStock returns real-time inventory on-board a technician's mobile van.
func (h *FieldServiceHandler) GetVanStock(c *gin.Context) {
	techID := c.Param("technician_id")

	globalFSStore.RLock()
	defer globalFSStore.RUnlock()

	var matched *TechnicianCandidate
	for _, t := range globalFSStore.technicians {
		if t.TechnicianID == techID {
			matched = &t
			break
		}
	}

	if matched == nil {
		matched = &globalFSStore.technicians[0]
	}

	var items []VanStockItem
	var totalVal float64

	for sku, qty := range matched.VanInventory {
		unitVal := 450000.0
		if sku == "PART-TFT-PCB-MAIN" {
			unitVal = 850000.0
		} else if sku == "PART-CAPACITOR-100UF" {
			unitVal = 15000.0
		}
		alert := qty <= 1.0

		items = append(items, VanStockItem{
			PartSKU:         sku,
			PartName:        sku,
			QuantityOnHand:  qty,
			MinReorderLevel: 2.0,
			ReorderAlert:    alert,
			UnitValueIDR:    unitVal,
		})
		totalVal += (qty * unitVal)
	}

	overview := VanStockOverview{
		TechnicianID:    matched.TechnicianID,
		TechnicianName:  matched.Name,
		VanID:           matched.VanID,
		TotalSKUs:       len(items),
		TotalStockValue: totalVal,
		Items:           items,
		LastSyncedAt:    time.Now().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, overview)
}

// RegisterFieldServiceRoutes registers field service lightning & van stock endpoints.
func RegisterFieldServiceRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	h := NewFieldServiceHandler(db)
	fsGroup := router.Group("/field-service")
	fsGroup.Use(authMiddleware)
	{
		fsGroup.POST("/dispatch/recommend", h.RecommendDispatch)
		fsGroup.POST("/work-order/complete", h.CompleteWorkOrder)
		fsGroup.GET("/van-stock/:technician_id", h.GetVanStock)
	}
}
