package currency

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	accountSvc *account.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(db *gorm.DB, repo *Repository, accountSvc *account.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// -------------------------------------------------------------
// CURRENCY & RATES MANAGEMENT
// -------------------------------------------------------------

func (s *Service) ListCurrencies(activeOnly bool) ([]Currency, error) {
	return s.repo.ListCurrencies(activeOnly)
}

func (s *Service) GetCurrencyByID(id uint) (*Currency, error) {
	return s.repo.GetCurrencyByID(id)
}

func (s *Service) GetCurrencyByName(name string) (*Currency, error) {
	return s.repo.GetCurrencyByName(name)
}

func (s *Service) GetBaseCurrency() (*Currency, error) {
	return s.repo.GetBaseCurrency()
}

func (s *Service) CreateCurrency(req CreateCurrencyRequest, companyID uint) (*Currency, error) {
	name := strings.ToUpper(strings.TrimSpace(req.Name))
	if name == "" {
		return nil, errors.New("kode mata uang wajib diisi (e.g. USD)")
	}

	existing, _ := s.repo.GetCurrencyByName(name)
	if existing != nil {
		return nil, fmt.Errorf("mata uang '%s' sudah terdaftar", name)
	}

	curr := &Currency{
		Name:          name,
		Symbol:        req.Symbol,
		FullName:      req.FullName,
		DecimalPlaces: req.DecimalPlaces,
		Rounding:      req.Rounding,
		Active:        true,
		IsBase:        req.IsBase,
	}
	if curr.DecimalPlaces == 0 && name != "JPY" {
		curr.DecimalPlaces = 2
	}
	if curr.Rounding == 0 {
		curr.Rounding = 0.01
	}

	if err := s.repo.CreateCurrency(curr); err != nil {
		return nil, err
	}
	return curr, nil
}

func (s *Service) CreateRate(req CreateRateRequest, companyID uint) (*CurrencyRate, error) {
	curr, err := s.repo.GetCurrencyByID(req.CurrencyID)
	if err != nil {
		return nil, errors.New("mata uang tidak ditemukan")
	}

	if curr.IsBase {
		return nil, errors.New("mata uang basis perusahaan (IDR) selalu memiliki nilai kurs 1.0")
	}

	if req.DirectRate <= 0 {
		return nil, errors.New("nilai kurs pasar (direct rate) harus lebih besar dari 0")
	}

	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	rateObj := &CurrencyRate{
		CurrencyID: req.CurrencyID,
		Name:       dateStr,
		DirectRate: req.DirectRate,
		Rate:       1.0 / req.DirectRate, // Standard Odoo format
		Source:     req.Source,
		CompanyID:  companyID,
	}
	if rateObj.Source == "" {
		rateObj.Source = "Bank Indonesia JISDOR"
	}

	if err := s.repo.CreateRate(rateObj); err != nil {
		return nil, err
	}
	return rateObj, nil
}

func (s *Service) ListRates(currencyID uint, companyID uint) ([]CurrencyRate, error) {
	return s.repo.ListRates(currencyID, companyID)
}

func (s *Service) GetLatestRate(currencyID uint, date string, companyID uint) (float64, error) {
	curr, err := s.repo.GetCurrencyByID(currencyID)
	if err != nil {
		return 0, errors.New("mata uang tidak ditemukan")
	}
	if curr.IsBase {
		return 1.0, nil
	}

	rate, err := s.repo.GetLatestRate(currencyID, date, companyID)
	if err != nil {
		return 0, fmt.Errorf("tidak ada data kurs aktif untuk mata uang '%s' pada tanggal %s", curr.Name, date)
	}
	return rate.DirectRate, nil
}

// -------------------------------------------------------------
// CURRENCY CONVERSION ENGINE
// -------------------------------------------------------------

func (s *Service) Convert(amount float64, fromCurID, toCurID uint, date string, companyID uint) (*CurrencyConversionResponse, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	fromCur, err := s.repo.GetCurrencyByID(fromCurID)
	if err != nil {
		return nil, errors.New("mata uang asal tidak ditemukan")
	}
	toCur, err := s.repo.GetCurrencyByID(toCurID)
	if err != nil {
		return nil, errors.New("mata uang tujuan tidak ditemukan")
	}

	if fromCurID == toCurID {
		return &CurrencyConversionResponse{
			OriginalAmount:  amount,
			FromCurrency:    fromCur.Name,
			ToCurrency:      toCur.Name,
			ConvertedAmount: amount,
			AppliedRate:     1.0,
			RateDate:        date,
		}, nil
	}

	var fromDirectRate float64 = 1.0
	if !fromCur.IsBase {
		fromDirectRate, err = s.GetLatestRate(fromCurID, date, companyID)
		if err != nil {
			return nil, err
		}
	}

	var toDirectRate float64 = 1.0
	if !toCur.IsBase {
		toDirectRate, err = s.GetLatestRate(toCurID, date, companyID)
		if err != nil {
			return nil, err
		}
	}

	// 1. Convert from source to Base IDR
	baseAmount := amount * fromDirectRate

	// 2. Convert from Base IDR to target currency
	convertedAmount := baseAmount / toDirectRate
	appliedRate := fromDirectRate / toDirectRate

	return &CurrencyConversionResponse{
		OriginalAmount:  amount,
		FromCurrency:    fromCur.Name,
		ToCurrency:      toCur.Name,
		ConvertedAmount: math.Round(convertedAmount*100) / 100,
		AppliedRate:     appliedRate,
		RateDate:        date,
	}, nil
}

// -------------------------------------------------------------
// UNREALIZED FOREX REVALUATION (PSAK 10 / IAS 21)
// -------------------------------------------------------------

func (s *Service) ComputeUnrealizedRevaluation(req ComputeRevaluationRequest, companyID, userID uint) (*ForexRevaluation, error) {
	curr, err := s.repo.GetCurrencyByID(req.CurrencyID)
	if err != nil {
		return nil, errors.New("mata uang revaluasi tidak ditemukan")
	}
	if curr.IsBase {
		return nil, errors.New("mata uang basis (IDR) tidak dapat direvaluasi")
	}

	if req.ClosingRate <= 0 {
		return nil, errors.New("kurs penutupan (closing rate) harus lebih besar dari 0")
	}

	if req.RevaluationDate == "" {
		req.RevaluationDate = time.Now().Format("2006-01-02")
	}

	// Scan open posted AccountMove lines that have foreign currency exposure
	var openMoves []account.AccountMove
	err = s.db.Preload("Partner").
		Preload("LineIDs.Account").
		Where("company_id = ? AND state = ? AND payment_state != ? AND amount_residual > 0", companyID, "posted", "paid").
		Where("currency_id = ?", req.CurrencyID).
		Find(&openMoves).Error
	if err != nil {
		return nil, fmt.Errorf("gagal memindai invoice terbuka: %w", err)
	}

	var lines []ForexRevaluationLine
	var totalGain, totalLoss float64

	for _, move := range openMoves {
		if move.CurrencyRate <= 0 {
			move.CurrencyRate = 1.0
		}

		// Calculate open foreign amount
		foreignAmt := move.AmountCurrency
		if foreignAmt <= 0 && move.CurrencyRate > 0 {
			foreignAmt = move.AmountResidual / move.CurrencyRate
		} else if move.AmountTotal > 0 {
			ratio := move.AmountResidual / move.AmountTotal
			foreignAmt = move.AmountCurrency * ratio
		}

		if foreignAmt <= 0 {
			continue
		}

		// Historical book value in IDR
		histAmt := foreignAmt * move.CurrencyRate
		// Revalued value at closing spot rate in IDR
		revalAmt := foreignAmt * req.ClosingRate

		for _, ml := range move.LineIDs {
			isAR := ml.Account.Type == "asset_receivable" && move.MoveType == "out_invoice"
			isAP := ml.Account.Type == "liability_payable" && move.MoveType == "in_invoice"

			if !isAR && !isAP {
				continue
			}

			var gainLoss float64
			var glType string

			if isAR {
				// Piutang: jika kurs naik -> untung (Gain)
				gainLoss = revalAmt - histAmt
			} else {
				// Hutang: jika kurs naik -> rugi (Loss), jika kurs turun -> untung (Gain)
				gainLoss = histAmt - revalAmt
			}

			if gainLoss >= 0 {
				glType = "gain"
				totalGain += gainLoss
			} else {
				glType = "loss"
				totalLoss += math.Abs(gainLoss)
			}

			lines = append(lines, ForexRevaluationLine{
				MoveID:           move.ID,
				MoveName:         move.Name,
				MoveType:         move.MoveType,
				MoveLineID:       ml.ID,
				PartnerID:        move.PartnerID,
				PartnerName:      move.Partner.Name,
				AccountID:        ml.AccountID,
				AccountCode:      ml.Account.Code,
				AccountName:      ml.Account.Name,
				AccountType:      ml.Account.Type,
				AmountCurrency:   math.Round(foreignAmt*100) / 100,
				HistoricalRate:   move.CurrencyRate,
				HistoricalAmount: math.Round(histAmt*100) / 100,
				ClosingRate:      req.ClosingRate,
				RevaluedAmount:   math.Round(revalAmt*100) / 100,
				GainLoss:         math.Round(gainLoss*100) / 100,
				GainLossType:     glType,
			})
		}
	}

	netGainLoss := totalGain - totalLoss
	docName := s.repo.GetNextRevaluationSequence(companyID)

	rev := &ForexRevaluation{
		Name:            docName,
		RevaluationDate: req.RevaluationDate,
		CurrencyID:      req.CurrencyID,
		ClosingRate:     req.ClosingRate,
		State:           "draft",
		TotalGain:       math.Round(totalGain*100) / 100,
		TotalLoss:       math.Round(totalLoss*100) / 100,
		NetGainLoss:     math.Round(netGainLoss*100) / 100,
		Notes:           req.Notes,
		Lines:           lines,
		CompanyID:       companyID,
		UserID:          userID,
	}

	if err := s.repo.CreateRevaluation(rev); err != nil {
		return nil, fmt.Errorf("gagal menyimpan revaluasi valas: %w", err)
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"account.multicurrency.revaluation",
			rev.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Draf revaluasi kurs valas %s dibuat. %d pos terbuka direvaluasi pada kurs Rp %.2f (Total Laba: Rp %.2f, Total Rugi: Rp %.2f, Net: Rp %.2f).",
				curr.Name, len(lines), req.ClosingRate, totalGain, totalLoss, netGainLoss),
			"revaluation:draft",
			companyID,
		)
	}

	return s.repo.GetRevaluationByID(rev.ID, companyID)
}

