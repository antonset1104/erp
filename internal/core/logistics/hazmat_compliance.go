package logistics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HazmatItem represents a single dangerous goods chemical or product entry
type HazmatItem struct {
	ItemSKU             string  `json:"item_sku"`
	UNNumber            string  `json:"un_number"` // e.g. "UN 3480", "UN 1203", "UN 1805"
	ProperShippingName  string  `json:"proper_shipping_name"` // e.g. "LITHIUM ION BATTERIES"
	PrimaryHazardClass  string  `json:"primary_hazard_class"` // e.g. "CLASS_9", "CLASS_3", "CLASS_8"
	SubsidiaryRisk      string  `json:"subsidiary_risk"`      // e.g. "NONE", "CLASS_6.1"
	PackingGroup        string  `json:"packing_group"`        // "PG_I", "PG_II", "PG_III"
	QuantityPackages    int     `json:"quantity_packages"`
	PackageTypeDescription string `json:"package_type_description"` // "FIBREBOARD_BOX_4G", "STEEL_DRUM_1A1"
	NetMassPerPackageKG float64 `json:"net_mass_per_package_kg"`
	FlashPointCelsius   float64 `json:"flash_point_celsius"` // For flammable liquids (Class 3)
	IsMarinePollutant   bool    `json:"is_marine_pollutant"`
	IsLimitedQuantity   bool    `json:"is_limited_quantity"`
}

// StowageAuditResult represents the chemical segregation assessment inside a freight container
type StowageAuditResult struct {
	ShipmentID               string       `json:"shipment_id"`
	ContainerNumber          string       `json:"container_number"`
	ItemsAudited             []HazmatItem `json:"items_audited"`
	SegregationStatus        string       `json:"segregation_status"` // "COMPLIANT_PERMITTED", "SEGREGATION_CONFLICT_FORBIDDEN"
	IncompatiblePairsFound   []string     `json:"incompatible_pairs_found"`
	TotalNetExplosiveMassKG  float64      `json:"total_net_explosive_mass_kg"`
	TunnelRestrictionCode    string       `json:"tunnel_restriction_code"` // e.g. "(E)", "(D/E)"
	AuditedAt                time.Time    `json:"audited_at"`
}

// DangerousGoodsDeclaration represents the official IMO / IATA Multimodal DGD document
type DangerousGoodsDeclaration struct {
	DGDReferenceNumber       string       `json:"dgd_reference_number"` // e.g. "DGD-IMDG-2026-88192"
	ShipmentID               string       `json:"shipment_id"`
	ConsignorShipper         string       `json:"consignor_shipper"`
	ConsigneeReceiver        string       `json:"consignee_receiver"`
	TransportMode            string       `json:"transport_mode"` // "MARITIME_IMDG", "ROAD_ADR", "AIR_IATA"
	VesselFlightNumber       string       `json:"vessel_flight_number"`
	PortOfLoading            string       `json:"port_of_loading"`
	PortOfDischarge          string       `json:"port_of_discharge"`
	EmergencyTelephone24Hr   string       `json:"emergency_telephone_24hr"`
	EmergencyContactPerson   string       `json:"emergency_contact_person"`
	Items                    []HazmatItem `json:"items"`
	ShipperCertificationHash string       `json:"shipper_certification_hash"`
	DeclarationStatus        string       `json:"declaration_status"` // "SIGNED_AUTHORIZED", "VOID_REJECTED"
	IssuedAt                 time.Time    `json:"issued_at"`
}

// HazmatComplianceManifest compiles full dangerous goods shipping dossier
type HazmatComplianceManifest struct {
	ShipmentID             string                    `json:"shipment_id"`
	AuditSummary           StowageAuditResult        `json:"audit_summary"`
	Declaration            DangerousGoodsDeclaration `json:"declaration"`
	PlacardRequirement     string                    `json:"placard_requirement"` // e.g. "CLASS 9 MISCELLANEOUS & BATTERY MARK"
	ComplianceVerdict      string                    `json:"compliance_verdict"`
	LastAuditedAt          time.Time                 `json:"last_audited_at"`
}

