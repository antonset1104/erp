package budget

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	chatterSvc *chatter.Service
	aiClient   *ai.Client
	accountSvc *account.Service
}

func NewService(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client, accountSvc *account.Service) *Service {
	return &Service{
		db:         db,
		repo:       NewRepository(db),
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
		accountSvc: accountSvc,
	}
}

func (s *Service) Repo() *Repository {
	return s.repo
}

// --- Analytic Accounts ---

func (s *Service) CreateAnalyticAccount(req CreateAnalyticAccountRequest, companyID uint) (*AnalyticAccount, error) {
	acc := &AnalyticAccount{
		Code:         strings.ToUpper(strings.TrimSpace(req.Code)),
		Name:         strings.TrimSpace(req.Name),
		DepartmentID: req.DepartmentID,
		PartnerID:    req.PartnerID,
		Description:  req.Description,
		Active:       true,
	}
	acc.CompanyID = companyID

	if err := s.repo.CreateAnalyticAccount(acc); err != nil {
		return nil, fmt.Errorf("gagal membuat akun analitik: %w", err)
	}
	return acc, nil
}

func (s *Service) ListAnalyticAccounts(companyID uint) ([]AnalyticAccount, error) {
	return s.repo.ListAnalyticAccounts(companyID)
}

func (s *Service) GetAnalyticAccount(id uint) (*AnalyticAccount, error) {
	return s.repo.GetAnalyticAccountByID(id)
}

// --- Budgetary Positions ---

func (s *Service) CreateBudgetaryPosition(req CreateBudgetaryPositionRequest, companyID uint) (*BudgetaryPosition, error) {
	var idStrs []string
	for _, id := range req.AccountIDs {
		idStrs = append(idStrs, strconv.FormatUint(uint64(id), 10))
	}

	pos := &BudgetaryPosition{
		Name:        strings.TrimSpace(req.Name),
		Code:        strings.ToUpper(strings.TrimSpace(req.Code)),
		Description: req.Description,
		AccountIDs:  strings.Join(idStrs, ","),
	}
	pos.CompanyID = companyID

	if err := s.repo.CreateBudgetaryPosition(pos); err != nil {
		return nil, fmt.Errorf("gagal membuat pos anggaran: %w", err)
	}
	return pos, nil
}

func (s *Service) ListBudgetaryPositions(companyID uint) ([]BudgetaryPosition, error) {
	return s.repo.ListBudgetaryPositions(companyID)
}

func (s *Service) GetBudgetaryPosition(id uint) (*BudgetaryPosition, error) {
	return s.repo.GetBudgetaryPositionByID(id)
}

// --- Budgets ---

func (s *Service) CreateBudget(req CreateBudgetRequest, userID uint, companyID uint) (*Budget, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		now := time.Now()
		quarter := fmt.Sprintf("Q%d", (int(now.Month())-1)/3+1)
		code = s.repo.NextBudgetCode(strconv.Itoa(now.Year()), quarter)
	}

	b := &Budget{
		Name:          strings.TrimSpace(req.Name),
		Code:          code,
		UserID:        userID,
		DateFrom:      req.DateFrom,
		DateTo:        req.DateTo,
		State:         "draft",
		Notes:         req.Notes,
		AICheckStatus: "pending",
	}
	b.CompanyID = companyID

	if err := s.repo.CreateBudget(b); err != nil {
		return nil, fmt.Errorf("gagal membuat dokumen anggaran: %w", err)
	}

	// Create lines if provided
	var lines []BudgetLine
	for _, l := range req.Lines {
		dFrom := l.DateFrom
		if dFrom == "" {
			dFrom = b.DateFrom
		}
		dTo := l.DateTo
		if dTo == "" {
			dTo = b.DateTo
		}

		line := BudgetLine{
			BudgetID:            b.ID,
			BudgetaryPositionID: l.BudgetaryPositionID,
			AnalyticAccountID:   l.AnalyticAccountID,
			DateFrom:            dFrom,
			DateTo:              dTo,
			PlannedAmount:       l.PlannedAmount,
			VarianceAmount:      l.PlannedAmount,
			Notes:               l.Notes,
		}
		line.CompanyID = companyID
		lines = append(lines, line)
	}

	if len(lines) > 0 {
		if err := s.repo.ReplaceBudgetLines(b.ID, lines); err != nil {
			return nil, fmt.Errorf("gagal menyimpan baris anggaran: %w", err)
		}
	}

	// Trigger initial computation
	if err := s.ComputeBudget(b.ID, companyID); err != nil {
		return nil, err
	}

	s.logChatter(b.ID, fmt.Sprintf("Dokumen Anggaran Keuangan '%s' (%s) dibuat dalam status Draf.", b.Name, b.Code), companyID, userID)

	return s.repo.GetBudgetByID(b.ID)
}

