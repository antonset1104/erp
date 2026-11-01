package lowcode_studio

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"gorm.io/gorm"
)

type Service interface {
	// Models
	CreateModel(req CreateCustomModelRequest, companyID uint) (*CustomModel, error)
	GetModelByID(id uint) (*CustomModel, error)
	GetModelByTechnicalName(name string) (*CustomModel, error)
	ListModels(companyID uint) ([]CustomModel, error)
	UpdateModel(id uint, req CreateCustomModelRequest) (*CustomModel, error)
	DeleteModel(id uint) error

	// Fields
	AddField(modelID uint, req CreateCustomFieldRequest) (*CustomField, error)
	DeleteField(fieldID uint) error

	// Records
	CreateRecord(modelID uint, req SaveCustomRecordRequest, companyID uint, userID uint) (*CustomRecord, error)
	GetRecordByID(recordID uint) (*CustomRecord, error)
	ListRecords(modelID uint, companyID uint, page, limit int) ([]CustomRecord, int64, error)
	UpdateRecord(recordID uint, req SaveCustomRecordRequest) (*CustomRecord, error)
	DeleteRecord(recordID uint) error

	// Core Extension
	SaveCoreExtension(req SaveCoreExtensionRequest, companyID uint) (*CoreModelExtensionValue, error)
	GetCoreExtension(coreModel string, recordID uint) (*CoreModelExtensionValue, error)

	// AI Schema Architect Copilot
	GenerateSchemaWithAI(prompt string, category string) (*AISchemaGenerationResponse, error)

	// Analytics
	GetStats(companyID uint) (*LowCodeStudioStats, error)
}