func (s *Service) PostRevaluation(revaluationID uint, companyID, userID uint) (*ForexRevaluation, error) {
	rev, err := s.repo.GetRevaluationByID(revaluationID, companyID)
	if err != nil {
		return nil, errors.New("dokumen revaluasi valas tidak ditemukan")
	}

	if rev.State == "posted" {
		return nil, errors.New("revaluasi kurs ini sudah diposting ke General Ledger")
	}
	if rev.State == "cancel" {
		return nil, errors.New("tidak dapat memposting dokumen revaluasi yang dibatalkan")
	}

	if len(rev.Lines) == 0 {
		return nil, errors.New("tidak ada baris piutang/hutang valas untuk direvaluasi")
	}

	// 1. Ensure Forex Gain & Loss Accounts exist
	gainAcc := s.ensureAccount("710110", "Laba Selisih Kurs Belum Terealisasi (Unrealized Forex Gain)", "income", companyID)
	lossAcc := s.ensureAccount("620110", "Rugi Selisih Kurs Belum Terealisasi (Unrealized Forex Loss)", "expense", companyID)

	// 2. Find Miscellaneous Journal (MISC)
	var miscJournal account.AccountJournal
	if err := s.db.Where("company_id = ? AND type = ?", companyID, "general").First(&miscJournal).Error; err != nil {
		if err := s.db.Where("company_id = ?", companyID).First(&miscJournal).Error; err != nil {
			return nil, errors.New("jurnal umum (general/misc) tidak ditemukan")
		}
	}

	// 3. Construct Balanced General Ledger Move
	moveLines := make([]account.AccountMoveLine, 0)
	var totalDebit, totalCredit float64

	for _, l := range rev.Lines {
		diff := math.Abs(l.GainLoss)
		if diff <= 0.01 {
			continue
		}

		label := fmt.Sprintf("Penyesuaian Kurs Akhir Periode %s (%s) - %s @ Rp %.0f -> Rp %.0f",
			rev.Currency.Name, l.MoveName, l.PartnerName, l.HistoricalRate, l.ClosingRate)

		partnerIDVal := l.PartnerID

		if l.AccountType == "asset_receivable" {
			if l.GainLossType == "gain" {
				// Piutang bertambah (Debit AR, Credit Forex Gain)
				moveLines = append(moveLines, account.AccountMoveLine{
					AccountID: l.AccountID,
					PartnerID: &partnerIDVal,
					Name:      label,
					Debit:     diff,
					Credit:    0,
					Balance:   diff,
				})
				moveLines = append(moveLines, account.AccountMoveLine{
					AccountID: gainAcc.ID,
					Name:      label,
					Debit:     0,
					Credit:    diff,
					Balance:   -diff,
				})
				totalDebit += diff
				totalCredit += diff
			} else {
				// Piutang berkurang (Debit Forex Loss, Credit AR)
				moveLines = append(moveLines, account.AccountMoveLine{
					AccountID: lossAcc.ID,
					Name:      label,
					Debit:     diff,
					Credit:    0,
					Balance:   diff,
				})
				moveLines = append(moveLines, account.AccountMoveLine{
					AccountID: l.AccountID,
					PartnerID: &partnerIDVal,
					Name:      label,
					Debit:     0,
					Credit:    diff,
					Balance:   -diff,
				})
				totalDebit += diff
				totalCredit += diff
			}
		} else if l.AccountType == "liability_payable" {
			if l.GainLossType == "gain" {
				// Hutang berkurang (Debit AP, Credit Forex Gain)
				moveLines = append(moveLines, account.AccountMoveLine{
					AccountID: l.AccountID,
					PartnerID: &partnerIDVal,
					Name:      label,
					Debit:     diff,
					Credit:    0,
					Balance:   diff,
				})
				moveLines = append(moveLines, account.AccountMoveLine{
					AccountID: gainAcc.ID,
					Name:      label,
					Debit:     0,
					Credit:    diff,
					Balance:   -diff,
				})
				totalDebit += diff
				totalCredit += diff
			} else {
				// Hutang bertambah (Debit Forex Loss, Credit AP)
				moveLines = append(moveLines, account.AccountMoveLine{
					AccountID: lossAcc.ID,
					Name:      label,
					Debit:     diff,
					Credit:    0,
					Balance:   diff,
				})
				moveLines = append(moveLines, account.AccountMoveLine{
					AccountID: l.AccountID,
					PartnerID: &partnerIDVal,
					Name:      label,
					Debit:     0,
					Credit:    diff,
					Balance:   -diff,
				})
				totalDebit += diff
				totalCredit += diff
			}
		}
	}

	if len(moveLines) == 0 {
		return nil, errors.New("tidak ada selisih kurs yang perlu dibukukan ke General Ledger")
	}

	glMove := &account.AccountMove{
		MoveType:       "entry",
		JournalID:      miscJournal.ID,
		PartnerID:      rev.Lines[0].PartnerID,
		InvoiceDate:    rev.RevaluationDate,
		Ref:            rev.Name,
		Note:           fmt.Sprintf("Jurnal Penyesuaian Revaluasi Kurs Valas (%s) per %s", rev.Currency.Name, rev.RevaluationDate),
		AmountUntaxed:  totalDebit,
		AmountTotal:    totalDebit,
		AmountResidual: 0,
		State:          "draft",
		PaymentState:   "paid",
		LineIDs:        moveLines,
	}
	glMove.CompanyID = companyID

	if err := s.accountSvc.CreateMove(glMove); err != nil {
		return nil, fmt.Errorf("gagal membuat jurnal revaluasi: %w", err)
	}

	postedMove, err := s.accountSvc.ActionPost(glMove.ID, companyID)
	if err != nil {
		return nil, fmt.Errorf("gagal memposting jurnal revaluasi: %w", err)
	}

	rev.JournalEntryID = &postedMove.ID
	rev.State = "posted"
	if err := s.repo.UpdateRevaluation(rev); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"account.multicurrency.revaluation",
			rev.ID,
			userID,
			"Pengguna",
			fmt.Sprintf("Revaluasi kurs diposting ke General Ledger! Jurnal '%s' berhasil diterbitkan (Debit/Kredit: Rp %.2f).",
				postedMove.Name, totalDebit),
			"revaluation:posted",
			companyID,
		)
	}

	return s.repo.GetRevaluationByID(rev.ID, companyID)
}