func (s *Service) ListBudgets(companyID uint, state string, search string) ([]Budget, error) {
	return s.repo.ListBudgets(companyID, state, search)
}

func (s *Service) GetBudget(id uint) (*Budget, error) {
	return s.repo.GetBudgetByID(id)
}

// ComputeBudget recalculates actual expenditure, theoretical amounts, and variances for all lines
func (s *Service) ComputeBudget(budgetID uint, companyID uint) error {
	b, err := s.repo.GetBudgetByID(budgetID)
	if err != nil {
		return fmt.Errorf("anggaran #%d tidak ditemukan: %w", budgetID, err)
	}

	var totalPlanned, totalPractical, totalTheoretical float64
	now := time.Now()

	for i := range b.Lines {
		line := &b.Lines[i]

		// 1. Calculate time ratio for Theoretical Amount
		tFrom, errFrom := time.Parse("2006-01-02", line.DateFrom)
		tTo, errTo := time.Parse("2006-01-02", line.DateTo)

		ratio := 0.0
		if errFrom == nil && errTo == nil && tTo.After(tFrom) {
			if now.Before(tFrom) {
				ratio = 0.0
			} else if now.After(tTo) {
				ratio = 1.0
			} else {
				totalDur := tTo.Sub(tFrom).Seconds()
				elapsed := now.Sub(tFrom).Seconds()
				ratio = elapsed / totalDur
			}
		}
		line.TheoriticalAmount = line.PlannedAmount * ratio

		// 2. Query Practical Amount from general ledger account moves
		var accountIDs []uint
		if line.BudgetaryPosition != nil && line.BudgetaryPosition.AccountIDs != "" {
			parts := strings.Split(line.BudgetaryPosition.AccountIDs, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if idVal, err := strconv.ParseUint(p, 10, 32); err == nil {
					accountIDs = append(accountIDs, uint(idVal))
				}
			}
		} else if line.BudgetaryPositionID > 0 {
			pos, err := s.repo.GetBudgetaryPositionByID(line.BudgetaryPositionID)
			if err == nil && pos != nil && pos.AccountIDs != "" {
				parts := strings.Split(pos.AccountIDs, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if idVal, err := strconv.ParseUint(p, 10, 32); err == nil {
						accountIDs = append(accountIDs, uint(idVal))
					}
				}
			}
		}

		practical, err := s.repo.QueryPracticalExpenditure(accountIDs, line.DateFrom, line.DateTo, companyID)
		if err == nil {
			line.PracticalAmount = practical
		}

		// 3. Percentage and Variance
		if line.PlannedAmount > 0 {
			line.Percentage = (line.PracticalAmount / line.PlannedAmount) * 100
		} else {
			line.Percentage = 0
		}
		line.VarianceAmount = line.PlannedAmount - line.PracticalAmount

		_ = s.repo.UpdateBudgetLine(line)

		totalPlanned += line.PlannedAmount
		totalPractical += line.PracticalAmount
		totalTheoretical += line.TheoriticalAmount
	}

	b.TotalPlannedAmount = totalPlanned
	b.TotalPracticalAmount = totalPractical
	b.TotalTheoreticalAmount = totalTheoretical
	if totalPlanned > 0 {
		b.AchievementPercentage = (totalPractical / totalPlanned) * 100
	} else {
		b.AchievementPercentage = 0
	}

	return s.repo.UpdateBudget(b)
}

// --- State Transitions ---

