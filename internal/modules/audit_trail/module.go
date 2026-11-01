package audit_trail

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/chatter"

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
	return "audit_trail"
}

func (m *Module) Service() *Service {
	return m.service
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&AuditLogRule{},
		&AuditLogEntry{},
		&AuditLogFieldDiff{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) Seed(db *gorm.DB) error {
	if err := m.AutoMigrate(db); err != nil {
		return err
	}

	var count int64
	db.Model(&AuditLogRule{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Enterprise Audit Trail rules and demo CDC logs...")

	// 1. Seed Default Audit Rules
	rules := []AuditLogRule{
		{
			ModelBase:     models.ModelBase{CompanyID: 1, Active: true},
			Name:          "Audit Jurnal & Faktur Akuntansi",
			ModelName:     "account.move",
			LogCreate:     true,
			LogWrite:      true,
			LogUnlink:     true,
			LogRead:       false,
			TrackFields:   "state,amount_total,partner_id,date,ref",
			CaptureDiff:   true,
			RetentionDays: 2555, // 7 years (SOX compliance)
			Description:   "Pengawasan ketat terhadap seluruh mutasi jurnal finansial, pengakuan piutang, dan pencatatan kas/bank.",
		},
		{
			ModelBase:     models.ModelBase{CompanyID: 1, Active: true},
			Name:          "Audit Data Kepegawaian & Payroll",
			ModelName:     "hr.employee",
			LogCreate:     true,
			LogWrite:      true,
			LogUnlink:     true,
			LogRead:       false,
			TrackFields:   "basic_salary,wage,salary,bank_account_id,job_title,work_email,active",
			CaptureDiff:   true,
			RetentionDays: 1825, // 5 years
			Description:   "Audit kepatuhan kompensasi karyawan, nomor rekening transfer payroll, dan jabatan struktural.",
		},
		{
			ModelBase:     models.ModelBase{CompanyID: 1, Active: true},
			Name:          "Audit Pesanan Penjualan Komersial",
			ModelName:     "sale.order",
			LogCreate:     true,
			LogWrite:      true,
			LogUnlink:     true,
			LogRead:       false,
			TrackFields:   "state,amount_total,partner_id,pricelist_id",
			CaptureDiff:   true,
			RetentionDays: 1095, // 3 years
			Description:   "Pencatatan riwayat diskon, status persetujuan order, dan perubahan nilai kontrak penjualan.",
		},
		{
			ModelBase:     models.ModelBase{CompanyID: 1, Active: true},
			Name:          "Audit Master Produk & Harga Jual",
			ModelName:     "product.product",
			LogCreate:     true,
			LogWrite:      true,
			LogUnlink:     true,
			LogRead:       false,
			TrackFields:   "list_price,standard_price,name,active",
			CaptureDiff:   true,
			RetentionDays: 1095,
			Description:   "Tracking perubahan harga jual dan harga pokok penjualan (HPP) untuk mencegah manipulasi margin.",
		},
		{
			ModelBase:     models.ModelBase{CompanyID: 1, Active: true},
			Name:          "Audit Kontrak Berlangganan SaaS",
			ModelName:     "sale.subscription",
			LogCreate:     true,
			LogWrite:      true,
			LogUnlink:     true,
			LogRead:       false,
			TrackFields:   "stage,recurring_total,recurring_interval,recurring_rule_type",
			CaptureDiff:   true,
			RetentionDays: 1095,
			Description:   "Pemantauan siklus hidup kontrak langganan, churn, jeda waktu (paused), dan nilai MRR.",
		},
	}

	for i := range rules {
		if err := db.Create(&rules[i]).Error; err != nil {
			log.Printf("⚠️ Gagal seeding audit rule: %v", err)
		}
	}

	// 2. Seed Initial CDC Audit Logs
	entries := []struct {
		Entry AuditLogEntry
		Diffs []AuditLogFieldDiff
	}{
		{
			Entry: AuditLogEntry{
				ModelBase:     models.ModelBase{CompanyID: 1, CreateUID: 1, WriteUID: 1, Active: true},
				LogCode:       "AUD/2026/00001",
				ModelName:     "sale.order",
				ResID:         1,
				ResName:       "SO/2026/00001",
				Action:        ActionCreate,
				UserID:        1,
				UserName:      "Mitchell Admin",
				UserEmail:     "admin@odoo-indonesia.com",
				IPAddress:     "192.168.1.10",
				UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
				Timestamp:     "2026-10-09 09:15:00",
				Summary:       "CREATE pada sale.order #1 (SO/2026/00001) oleh Mitchell Admin",
				OldValuesJSON: "{}",
				NewValuesJSON: `{"id":1,"name":"SO/2026/00001","partner_id":1,"amount_total":15000000,"state":"draft"}`,
				IsSuspicious:  false,
				AnomalyReason: "",
				RiskScore:     10,
			},
			Diffs: []AuditLogFieldDiff{
				{FieldName: "state", OldValue: "<nil>", NewValue: "draft", ValueType: "string", IsSensitive: true},
				{FieldName: "amount_total", OldValue: "<nil>", NewValue: "15000000", ValueType: "float", IsSensitive: true},
			},
		},
		{
			Entry: AuditLogEntry{
				ModelBase:     models.ModelBase{CompanyID: 1, CreateUID: 1, WriteUID: 1, Active: true},
				LogCode:       "AUD/2026/00002",
				ModelName:     "product.product",
				ResID:         1,
				ResName:       "PROD-001 (Meja Kantor Ergonomis)",
				Action:        ActionUpdate,
				UserID:        1,
				UserName:      "Mitchell Admin",
				UserEmail:     "admin@odoo-indonesia.com",
				IPAddress:     "192.168.1.10",
				UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
				Timestamp:     "2026-10-09 11:30:00",
				Summary:       "UPDATE pada product.product #1 oleh Mitchell Admin",
				OldValuesJSON: `{"id":1,"list_price":750000,"standard_price":450000}`,
				NewValuesJSON: `{"id":1,"list_price":800000,"standard_price":450000}`,
				IsSuspicious:  false,
				AnomalyReason: "",
				RiskScore:     15,
			},
			Diffs: []AuditLogFieldDiff{
				{FieldName: "list_price", OldValue: "750000", NewValue: "800000", ValueType: "float", IsSensitive: true},
			},
		},
		{
			Entry: AuditLogEntry{
				ModelBase:     models.ModelBase{CompanyID: 1, CreateUID: 2, WriteUID: 2, Active: true},
				LogCode:       "AUD/2026/00003",
				ModelName:     "hr.employee",
				ResID:         2,
				ResName:       "EMP-002 (Budi Santoso)",
				Action:        ActionUpdate,
				UserID:        2,
				UserName:      "HR Staff",
				UserEmail:     "hr@odoo-indonesia.com",
				IPAddress:     "10.0.8.44",
				UserAgent:     "PostmanRuntime/7.32.2",
				Timestamp:     "2026-10-09 23:45:10",
				Summary:       "UPDATE pada hr.employee #2 (EMP-002) oleh HR Staff",
				OldValuesJSON: `{"id":2,"wage":8500000,"job_title":"Junior Developer"}`,
				NewValuesJSON: `{"id":2,"wage":19000000,"job_title":"Senior Lead"}`,
				IsSuspicious:  true,
				AnomalyReason: "Aktivitas dilakukan di luar jam kerja resmi (23:00 WIB); Modifikasi data sensitif/finansial pada kolom 'wage'; Lonjakan nilai drastis (124%) pada 'wage'",
				RiskScore:     80,
			},
			Diffs: []AuditLogFieldDiff{
				{FieldName: "wage", OldValue: "8500000", NewValue: "19000000", ValueType: "float", IsSensitive: true},
				{FieldName: "job_title", OldValue: "Junior Developer", NewValue: "Senior Lead", ValueType: "string", IsSensitive: false},
			},
		},
		{
			Entry: AuditLogEntry{
				ModelBase:     models.ModelBase{CompanyID: 1, CreateUID: 1, WriteUID: 1, Active: true},
				LogCode:       "AUD/2026/00004",
				ModelName:     "account.move",
				ResID:         1,
				ResName:       "INV/2026/00001",
				Action:        ActionUpdate,
				UserID:        1,
				UserName:      "Mitchell Admin",
				UserEmail:     "admin@odoo-indonesia.com",
				IPAddress:     "192.168.1.10",
				UserAgent:     "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
				Timestamp:     "2026-10-09 14:10:00",
				Summary:       "UPDATE pada account.move #1 (INV/2026/00001) oleh Mitchell Admin",
				OldValuesJSON: `{"id":1,"state":"posted","amount_total":25000000}`,
				NewValuesJSON: `{"id":1,"state":"draft","amount_total":25000000}`,
				IsSuspicious:  true,
				AnomalyReason: "Modifikasi data sensitif/finansial pada kolom 'state'; Pembatalan (unpost) transaksi akuntansi yang telah dibukukan",
				RiskScore:     70,
			},
			Diffs: []AuditLogFieldDiff{
				{FieldName: "state", OldValue: "posted", NewValue: "draft", ValueType: "string", IsSensitive: true},
			},
		},
	}

	for _, item := range entries {
		entry := item.Entry
		if err := db.Create(&entry).Error; err == nil {
			for _, diff := range item.Diffs {
				diff.AuditLogID = entry.ID
				diff.CompanyID = 1
				diff.Active = true
				_ = db.Create(&diff).Error
			}
		}
	}

	log.Println("✅ Enterprise Audit Trail and CDC demo logs successfully seeded!")
	return nil
}