type service struct {
	repo       Repository
	db         *gorm.DB
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(repo Repository, db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client) Service {
	return &service{
		repo:       repo,
		db:         db,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// Models
func (s *service) CreateModel(req CreateCustomModelRequest, companyID uint) (*CustomModel, error) {
	if companyID == 0 {
		companyID = 1
	}

	if req.DisplayName == "" && req.Name != "" {
		req.DisplayName = req.Name
	}
	if req.DisplayName == "" {
		return nil, errors.New("display_name model kustom wajib diisi")
	}
	if req.MenuCategory == "" && req.Category != "" {
		req.MenuCategory = req.Category
	}

	techName := strings.TrimSpace(req.TechnicalName)
	if !strings.HasPrefix(techName, "x_") && !req.IsCoreExtension {
		techName = "x_" + techName
	}
	techName = strings.ToLower(strings.ReplaceAll(techName, " ", "_"))

	// Validasi duplikasi
	existing, _ := s.repo.GetModelByTechnicalName(techName)
	if existing != nil {
		return nil, fmt.Errorf("model teknikal '%s' sudah terdaftar", techName)
	}

	icon := req.Icon
	if icon == "" {
		icon = "fas fa-cube"
	}
	category := req.MenuCategory
	if category == "" {
		category = "Custom Apps"
	}

	model := &CustomModel{
		CompanyID:        companyID,
		TechnicalName:    techName,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		Icon:             icon,
		MenuCategory:     category,
		IsCoreExtension:  req.IsCoreExtension,
		CoreModelTarget:  req.CoreModelTarget,
		AllowChatter:     req.AllowChatter,
		AllowAttachments: req.AllowAttachments,
		IsActive:         true,
	}

	if err := s.repo.CreateModel(model); err != nil {
		return nil, err
	}

	// Insert initial fields jika disediakan
	for i, f := range req.InitialFields {
		fieldName := strings.TrimSpace(f.FieldName)
		if !strings.HasPrefix(fieldName, "x_") {
			fieldName = "x_" + fieldName
		}
		fieldName = strings.ToLower(strings.ReplaceAll(fieldName, " ", "_"))

		field := &CustomField{
			CustomModelID:    model.ID,
			FieldName:        fieldName,
			FieldLabel:       f.FieldLabel,
			FieldType:        f.FieldType,
			IsRequired:       f.IsRequired,
			IsReadonly:       f.IsReadonly,
			IsSearchable:     f.IsSearchable,
			DefaultValue:     f.DefaultValue,
			SelectionOptions: f.SelectionOptions,
			RelationModel:    f.RelationModel,
			SequenceOrder:    (i + 1) * 10,
			HelpTooltip:      f.HelpTooltip,
		}
		_ = s.repo.CreateField(field)
	}

	// Log Chatter
	if s.chatterSvc != nil {
		_ = s.chatterSvc.PostMessage(&chatter.MailMessage{
			ModelName:   "lowcode.model",
			ResID:       model.ID,
			AuthorName:  "Low-Code Studio",
			Body:        fmt.Sprintf("Entitas Model Dinamis '%s' (%s) berhasil dikonfigurasi dengan %d kolom awal.", model.DisplayName, model.TechnicalName, len(req.InitialFields)),
			MessageType: "comment",
		})
	}

	return s.repo.GetModelByID(model.ID)
}

func (s *service) GetModelByID(id uint) (*CustomModel, error) {
	return s.repo.GetModelByID(id)
}

func (s *service) GetModelByTechnicalName(name string) (*CustomModel, error) {
	return s.repo.GetModelByTechnicalName(name)
}

func (s *service) ListModels(companyID uint) ([]CustomModel, error) {
	return s.repo.ListModels(companyID)
}

func (s *service) UpdateModel(id uint, req CreateCustomModelRequest) (*CustomModel, error) {
	model, err := s.repo.GetModelByID(id)
	if err != nil {
		return nil, err
	}

	model.DisplayName = req.DisplayName
	model.Description = req.Description
	if req.Icon != "" {
		model.Icon = req.Icon
	}
	if req.MenuCategory != "" {
		model.MenuCategory = req.MenuCategory
	}
	model.AllowChatter = req.AllowChatter
	model.AllowAttachments = req.AllowAttachments

	if err := s.repo.UpdateModel(model); err != nil {
		return nil, err
	}
	return model, nil
}

func (s *service) DeleteModel(id uint) error {
	return s.repo.DeleteModel(id)
}

// Fields
func (s *service) AddField(modelID uint, req CreateCustomFieldRequest) (*CustomField, error) {
	if req.FieldLabel == "" && req.Label != "" {
		req.FieldLabel = req.Label
	}
	if !req.IsRequired && req.Required {
		req.IsRequired = true
	}
	if req.SelectionOptions == "" && req.OptionsJSON != "" {
		req.SelectionOptions = req.OptionsJSON
	}

	model, err := s.repo.GetModelByID(modelID)
	if err != nil {
		return nil, fmt.Errorf("model tidak ditemukan: %w", err)
	}

	fieldName := strings.TrimSpace(req.FieldName)
	if !strings.HasPrefix(fieldName, "x_") {
		fieldName = "x_" + fieldName
	}
	fieldName = strings.ToLower(strings.ReplaceAll(fieldName, " ", "_"))

	// Cek apakah nama field sudah ada pada model ini
	for _, f := range model.Fields {
		if f.FieldName == fieldName {
			return nil, fmt.Errorf("field '%s' sudah terdaftar pada model ini", fieldName)
		}
	}

	seq := req.SequenceOrder
	if seq == 0 {
		seq = (len(model.Fields) + 1) * 10
	}

	field := &CustomField{
		CustomModelID:    modelID,
		FieldName:        fieldName,
		FieldLabel:       req.FieldLabel,
		FieldType:        req.FieldType,
		IsRequired:       req.IsRequired,
		IsReadonly:       req.IsReadonly,
		IsSearchable:     req.IsSearchable,
		DefaultValue:     req.DefaultValue,
		SelectionOptions: req.SelectionOptions,
		RelationModel:    req.RelationModel,
		SequenceOrder:    seq,
		HelpTooltip:      req.HelpTooltip,
	}

	if err := s.repo.CreateField(field); err != nil {
		return nil, err
	}
	return field, nil
}

func (s *service) DeleteField(fieldID uint) error {
	return s.repo.DeleteField(fieldID)
}

// Records
func (s *service) CreateRecord(modelID uint, req SaveCustomRecordRequest, companyID uint, userID uint) (*CustomRecord, error) {
	if companyID == 0 {
		companyID = 1
	}
	if userID == 0 {
		userID = 1
	}

	model, err := s.repo.GetModelByID(modelID)
	if err != nil {
		return nil, fmt.Errorf("model tidak ditemukan: %w", err)
	}

	// Validasi field required
	for _, f := range model.Fields {
		if f.IsRequired {
			val, exists := req.Values[f.FieldName]
			if !exists || val == nil || fmt.Sprintf("%v", val) == "" {
				return nil, fmt.Errorf("kolom wajib '%s' (%s) harus diisi", f.FieldLabel, f.FieldName)
			}
		}
	}

	valuesBytes, err := json.Marshal(req.Values)
	if err != nil {
		return nil, fmt.Errorf("gagal memproses nilai JSON: %w", err)
	}

	displayName := req.DisplayName
	if displayName == "" {
		// Ambil dari field string pertama jika displayName kosong
		for _, f := range model.Fields {
			if f.FieldType == FieldTypeString {
				if v, ok := req.Values[f.FieldName]; ok && v != nil {
					displayName = fmt.Sprintf("%v", v)
					break
				}
			}
		}
		if displayName == "" {
			displayName = fmt.Sprintf("%s Record", model.DisplayName)
		}
	}

	prefix := strings.ToUpper(strings.TrimPrefix(model.TechnicalName, "x_"))
	if len(prefix) > 4 {
		prefix = prefix[:4]
	}
	recordCode := s.repo.GenerateRecordCode(prefix)

	record := &CustomRecord{
		CompanyID:     companyID,
		CustomModelID: modelID,
		RecordCode:    recordCode,
		DisplayName:   displayName,
		ValuesJSON:    string(valuesBytes),
		CreatedByID:   userID,
	}

	if err := s.repo.CreateRecord(record); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil && model.AllowChatter {
		_ = s.chatterSvc.PostMessage(&chatter.MailMessage{
			ModelName:   model.TechnicalName,
			ResID:       record.ID,
			AuthorName:  "Low-Code Engine",
			Body:        fmt.Sprintf("Baris data baru '%s' (%s) berhasil dicatat.", record.DisplayName, record.RecordCode),
			MessageType: "comment",
		})
	}

	return record, nil
}

func (s *service) GetRecordByID(recordID uint) (*CustomRecord, error) {
	return s.repo.GetRecordByID(recordID)
}

func (s *service) ListRecords(modelID uint, companyID uint, page, limit int) ([]CustomRecord, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	offset := (page - 1) * limit
	return s.repo.ListRecordsByModelID(modelID, companyID, limit, offset)
}

func (s *service) UpdateRecord(recordID uint, req SaveCustomRecordRequest) (*CustomRecord, error) {
	record, err := s.repo.GetRecordByID(recordID)
	if err != nil {
		return nil, err
	}

	model, err := s.repo.GetModelByID(record.CustomModelID)
	if err != nil {
		return nil, err
	}

	// Merge/Validate
	currentValues := make(map[string]interface{})
	_ = json.Unmarshal([]byte(record.ValuesJSON), &currentValues)
	for k, v := range req.Values {
		currentValues[k] = v
	}

	// Validasi field required
	for _, f := range model.Fields {
		if f.IsRequired {
			val, exists := currentValues[f.FieldName]
			if !exists || val == nil || fmt.Sprintf("%v", val) == "" {
				return nil, fmt.Errorf("kolom wajib '%s' (%s) harus diisi", f.FieldLabel, f.FieldName)
			}
		}
	}

	newBytes, err := json.Marshal(currentValues)
	if err != nil {
		return nil, err
	}

	if req.DisplayName != "" {
		record.DisplayName = req.DisplayName
	}
	record.ValuesJSON = string(newBytes)

	if err := s.repo.UpdateRecord(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *service) DeleteRecord(recordID uint) error {
	return s.repo.DeleteRecord(recordID)
}

// Core Extension
func (s *service) SaveCoreExtension(req SaveCoreExtensionRequest, companyID uint) (*CoreModelExtensionValue, error) {
	if companyID == 0 {
		companyID = 1
	}
	bytes, err := json.Marshal(req.Values)
	if err != nil {
		return nil, err
	}

	val := &CoreModelExtensionValue{
		CompanyID:     companyID,
		CoreModel:     req.CoreModel,
		RecordID:      req.RecordID,
		ExtensionJSON: string(bytes),
	}

	if err := s.repo.SaveCoreExtensionValue(val); err != nil {
		return nil, err
	}
	return val, nil
}

func (s *service) GetCoreExtension(coreModel string, recordID uint) (*CoreModelExtensionValue, error) {
	return s.repo.GetCoreExtensionValue(coreModel, recordID)
}

// AI Schema Architect Copilot
func (s *service) GenerateSchemaWithAI(prompt string, category string) (*AISchemaGenerationResponse, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, errors.New("deskripsi kebutuhan bisnis industri tidak boleh kosong")
	}

	// Cek apakah Cloudflare AI tersedia
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "You are a World-Class Enterprise ERP Data Architect (SAP/Odoo Studio/Salesforce certified). Given a business requirement in any industry, generate an exact JSON schema containing: model_technical_name (prefix x_), model_display_name, description, icon (FontAwesome), menu_category, architecture_notes, and recommended_fields (array of {field_name, field_label, field_type, is_required, selection_options, help_tooltip}). Supported field_types: string, text, integer, float, monetary, date, datetime, boolean, selection, many2one."
		userPrompt := fmt.Sprintf("Generate ERP Custom Schema for: %s (Category: %s). Return ONLY valid JSON.", prompt, category)

		aiResp, err := s.aiClient.CallRaw(userPrompt, systemPrompt)
		if err == nil && aiResp != "" {
			var result AISchemaGenerationResponse
			// Bersihkan markdown code blocks jika ada
			cleanJSON := strings.TrimPrefix(aiResp, "```json")
			cleanJSON = strings.TrimPrefix(cleanJSON, "```")
			cleanJSON = strings.TrimSuffix(cleanJSON, "```")
			cleanJSON = strings.TrimSpace(cleanJSON)

			if jsonErr := json.Unmarshal([]byte(cleanJSON), &result); jsonErr == nil && result.ModelTechnicalName != "" {
				result.Provider = "Cloudflare Workers AI (@cf/meta/llama-3-8b-instruct)"
				return &result, nil
			}
		}
	}

	// Fallback: Local Smart Heuristic Industry Architect Engine
	return s.generateLocalHeuristicSchema(prompt, category), nil
}

func (s *service) generateLocalHeuristicSchema(prompt string, category string) *AISchemaGenerationResponse {
	lower := strings.ToLower(prompt)

	if strings.Contains(lower, "medis") || strings.Contains(lower, "pasien") || strings.Contains(lower, "dokter") || strings.Contains(lower, "hospital") || strings.Contains(lower, "klinik") {
		return &AISchemaGenerationResponse{
			ModelTechnicalName: "x_medical_record",
			ModelDisplayName:   "Rekam Medis Pasien (EHR/EMR)",
			Description:        "Entitas rekam medis elektronik terintegrasi mencakup diagnosa klinis, riwayat alergi, tindakan poli, dan resep farmasi.",
			Icon:               "fas fa-heartbeat",
			MenuCategory:       "Healthcare & Klinik",
			ArchitectureNotes:  "Dirancang sesuai standar SNOMED-CT / ICD-10 dengan dukungan integrasi SatuSehat Kemenkes dan BPJS Kesehatan.",
			Provider:           "Local Smart Heuristic Engine (Healthcare Industry Blueprint)",
			RecommendedFields: []CreateCustomFieldRequest{
				{FieldName: "x_patient_mrn", FieldLabel: "Nomor Rekam Medis (No. RM)", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true, HelpTooltip: "Nomor registrasi unik pasien"},
				{FieldName: "x_patient_id", FieldLabel: "Nama Pasien (Kontak)", FieldType: FieldTypeMany2One, RelationModel: "res.partner", IsRequired: true, HelpTooltip: "Tautan ke master kontak pasien"},
				{FieldName: "x_blood_type", FieldLabel: "Golongan Darah", FieldType: FieldTypeSelection, SelectionOptions: `["A+", "A-", "B+", "B-", "AB+", "AB-", "O+", "O-"]`, IsRequired: true},
				{FieldName: "x_examination_date", FieldLabel: "Tanggal Pemeriksaan", FieldType: FieldTypeDateTime, IsRequired: true},
				{FieldName: "x_systolic_bp", FieldLabel: "Tekanan Darah Sistolik (mmHg)", FieldType: FieldTypeInteger, IsRequired: false},
				{FieldName: "x_body_temp", FieldLabel: "Suhu Tubuh (°C)", FieldType: FieldTypeFloat, IsRequired: false},
				{FieldName: "x_symptoms", FieldLabel: "Keluhan Utama (Anamnesis)", FieldType: FieldTypeText, IsRequired: true},
				{FieldName: "x_icd10_diagnosis", FieldLabel: "Diagnosa Klinis (ICD-10)", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true},
				{FieldName: "x_is_bpjs", FieldLabel: "Pasien Rujukan BPJS", FieldType: FieldTypeBoolean, DefaultValue: "false"},
				{FieldName: "x_doctor_notes", FieldLabel: "Instruksi Medis & Resep Obat", FieldType: FieldTypeText, IsRequired: false},
			},
		}
	}

	if strings.Contains(lower, "hotel") || strings.Contains(lower, "kamar") || strings.Contains(lower, "tamu") || strings.Contains(lower, "villa") || strings.Contains(lower, "booking") {
		return &AISchemaGenerationResponse{
			ModelTechnicalName: "x_hotel_reservation",
			ModelDisplayName:   "Reservasi Kamar Hotel & Villa",
			Description:        "Entitas sistem manajemen properti perhotelan (PMS) untuk reservasi kamar, status tamu, folio tagihan, dan housekeeping.",
			Icon:               "fas fa-hotel",
			MenuCategory:       "Hospitality & Hotel",
			ArchitectureNotes:  "Mendukung alur check-in/check-out otomatis, integrasi key-card, dan folio billing terhubung ke Akuntansi General Ledger.",
			Provider:           "Local Smart Heuristic Engine (Hospitality Industry Blueprint)",
			RecommendedFields: []CreateCustomFieldRequest{
				{FieldName: "x_reservation_code", FieldLabel: "Kode Booking", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true},
				{FieldName: "x_guest_id", FieldLabel: "Tamu / Pelanggan", FieldType: FieldTypeMany2One, RelationModel: "res.partner", IsRequired: true},
				{FieldName: "x_room_number", FieldLabel: "Nomor Kamar", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true},
				{FieldName: "x_room_type", FieldLabel: "Tipe Kamar", FieldType: FieldTypeSelection, SelectionOptions: `["Deluxe King", "Superior Twin", "Executive Suite", "Presidential Suite", "Family Villa"]`, IsRequired: true},
				{FieldName: "x_check_in", FieldLabel: "Tanggal Check-In", FieldType: FieldTypeDateTime, IsRequired: true},
				{FieldName: "x_check_out", FieldLabel: "Tanggal Check-Out", FieldType: FieldTypeDateTime, IsRequired: true},
				{FieldName: "x_total_guests", FieldLabel: "Jumlah Tamu", FieldType: FieldTypeInteger, IsRequired: true, DefaultValue: "2"},
				{FieldName: "x_nightly_rate", FieldLabel: "Tarif per Malam", FieldType: FieldTypeMonetary, IsRequired: true},
				{FieldName: "x_deposit_amount", FieldLabel: "Uang Jaminan (Deposit)", FieldType: FieldTypeMonetary, IsRequired: false},
				{FieldName: "x_is_vip", FieldLabel: "Tamu VIP / Loyalty Member", FieldType: FieldTypeBoolean, DefaultValue: "false"},
			},
		}
	}

	if strings.Contains(lower, "tambang") || strings.Contains(lower, "mining") || strings.Contains(lower, "batubara") || strings.Contains(lower, "nikel") || strings.Contains(lower, "ritase") {
		return &AISchemaGenerationResponse{
			ModelTechnicalName: "x_mining_production_haul",
			ModelDisplayName:   "Ritase Pengangkutan Tambang (Hauling)",
			Description:        "Pencatatan produksi pit, ritase dump truck, jembatan timbang (weighbridge), dan pengujian kadar kalori/kandungan mineral.",
			Icon:               "fas fa-mountain",
			MenuCategory:       "Mining & Resources",
			ArchitectureNotes:  "Kompatibel dengan sensor telematika fleet GPS, jembatan timbang terotomasi, dan sertifikasi surveyor kualitas tambang.",
			Provider:           "Local Smart Heuristic Engine (Mining & Heavy Industry Blueprint)",
			RecommendedFields: []CreateCustomFieldRequest{
				{FieldName: "x_ticket_number", FieldLabel: "Nomor Tiket Timbang", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true},
				{FieldName: "x_haul_date", FieldLabel: "Tanggal & Jam Hauling", FieldType: FieldTypeDateTime, IsRequired: true},
				{FieldName: "x_pit_block", FieldLabel: "Lokasi Blok Pit Tambang", FieldType: FieldTypeString, IsRequired: true},
				{FieldName: "x_truck_unit", FieldLabel: "Unit Dump Truck / DT No.", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true},
				{FieldName: "x_driver_id", FieldLabel: "Operator / Supir", FieldType: FieldTypeMany2One, RelationModel: "res.partner", IsRequired: true},
				{FieldName: "x_gross_weight_ton", FieldLabel: "Berat Bruto (Ton)", FieldType: FieldTypeFloat, IsRequired: true},
				{FieldName: "x_tare_weight_ton", FieldLabel: "Berat Tara (Ton)", FieldType: FieldTypeFloat, IsRequired: true},
				{FieldName: "x_net_tonnage", FieldLabel: "Tonase Bersih (Net Ton)", FieldType: FieldTypeFloat, IsRequired: true},
				{FieldName: "x_calorie_gar", FieldLabel: "Kadar Kalori (GAR / Grade)", FieldType: FieldTypeInteger, IsRequired: false},
				{FieldName: "x_destination_stockpile", FieldLabel: "Tujuan Stockpile / Jetty", FieldType: FieldTypeSelection, SelectionOptions: `["Stockpile Utama KM 12", "ROM Pad Pit A", "Jetty Port Loading 1", "Crusher Plant B"]`, IsRequired: true},
			},
		}
	}

	// Default Generic Enterprise Model
	formattedName := strings.ToLower(strings.ReplaceAll(prompt, " ", "_"))
	if len(formattedName) > 20 {
		formattedName = formattedName[:20]
	}

	return &AISchemaGenerationResponse{
		ModelTechnicalName: "x_" + formattedName,
		ModelDisplayName:   prompt,
		Description:        fmt.Sprintf("Entitas kustom terkonfigurasi otomatis untuk mengakomodir kebutuhan proses bisnis: %s", prompt),
		Icon:               "fas fa-project-diagram",
		MenuCategory:       "Aplikasi Industri Kustom",
		ArchitectureNotes:  "Skema bisnis dinamis yang dilengkapi audit trail, otomatisasi formulir web, dan validasi tipe data universal.",
		Provider:           "Local Smart Heuristic Engine (Universal Entity Blueprint)",
		RecommendedFields: []CreateCustomFieldRequest{
			{FieldName: "x_code", FieldLabel: "Kode / No. Referensi", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true},
			{FieldName: "x_name", FieldLabel: "Nama / Judul", FieldType: FieldTypeString, IsRequired: true, IsSearchable: true},
			{FieldName: "x_transaction_date", FieldLabel: "Tanggal Transaksi", FieldType: FieldTypeDate, IsRequired: true},
			{FieldName: "x_partner_id", FieldLabel: "Mitra Bisnis Terkait", FieldType: FieldTypeMany2One, RelationModel: "res.partner", IsRequired: false},
			{FieldName: "x_amount", FieldLabel: "Nilai Transaksi", FieldType: FieldTypeMonetary, IsRequired: false},
			{FieldName: "x_status", FieldLabel: "Status Alur Kerja", FieldType: FieldTypeSelection, SelectionOptions: `["Draft", "In Progress", "Verified", "Approved", "Done"]`, IsRequired: true},
			{FieldName: "x_notes", FieldLabel: "Catatan & Keterangan", FieldType: FieldTypeText, IsRequired: false},
		},
	}
}

// Stats
func (s *service) GetStats(companyID uint) (*LowCodeStudioStats, error) {
	return s.repo.GetStats(companyID)
}