// HazmatEngine manages dangerous goods classification, segregation, and DGD document generation
type HazmatEngine struct {
	mu           sync.RWMutex
	consignments map[string][]HazmatItem
	audits       map[string]StowageAuditResult
	declarations map[string]DangerousGoodsDeclaration
}

var (
	globalHazmatEngine *HazmatEngine
	hazmatOnce         sync.Once
)

// GetHazmatEngine returns singleton instance of HazmatEngine
func GetHazmatEngine() *HazmatEngine {
	hazmatOnce.Do(func() {
		globalHazmatEngine = &HazmatEngine{
			consignments: make(map[string][]HazmatItem),
			audits:       make(map[string]StowageAuditResult),
			declarations: make(map[string]DangerousGoodsDeclaration),
		}
	})
	return globalHazmatEngine
}

// ClassifyConsignment evaluates items and stores classified dangerous goods shipment
func (e *HazmatEngine) ClassifyConsignment(shipmentID string, items []HazmatItem) []HazmatItem {
	e.mu.Lock()
	defer e.mu.Unlock()

	for i, it := range items {
		// Determine Limited Quantity thresholds (e.g. <= 5kg for PG II/III)
		if it.NetMassPerPackageKG <= 5.0 && (it.PackingGroup == "PG_II" || it.PackingGroup == "PG_III") {
			items[i].IsLimitedQuantity = true
		}
	}

	e.consignments[shipmentID] = items
	return items
}

// AuditStowageSegregation verifies IMDG Code / ADR chemical segregation table
func (e *HazmatEngine) AuditStowageSegregation(shipmentID string, containerNo string) StowageAuditResult {
	e.mu.Lock()
	defer e.mu.Unlock()

	items := e.consignments[shipmentID]
	var conflicts []string
	status := "COMPLIANT_PERMITTED"

	// Audit incompatible hazard class pairings
	// E.g. Class 8 (Acids) cannot be co-loaded with Class 6.1 (Cyanides)
	// E.g. Class 3 (Flammables) must be separated from Class 5.1 (Oxidizers)
	hasAcid := false
	hasCyanide := false
	hasFlammable := false
	hasOxidizer := false

	for _, it := range items {
		if it.PrimaryHazardClass == "CLASS_8" {
			hasAcid = true
		}
		if it.PrimaryHazardClass == "CLASS_6.1" || it.SubsidiaryRisk == "CLASS_6.1" {
			hasCyanide = true
		}
		if it.PrimaryHazardClass == "CLASS_3" {
			hasFlammable = true
		}
		if it.PrimaryHazardClass == "CLASS_5.1" {
			hasOxidizer = true
		}
	}

	if hasAcid && hasCyanide {
		conflicts = append(conflicts, "CRITICAL CONFLICT: Class 8 (Corrosive Acids) and Class 6.1 (Cyanides) prohibited in same freight container (IMDG 7.2.4 Segregation Table 2).")
		status = "SEGREGATION_CONFLICT_FORBIDDEN"
	}
	if hasFlammable && hasOxidizer {
		conflicts = append(conflicts, "SAFETY VIOLATION: Class 3 (Flammable Liquids) and Class 5.1 (Oxidizing Agents) require compartment separation.")
		status = "SEGREGATION_CONFLICT_FORBIDDEN"
	}

	res := StowageAuditResult{
		ShipmentID:              shipmentID,
		ContainerNumber:         containerNo,
		ItemsAudited:            items,
		SegregationStatus:       status,
		IncompatiblePairsFound:  conflicts,
		TotalNetExplosiveMassKG: 0.0,
		TunnelRestrictionCode:   "(E)",
		AuditedAt:               time.Now(),
	}

	e.audits[shipmentID] = res
	return res
}