func (s *Service) CancelRevaluation(revaluationID uint, companyID, userID uint) (*ForexRevaluation, error) {
	rev, err := s.repo.GetRevaluationByID(revaluationID, companyID)
	if err != nil {
		return nil, errors.New("dokumen revaluasi valas tidak ditemukan")
	}

	if rev.State == "posted" {
		return nil, errors.New("revaluasi yang telah diposting ke jurnal buku besar tidak dapat dibatalkan langsung")
	}

	rev.State = "cancel"
	if err := s.repo.UpdateRevaluation(rev); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		_ = s.chatterSvc.LogTracking(
			"account.multicurrency.revaluation",
			rev.ID,
			userID,
			"Pengguna",
			"Dokumen revaluasi valas dibatalkan.",
			"revaluation:cancel",
			companyID,
		)
	}

	return rev, nil
}

func (s *Service) GetRevaluationByID(id uint, companyID uint) (*ForexRevaluation, error) {
	return s.repo.GetRevaluationByID(id, companyID)
}

func (s *Service) ListRevaluations(companyID uint, state string) ([]ForexRevaluation, error) {
	return s.repo.ListRevaluations(companyID, state)
}

// -------------------------------------------------------------
// REALIZED FOREX GAIN / LOSS ENGINE (SETTLEMENT)
// -------------------------------------------------------------

