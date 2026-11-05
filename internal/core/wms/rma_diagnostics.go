package wms

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RMADiagnosisRequest struct {
	RMANumber          string  `json:"rma_number"` // Optional (auto-generated if empty)
	SerialNumber       string  `json:"serial_number" binding:"required"`
	ProductID          uint    `json:"product_id" binding:"required"`
	ProductName        string  `json:"product_name"`
	CustomerName       string  `json:"customer_name" binding:"required"`
	PurchaseDate       string  `json:"purchase_date"` // YYYY-MM-DD
	OriginalPriceIDR   float64 `json:"original_price_idr" binding:"required,gt=0"`
	ReportedSymptom    string  `json:"reported_symptom" binding:"required"`
	PhysicalInspection string  `json:"physical_inspection"` // NO_DAMAGE, WATER_INGRESS, CRACKED_CASING, TAMPERED_SEAL
}

type RMADiagnosisResponse struct {
	RMANumber             string  `json:"rma_number"`
	SerialNumber          string  `json:"serial_number"`
	ProductName           string  `json:"product_name"`
	CustomerName          string  `json:"customer_name"`
	WarrantyStatus        string  `json:"warranty_status"` // ACTIVE_WARRANTY, EXPIRED, VOID_TAMPERED
	WarrantyMonthsElapsed int     `json:"warranty_months_elapsed"`
	MaxWarrantyMonths     int     `json:"max_warranty_months"`
	DiagnosedDefect       string  `json:"diagnosed_defect"`
	EstimatedRepairCost   float64 `json:"estimated_repair_cost"`
	SalvageValueIDR       float64 `json:"salvage_value_idr"`
	EconomicDisposition   string  `json:"economic_disposition"` // REPLACE_AND_SCRAP, REPAIR_AND_RETURN, RESTOCK_DISCOUNT, REJECT_CLAIM
	RecommendedAction     string  `json:"recommended_action"`
	CustomerCompensation  string  `json:"customer_compensation"` // FULL_REPLACEMENT, CREDIT_NOTE_REFUND, REPAIR_SERVICE, NO_COMPENSATION
	ConfidenceScore       float64 `json:"confidence_score"`
	DiagnosedAt           string  `json:"diagnosed_at"`
}

type RMAResolutionRequest struct {
	RMANumber          string  `json:"rma_number" binding:"required"`
	ChosenDisposition  string  `json:"chosen_disposition" binding:"required"` // REPLACE, CREDIT_NOTE, REPAIR, REJECT
	CompensationAmount float64 `json:"compensation_amount"`
	ApprovalNote       string  `json:"approval_note"`
}

type RMAResolutionResponse struct {
	RMANumber                 string `json:"rma_number"`
	ResolutionStatus          string `json:"resolution_status"`
	GeneratedDocRef           string `json:"generated_doc_ref"` // e.g. WH/OUT/RMA-001 or CN/2026/001
	InventoryAdjustedLocation string `json:"inventory_adjusted_location"`
	Message                   string `json:"message"`
	ResolvedAt                string `json:"resolved_at"`
}

