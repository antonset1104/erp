package chatter

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

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "chatter"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&MailMessage{})
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/chatter")
	group.Use(authMiddleware)
	{
		group.GET("/:model/:res_id", m.handler.ListMessages)
		group.POST("/:model/:res_id", m.handler.PostMessage)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&MailMessage{}).Count(&count)
	if count == 0 {
		sampleLogs := []MailMessage{
			{
				ModelName:   "sale.order",
				ResID:       1,
				AuthorID:    1,
				AuthorName:  "admin@erp.local",
				Body:        "Sales Order created and quotation sent to customer.",
				MessageType: "notification",
				Tracking:    "Status: draft -> sent",
			},
			{
				ModelName:   "sale.order",
				ResID:       1,
				AuthorID:    1,
				AuthorName:  "admin@erp.local",
				Body:        "Customer confirmed quotation via email. Delivery order scheduled.",
				MessageType: "comment",
			},
			{
				ModelName:   "account.move",
				ResID:       1,
				AuthorID:    1,
				AuthorName:  "admin@erp.local",
				Body:        "Invoice validated and posted to General Ledger.",
				MessageType: "tracking",
				Tracking:    "Status: draft -> posted",
			},
		}

		for i := range sampleLogs {
			sampleLogs[i].CompanyID = 1
			db.Create(&sampleLogs[i])
		}
		log.Println("[Chatter] Seeded sample audit trail activity logs")
	}

	return nil
}