func (s *Service) CalculateRealizedGainLoss(req RealizedGainLossCalculationRequest, companyID uint) (*RealizedGainLossResult, error) {
	move, err := s.accountSvc.GetMove(req.MoveID, companyID)
	if err != nil {
		return nil, errors.New("dokumen faktur tidak ditemukan")
	}

	if move.CurrencyID == nil {
		return nil, errors.New("faktur ini dicatat dalam mata uang basis lokal (tidak ada selisih kurs valas)")
	}

	curr, err := s.repo.GetCurrencyByID(*move.CurrencyID)
	if err != nil {
		return nil, errors.New("mata uang faktur tidak ditemukan")
	}

	histRate := move.CurrencyRate
	if histRate <= 0 {
		histRate = 1.0
	}

	foreignPaid := req.PaymentAmount
	histIDR := foreignPaid * histRate
	paidIDR := foreignPaid * req.PaymentRate

	var gainLoss float64
	var glType string
	var accCode, accName string

	if move.MoveType == "out_invoice" {
		// Piutang Pelanggan: Jika kurs bayar > kurs faktur -> Laba (Gain)
		gainLoss = paidIDR - histIDR
	} else {
		// Hutang Pemasok: Jika kurs bayar < kurs faktur -> Laba (Gain)
		gainLoss = histIDR - paidIDR
	}

	if gainLoss >= 0 {
		glType = "gain"
		accCode = "710100"
		accName = "Pendapatan Laba Selisih Kurs Terealisasi (Realized Forex Gain)"
	} else {
		glType = "loss"
		accCode = "620100"
		accName = "Beban Kerugian Selisih Kurs Terealisasi (Realized Forex Loss)"
	}

	return &RealizedGainLossResult{
		MoveID:             move.ID,
		MoveName:           move.Name,
		CurrencyName:       curr.Name,
		ForeignAmountPaid:  math.Round(foreignPaid*100) / 100,
		HistoricalRate:     histRate,
		PaymentRate:        req.PaymentRate,
		HistoricalIDR:      math.Round(histIDR*100) / 100,
		ActualPaidIDR:      math.Round(paidIDR*100) / 100,
		RealizedGainLoss:   math.Round(math.Abs(gainLoss)*100) / 100,
		GainLossType:       glType,
		JournalAccountCode: accCode,
		JournalAccountName: accName,
	}, nil
}

