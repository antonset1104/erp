package helpdesk

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
	return "helpdesk"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&HelpdeskTeam{}, &HelpdeskTicket{})
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/helpdesk")
	group.Use(authMiddleware)
	{
		// Tickets
		group.GET("/tickets", m.handler.ListTickets)
		group.POST("/tickets", m.handler.CreateTicket)
		group.GET("/tickets/:id", m.handler.GetTicket)
		group.PUT("/tickets/:id/stage", m.handler.UpdateTicketStage)
		group.PUT("/tickets/:id/assign", m.handler.AssignTicket)
		group.PUT("/tickets/:id/resolve", m.handler.ResolveTicket)

		// Teams
		group.GET("/teams", m.handler.ListTeams)
		group.POST("/teams", m.handler.CreateTeam)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var teamCount int64
	db.Model(&HelpdeskTeam{}).Count(&teamCount)
	if teamCount == 0 {
		userID1 := uint(1) // Admin user

		team1 := HelpdeskTeam{
			Name:     "Customer Care & Support",
			Code:     "CARE",
			Email:    "care@odoo-go.id",
			LeaderID: &userID1,
			Color:    "#3b82f6",
			Active:   true,
		}
		team1.CompanyID = 1
		team1.CreateUID = 1
		team1.WriteUID = 1
		db.Create(&team1)

		team2 := HelpdeskTeam{
			Name:     "Technical Support Tier 2",
			Code:     "TECH",
			Email:    "tech-support@odoo-go.id",
			LeaderID: &userID1,
			Color:    "#8b5cf6",
			Active:   true,
		}
		team2.CompanyID = 1
		team2.CreateUID = 1
		team2.WriteUID = 1
		db.Create(&team2)

		team3 := HelpdeskTeam{
			Name:     "Billing & Invoicing Help",
			Code:     "BILL",
			Email:    "billing@odoo-go.id",
			LeaderID: &userID1,
			Color:    "#10b981",
			Active:   true,
		}
		team3.CompanyID = 1
		team3.CreateUID = 1
		team3.WriteUID = 1
		db.Create(&team3)

		var ticketCount int64
		db.Model(&HelpdeskTicket{}).Count(&ticketCount)
		if ticketCount == 0 {
			partnerID1 := uint(1) // PT Maju Bersama
			partnerID2 := uint(2) // CV Berkah Sejahtera

			t1 := HelpdeskTicket{
				TicketNumber: "TCK/2026/00001",
				Name:         "Permintaan Lisensi Tambahan POS Kasir Cabang Bandung",
				TeamID:       &team3.ID,
				PartnerID:    &partnerID1,
				PartnerName:  "PT Maju Bersama Teknologi",
				PartnerEmail: "finance@majubersama.id",
				PartnerPhone: "0812-3456-7890",
				UserID:       &userID1,
				Priority:     "1",
				Category:     "billing",
				Stage:        "in_progress",
				Description:  "Mohon invoice penambahan 2 terminal POS Android untuk outlet baru di Mall Paris Van Java Bandung.",
			}
			t1.CompanyID = 1
			t1.CreateUID = 1
			t1.WriteUID = 1
			db.Create(&t1)

			t2 := HelpdeskTicket{
				TicketNumber: "TCK/2026/00002",
				Name:         "Kendala Sinkronisasi Stok Antar Gudang Cabang Surabaya",
				TeamID:       &team2.ID,
				PartnerID:    &partnerID2,
				PartnerName:  "CV Berkah Sejahtera",
				PartnerEmail: "ops@berkahsejahtera.co.id",
				PartnerPhone: "0813-9876-5432",
				UserID:       nil,
				Priority:     "3",
				Category:     "technical",
				Stage:        "new",
				Description:  "Ketika melakukan transfer stok WH/INT/00001 dari JKT ke SBY, kuantitas reserved sempat delay 5 menit.",
			}
			t2.CompanyID = 1
			t2.CreateUID = 1
			t2.WriteUID = 1
			db.Create(&t2)

			t3 := HelpdeskTicket{
				TicketNumber: "TCK/2026/00003",
				Name:         "Klaim Garansi & Servis Berkala Mesin Server Dell R750",
				TeamID:       &team1.ID,
				PartnerID:    &partnerID1,
				PartnerName:  "PT Maju Bersama Teknologi",
				PartnerEmail: "it@majubersama.id",
				PartnerPhone: "0812-3456-7890",
				UserID:       &userID1,
				Priority:     "2",
				Category:     "warranty",
				Stage:        "solved",
				Description:  "Pengecekan kipas redundant power supply untuk server fixed asset utama.",
				Resolution:   "Tim teknisi rekanan Dell telah melakukan penggantian modul fan di data center dan sistem beroperasi normal 100%.",
				CloseDate:    "2026-10-04 14:30:00",
			}
			t3.CompanyID = 1
			t3.CreateUID = 1
			t3.WriteUID = 1
			db.Create(&t3)
		}

		log.Println("[Helpdesk] Seeded default Teams and Support Tickets")
	}
	return nil
}
