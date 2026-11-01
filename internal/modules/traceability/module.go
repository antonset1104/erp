package traceability

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/barcode"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/product"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db       *gorm.DB
	service  *Service
	handler  *Handler
	aiClient *ai.Client
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(db, repo, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		db:       db,
		service:  service,
		handler:  handler,
		aiClient: aiClient,
	}
}

func (m *Module) Name() string {
	return "traceability"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&LotTraceRecord{},
		&ProductRecall{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/traceability")
	group.Use(authMiddleware)
	{
		group.GET("/stats", m.handler.GetStats)
		group.GET("/traces", m.handler.ListTraces)
		group.POST("/record", m.handler.RecordMovement)
		group.GET("/tree/:lot_id", m.handler.GetTraceabilityTree)
		group.POST("/fefo", m.handler.AllocateFEFO)
		group.GET("/expiry-audit", m.handler.AuditExpiryRisk)
		group.POST("/recalls", m.handler.InitiateRecall)
		group.GET("/recalls", m.handler.ListRecalls)
		group.GET("/recalls/:id", m.handler.GetRecall)
		group.POST("/recalls/:id/complete", m.handler.CompleteRecall)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	if err := m.AutoMigrate(db); err != nil {
		return err
	}

	var traceCount int64
	db.Model(&LotTraceRecord{}).Count(&traceCount)
	if traceCount > 0 {
		return nil
	}

	// Find products to associate with demo lots
	var rawProd, finProd product.Product
	errRaw := db.Where("name LIKE ?", "%Raw%").First(&rawProd).Error
	if errRaw != nil {
		errRaw = db.First(&rawProd).Error
	}

	errFin := db.Where("id != ?", rawProd.ID).First(&finProd).Error
	if errFin != nil {
		finProd = rawProd
	}

	if rawProd.ID == 0 {
		return nil // No products in DB yet
	}

	now := time.Now()
	todayStr := now.Format("2006-01-02 15:04:05")
	nearExpDate := now.AddDate(0, 0, 18).Format("2006-01-02")
	expiredDate := now.AddDate(0, -1, -5).Format("2006-01-02")
	freshDate := now.AddDate(1, 6, 0).Format("2006-01-02")

	// 1. Seed Raw Material Lot (Active Pharma / Chemical Precursor)
	rawLot := barcode.StockProductionLot{
		Name:           "LOT-RAW-PRECURSOR-2026",
		ProductID:      rawProd.ID,
		Barcode:        "LOT-RAW-PRECURSOR-2026",
		Ref:            "PO-VEND-CHEM-991",
		ExpirationDate: freshDate,
		Qty:            100.0,
		Notes:          "Bahan baku impor kualitas farmasi tinggi dari Shanghai ChemCorp",
	}
	rawLot.CompanyID = 1
	_ = db.Create(&rawLot)

	// Record Receipt Trace for Raw Lot
	rawReceiptTrace := LotTraceRecord{
		LotID:          rawLot.ID,
		LotName:        rawLot.Name,
		ProductID:      rawProd.ID,
		ProductName:    rawProd.Name,
		EventType:      EventReceipt,
		Reference:      "WH/IN/00088",
		SourceDocument: "PO00021",
		PartnerName:    "PT Kimia Farma Trading & Distribution",
		Quantity:       150.0,
		UoM:            "kg",
		Date:           now.AddDate(0, -2, 0).Format("2006-01-02 15:04:05"),
		ExpirationDate: freshDate,
		Notes:          "Penerimaan barang dari kapal Tanjung Priok",
	}
	rawReceiptTrace.CompanyID = 1
	_ = db.Create(&rawReceiptTrace)

	// 2. Seed Finished Good Lot produced from Raw Lot
	finLot := barcode.StockProductionLot{
		Name:           "LOT-FIN-MEDICINE-2026",
		ProductID:      finProd.ID,
		Barcode:        "LOT-FIN-MEDICINE-2026",
		Ref:            "MO/2026/00010",
		ExpirationDate: freshDate,
		Qty:            5000.0,
		Notes:          "Bets produksi obat jadi steril tablet batch 10",
	}
	finLot.CompanyID = 1
	_ = db.Create(&finLot)

	// Raw lot consumed in MO
	moConsumeTrace := LotTraceRecord{
		LotID:           rawLot.ID,
		LotName:         rawLot.Name,
		ProductID:       rawProd.ID,
		ProductName:     rawProd.Name,
		EventType:       EventMOConsume,
		Reference:       "MO/2026/00010",
		SourceDocument:  "MO/2026/00010",
		ProducedLotID:   &finLot.ID,
		ProducedLotName: finLot.Name,
		Quantity:        50.0,
		UoM:             "kg",
		Date:            now.AddDate(0, -1, 0).Format("2006-01-02 15:04:05"),
		ExpirationDate:  freshDate,
		Notes:           "Konsumsi bahan baku di ruang pencampuran formulasi",
	}
	moConsumeTrace.CompanyID = 1
	_ = db.Create(&moConsumeTrace)

	// Finished lot produced in MO
	moProduceTrace := LotTraceRecord{
		LotID:           finLot.ID,
		LotName:         finLot.Name,
		ProductID:       finProd.ID,
		ProductName:     finProd.Name,
		EventType:       EventMOProduce,
		Reference:       "MO/2026/00010",
		SourceDocument:  "MO/2026/00010",
		ConsumedLotID:   &rawLot.ID,
		ConsumedLotName: rawLot.Name,
		Quantity:        8000.0,
		UoM:             "Box",
		Date:            now.AddDate(0, -1, 0).Format("2006-01-02 15:04:05"),
		ExpirationDate:  freshDate,
		Notes:           "Penyelesaian produksi dan pengemasan primer/sekunder",
	}
	moProduceTrace.CompanyID = 1
	_ = db.Create(&moProduceTrace)

	// Delivery 1 of finished lot to Hospital Siloam
	delTrace1 := LotTraceRecord{
		LotID:          finLot.ID,
		LotName:        finLot.Name,
		ProductID:      finProd.ID,
		ProductName:    finProd.Name,
		EventType:      EventDelivery,
		Reference:      "WH/OUT/00091",
		SourceDocument: "SO00045",
		PartnerName:    "RS Siloam Hospitals Kebon Jeruk",
		Quantity:       2000.0,
		UoM:            "Box",
		Date:           now.AddDate(0, 0, -14).Format("2006-01-02 15:04:05"),
		ExpirationDate: freshDate,
		Notes:          "Pengiriman ambulans farmasi termokontrol",
	}
	delTrace1.CompanyID = 1
	_ = db.Create(&delTrace1)

	// Delivery 2 of finished lot to Kimia Farma Pharmacy
	delTrace2 := LotTraceRecord{
		LotID:          finLot.ID,
		LotName:        finLot.Name,
		ProductID:      finProd.ID,
		ProductName:    finProd.Name,
		EventType:      EventDelivery,
		Reference:      "WH/OUT/00092",
		SourceDocument: "SO00046",
		PartnerName:    "Apotek Kimia Farma Matraman",
		Quantity:       1000.0,
		UoM:            "Box",
		Date:           now.AddDate(0, 0, -7).Format("2006-01-02 15:04:05"),
		ExpirationDate: freshDate,
		Notes:          "Pengiriman retail obat generik berlogo",
	}
	delTrace2.CompanyID = 1
	_ = db.Create(&delTrace2)

	// 3. Seed Near-Expiry Lot (expiring in 18 days)
	nearExpLot := barcode.StockProductionLot{
		Name:           "LOT-NEAR-EXP-AMOX-2026",
		ProductID:      finProd.ID,
		Barcode:        "LOT-NEAR-EXP-AMOX-2026",
		ExpirationDate: nearExpDate,
		Qty:            450.0,
		Notes:          "Lot mendekati tanggal kadaluwarsa (18 hari tersisa)",
	}
	nearExpLot.CompanyID = 1
	_ = db.Create(&nearExpLot)

	// 4. Seed Expired Lot (expired 35 days ago)
	expLot := barcode.StockProductionLot{
		Name:           "LOT-EXPIRED-CEFIXIME-99",
		ProductID:      finProd.ID,
		Barcode:        "LOT-EXPIRED-CEFIXIME-99",
		ExpirationDate: expiredDate,
		Qty:            120.0,
		Notes:          "Lot telah melewati masa kedaluwarsa. Menunggu instruksi pemusnahan",
	}
	expLot.CompanyID = 1
	_ = db.Create(&expLot)

	log.Printf("[Traceability] Seeded demo lot traceability genealogy & expiry audit batches at %s", todayStr)
	return nil
}