// -------------------------------------------------------------
// CLOUDFLARE WORKERS AI FOREX RISK & CURRENCY EXPOSURE COPILOT
// -------------------------------------------------------------

func (s *Service) ForexRiskAIAudit(currencyID uint, companyID uint) (*ForexRiskReport, error) {
	curr, err := s.repo.GetCurrencyByID(currencyID)
	if err != nil {
		return nil, errors.New("mata uang tidak ditemukan")
	}

	spotRate, _ := s.GetLatestRate(currencyID, "", companyID)
	if spotRate <= 0 {
		spotRate = 16000.0 // Default sensible fallback
	}

	// 1. Calculate open receivables (AR) in foreign currency
	var openAR float64
	s.db.Model(&account.AccountMove{}).
		Where("company_id = ? AND currency_id = ? AND move_type = ? AND state = ? AND payment_state != ?",
			companyID, currencyID, "out_invoice", "posted", "paid").
		Select("COALESCE(SUM(amount_currency), 0)").
		Scan(&openAR)

	// 2. Calculate open payables (AP) in foreign currency
	var openAP float64
	s.db.Model(&account.AccountMove{}).
		Where("company_id = ? AND currency_id = ? AND move_type = ? AND state = ? AND payment_state != ?",
			companyID, currencyID, "in_invoice", "posted", "paid").
		Select("COALESCE(SUM(amount_currency), 0)").
		Scan(&openAP)

	netExposureFC := openAR - openAP
	netExposureIDR := netExposureFC * spotRate

	shock2Pct := netExposureIDR * 0.02
	shock5Pct := netExposureIDR * 0.05
	shock10Pct := netExposureIDR * 0.10

	riskLevel := "low"
	if math.Abs(netExposureIDR) > 100000000 {
		riskLevel = "high"
	} else if math.Abs(netExposureIDR) > 25000000 {
		riskLevel = "medium"
	}

	prompt := fmt.Sprintf(`Anda adalah CFO & Senior FX Treasury Risk Advisor Enterprise.
Lakukan analisis eksposur risiko valas (Foreign Exchange Exposure & Sensitivity Analysis):
- Mata Uang Asing: %s
- Kurs Spot Hari Ini: Rp %.2f / %s
- Piutang Dagang Terbuka (AR Valas): %.2f %s (Rp %.2f)
- Hutang Dagang Terbuka (AP Valas): %.2f %s (Rp %.2f)
- Posisi Eksposur Bersih (Net Open Position): %.2f %s (Rp %.2f)
- Sensitivitas Depresiasi Rupiah +2%%: Dampak Rp %.2f
- Sensitivitas Depresiasi Rupiah +5%%: Dampak Rp %.2f
- Tingkat Risiko: %s

Berikan respons JSON murni dengan format:
{
  "risk_level": "%s",
  "executive_summary": "Ringkasan analisis eksposur valas dan dampak terhadap arus kas operasional",
  "action_recommendations": [
    "Rekomendasi 1",
    "Rekomendasi 2",
    "Rekomendasi 3"
  ]
}`, curr.Name, spotRate, curr.Name, openAR, curr.Name, openAR*spotRate,
		openAP, curr.Name, openAP*spotRate, netExposureFC, curr.Name, netExposureIDR,
		shock2Pct, shock5Pct, riskLevel, riskLevel)

	report := &ForexRiskReport{
		BaseCurrency:          "IDR",
		ForeignCurrency:       curr.Name,
		SpotRate:              spotRate,
		OpenReceivablesFC:     math.Round(openAR*100) / 100,
		OpenPayablesFC:        math.Round(openAP*100) / 100,
		NetExposureFC:         math.Round(netExposureFC*100) / 100,
		NetExposureIDR:        math.Round(netExposureIDR*100) / 100,
		SensitivityShock2Pct:  math.Round(shock2Pct*100) / 100,
		SensitivityShock5Pct:  math.Round(shock5Pct*100) / 100,
		SensitivityShock10Pct: math.Round(shock10Pct*100) / 100,
		RiskLevel:             riskLevel,
		Engine:                "local_heuristic_engine",
	}

	// Try Cloudflare Workers AI if configured
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := "You are a senior Treasury & Forex Risk Analyst for an Indonesian enterprise ERP. Return JSON with risk_level, executive_summary, and action_recommendations."
		aiResp, err := s.aiClient.CallRaw(prompt, systemPrompt)
		if err == nil && aiResp != "" {
			var parsed struct {
				RiskLevel        string   `json:"risk_level"`
				ExecutiveSummary string   `json:"executive_summary"`
				Recommendations  []string `json:"action_recommendations"`
			}
			jsonStr := strings.TrimSpace(aiResp)
			if idx := strings.Index(jsonStr, "{"); idx != -1 {
				if endIdx := strings.LastIndex(jsonStr, "}"); endIdx != -1 {
					jsonStr = jsonStr[idx : endIdx+1]
				}
			}
			if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
				if parsed.ExecutiveSummary != "" {
					report.ExecutiveSummary = parsed.ExecutiveSummary
				}
				if len(parsed.Recommendations) > 0 {
					report.ActionRecommendations = parsed.Recommendations
				}
				if parsed.RiskLevel != "" {
					report.RiskLevel = parsed.RiskLevel
				}
				report.Engine = "@cf/meta/llama-3.1-8b-instruct"
			}
		}
	}

	// Fallback heuristic if empty
	if report.ExecutiveSummary == "" {
		if netExposureFC > 0 {
			report.ExecutiveSummary = fmt.Sprintf("Posisi eksposur valas %s berada pada Net Long sebesar %.2f %s. Depresiasi Rupiah akan memberikan potensi keuntungan kurs positif (Forex Gain), namun apresiasi Rupiah dapat mengikis margin kas.",
				curr.Name, netExposureFC, curr.Name)
		} else if netExposureFC < 0 {
			report.ExecutiveSummary = fmt.Sprintf("Posisi eksposur valas %s berada pada Net Short sebesar %.2f %s. Perusahaan rentan terhadap pelemahan nilai tukar Rupiah yang berisiko memperbesar beban pelunasan hutang impor.",
				curr.Name, math.Abs(netExposureFC), curr.Name)
		} else {
			report.ExecutiveSummary = fmt.Sprintf("Posisi eksposur valas %s berada dalam kondisi Natural Hedge seimbang antara piutang ekspor dan hutang impor.", curr.Name)
		}

		report.ActionRecommendations = []string{
			"Gunakan instrumen Forward Contract atau DNDF (Domestic Non-Deliverable Forward) untuk mengunci kurs transaksi material.",
			"Terapkan penagihan piutang valas yang lebih agresif untuk memitigasi volatilitas mata uang regional.",
			"Lakukan revaluasi kurs bulanan berkala (PSAK 10) agar laporan posisi keuangan merefleksikan nilai wajar terkini.",
		}
	}

	return report, nil
}

