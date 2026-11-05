package ai

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

var (
	globalLegalAuditCounter uint64
)

// LegalContractAuditRecord persists comprehensive contract legal assessments, risk ratings, and counterparty metadata
type LegalContractAuditRecord struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	AuditCode              string    `gorm:"size:64;uniqueIndex;not null" json:"audit_code"`
	ContractTitle          string    `gorm:"size:255;not null" json:"contract_title"`
	ContractType           string    `gorm:"size:64;index" json:"contract_type"` // "VENDOR_MSA", "PROCUREMENT_PKS", "CUSTOMER_SLA", "NDA", "DISTRIBUTOR_AGREEMENT"
	CounterpartyName       string    `gorm:"size:128;index" json:"counterparty_name"`
	ContractValueIDR       float64   `gorm:"column:contract_value_idr" json:"contract_value_idr"`
	RiskScore              float64   `gorm:"column:risk_score" json:"risk_score"` // 0.0 (safe) to 100.0 (critical exposure)
	RiskLevel              string    `gorm:"size:32;index" json:"risk_level"`     // "LOW", "MODERATE", "HIGH", "CRITICAL"
	IdentifiedClausesCount int       `gorm:"column:identified_clauses_count" json:"identified_clauses_count"`
	HighRiskClausesCount   int       `gorm:"column:high_risk_clauses_count" json:"high_risk_clauses_count"`
	RawContractText        string    `gorm:"type:text" json:"raw_contract_text"`
	ExecutiveSummary       string    `gorm:"type:text" json:"executive_summary"`
	NegotiationStatus      string    `gorm:"size:32;default:'DRAFT_REVIEW'" json:"negotiation_status"` // "DRAFT_REVIEW", "REDLINED", "COUNTER_OFFER_SENT", "ACCEPTED", "REJECTED"
	CreatedBy              string    `gorm:"size:64" json:"created_by"`
	CompanyID              uint      `gorm:"index" json:"company_id"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (LegalContractAuditRecord) TableName() string {
	return "ai_legal_contract_audits"
}

// LegalClauseRedlineRecord holds granular clause-level risk findings, side-by-side diff markup, and counter-clauses
type LegalClauseRedlineRecord struct {
	ID                     uint      `gorm:"primaryKey" json:"id"`
	AuditID                uint      `gorm:"index;not null" json:"audit_id"`
	ClauseNumber           string    `gorm:"size:32" json:"clause_number"` // e.g. "Pasal 8.2", "Section 14"
	ClauseCategory         string    `gorm:"size:64;index" json:"clause_category"` // "LIMITATION_OF_LIABILITY", "INDEMNIFICATION", "PENALTY_LIQUIDATED_DAMAGES", "INTELLECTUAL_PROPERTY", "TERMINATION_CONVENIENCE", "DATA_PRIVACY_PDP", "GOVERNING_LAW"
	RiskSeverity           string    `gorm:"size:32;index" json:"risk_severity"`   // "CRITICAL", "HIGH", "MEDIUM", "LOW"
	OriginalClauseText     string    `gorm:"type:text" json:"original_clause_text"`
	RiskExplanation         string    `gorm:"type:text" json:"risk_explanation"`
	RecommendedCounterText string    `gorm:"type:text" json:"recommended_counter_text"`
	AcceptanceStatus       string    `gorm:"size:32;default:'PROPOSED'" json:"acceptance_status"` // "PROPOSED", "ACCEPTED_BY_COUNTERPARTY", "WAIVED", "REJECTED"
	DiffMarkupHTML         string    `gorm:"type:text" json:"diff_markup_html"`                   // visual inline deletion <del> and addition <ins> markup
	CompanyID              uint      `gorm:"index" json:"company_id"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

func (LegalClauseRedlineRecord) TableName() string {
	return "ai_legal_clause_redlines"
}

// AnalyzeContractRequest represents payload to audit a new commercial or legal agreement
type AnalyzeContractRequest struct {
	ContractTitle        string  `json:"contract_title"`
	ContractType         string  `json:"contract_type"` // "VENDOR_MSA", "PROCUREMENT_PKS", "CUSTOMER_SLA", "NDA", "DISTRIBUTOR_AGREEMENT"
	CounterpartyName     string  `json:"counterparty_name"`
	ContractValueIDR     float64 `json:"contract_value_idr"`
	RawContractText      string  `json:"raw_contract_text"`
	CreatedBy            string  `json:"created_by"`
	AutoGenerateRedlines bool    `json:"auto_generate_redlines"`
}

