package lowcode_studio

import (
	"log"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db         *gorm.DB
	repo       Repository
	service    Service
	handler    *Handler
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	repo := NewRepository(db)
	service := NewService(repo, db, chatterSvc, aiClient)
	handler := NewHandler(service)

	return &Module{
		db:         db,
		repo:       repo,
		service:    service,
		handler:    handler,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

func (m *Module) Name() string {
	return "lowcode_studio"
}

func (m *Module) Service() Service {
	return m.service
}

func (m *Module) Repository() Repository {
	return m.repo
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	m.db = db
	return m.repo.AutoMigrate()
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(r, authMiddleware)
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&CustomModel{}).Count(&count)
	if count > 0 {
		return nil
	}

	log.Println("🌱 Seeding Universal Low-Code Studio Default Industry Blueprints...")

	// 1. Blueprint Medis (Healthcare EMR)
	medModel := &CustomModel{
		CompanyID:        1,
		TechnicalName:    "x_medical_emr",
		DisplayName:      "Rekam Medis Pasien (EHR/EMR)",
		Description:      "Sistem pencatatan riwayat kesehatan pasien, pemeriksaan poliklinik, dan resep farmasi.",
		Icon:             "fas fa-heartbeat",
		MenuCategory:     "Healthcare & Klinik",
		IsCoreExtension:  false,
		AllowChatter:     true,
		AllowAttachments: true,
		IsActive:         true,
	}
	_ = db.Create(medModel)

	medFields := []CustomField{
		{CustomModelID: medModel.ID, FieldName: "x_mrn", FieldLabel: "Nomor RM (MRN)", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true, SequenceOrder: 10},
		{CustomModelID: medModel.ID, FieldName: "x_patient_name", FieldLabel: "Nama Pasien", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true, SequenceOrder: 20},
		{CustomModelID: medModel.ID, FieldName: "x_blood_type", FieldLabel: "Golongan Darah", FieldType: FieldTypeSelection, SelectionOptions: `["A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"]`, IsRequired: true, SequenceOrder: 30},
		{CustomModelID: medModel.ID, FieldName: "x_symptoms", FieldLabel: "Anamnesis / Keluhan", FieldType: FieldTypeText, IsRequired: true, SequenceOrder: 40},
		{CustomModelID: medModel.ID, FieldName: "x_diagnosis_code", FieldLabel: "Kode ICD-10", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true, SequenceOrder: 50},
		{CustomModelID: medModel.ID, FieldName: "x_is_bpjs", FieldLabel: "Pasien BPJS", FieldType: FieldTypeBoolean, DefaultValue: "true", SequenceOrder: 60},
	}
	for _, f := range medFields {
		_ = db.Create(&f)
	}

	_ = db.Create(&CustomRecord{
		CompanyID:     1,
		CustomModelID: medModel.ID,
		RecordCode:    "EMR/2026/00001",
		DisplayName:   "Budi Santoso (RM-99214)",
		ValuesJSON:    `{"x_mrn":"RM-99214","x_patient_name":"Budi Santoso","x_blood_type":"O+","x_symptoms":"Batuk berdahak 3 hari disertai demam tinggi 38.5C","x_diagnosis_code":"J06.9 (Infeksi Saluran Pernapasan Akut)","x_is_bpjs":true}`,
		CreatedByID:   1,
	})

	// 2. Blueprint Tambang (Mining & Hauling)
	miningModel := &CustomModel{
		CompanyID:        1,
		TechnicalName:    "x_mining_haul",
		DisplayName:      "Ritase Pengangkutan Batubara",
		Description:      "Pencatatan produksi pit, armada dump truck, tonase timbang, dan kadar kalori tambang.",
		Icon:             "fas fa-mountain",
		MenuCategory:     "Mining & Resources",
		IsCoreExtension:  false,
		AllowChatter:     true,
		AllowAttachments: true,
		IsActive:         true,
	}
	_ = db.Create(miningModel)

	miningFields := []CustomField{
		{CustomModelID: miningModel.ID, FieldName: "x_ticket_no", FieldLabel: "No. Tiket Timbang", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true, SequenceOrder: 10},
		{CustomModelID: miningModel.ID, FieldName: "x_truck_unit", FieldLabel: "Unit Dump Truck", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true, SequenceOrder: 20},
		{CustomModelID: miningModel.ID, FieldName: "x_net_tonnage", FieldLabel: "Tonase Bersih (Ton)", FieldType: FieldTypeFloat, IsRequired: true, SequenceOrder: 30},
		{CustomModelID: miningModel.ID, FieldName: "x_calorie_gar", FieldLabel: "Kalori (GAR / Grade)", FieldType: FieldTypeInteger, IsRequired: false, SequenceOrder: 40},
		{CustomModelID: miningModel.ID, FieldName: "x_destination", FieldLabel: "Tujuan Stockpile", FieldType: FieldTypeSelection, SelectionOptions: `["Stockpile KM 14", "ROM Pad Pit B", "Jetty Port 1"]`, IsRequired: true, SequenceOrder: 50},
	}
	for _, f := range miningFields {
		_ = db.Create(&f)
	}

	_ = db.Create(&CustomRecord{
		CompanyID:     1,
		CustomModelID: miningModel.ID,
		RecordCode:    "HAUL/2026/00001",
		DisplayName:   "DT-8821 (34.8 Ton - Pit A)",
		ValuesJSON:    `{"x_ticket_no":"WB-2026-90412","x_truck_unit":"DT-8821","x_net_tonnage":34.8,"x_calorie_gar":5600,"x_destination":"Stockpile KM 14"}`,
		CreatedByID:   1,
	})

	// 3. Blueprint Hospitality (Hotel Reservation)
	hotelModel := &CustomModel{
		CompanyID:        1,
		TechnicalName:    "x_hotel_reservation",
		DisplayName:      "Reservasi Kamar Hotel",
		Description:      "Manajemen pemesanan kamar hotel, status tamu, tipe suite, dan deposit.",
		Icon:             "fas fa-hotel",
		MenuCategory:     "Hospitality & Hotel",
		IsCoreExtension:  false,
		AllowChatter:     true,
		AllowAttachments: true,
		IsActive:         true,
	}
	_ = db.Create(hotelModel)

	hotelFields := []CustomField{
		{CustomModelID: hotelModel.ID, FieldName: "x_booking_ref", FieldLabel: "Kode Reservasi", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true, SequenceOrder: 10},
		{CustomModelID: hotelModel.ID, FieldName: "x_guest_name", FieldLabel: "Nama Tamu", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true, SequenceOrder: 20},
		{CustomModelID: hotelModel.ID, FieldName: "x_room_type", FieldLabel: "Tipe Kamar", FieldType: FieldTypeSelection, SelectionOptions: `["Deluxe King", "Superior Twin", "Presidential Suite"]`, IsRequired: true, SequenceOrder: 30},
		{CustomModelID: hotelModel.ID, FieldName: "x_nightly_rate", FieldLabel: "Tarif per Malam", FieldType: FieldTypeMonetary, IsRequired: true, SequenceOrder: 40},
		{CustomModelID: hotelModel.ID, FieldName: "x_is_vip", FieldLabel: "Tamu VIP", FieldType: FieldTypeBoolean, DefaultValue: "false", SequenceOrder: 50},
	}
	for _, f := range hotelFields {
		_ = db.Create(&f)
	}

	_ = db.Create(&CustomRecord{
		CompanyID:     1,
		CustomModelID: hotelModel.ID,
		RecordCode:    "HTL/2026/00001",
		DisplayName:   "HTL-BK-202601 (Sarah Jenkins)",
		ValuesJSON:    `{"x_booking_ref":"HTL-BK-202601","x_guest_name":"Sarah Jenkins","x_room_type":"Presidential Suite","x_nightly_rate":4500000,"x_is_vip":true}`,
		CreatedByID:   1,
	})

	log.Println("✅ Universal Low-Code Studio Initial Seeds Complete.")
	return nil
}