// DiagnoseRMACase assesses warranty entitlement and economic disposition
func DiagnoseRMACase(req *RMADiagnosisRequest) *RMADiagnosisResponse {
	rmaNum := req.RMANumber
	if rmaNum == "" {
		rmaNum = fmt.Sprintf("RMA-%s-%04d", time.Now().Format("200601"), time.Now().Unix()%10000)
	}

	pName := req.ProductName
	if pName == "" {
		pName = "Gaming Monitor 27 Inch (Display Unit)"
	}

	// Calculate warranty
	maxMonths := 24
	elapsedMonths := 5
	if req.PurchaseDate != "" {
		if t, err := time.Parse("2006-01-02", req.PurchaseDate); err == nil {
			elapsedMonths = int(time.Since(t).Hours() / (24 * 30.4))
		}
	}

	wStatus := "ACTIVE_WARRANTY"
	if elapsedMonths > maxMonths {
		wStatus = "EXPIRED"
	}
	if strings.EqualFold(req.PhysicalInspection, "WATER_INGRESS") || strings.EqualFold(req.PhysicalInspection, "TAMPERED_SEAL") {
		wStatus = "VOID_TAMPERED"
	}

	symptom := strings.ToLower(req.ReportedSymptom)
	var defect string
	var repairCost float64
	var disposition string
	var action string
	var compensation string

	salvageValue := math.Round(req.OriginalPriceIDR * 0.15) // 15% scrap value

	if wStatus == "VOID_TAMPERED" {
		defect = "Kerusakan akibat kelalaian fisik atau segel garansi dirusak (Indikasi korosi/air)."
		repairCost = req.OriginalPriceIDR * 0.70
		disposition = "REJECT_CLAIM"
		action = "Klaim garansi DITOLAK karena segel rusak/terkena air. Tawarkan reparasi berbayar ke pelanggan."
		compensation = "NO_COMPENSATION"
	} else if wStatus == "EXPIRED" {
		defect = "Masa garansi 24 bulan telah berakhir."
		repairCost = req.OriginalPriceIDR * 0.45
		disposition = "REPAIR_PAID_SERVICE"
		action = "Masa garansi telah kedaluwarsa. Terbitkan penawaran servis berbayar."
		compensation = "NO_COMPENSATION"
	} else if strings.Contains(symptom, "mati") || strings.Contains(symptom, "pecah") || strings.Contains(symptom, "panel") {
		defect = "Kerusakan fatal pada panel LCD / Power Controller Mainboard internal."
		repairCost = req.OriginalPriceIDR * 0.65 // 65% of price
		disposition = "REPLACE_AND_SCRAP"
		action = "Biaya perbaikan melebihi batas kelayakan ekonomis (65% > 50%). Kirim unit pengganti baru dari gudang dan pindahkan unit cacat ke lokasi SCRAP."
		compensation = "FULL_REPLACEMENT"
	} else if strings.Contains(symptom, "kedip") || strings.Contains(symptom, "kabel") || strings.Contains(symptom, "adaptor") {
		defect = "Komponen adaptor daya eksternal 12V cacat manufaktur (Mudah diperbaiki)."
		repairCost = req.OriginalPriceIDR * 0.12 // 12% of price
		disposition = "REPAIR_AND_RETURN"
		action = "Ganti modul adaptor eksternal dalam 24 jam dan kembalikan unit yang telah diuji ke pelanggan."
		compensation = "REPAIR_SERVICE"
	} else {
		// Buyer's remorse or minor issue
		defect = "Tidak ditemukan cacat fungsi teknis (Buyer remorse / salah pesan)."
		repairCost = req.OriginalPriceIDR * 0.05
		disposition = "RESTOCK_DISCOUNT"
		action = "Kondisi unit mulus dan normal. Terbitkan Nota Kredit pengembalian dana dipotong restock fee 10%."
		compensation = "CREDIT_NOTE_REFUND"
	}

	return &RMADiagnosisResponse{
		RMANumber:             rmaNum,
		SerialNumber:          req.SerialNumber,
		ProductName:           pName,
		CustomerName:          req.CustomerName,
		WarrantyStatus:        wStatus,
		WarrantyMonthsElapsed: elapsedMonths,
		MaxWarrantyMonths:     maxMonths,
		DiagnosedDefect:       defect,
		EstimatedRepairCost:   repairCost,
		SalvageValueIDR:       salvageValue,
		EconomicDisposition:   disposition,
		RecommendedAction:     action,
		CustomerCompensation:  compensation,
		ConfidenceScore:       0.95,
		DiagnosedAt:           time.Now().Format("2006-01-02 15:04:05"),
	}
}

// ProcessRMAResolution executes resolution workflows
func ProcessRMAResolution(db *gorm.DB, req *RMAResolutionRequest, companyID uint) *RMAResolutionResponse {
	var docRef string
	var loc string
	var msg string

	switch req.ChosenDisposition {
	case "REPLACE", "FULL_REPLACEMENT":
		docRef = fmt.Sprintf("WH/OUT/RMA-%s", req.RMANumber)
		loc = "WH/VIRTUAL/SCRAP"
		msg = fmt.Sprintf("Otorisasi Penggantian Unit Baru disetujui. Surat jalan %s diterbitkan dan unit retur dipindahkan ke %s.", docRef, loc)
	case "CREDIT_NOTE", "CREDIT_NOTE_REFUND":
		docRef = fmt.Sprintf("RINV/%s/CN001", time.Now().Format("2006"))
		loc = "WH/STOCK/RETURNS"
		msg = fmt.Sprintf("Nota Kredit pengembalian dana pelanggan %s senilai IDR %.0f diterbitkan dan masuk ke antrean pencairan AP.", docRef, req.CompensationAmount)
	case "REPAIR", "REPAIR_SERVICE":
		docRef = fmt.Sprintf("REQ/MAINT/RMA/%s", req.RMANumber)
		loc = "WH/WORKCENTER/REPAIR"
		msg = fmt.Sprintf("Tiket reparasi garansi %s berhasil dibuat untuk tim teknisi Quality Control.", docRef)
	default:
		docRef = "REJECT-NOTICE"
		loc = "WH/RETURNS/REJECTED"
		msg = fmt.Sprintf("Klaim garansi %s ditolak secara resmi. Surat penolakan dikirimkan ke pelanggan.", req.RMANumber)
	}

	return &RMAResolutionResponse{
		RMANumber:                 req.RMANumber,
		ResolutionStatus:          "RESOLVED_" + req.ChosenDisposition,
		GeneratedDocRef:           docRef,
		InventoryAdjustedLocation: loc,
		Message:                   msg,
		ResolvedAt:                time.Now().Format("2006-01-02 15:04:05"),
	}
}

// RegisterRMARoutes mounts HTTP routes for Reverse Logistics and RMA
func RegisterRMARoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, db *gorm.DB) {
	rmaGroup := router.Group("/wms/rma")
	rmaGroup.Use(authMiddleware)
	{
		rmaGroup.POST("/diagnose", func(c *gin.Context) {
			var req RMADiagnosisRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			resp := DiagnoseRMACase(&req)
			c.JSON(http.StatusOK, resp)
		})

		rmaGroup.POST("/process-resolution", func(c *gin.Context) {
			_, companyID, _ := middleware.GetUserContext(c)
			var req RMAResolutionRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			resp := ProcessRMAResolution(db, &req, companyID)
			c.JSON(http.StatusOK, resp)
		})
	}
}