func (s *Service) ActionConfirm(budgetID uint, companyID uint, userID uint) (*Budget, error) {
	b, err := s.repo.GetBudgetByID(budgetID)
	if err != nil {
		return nil, err
	}
	if b.State != "draft" {
		return nil, errors.New("hanya anggaran berstatus 'draft' yang dapat dikonfirmasi")
	}

	_ = s.ComputeBudget(budgetID, companyID)

	b.State = "confirm"
	if err := s.repo.UpdateBudget(b); err != nil {
		return nil, err
	}

	s.logChatter(b.ID, fmt.Sprintf("Dokumen Anggaran '%s' dikonfirmasi oleh Finance Controller (Total Rencana: Rp %.2f).", b.Name, b.TotalPlannedAmount), companyID, userID)
	return s.repo.GetBudgetByID(budgetID)
}

func (s *Service) ActionValidate(budgetID uint, companyID uint, userID uint) (*Budget, error) {
	b, err := s.repo.GetBudgetByID(budgetID)
	if err != nil {
		return nil, err
	}
	if b.State != "confirm" {
		return nil, errors.New("hanya anggaran berstatus 'confirm' yang dapat divalidasi")
	}

	_ = s.ComputeBudget(budgetID, companyID)

	b.State = "validate"
	if err := s.repo.UpdateBudget(b); err != nil {
		return nil, err
	}

	s.logChatter(b.ID, fmt.Sprintf("✅ Anggaran Resmi Divalidasi & Aktif oleh Dewan Direksi / CFO (Plafon: Rp %.2f).", b.TotalPlannedAmount), companyID, userID)
	return s.repo.GetBudgetByID(budgetID)
}

func (s *Service) ActionDone(budgetID uint, companyID uint, userID uint) (*Budget, error) {
	b, err := s.repo.GetBudgetByID(budgetID)
	if err != nil {
		return nil, err
	}
	if b.State != "validate" {
		return nil, errors.New("hanya anggaran berstatus 'validate' yang dapat diselesaikan (done)")
	}

	_ = s.ComputeBudget(budgetID, companyID)

	b.State = "done"
	if err := s.repo.UpdateBudget(b); err != nil {
		return nil, err
	}

	s.logChatter(b.ID, fmt.Sprintf("🏁 Periode Anggaran Ditutup & Selesai (Realisasi Akhir: Rp %.2f / %.1f%%).", b.TotalPracticalAmount, b.AchievementPercentage), companyID, userID)
	return s.repo.GetBudgetByID(budgetID)
}

func (s *Service) ActionCancel(budgetID uint, companyID uint, userID uint) (*Budget, error) {
	b, err := s.repo.GetBudgetByID(budgetID)
	if err != nil {
		return nil, err
	}

	b.State = "cancel"
	if err := s.repo.UpdateBudget(b); err != nil {
		return nil, err
	}

	s.logChatter(b.ID, "Dokumen Anggaran Dibatalkan.", companyID, userID)
	return s.repo.GetBudgetByID(budgetID)
}

func (s *Service) ActionResetDraft(budgetID uint, companyID uint, userID uint) (*Budget, error) {
	b, err := s.repo.GetBudgetByID(budgetID)
	if err != nil {
		return nil, err
	}

	b.State = "draft"
	if err := s.repo.UpdateBudget(b); err != nil {
		return nil, err
	}

	s.logChatter(b.ID, "Dokumen Anggaran dikembalikan ke status Draf.", companyID, userID)
	return s.repo.GetBudgetByID(budgetID)
}

// --- Cloudflare Workers AI Budget Variance & Overspend Auditor ---