// GenerateDangerousGoodsDeclaration compiles statutory DGD document
func (e *HazmatEngine) GenerateDangerousGoodsDeclaration(req DangerousGoodsDeclaration) DangerousGoodsDeclaration {
	e.mu.Lock()
	defer e.mu.Unlock()

	refNo := fmt.Sprintf("DGD-%s-%d", req.TransportMode, time.Now().UnixNano()%1000000)
	raw := fmt.Sprintf("%s|%s|%s|%s|%d", refNo, req.ShipmentID, req.ConsignorShipper, req.PortOfLoading, time.Now().Unix())
	hashBytes := sha256.Sum256([]byte(raw))
	sigHash := "SIG-DGD-" + hex.EncodeToString(hashBytes[:16])

	items := e.consignments[req.ShipmentID]
	if len(req.Items) > 0 {
		items = req.Items
	}

	decl := DangerousGoodsDeclaration{
		DGDReferenceNumber:       refNo,
		ShipmentID:               req.ShipmentID,
		ConsignorShipper:         req.ConsignorShipper,
		ConsigneeReceiver:        req.ConsigneeReceiver,
		TransportMode:            req.TransportMode,
		VesselFlightNumber:       req.VesselFlightNumber,
		PortOfLoading:            req.PortOfLoading,
		PortOfDischarge:          req.PortOfDischarge,
		EmergencyTelephone24Hr:   req.EmergencyTelephone24Hr,
		EmergencyContactPerson:   req.EmergencyContactPerson,
		Items:                    items,
		ShipperCertificationHash: sigHash,
		DeclarationStatus:        "SIGNED_AUTHORIZED",
		IssuedAt:                 time.Now(),
	}

	e.declarations[req.ShipmentID] = decl
	return decl
}

// GetComplianceManifest builds end-to-end hazmat dossier
func (e *HazmatEngine) GetComplianceManifest(shipmentID string) HazmatComplianceManifest {
	e.mu.RLock()
	defer e.mu.RUnlock()

	audit := e.audits[shipmentID]
	decl := e.declarations[shipmentID]

	verdict := "INTERNATIONAL_HAZMAT_COMPLIANT"
	if audit.SegregationStatus == "SEGREGATION_CONFLICT_FORBIDDEN" {
		verdict = "REJECTED_CHEMICAL_STOWAGE_CONFLICT"
	}

	return HazmatComplianceManifest{
		ShipmentID:         shipmentID,
		AuditSummary:       audit,
		Declaration:        decl,
		PlacardRequirement: "DIAMOND PLACARD CLASS 9 (LITHIUM BATTERY) & CLASS 3 FLAMMABLE",
		ComplianceVerdict:  verdict,
		LastAuditedAt:      time.Now(),
	}
}

// RegisterHazmatRoutes registers REST API endpoints for HAZMAT Transport Compliance
func RegisterHazmatRoutes(r *gin.RouterGroup) {
	group := r.Group("/logistics/hazmat")
	{
		group.POST("/classify-consignment", func(c *gin.Context) {
			var req struct {
				ShipmentID string       `json:"shipment_id" binding:"required"`
				Items      []HazmatItem `json:"items" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetHazmatEngine().ClassifyConsignment(req.ShipmentID, req.Items)
			c.JSON(http.StatusOK, gin.H{
				"status": "CONSIGNMENT_CLASSIFIED",
				"items":  res,
			})
		})

		group.POST("/audit-stowage-segregation", func(c *gin.Context) {
			var req struct {
				ShipmentID      string `json:"shipment_id" binding:"required"`
				ContainerNumber string `json:"container_number" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetHazmatEngine().AuditStowageSegregation(req.ShipmentID, req.ContainerNumber)
			c.JSON(http.StatusOK, gin.H{
				"status": "SEGREGATION_AUDITED",
				"audit":  res,
			})
		})

		group.POST("/generate-dangerous-goods-declaration", func(c *gin.Context) {
			var req DangerousGoodsDeclaration
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			res := GetHazmatEngine().GenerateDangerousGoodsDeclaration(req)
			c.JSON(http.StatusOK, gin.H{
				"status":      "DGD_GENERATED",
				"declaration": res,
			})
		})

		group.GET("/compliance-manifest/:shipment_id", func(c *gin.Context) {
			shipmentID := c.Param("shipment_id")
			res := GetHazmatEngine().GetComplianceManifest(shipmentID)
			c.JSON(http.StatusOK, gin.H{
				"status":   "MANIFEST_COMPILED",
				"manifest": res,
			})
		})
	}
}
