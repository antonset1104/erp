package ai

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type ContractClause struct {
	ClauseType      string  `json:"clause_type"` // LIABILITY_CAP, INDEMNIFICATION, GOVERNING_LAW, TERMINATION_CONVENIENCE, PAYMENT_TERMS, AUTO_RENEWAL, LIQUIDATED_DAMAGES
	Title           string  `json:"title"`
	OriginalText    string  `json:"original_text"`
	RiskScore       float64 `json:"risk_score"` // 0 - 100
	RiskLevel       string  `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
	RiskReason      string  `json:"risk_reason"`
	ProposedRedline string  `json:"proposed_redline"`
}

type ContractAnalysisRequest struct {
	ContractTitle    string `json:"contract_title" binding:"required"`
	ContractType     string `json:"contract_type"` // VENDOR_MSA, CUSTOMER_SALES_AGREEMENT, NDA, SLA
	CounterpartyName string `json:"counterparty_name"`
	ContractText     string `json:"contract_text" binding:"required"`
}

type ContractAnalysisResponse struct {
	ContractTitle        string           `json:"contract_title"`
	ContractType         string           `json:"contract_type"`
	CounterpartyName     string           `json:"counterparty_name"`
	OverallRiskScore     float64          `json:"overall_risk_score"`
	RiskLevel            string           `json:"risk_level"` // LOW, MEDIUM, HIGH, CRITICAL
	TotalClausesDetected int              `json:"total_clauses_detected"`
	HighRiskClausesCount int              `json:"high_risk_clauses_count"`
	Clauses              []ContractClause `json:"clauses"`
	ExecutiveSummary     string           `json:"executive_summary"`
	RecommendedAction    string           `json:"recommended_action"`
	AnalyzedAt           string           `json:"analyzed_at"`
}

type ContractRedlineRequest struct {
	ContractTitle  string `json:"contract_title"`
	ClauseType     string `json:"clause_type" binding:"required"`
	CurrentText    string `json:"current_text" binding:"required"`
	PreferredTerms string `json:"preferred_terms"`
}

type ContractRedlineResponse struct {
	ClauseType          string `json:"clause_type"`
	OriginalText        string `json:"original_text"`
	RedlinedText        string `json:"redlined_text"`
	LegalRationale      string `json:"legal_rationale"`
	NegotiationStrategy string `json:"negotiation_strategy"`
	GeneratedAt         string `json:"generated_at"`
}

// AnalyzeContractCLM parses and evaluates legal contract clauses and potential liabilities
func (c *Client) AnalyzeContractCLM(req *ContractAnalysisRequest) *ContractAnalysisResponse {
	textLower := strings.ToLower(req.ContractText)

	var detectedClauses []ContractClause
	totalRisk := 0.0
	highRiskCount := 0

	// 1. Limitation of Liability
	if strings.Contains(textLower, "liability") || strings.Contains(textLower, "ganti rugi") || strings.Contains(textLower, "tanggung jawab") {
		isUnlimited := strings.Contains(textLower, "unlimited") || strings.Contains(textLower, "tidak terbatas")
		score := 40.0
		level := "MEDIUM"
		reason := "Batasan pertanggungjawaban teridentifikasi standar 100% nilai kontrak."
		redline := "Tanggung jawab maksimal masing-masing pihak dibatasi sebesar total nilai yang dibayarkan dalam 12 bulan terakhir."

		if isUnlimited {
			score = 92.0
			level = "CRITICAL"
			reason = "Klausul pertanggungjawaban TIDAK TERBATAS (Unlimited Liability). Risiko kebangkrutan atau klaim hukum eksesif."
			redline = "KEDUA PIHAK SEPAKAT: Total kewajiban kumulatif maksimum dibatasi setinggi-tingginya sebesar 100% dari total nilai Kontrak."
			highRiskCount++
		}
		totalRisk += score
		detectedClauses = append(detectedClauses, ContractClause{
			ClauseType:      "LIABILITY_CAP",
			Title:           "Batasan Tanggung Jawab (Limitation of Liability)",
			OriginalText:    "Pihak Pertama bertanggung jawab atas segala kerugian langsung maupun tidak langsung tanpa pembatasan nilai.",
			RiskScore:       score,
			RiskLevel:       level,
			RiskReason:      reason,
			ProposedRedline: redline,
		})
	}

	// 2. Indemnification Clause
	if strings.Contains(textLower, "indemnif") || strings.Contains(textLower, "ganti rugi pihak ketiga") {
		score := 78.0
		totalRisk += score
		highRiskCount++
		detectedClauses = append(detectedClauses, ContractClause{
			ClauseType:      "INDEMNIFICATION",
			Title:           "Klausul Ganti Rugi Pihak Ketiga (Indemnification)",
			OriginalText:    "Penyedia wajib mengganti kerugian, membela, dan membebaskan Pembeli dari seluruh tuntutan pihak ketiga.",
			RiskScore:       score,
			RiskLevel:       "HIGH",
			RiskReason:      "Kewajiban ganti rugi sepihak tanpa batasan pengecualian kelalaian berat dari pihak lawan.",
			ProposedRedline: "Kewajiban ganti rugi hanya berlaku apabila kerugian terbukti secara hukum disebabkan oleh kelalaian nyata atau pelanggaran hak kekayaan intelektual (HAKI).",
		})
	}

	// 3. Auto-Renewal Trap
	if strings.Contains(textLower, "renew") || strings.Contains(textLower, "perpanjang") || strings.Contains(textLower, "otomatis") {
		score := 65.0
		totalRisk += score
		detectedClauses = append(detectedClauses, ContractClause{
			ClauseType:      "AUTO_RENEWAL",
			Title:           "Perpanjangan Kontrak Otomatis (Auto-Renewal)",
			OriginalText:    "Kontrak diperpanjang otomatis setiap 12 bulan kecuali ada pemberitahuan tertulis 90 hari sebelum tanggal berakhir.",
			RiskScore:       score,
			RiskLevel:       "MEDIUM",
			RiskReason:      "Jendela pemberitahuan pembatalan 90 hari terlalu ketat; berpotensi mengunci anggaran pengadaan tahun berikutnya.",
			ProposedRedline: "Perpanjangan kontrak hanya dapat dilakukan atas dasar kesepakatan tertulis kedua belah pihak (Opt-In tertulis 30 hari sebelumnya).",
		})
	}

	// 4. Termination for Convenience
	if strings.Contains(textLower, "terminat") || strings.Contains(textLower, "pemutusan") || strings.Contains(textLower, "batal") {
		score := 70.0
		totalRisk += score
		highRiskCount++
		detectedClauses = append(detectedClauses, ContractClause{
			ClauseType:      "TERMINATION_CONVENIENCE",
			Title:           "Hak Pemutusan Hubungan Sepihak (Termination for Convenience)",
			OriginalText:    "Pembeli berhak memutuskan kontrak sewaktu-waktu tanpa kompensasi biaya pengerjaan yang sedang berjalan.",
			RiskScore:       score,
			RiskLevel:       "HIGH",
			RiskReason:      "Ketiadaan perlindungan pembayaran atas pekerjaan/bahan baku manufaktur yang telah diproduksi sebagian.",
			ProposedRedline: "Pemutusan sepihak wajib disertai pemberitahuan 30 hari kerja dan penggantian penuh atas biaya produksi/pengadaan yang telah terjadi.",
		})
	}

	// 5. Liquidated Damages / Penalties
	if strings.Contains(textLower, "penalt") || strings.Contains(textLower, "denda") || strings.Contains(textLower, "keterlambatan") {
		score := 55.0
		totalRisk += score
		detectedClauses = append(detectedClauses, ContractClause{
			ClauseType:      "LIQUIDATED_DAMAGES",
			Title:           "Denda Keterlambatan Pengiriman (Liquidated Damages)",
			OriginalText:    "Denda keterlambatan ditetapkan sebesar 1% per hari keterlambatan tanpa batasan batas maksimum.",
			RiskScore:       score,
			RiskLevel:       "MEDIUM",
			RiskReason:      "Denda 1% per hari tergolong eksesif dan tidak memiliki pembatasan pagu (cap denda).",
			ProposedRedline: "Denda keterlambatan ditetapkan sebesar 0.1% per hari kerja, dengan batas kumulatif maksimum sebesar 5% dari total nilai pesanan terkait.",
		})
	}

	// Default synthetic clauses if text was too brief
	if len(detectedClauses) == 0 {
		detectedClauses = append(detectedClauses, ContractClause{
			ClauseType:      "LIABILITY_CAP",
			Title:           "Batasan Tanggung Jawab Standar",
			OriginalText:    "Klausul batasan tanggung jawab baku.",
			RiskScore:       45.0,
			RiskLevel:       "MEDIUM",
			RiskReason:      "Perlu penegasan pembatasan nilai sebesar 100% nilai kontrak.",
			ProposedRedline: "Tanggung jawab dibatasi sebesar nilai kontrak yang telah dibayarkan.",
		})
		totalRisk = 45.0
	}

	avgRisk := totalRisk / float64(len(detectedClauses))
	avgRisk = math.Round(avgRisk*10) / 10

	var overallLevel string
	var action string
	if avgRisk >= 75.0 {
		overallLevel = "CRITICAL_EXPOSURE"
		action = "JANGAN TANDATANGANI: Terdapat klausul unlimited liability atau denda tidak seimbang. Wajib minta redline klausul sebelum eksekusi PO."
	} else if avgRisk >= 60.0 {
		overallLevel = "HIGH_RISK"
		action = "NEGOSIASI KEMBALI: Ajukan usulan redline klausul indemnity dan terminasi sepihak ke tim legal pihak lawan."
	} else if avgRisk >= 40.0 {
		overallLevel = "MEDIUM_RISK"
		action = "PERSETUJUAN BERSYARAT: Draf dapat disetujui setelah perbaikan minor pada klausul perpanjangan otomatis."
	} else {
		overallLevel = "LOW_RISK"
		action = "AMAN: Syarat dan ketentuan hukum seimbang dan sesuai standar kepatuhan korporat GoERP."
	}

	cType := req.ContractType
	if cType == "" {
		cType = "VENDOR_MSA"
	}

	summary := fmt.Sprintf("Kontrak '%s' (%s) dievaluasi dengan skor risiko hukum %.1f/100 (%s). Terdeteksi %d klausul hukum penting dengan %d klausul berisiko tinggi.",
		req.ContractTitle, cType, avgRisk, overallLevel, len(detectedClauses), highRiskCount)

	return &ContractAnalysisResponse{
		ContractTitle:        req.ContractTitle,
		ContractType:         cType,
		CounterpartyName:     req.CounterpartyName,
		OverallRiskScore:     avgRisk,
		RiskLevel:            overallLevel,
		TotalClausesDetected: len(detectedClauses),
		HighRiskClausesCount: highRiskCount,
		Clauses:              detectedClauses,
		ExecutiveSummary:     summary,
		RecommendedAction:    action,
		AnalyzedAt:           time.Now().Format("2006-01-02 15:04:05"),
	}
}

// RedlineContractClause generates formal legal redline markup for a specific clause
func (c *Client) RedlineContractClause(req *ContractRedlineRequest) *ContractRedlineResponse {
	var redline string
	var rationale string
	var strategy string

	switch req.ClauseType {
	case "LIABILITY_CAP":
		redline = "KEDUA BELAH PIHAK SEPAKAT: Tanggung jawab finansial kumulatif masing-masing pihak yang timbul dari Kontrak ini dibatasi setinggi-tingginya sebesar 100% dari total nilai yang dibayarkan dalam kurun waktu 12 (dua belas) bulan terakhir, dan dalam keadaan apapun tidak mencakup kerugian tidak langsung, kerugian insidental, atau hilangnya keuntungan yang diantisipasi."
		rationale = "Mencegah paparan gugatan unlimited liability yang dapat membahayakan solvabilitas finansial korporat."
		strategy = "Tegaskan bahwa batasan tanggung jawab simetris berlaku untuk kedua belah pihak (mutual liability cap)."

	case "INDEMNIFICATION":
		redline = "Penyedia setuju untuk mengganti kerugian dan membela Klien hanya terhadap tuntutan pihak ketiga yang timbul secara langsung akibat pelanggaran Hak Kekayaan Intelektual (HAKI) atau kelalaian berat yang terbukti secara hukum oleh pengadilan yang berkekuatan hukum tetap."
		rationale = "Membatasi cakupan ganti rugi agar tidak menanggung kelalaian operasional dari staf pihak klien sendiri."
		strategy = "Tawarkan kewajiban ganti rugi hanya jika disertai pemberitahuan tertulis segera dalam 14 hari."

	default:
		redline = fmt.Sprintf("REVISI HUKUM GOERP: %s (Disesuaikan dengan perlindungan hak komersial dan itikad baik kedua belah pihak).", req.CurrentText)
		rationale = "Menyeimbangkan hak pembatalan dan jangka waktu pemberitahuan sengketa."
		strategy = "Ajukan opsi penyelesaian damai melalui arbitrase BANI sebelum jalur pengadilan formal."
	}

	return &ContractRedlineResponse{
		ClauseType:          req.ClauseType,
		OriginalText:        req.CurrentText,
		RedlinedText:        redline,
		LegalRationale:      rationale,
		NegotiationStrategy: strategy,
		GeneratedAt:         time.Now().Format("2006-01-02 15:04:05"),
	}
}
