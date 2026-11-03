package pos

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	svc     *Service
	handler *Handler
}

func NewModule(db *gorm.DB) *Module {
	svc := NewService(db)
	handler := NewHandler(svc)
	return &Module{
		db:      db,
		svc:     svc,
		handler: handler,
	}
}

func (m *Module) Name() string {
	return "pos"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&PosConfig{},
		&PosSession{},
		&PosOrder{},
		&PosOrderLine{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/pos")
	group.Use(authMiddleware)
	{
		group.GET("/configs", m.handler.ListConfigs)
		group.GET("/sessions", m.handler.ListSessions)
		group.POST("/sessions/open", m.handler.OpenSession)
		group.POST("/sessions/:id/close", m.handler.CloseSession)
		group.GET("/orders", m.handler.ListOrders)
		group.POST("/orders", m.handler.CreateOrder)
		group.POST("/sync-offline-batch", m.handler.SyncOfflineBatch)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&PosConfig{}).Count(&count)
	if count == 0 {
		config := PosConfig{
			Name:          "Main Cashier Terminal (Shop Front)",
			ReceiptHeader: "PT Golang ERP Nusantara\nJl. Sudirman 52, Jakarta",
			ReceiptFooter: "Terima Kasih Atas Kunjungan Anda!\nBarang yang sudah dibeli tidak dapat ditukar.",
		}
		config.CompanyID = 1
		db.Create(&config)

		// Seed a sample opened session
		session := PosSession{
			Name:                     "POS/SESS/00001",
			ConfigID:                 config.ID,
			UserID:                   1, // Admin
			State:                    "opened",
			StartAt:                  "2026-10-04 08:00:00",
			CashRegisterBalanceStart: 1000000, // 1 Million initial cash float
			TotalOrders:              1,
			TotalRevenue:             1450000,
		}
		session.CompanyID = 1
		db.Create(&session)

		// Seed a sample POS Order
		order := PosOrder{
			Name:          "POS/2026/00001",
			SessionID:     session.ID,
			DateOrder:     "2026-10-04 09:15:00",
			AmountTotal:   1450000,
			AmountPaid:    1500000,
			AmountChange:  50000,
			PaymentMethod: "cash",
			State:         "paid",
			Lines: []PosOrderLine{
				{
					ProductID:     2, // Wireless Ergonomic Mouse
					ProductName:   "Wireless Ergonomic Mouse",
					Qty:           1,
					PriceUnit:     1450000,
					PriceSubtotal: 1450000,
				},
			},
			Note: "Walk-in retail customer",
		}
		order.CompanyID = 1
		for i := range order.Lines {
			order.Lines[i].CompanyID = 1
		}
		db.Create(&order)

		log.Println("[POS] Seeded default Point of Sale Terminal, Session & Sample Order")
	}
	return nil
}