// -------------------------------------------------------------
// STATS & HELPERS
// -------------------------------------------------------------

func (s *Service) GetStats(companyID uint) (*CurrencyStats, error) {
	stats, err := s.repo.GetStats(companyID)
	if err != nil {
		return nil, err
	}

	// Sum open USD AR and AP
	var usdCurr Currency
	if err := s.db.Where("name = ?", "USD").First(&usdCurr).Error; err == nil {
		s.db.Model(&account.AccountMove{}).
			Where("company_id = ? AND currency_id = ? AND move_type = ? AND state = ? AND payment_state != ?",
				companyID, usdCurr.ID, "out_invoice", "posted", "paid").
			Select("COALESCE(SUM(amount_currency), 0)").
			Scan(&stats.OpenUSDReceivables)

		s.db.Model(&account.AccountMove{}).
			Where("company_id = ? AND currency_id = ? AND move_type = ? AND state = ? AND payment_state != ?",
				companyID, usdCurr.ID, "in_invoice", "posted", "paid").
			Select("COALESCE(SUM(amount_currency), 0)").
			Scan(&stats.OpenUSDPayables)

		stats.NetUSDExposure = stats.OpenUSDReceivables - stats.OpenUSDPayables
	}

	return stats, nil
}

func (s *Service) ensureAccount(code, name, accType string, companyID uint) *account.AccountAccount {
	var acc account.AccountAccount
	if err := s.db.Where("code = ?", code).First(&acc).Error; err == nil {
		return &acc
	}

	acc = account.AccountAccount{
		Code: code,
		Name: name,
		Type: accType,
	}
	acc.CompanyID = companyID
	s.db.Create(&acc)
	return &acc
}
