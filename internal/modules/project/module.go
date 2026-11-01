package project

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
	return "project"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&Project{}, &Task{}, &TimesheetLine{})
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Projects Group
	projGroup := api.Group("/projects")
	projGroup.Use(authMiddleware)
	{
		projGroup.GET("", m.handler.ListProjects)
		projGroup.POST("", m.handler.CreateProject)
		projGroup.GET("/:id", m.handler.GetProject)
	}

	// Tasks Group
	taskGroup := api.Group("/tasks")
	taskGroup.Use(authMiddleware)
	{
		taskGroup.GET("", m.handler.ListTasks)
		taskGroup.POST("", m.handler.CreateTask)
		taskGroup.GET("/:id", m.handler.GetTask)
		taskGroup.PUT("/:id/stage", m.handler.UpdateTaskStage)
		taskGroup.POST("/:id/timesheets", m.handler.LogTimesheet)
		taskGroup.GET("/:id/timesheets", m.handler.ListTimesheets)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&Project{}).Count(&count)
	if count == 0 {
		partnerID1 := uint(1) // PT Maju Bersama Teknologi
		partnerID2 := uint(2) // CV Berkah Sejahtera
		userID1 := uint(1)    // Administrator

		proj1 := Project{
			Name:        "Implementasi ERP Cloud Odoo-Go",
			PartnerID:   &partnerID1,
			UserID:      &userID1,
			Status:      "active",
			Color:       "#6366f1",
			Description: "Deployment sistem ERP modern modular di cloud untuk integrasi manufaktur, ritel POS dan keuangan.",
		}
		proj1.CompanyID = 1
		proj1.CreateUID = 1
		proj1.WriteUID = 1
		if err := db.Create(&proj1).Error; err != nil {
			return err
		}

		proj2 := Project{
			Name:        "Migrasi Infrastruktur & Jaringan Cabang",
			PartnerID:   &partnerID2,
			UserID:      &userID1,
			Status:      "active",
			Color:       "#10b981",
			Description: "Pemasangan server backup lokal dan router switch kantor cabang baru.",
		}
		proj2.CompanyID = 1
		proj2.CreateUID = 1
		proj2.WriteUID = 1
		if err := db.Create(&proj2).Error; err != nil {
			return err
		}

		// Tasks for Proj1
		task1 := Task{
			ProjectID:      proj1.ID,
			Name:           "Setup Database PostgreSQL & Auto-Migration",
			PartnerID:      &partnerID1,
			UserID:         &userID1,
			Stage:          "done",
			Priority:       "2",
			PlannedHours:   16,
			EffectiveHours: 16,
			Progress:       100,
			Description:    "Konfigurasi database multi-tenant, tabel double-entry stock, dan entri buku besar balance.",
		}
		task1.CompanyID = 1
		task1.CreateUID = 1
		task1.WriteUID = 1
		db.Create(&task1)

		task2 := Task{
			ProjectID:      proj1.ID,
			Name:           "Kustomisasi Laporan Laba Rugi & Neraca Saldo",
			PartnerID:      &partnerID1,
			UserID:         &userID1,
			Stage:          "in_progress",
			Priority:       "3",
			PlannedHours:   24,
			EffectiveHours: 12,
			Progress:       50,
			Description:    "Implementasi kalkulasi laporan laba rugi real-time dan neraca berimbang sesuai SAK Indonesia.",
		}
		task2.CompanyID = 1
		task2.CreateUID = 1
		task2.WriteUID = 1
		db.Create(&task2)

		task3 := Task{
			ProjectID:      proj1.ID,
			Name:           "Pelatihan Kasir POS & SOP Gudang",
			PartnerID:      &partnerID1,
			UserID:         &userID1,
			Stage:          "todo",
			Priority:       "1",
			PlannedHours:   8,
			EffectiveHours: 0,
			Progress:       0,
			Description:    "Workshop pengoperasian checkout kasir ritel POS dan surat jalan gudang.",
		}
		task3.CompanyID = 1
		task3.CreateUID = 1
		task3.WriteUID = 1
		db.Create(&task3)

		// Seed a timesheet
		ts := TimesheetLine{
			ProjectID:  proj1.ID,
			TaskID:     task2.ID,
			UserID:     userID1,
			Date:       "2026-10-04",
			UnitAmount: 4.5,
			Name:       "Slicing query SQL Trial Balance & kalkulasi saldo akun debit/kredit",
		}
		ts.CompanyID = 1
		ts.CreateUID = 1
		ts.WriteUID = 1
		db.Create(&ts)

		log.Println("[Project] Seeded default Projects, Tasks & Timesheet lines")
	}
	return nil
}