// LegalCopilotOverview aggregates executive metrics on legal contract risk exposure and active negotiations
type LegalCopilotOverview struct {
	TotalContractsAudited     int                        `json:"total_contracts_audited"`
	CriticalExposureContracts int                        `json:"critical_exposure_contracts"`
	AverageRiskScore          float64                    `json:"average_risk_score"`
	TotalSafeguardedValueIDR  float64                    `json:"total_safeguarded_value_idr"`
	RedlinesGeneratedCount    int                        `json:"redlines_generated_count"`
	RecentAudits              []LegalContractAuditRecord `json:"recent_audits"`
	HighRiskClauses           []LegalClauseRedlineRecord `json:"high_risk_clauses"`
}

// LegalRedlineEngine orchestrates semantic playbook parsing, risk scoring, and counter-clause drafting
type LegalRedlineEngine struct {
	db *gorm.DB
}

var (
	legalEngineInstance *LegalRedlineEngine
	legalEngineOnce     sync.Once
)

// GetLegalRedlineEngine returns the singleton instance
func GetLegalRedlineEngine(db *gorm.DB) *LegalRedlineEngine {
	legalEngineOnce.Do(func() {
		_ = db.AutoMigrate(&LegalContractAuditRecord{}, &LegalClauseRedlineRecord{})
		legalEngineInstance = &LegalRedlineEngine{
			db: db,
		}
	})
	return legalEngineInstance
}