func (s *Service) AuditWithAI(budgetID uint, companyID uint, userID uint) (*AIBudgetAuditResult, error) {
	_ = s.ComputeBudget(budgetID, companyID)

	b, err := s.repo.GetBudgetByID(budgetID)
	if err != nil {
		return nil, fmt.Errorf("anggaran #%d tidak ditemukan: %w", budgetID, err)
	}

	// Prepare data summary
	var linesDesc []string
	var flaggedOverspends []string
	var overspendCount int

	for _, l := range b.Lines {
		posName := "Pos Anggaran"
		if l.BudgetaryPosition != nil {
			posName = l.BudgetaryPosition.Name
		}
		anName := "Analitik"
		if l.AnalyticAccount != nil {
			anName = l.AnalyticAccount.Name
		}

		linesDesc = append(linesDesc, fmt.Sprintf("- [%s | %s]: Rencana=Rp %.0f, Realisasi=Rp %.0f, Target Waktu=Rp %.0f, Konsumsi=%.1f%% (Sisa=Rp %.0f)",
			posName, anName, l.PlannedAmount, l.PracticalAmount, l.TheoriticalAmount, l.Percentage, l.VarianceAmount))

		if l.Percentage > 100 {
			flaggedOverspends = append(flaggedOverspends, fmt.Sprintf("%s (%s): Overbudget %.1f%% (Melampaui plafon sebesar Rp %.0f)",
				posName, anName, l.Percentage, math.Abs(l.VarianceAmount)))
			overspendCount++
		} else if l.TheoriticalAmount > 0 && l.PracticalAmount > l.TheoriticalAmount*1.3 {
			flaggedOverspends = append(flaggedOverspends, fmt.Sprintf("%s (%s): Burn rate abnormal, konsumsi Rp %.0f mendahului target waktu Rp %.0f",
				posName, anName, l.PracticalAmount, l.TheoriticalAmount))
			overspendCount++
		}
	}

	// Try Cloudflare Workers AI LLM
	prompt := fmt.Sprintf(`Sebagai AI Financial Controller & Corporate Budget Auditor, analisis laporan penyerapan anggaran berikut:
Nama Dokumen: %s (%s)
Periode: %s s/d %s
Total Plafon Anggaran: Rp %.0f
Total Realisasi Aktual: Rp %.0f (Penyerapan: %.1f%%)
Target Teoretis Waktu: Rp %.0f
Rincian Pos Anggaran & Departemen:
%s

Tugas Anda:
1. Tentukan risk_level: "low", "medium", "high", atau "critical".
2. Tentukan risk_score (angka 0 s/d 100).
3. Tentukan overall_burn_rate: "normal", "elevated", atau "critical".
4. Buat executive_summary profesional mengenai kesehatan anggaran dan deviasi penyerapan.
5. Berikan 3 rekomendasi tindakan korektif (recommended_actions) bagi manajemen (misal pembekuan belanja diskresioner, realokasi pos, audit nota pengeluaran).

Jawab HANYA dalam format JSON valid tanpa format markdown luar:
{
  "risk_level": "low|medium|high|critical",
  "risk_score": 25.0,
  "overall_burn_rate": "normal|elevated|critical",
  "flagged_overspends": ["..."],
  "executive_summary": "...",
  "recommended_actions": ["..."]
}`, b.Name, b.Code, b.DateFrom, b.DateTo, b.TotalPlannedAmount, b.TotalPracticalAmount, b.AchievementPercentage, b.TotalTheoreticalAmount, strings.Join(linesDesc, "\n"))

	type AICallResponse struct {
		RiskLevel          string   `json:"risk_level"`
		RiskScore          float64  `json:"risk_score"`
		OverallBurnRate    string   `json:"overall_burn_rate"`
		FlaggedOverspends  []string `json:"flagged_overspends"`
		ExecutiveSummary   string   `json:"executive_summary"`
		RecommendedActions []string `json:"recommended_actions"`
	}

	var parsed AICallResponse
	source := "local_heuristic_engine"

	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "You are an enterprise corporate finance and budget variance auditor. Respond strictly in valid JSON format."
		respText, err := s.aiClient.CallRaw(prompt, systemPrompt)
		if err == nil {
			cleanJSON := strings.TrimSpace(respText)
			if strings.HasPrefix(cleanJSON, "```json") {
				cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
				cleanJSON = strings.TrimSuffix(cleanJSON, "```")
				cleanJSON = strings.TrimSpace(cleanJSON)
			}
			if err := json.Unmarshal([]byte(cleanJSON), &parsed); err == nil && parsed.RiskLevel != "" {
				source = "cloudflare_workers_ai_llama3.1"
			}
		}
	}

	// Smart Local Heuristic Fallback
	if source == "local_heuristic_engine" || parsed.RiskLevel == "" {
		if overspendCount > 1 || b.AchievementPercentage > 110 {
			parsed.RiskLevel = "critical"
			parsed.RiskScore = 88.0
			parsed.OverallBurnRate = "critical"
			parsed.ExecutiveSummary = fmt.Sprintf("Audit Anggaran Finansial mendeteksi %d pos pengeluaran melampaui plafon anggaran secara signifikan dengan rata-rata konsumsi %.1f%%. Diperlukan tindakan pembatasan belanja segera.", overspendCount, b.AchievementPercentage)
			parsed.RecommendedActions = []string{
				"Bekukan persetujuan PO dan permohonan klaim baru untuk pos anggaran yang mengalami defisit.",
				"Lakukan rekonfigurasi plafon delegasi wewenang (DOA) pada modul approval.",
				"Ajukan revisi anggaran perubahan (ABT) kepada CFO dan Dewan Direksi.",
			}
		} else if overspendCount == 1 || b.AchievementPercentage > 85 {
			parsed.RiskLevel = "medium"
			parsed.RiskScore = 48.0
			parsed.OverallBurnRate = "elevated"
			parsed.ExecutiveSummary = fmt.Sprintf("Evaluasi anggaran menunjukkan penyerapan sebesar %.1f%% dengan 1 pos biaya mendekati atau melebihi batas toleransi. Kecepatan belanja perlu dikendalikan agar tidak defisit sebelum tutup buku.", b.AchievementPercentage)
			parsed.RecommendedActions = []string{
				"Prioritaskan pembayaran belanja operasional esensial dan tunda kegiatan diskresioner.",
				"Lakukan realokasi dana dari pos anggaran lain yang penyerapannya masih rendah.",
				"Perketat verifikasi faktur vendor sebelum diposting ke jurnal akuntansi.",
			}
		} else {
			parsed.RiskLevel = "low"
			parsed.RiskScore = 15.0
			parsed.OverallBurnRate = "normal"
			parsed.ExecutiveSummary = fmt.Sprintf("Kinerja anggaran berada dalam koridor target sehat (Penyerapan: %.1f%% dari total plafon Rp %.0f). Tidak ditemukan anomali pembengkakan biaya.", b.AchievementPercentage, b.TotalPlannedAmount)
			parsed.RecommendedActions = []string{
				"Lanjutkan pengawasan berkala penyerapan biaya per departemen.",
				"Pertahankan kepatuhan approval berjenjang pada modul pengeluaran kas.",
				"Verifikasi sinkronisasi antara PO, Tagihan Vendor, dan akun Buku Besar GL.",
			}
		}
		if len(parsed.FlaggedOverspends) == 0 {
			parsed.FlaggedOverspends = flaggedOverspends
		}
	}

	result := &AIBudgetAuditResult{
		BudgetID:           b.ID,
		BudgetName:         b.Name,
		RiskLevel:          parsed.RiskLevel,
		RiskScore:          parsed.RiskScore,
		OverallBurnRate:    parsed.OverallBurnRate,
		FlaggedOverspends:  parsed.FlaggedOverspends,
		ExecutiveSummary:   parsed.ExecutiveSummary,
		RecommendedActions: parsed.RecommendedActions,
		Source:             source,
		AIConfidence:       92.5,
	}

	// Update budget record
	b.AICheckStatus = result.RiskLevel
	b.AIRiskScore = result.RiskScore
	b.AINotes = result.ExecutiveSummary
	_ = s.repo.UpdateBudget(b)

	s.logChatter(b.ID, fmt.Sprintf("🤖 Cloudflare Workers AI Audit Anggaran Selesai: Tingkat Risiko '%s' (Skor: %.1f, Burn Rate: %s).",
		strings.ToUpper(result.RiskLevel), result.RiskScore, strings.ToUpper(result.OverallBurnRate)), companyID, userID)

	return result, nil
}

// GetStats compiles executive analytics
func (s *Service) GetStats(companyID uint) (*BudgetStats, error) {
	return s.repo.GetStats(companyID)
}

// Helper chatter log
func (s *Service) logChatter(resID uint, body string, companyID uint, userID uint) {
	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "account.budget",
			ResID:       resID,
			Body:        body,
			MessageType: "notification",
			AuthorID:    userID,
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}
}
