package recruitment

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/hr"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	repo    *Repository
	service *Service
	handler *Handler
}

func NewModule(db *gorm.DB, hrSvc *hr.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	svc := NewService(db, repo, hrSvc, chatterSvc, aiClient)
	h := NewHandler(svc)

	return &Module{
		db:      db,
		repo:    repo,
		service: svc,
		handler: h,
	}
}

func (m *Module) Name() string {
	return "recruitment"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&RecruitmentStage{},
		&JobOpening{},
		&Applicant{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/recruitment")
	group.Use(authMiddleware)
	{
		// Stages
		group.GET("/stages", m.handler.ListStages)
		group.POST("/stages", m.handler.CreateStage)

		// Job Openings
		group.GET("/openings", m.handler.ListJobOpenings)
		group.GET("/openings/:id", m.handler.GetJobOpening)
		group.POST("/openings", m.handler.CreateJobOpening)
		group.PUT("/openings/:id/state", m.handler.UpdateJobOpeningState)

		// Applicants
		group.GET("/applicants", m.handler.ListApplicants)
		group.GET("/applicants/:id", m.handler.GetApplicant)
		group.POST("/applicants", m.handler.CreateApplicant)
		group.PUT("/applicants/:id/stage", m.handler.UpdateApplicantStage)
		group.DELETE("/applicants/:id", m.handler.DeleteApplicant)

		// Cloudflare Workers AI Talent Screening & Actions
		group.POST("/applicants/:id/ai-screening", m.handler.RunAIScreening)
		group.POST("/applicants/:id/hire", m.handler.ActionHire)
		group.POST("/applicants/:id/refuse", m.handler.ActionRefuse)

		// Analytics
		group.GET("/stats", m.handler.GetStats)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	// 1. Seed Recruitment Stages
	var stageCount int64
	db.Model(&RecruitmentStage{}).Count(&stageCount)
	if stageCount == 0 {
		stages := []RecruitmentStage{
			{Name: "1. Initial Screening", Sequence: 10, Folded: false},
			{Name: "2. Technical Review", Sequence: 20, Folded: false},
			{Name: "3. User & HR Interview", Sequence: 30, Folded: false},
			{Name: "4. Contract Proposal", Sequence: 40, Folded: false},
			{Name: "5. Hired / Diterima", Sequence: 50, Folded: true, IsHired: true},
			{Name: "Refused / Ditolak", Sequence: 99, Folded: true, IsRefused: true},
		}
		for i := range stages {
			stages[i].CompanyID = 1
			db.Create(&stages[i])
		}
		log.Println("[Recruitment] Seeded 6 default recruitment stages")
	}

	// 2. Seed Sample Job Openings
	var openingCount int64
	db.Model(&JobOpening{}).Count(&openingCount)
	if openingCount == 0 {
		// Find IT and Finance departments
		var itDept, finDept hr.Department
		db.Where("name LIKE ? AND company_id = 1", "%IT%").First(&itDept)
		if itDept.ID == 0 {
			db.Where("company_id = 1").First(&itDept)
		}
		db.Where("name LIKE ? AND company_id = 1", "%Finan%").First(&finDept)
		if finDept.ID == 0 {
			finDept = itDept
		}

		openings := []JobOpening{
			{
				Title:             "Senior Golang & Edge Systems Engineer",
				DepartmentID:      itDept.ID,
				ExpectedEmployees: 2,
				CurrentHired:      0,
				Requirements:      "Kualifikasi: Minimal 3 tahun pengalaman backend development dengan Golang, Gin framework, GORM, database PostgreSQL & SQLite. Berpengalaman dengan Cloudflare Workers, Edge caching, Docker, dan arsitektur Modular Monolith/Microservices. Memahami konsep ERP Enterprise ERP, clean code, dan high concurrency.",
				Description:       "Bertanggung jawab atas arsitektur backend ERP, integrasi Cloudflare Workers AI edge copilot, optimalisasi latensi database, dan sinkronisasi modul multi-company.",
				SalaryMin:         16000000,
				SalaryMax:         25000000,
				State:             "open",
			},
			{
				Title:             "Finance & Tax Accounting Specialist",
				DepartmentID:      finDept.ID,
				ExpectedEmployees: 1,
				CurrentHired:      0,
				Requirements:      "Kualifikasi: Pendidikan S1 Akuntansi/Keuangan. Pengalaman minimal 2 tahun dalam pembukuan Buku Besar (General Ledger), rekonsiliasi bank, pelaporan pajak PPh 21, PPh 23, dan PPN Keluaran/Masukan. Terbiasa mengoperasikan software ERP Akuntansi dan Microsoft Excel tingkat lanjut.",
				Description:       "Mengelola laporan keuangan bulanan, audit kas operasional, validasi klaim reimbursement karyawan, dan kepatuhan perpajakan perusahaan.",
				SalaryMin:         8000000,
				SalaryMax:         13000000,
				State:             "open",
			},
		}

		for i := range openings {
			openings[i].CompanyID = 1
			db.Create(&openings[i])
		}
		log.Println("[Recruitment] Seeded 2 sample job openings")
	}

	// 3. Seed Sample Applicants with Resumes
	var appCount int64
	db.Model(&Applicant{}).Count(&appCount)
	if appCount == 0 {
		var opening JobOpening
		db.Where("company_id = 1").First(&opening)

		var stage RecruitmentStage
		db.Where("company_id = 1 AND is_hired = 0 AND is_refused = 0").Order("sequence asc").First(&stage)

		if opening.ID > 0 && stage.ID > 0 {
			sampleResumeArya := `RINGKASAN PROFESIONAL:
Software Engineer dengan pengalaman 4 tahun mengembangkan sistem backend terdistribusi berskala besar berbasis Golang dan PostgreSQL. Terbiasa merancang RESTful API performa tinggi dengan Gin dan GORM, arsitektur modular monolith, serta deployment container Docker di lingkungan Linux.

KEAHLIAN TEKNIS:
- Bahasa: Golang (Advanced), SQL, JavaScript, Bash
- Framework & Tools: Gin, GORM, Redis, PostgreSQL, SQLite, Git, Docker, Linux
- Cloud & Infrastructure: Cloudflare Workers, Edge Networking, AWS EC2/S3
- Domain: Sistem ERP, Akuntansi Finansial, E-Commerce Payment Gateway

PENGALAMAN KERJA:
Backend Developer - PT Solusi Digital Nusantara (2022 - Sekarang)
- Membangun microservices transaksi keuangan dengan throughput 1.500 TPS menggunakan Golang.
- Mengintegrasikan Cloudflare CDN dan Workers AI untuk kompresi data dan deteksi anomali.
- Mengurangi beban latensi database sebesar 35% dengan indexing komposit dan query tuning.`

			sampleApplicant := Applicant{
				Name:           "Arya Pratama, S.Kom.",
				Email:          "arya.pratama.dev@gmail.com",
				Phone:          "+62 812-9876-5432",
				JobOpeningID:   opening.ID,
				DepartmentID:   opening.DepartmentID,
				StageID:        stage.ID,
				SalaryExpected: 18000000,
				ResumeText:     sampleResumeArya,
				State:          "new",
			}
			sampleApplicant.CompanyID = 1
			db.Create(&sampleApplicant)
			log.Println("[Recruitment] Seeded 1 sample applicant (Arya Pratama)")
		}
	}

	return nil
}