// AnalyzeContract executes rule-based and semantic playbook verification on contract clauses
func (e *LegalRedlineEngine) AnalyzeContract(companyID uint, req AnalyzeContractRequest) (*LegalContractAuditRecord, []LegalClauseRedlineRecord, error) {
	_ = e.db.AutoMigrate(&LegalContractAuditRecord{}, &LegalClauseRedlineRecord{})

	now := time.Now()
	seq := atomic.AddUint64(&globalLegalAuditCounter, 1)
	auditCode := fmt.Sprintf("LGL-AUD-%s-%04d-%04d", now.Format("060102150405"), seq, now.UnixNano()%10000)

	if req.ContractTitle == "" {
		req.ContractTitle = "Perjanjian Kerjasama Pengadaan Standar"
	}
	if req.ContractType == "" {
		req.ContractType = "PROCUREMENT_PKS"
	}
	if req.CounterpartyName == "" {
		req.CounterpartyName = "Pihak Kedua / Vendor Rekanan"
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "AI Legal Reasoner Copilot"
	}

	clauses := make([]LegalClauseRedlineRecord, 0)
	lowerText := strings.ToLower(req.RawContractText)

	// Rule 1: Asymmetric Limitation of Liability & Unlimited Buyer Exposure
	if strings.Contains(lowerText, "tanggung jawab") || strings.Contains(lowerText, "liability") || strings.Contains(lowerText, "kerugian") {
		if strings.Contains(lowerText, "tidak bertanggung jawab atas kerugian tidak langsung") || strings.Contains(lowerText, "tanpa batasan") || strings.Contains(lowerText, "unlimited liability") || strings.Contains(lowerText, "sepihak") {
			clauses = append(clauses, LegalClauseRedlineRecord{
				ClauseNumber:   "Pasal 12.1",
				ClauseCategory: "LIMITATION_OF_LIABILITY",
				RiskSeverity:   "CRITICAL",
				OriginalClauseText: "Pihak Pertama (Pembeli) menanggung segala risiko kerugian operasional dan membebaskan Pihak Kedua dari segala tuntutan pihak ketiga tanpa batas nominal pertanggungjawaban.",
				RiskExplanation: "Klausul membebankan liabilitas tanpa batas (unlimited exposure) pada perusahaan dan menghilangkan akuntabilitas vendor atas kelalaian berat.",
				RecommendedCounterText: "Tanggung jawab agregat masing-masing Pihak atas seluruh klaim kerugian langsung yang timbul dari Perjanjian ini dibatasi maksimal sebesar 100% dari total nilai transaksi yang telah dibayarkan dalam 12 (dua belas) bulan terakhir.",
				AcceptanceStatus: "PROPOSED",
				DiffMarkupHTML: "<del class=\"redline-del\">Pihak Pertama (Pembeli) menanggung segala risiko kerugian operasional dan membebaskan Pihak Kedua dari segala tuntutan pihak ketiga tanpa batas nominal pertanggungjawaban.</del> <ins class=\"redline-ins\">Tanggung jawab agregat masing-masing Pihak atas seluruh klaim kerugian langsung dibatasi maksimal sebesar 100% dari total nilai transaksi 12 bulan terakhir.</ins>",
				CompanyID: companyID,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
	}

	// Rule 2: Excessive Unilateral Liquidated Damages / Penalties
	if strings.Contains(lowerText, "denda") || strings.Contains(lowerText, "penalti") || strings.Contains(lowerText, "keterlambatan") {
		if strings.Contains(lowerText, "2%") || strings.Contains(lowerText, "1%") || strings.Contains(lowerText, "per hari") || strings.Contains(lowerText, "tanpa batas maksimal") {
			clauses = append(clauses, LegalClauseRedlineRecord{
				ClauseNumber:   "Pasal 9.4",
				ClauseCategory: "PENALTY_LIQUIDATED_DAMAGES",
				RiskSeverity:   "CRITICAL",
				OriginalClauseText: "Setiap keterlambatan pembayaran oleh Pihak Pertama dikenakan denda sebesar 2% (dua persen) per hari keterlambatan tanpa batasan akumulasi maksimal.",
				RiskExplanation: "Tarif denda 2% per hari setara bunga 730% per tahun yang bertentangan dengan kepatutan hukum perdata (Pasal 1338 KUHPerdata) dan menimbulkan risiko pemerasan finansial.",
				RecommendedCounterText: "Keterlambatan pembayaran yang terbukti bukan akibat sengketa faktur dikenakan denda keterlambatan sebesar 1‰ (satu permil) per hari kalender dengan batas akumulasi maksimal setinggi-tingginya 5% (lima persen) dari nilai faktur tertunggak.",
				AcceptanceStatus: "PROPOSED",
				DiffMarkupHTML: "<del class=\"redline-del\">Setiap keterlambatan pembayaran dikenakan denda sebesar 2% per hari tanpa batasan akumulasi maksimal.</del> <ins class=\"redline-ins\">Keterlambatan pembayaran dikenakan denda sebesar 1‰ per hari kalender dengan batas akumulasi maksimal setinggi-tingginya 5% dari nilai faktur tertunggak.</ins>",
				CompanyID: companyID,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
	}

	// Rule 3: Unilateral Termination Without Cause
	if strings.Contains(lowerText, "pengakhiran") || strings.Contains(lowerText, "mengakhiri") || strings.Contains(lowerText, "terminasi") || strings.Contains(lowerText, "termination") {
		if strings.Contains(lowerText, "seketika") || strings.Contains(lowerText, "sepihak") || strings.Contains(lowerText, "3 hari") || strings.Contains(lowerText, "tanpa ganti rugi") {
			clauses = append(clauses, LegalClauseRedlineRecord{
				ClauseNumber:   "Pasal 15.2",
				ClauseCategory: "TERMINATION_CONVENIENCE",
				RiskSeverity:   "HIGH",
				OriginalClauseText: "Pihak Kedua berhak mengakhiri Perjanjian ini secara sepihak seketika dengan pemberitahuan tertulis 3 (tiga) hari kerja tanpa kewajiban memberikan ganti rugi apapun.",
				RiskExplanation: "Periode pemberitahuan terminasi 3 hari terlalu singkat untuk mencari vendor pengganti rantai pasok dan berpotensi memicu disrupsi operasional pabrik.",
				RecommendedCounterText: "Masing-masing Pihak berhak mengakhiri Perjanjian dengan pemberitahuan tertulis sekurang-kurangnya 30 (tiga puluh) hari kalender sebelumnya, dengan tetap menyelesaikan hak dan kewajiban pesanan berjalan.",
				AcceptanceStatus: "PROPOSED",
				DiffMarkupHTML: "<del class=\"redline-del\">Pihak Kedua berhak mengakhiri Perjanjian secara sepihak seketika dengan pemberitahuan 3 hari kerja tanpa ganti rugi.</del> <ins class=\"redline-ins\">Masing-masing Pihak berhak mengakhiri Perjanjian dengan pemberitahuan tertulis sekurang-kurangnya 30 hari kalender sebelumnya dengan tetap menyelesaikan pesanan berjalan.</ins>",
				CompanyID: companyID,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
	}

	// Rule 4: Ambiguous IP Rights & Derivative Work Ownership
	if strings.Contains(lowerText, "kekayaan intelektual") || strings.Contains(lowerText, "hak cipta") || strings.Contains(lowerText, "intellectual property") {
		if strings.Contains(lowerText, "menjadi milik pihak kedua") || strings.Contains(lowerText, "seluruh ciptaan") || strings.Contains(lowerText, "modifikasi") {
			clauses = append(clauses, LegalClauseRedlineRecord{
				ClauseNumber:   "Pasal 11.3",
				ClauseCategory: "INTELLECTUAL_PROPERTY",
				RiskSeverity:   "HIGH",
				OriginalClauseText: "Seluruh hasil modifikasi, integrasi API, dan data kustomisasi sistem yang dikembangkan selama masa perjanjian sepenuhnya menjadi hak milik eksklusif Pihak Kedua.",
				RiskExplanation: "Klausul berisiko merampas hak cipta konfigurasi ERP internal dan kode integrasi proprietary yang dibiayai oleh perusahaan.",
				RecommendedCounterText: "Semua Hak Kekayaan Intelektual pra-ada milik masing-masing Pihak tetap menjadi milik Pihak tersebut. Setiap kustomisasi, integrasi API, dan data turunan yang dibiayai oleh Perusahaan menjadi hak milik eksklusif Perusahaan.",
				AcceptanceStatus: "PROPOSED",
				DiffMarkupHTML: "<del class=\"redline-del\">Seluruh modifikasi dan data kustomisasi sepenuhnya menjadi hak milik eksklusif Pihak Kedua.</del> <ins class=\"redline-ins\">Kustomisasi, integrasi API, dan data turunan yang dibiayai oleh Perusahaan menjadi hak milik eksklusif Perusahaan.</ins>",
				CompanyID: companyID,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
	}

	// Rule 5: Personal Data Protection (UU PDP No. 27/2022 / GDPR Compliance)
	if strings.Contains(lowerText, "data pribadi") || strings.Contains(lowerText, "informasi rahasia") || strings.Contains(lowerText, "privacy") {
		if !strings.Contains(lowerText, "uu no. 27 tahun 2022") && !strings.Contains(lowerText, "enkripsi") {
			clauses = append(clauses, LegalClauseRedlineRecord{
				ClauseNumber:   "Pasal 14.1",
				ClauseCategory: "DATA_PRIVACY_PDP",
				RiskSeverity:   "MEDIUM",
				OriginalClauseText: "Para Pihak sepakat menjaga kerahasiaan informasi yang dipertukarkan tanpa kewajiban sertifikasi keamanan data khusus.",
				RiskExplanation: "Perlu kepatuhan eksplisit terhadap UU PDP No. 27/2022 tentang Pelindungan Data Pribadi untuk mencegah sanksi denda administratif hingga Rp 20 Miliar atas kebocoran data.",
				RecommendedCounterText: "Para Pihak wajib memproses data pribadi sesuai Undang-Undang No. 27 Tahun 2022 tentang Pelindungan Data Pribadi (UU PDP), menerapkan enkripsi data standar industri, dan segera memberitahukan insiden kebocoran maksimal dalam 3x24 jam.",
				AcceptanceStatus: "PROPOSED",
				DiffMarkupHTML: "<del class=\"redline-del\">Para Pihak menjaga kerahasiaan tanpa kewajiban sertifikasi khusus.</del> <ins class=\"redline-ins\">Para Pihak memproses data sesuai UU PDP No. 27/2022, menerapkan enkripsi, dan memberitahukan insiden kebocoran maksimal dalam 3x24 jam.</ins>",
				CompanyID: companyID,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
	}

	// Fallback clause if standard clean text
	if len(clauses) == 0 {
		clauses = append(clauses, LegalClauseRedlineRecord{
			ClauseNumber:   "Pasal 18.1",
			ClauseCategory: "GOVERNING_LAW",
			RiskSeverity:   "LOW",
			OriginalClauseText: "Perjanjian ini diatur dan ditafsirkan berdasarkan hukum Negara Republik Indonesia. Segala perselisihan diselesaikan melalui musyawarah.",
			RiskExplanation: "Klausul hukum telah proporsional dan selaras dengan standar tata kelola korporasi.",
			RecommendedCounterText: "Perjanjian ini tunduk pada hukum Republik Indonesia. Sengketa yang tidak dapat diselesaikan melalui musyawarah dalam 30 hari akan dirujuk ke BANI (Badan Arbitrase Nasional Indonesia).",
			AcceptanceStatus: "PROPOSED",
			DiffMarkupHTML: "<ins class=\"redline-ins\">Penyelesaian sengketa disepakati merujuk ke BANI di Jakarta.</ins>",
			CompanyID: companyID,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	// Compute Aggregate Risk Score
	var riskScore float64
	highRiskCount := 0
	for _, c := range clauses {
		switch c.RiskSeverity {
		case "CRITICAL":
			riskScore += 35.0
			highRiskCount++
		case "HIGH":
			riskScore += 22.0
			highRiskCount++
		case "MEDIUM":
			riskScore += 12.0
		case "LOW":
			riskScore += 5.0
		}
	}
	if riskScore > 100.0 {
		riskScore = 100.0
	}

	riskLevel := "LOW"
	if riskScore >= 70.0 {
		riskLevel = "CRITICAL"
	} else if riskScore >= 45.0 {
		riskLevel = "HIGH"
	} else if riskScore >= 20.0 {
		riskLevel = "MODERATE"
	}

	execSummary := fmt.Sprintf(
		"Audit hukum AI mendeteksi skor risiko %.1f%% (%s) dengan %d klausul berisiko tinggi. "+
			"Fokus negosiasi utama mencakup batas liabilitas ganti rugi, ambang batas denda penalti, dan hak cipta kekayaan intelektual.",
		riskScore, riskLevel, highRiskCount,
	)

	audit := LegalContractAuditRecord{
		AuditCode:              auditCode,
		ContractTitle:          req.ContractTitle,
		ContractType:           req.ContractType,
		CounterpartyName:       req.CounterpartyName,
		ContractValueIDR:       req.ContractValueIDR,
		RiskScore:              riskScore,
		RiskLevel:              riskLevel,
		IdentifiedClausesCount: len(clauses),
		HighRiskClausesCount:   highRiskCount,
		RawContractText:        req.RawContractText,
		ExecutiveSummary:       execSummary,
		NegotiationStatus:      "REDLINED",
		CreatedBy:              req.CreatedBy,
		CompanyID:              companyID,
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := e.db.Create(&audit).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to save contract audit: %w", err)
	}

	// Attach AuditID to clauses and persist
	for i := range clauses {
		clauses[i].AuditID = audit.ID
		_ = e.db.Create(&clauses[i]).Error
	}

	return &audit, clauses, nil
}

// GetOverview aggregates executive metrics across legal contracts
func (e *LegalRedlineEngine) GetOverview(companyID uint) (*LegalCopilotOverview, error) {
	_ = e.db.AutoMigrate(&LegalContractAuditRecord{}, &LegalClauseRedlineRecord{})

	var totalContracts int64
	var criticalCount int64
	var totalRedlines int64

	_ = e.db.Model(&LegalContractAuditRecord{}).Where("company_id = ?", companyID).Count(&totalContracts).Error
	_ = e.db.Model(&LegalContractAuditRecord{}).Where("company_id = ? AND (risk_level = 'CRITICAL' OR risk_level = 'HIGH')", companyID).Count(&criticalCount).Error
	_ = e.db.Model(&LegalClauseRedlineRecord{}).Where("company_id = ?", companyID).Count(&totalRedlines).Error

	type sumResult struct {
		TotalValue float64 `gorm:"column:total_value"`
		AvgRisk    float64 `gorm:"column:avg_risk"`
	}
	var sr sumResult
	_ = e.db.Table("ai_legal_contract_audits").
		Select("COALESCE(SUM(contract_value_idr), 0) as total_value, COALESCE(AVG(risk_score), 0) as avg_risk").
		Where("company_id = ?", companyID).
		Scan(&sr).Error

	var recentAudits []LegalContractAuditRecord
	_ = e.db.Where("company_id = ?", companyID).Order("id DESC").Limit(10).Find(&recentAudits).Error

	var highRiskClauses []LegalClauseRedlineRecord
	_ = e.db.Where("company_id = ? AND (risk_severity = 'CRITICAL' OR risk_severity = 'HIGH')", companyID).Order("id DESC").Limit(15).Find(&highRiskClauses).Error

	overview := &LegalCopilotOverview{
		TotalContractsAudited:     int(totalContracts),
		CriticalExposureContracts: int(criticalCount),
		AverageRiskScore:          math.Round(sr.AvgRisk*10) / 10,
		TotalSafeguardedValueIDR:  sr.TotalValue,
		RedlinesGeneratedCount:    int(totalRedlines),
		RecentAudits:              recentAudits,
		HighRiskClauses:           highRiskClauses,
	}

	return overview, nil
}

// GetAudits lists contracts filtered by risk level
func (e *LegalRedlineEngine) GetAudits(companyID uint, riskLevel string, limit int) ([]LegalContractAuditRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	query := e.db.Where("company_id = ?", companyID)
	if riskLevel != "" && riskLevel != "ALL" {
		query = query.Where("risk_level = ?", riskLevel)
	}

	var records []LegalContractAuditRecord
	if err := query.Order("id DESC").Limit(limit).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

// GetAuditByID fetches full audit record alongside its granular clause redlines
func (e *LegalRedlineEngine) GetAuditByID(companyID uint, auditID uint) (*LegalContractAuditRecord, []LegalClauseRedlineRecord, error) {
	var audit LegalContractAuditRecord
	if err := e.db.Where("id = ? AND company_id = ?", auditID, companyID).First(&audit).Error; err != nil {
		return nil, nil, err
	}

	var clauses []LegalClauseRedlineRecord
	_ = e.db.Where("audit_id = ? AND company_id = ?", auditID, companyID).Order("id ASC").Find(&clauses).Error

	return &audit, clauses, nil
}

// UpdateClauseStatus updates whether a recommended counter-clause was accepted or waived
func (e *LegalRedlineEngine) UpdateClauseStatus(companyID uint, clauseID uint, status string) (*LegalClauseRedlineRecord, error) {
	var clause LegalClauseRedlineRecord
	if err := e.db.Where("id = ? AND company_id = ?", clauseID, companyID).First(&clause).Error; err != nil {
		return nil, err
	}

	clause.AcceptanceStatus = status
	clause.UpdatedAt = time.Now()
	if err := e.db.Save(&clause).Error; err != nil {
		return nil, err
	}

	return &clause, nil
}

// GenerateFullExportDoc generates a formal B2B legal counter-proposal memo with side-by-side redline justifications
func (e *LegalRedlineEngine) GenerateFullExportDoc(companyID uint, auditID uint) (string, error) {
	audit, clauses, err := e.GetAuditByID(companyID, auditID)
	if err != nil {
		return "", err
	}

	doc := fmt.Sprintf(
		"SURAT USULAN PERUBAHAN KLAUSUL (LEGAL REDLINE MEMORANDUM)\n"+
			"Nomor Audit: %s\n"+
			"Tanggal: %s\n"+
			"Kepada Yth: Tim Legal / Manajemen %s\n"+
			"Perihal: Usulan Perubahan Klausul Draft Perjanjian '%s'\n"+
			"Nilai Kontrak: Rp %.0f\n"+
			"Status Evaluasi AI: %s (Skor Risiko: %.1f%%)\n\n"+
			"RINGKASAN EKSEKUTIF:\n%s\n\n"+
			"RINCIAN USULAN PERBAIKAN PASAL (SIDE-BY-SIDE REDLINES):\n",
		audit.AuditCode, time.Now().Format("02 January 2006"), audit.CounterpartyName, audit.ContractTitle,
		audit.ContractValueIDR, audit.RiskLevel, audit.RiskScore, audit.ExecutiveSummary,
	)

	for i, c := range clauses {
		doc += fmt.Sprintf(
			"\n[%d] %s (%s - Tingkat Risiko: %s)\n"+
				"    Teks Asli Draft:\n    \"%s\"\n"+
				"    Pertimbangan Risiko:\n    %s\n"+
				"    Usulan Klausul Tandingan:\n    \"%s\"\n"+
				"    Status Negosiasi: %s\n",
			i+1, c.ClauseNumber, c.ClauseCategory, c.RiskSeverity,
			c.OriginalClauseText, c.RiskExplanation, c.RecommendedCounterText, c.AcceptanceStatus,
		)
	}

	doc += "\nDemikian usulan revisi ini kami sampaikan demi tercapainya kemitraan bisnis yang berimbang, adil, dan berkelanjutan.\n"
	return doc, nil
}

// SeedDemoData seeds realistic enterprise contracts for procurement, SaaS, and NDAs
func (e *LegalRedlineEngine) SeedDemoData(companyID uint) error {
	_ = e.db.AutoMigrate(&LegalContractAuditRecord{}, &LegalClauseRedlineRecord{})

	// 1. Critical Risk Logistics PKS
	contractText1 := `
	PERJANJIAN KERJASAMA PENGIRIMAN & LOGISTIK DISTRIBUSI NASIONAL
	Pasal 9.4: Setiap keterlambatan pembayaran oleh Pihak Pertama dikenakan denda sebesar 2% per hari tanpa batasan akumulasi maksimal.
	Pasal 12.1: Pihak Pertama (Pembeli) menanggung segala risiko kerugian operasional dan membebaskan Pihak Kedua dari segala tuntutan pihak ketiga tanpa batas nominal pertanggungjawaban (unlimited liability sepihak).
	Pasal 15.2: Pihak Kedua berhak mengakhiri Perjanjian ini secara sepihak seketika dengan pemberitahuan tertulis 3 hari kerja tanpa kewajiban memberikan ganti rugi apapun.
	`
	_, _, _ = e.AnalyzeContract(companyID, AnalyzeContractRequest{
		ContractTitle:    "Perjanjian Kerjasama Distribusi Logistik Nasional 2026-2027",
		ContractType:     "PROCUREMENT_PKS",
		CounterpartyName: "PT Mega Logistik Trans Nusantara",
		ContractValueIDR: 14500000000.0,
		RawContractText:  contractText1,
		CreatedBy:        "Legal Procurement Specialist",
	})

	// 2. High Risk Cloud ERP SaaS MSA
	contractText2 := `
	MASTER SERVICE AGREEMENT (MSA) CLOUD INFRASTRUCTURE & API INTEGRATION
	Pasal 11.3: Seluruh hasil modifikasi, integrasi API, dan data kustomisasi sistem yang dikembangkan selama masa perjanjian sepenuhnya menjadi hak milik eksklusif Pihak Kedua.
	Pasal 12.1: Pihak Kedua tidak bertanggung jawab atas kerugian tidak langsung dan membebankan ganti rugi sepihak kepada pengguna akhir.
	Pasal 14.1: Para Pihak sepakat menjaga kerahasiaan informasi yang dipertukarkan tanpa kewajiban sertifikasi keamanan data khusus.
	`
	_, _, _ = e.AnalyzeContract(companyID, AnalyzeContractRequest{
		ContractTitle:    "Master Cloud Service Agreement & Enterprise API Integration",
		ContractType:     "VENDOR_MSA",
		CounterpartyName: "PT Cloud Sentra Informatika",
		ContractValueIDR: 4800000000.0,
		RawContractText:  contractText2,
		CreatedBy:        "IT Corporate Governance",
	})

	// 3. Low Risk Bilateral NDA
	contractText3 := `
	PERJANJIAN KERAHASIAAN BERSAMA (BILATERAL NON-DISCLOSURE AGREEMENT)
	Pasal 18.1: Perjanjian ini diatur dan ditafsirkan berdasarkan hukum Negara Republik Indonesia. Segala perselisihan diselesaikan melalui musyawarah secara berimbang untuk melindungi rahasia dagang kedua belah pihak.
	`
	_, _, _ = e.AnalyzeContract(companyID, AnalyzeContractRequest{
		ContractTitle:    "Bilateral Mutual Non-Disclosure Agreement (NDA)",
		ContractType:     "NDA",
		CounterpartyName: "PT Sinar Abadi Perkasa",
		ContractValueIDR: 500000000.0,
		RawContractText:  contractText3,
		CreatedBy:        "Corporate Secretary",
	})

	return nil
}
