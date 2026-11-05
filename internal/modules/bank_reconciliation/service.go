package bank_reconciliation

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"regexp"
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

// CreateStatement creates a new bank statement record
func (s *Service) CreateStatement(req CreateStatementRequest, companyID uint, userID uint) (*AccountBankStatement, error) {
	if req.JournalID == 0 {
		return nil, errors.New("journal bank wajib dipilih")
	}

	var journal account.AccountJournal
	if err := s.db.Where("id = ? AND company_id = ?", req.JournalID, companyID).First(&journal).Error; err != nil {
		return nil, errors.New("journal bank tidak ditemukan")
	}

	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		year := time.Now().Format("2006")
		var count int64
		s.db.Model(&AccountBankStatement{}).Where("company_id = ?", companyID).Count(&count)
		name = fmt.Sprintf("BNK/STAT/%s/%04d", year, count+1)
	}

	stmt := &AccountBankStatement{
		Name:           name,
		JournalID:      req.JournalID,
		Date:           dateStr,
		BalanceStart:   req.BalanceStart,
		BalanceEnd:     req.BalanceStart,
		BalanceEndReal: req.BalanceEndReal,
		Difference:     req.BalanceEndReal - req.BalanceStart,
		State:          "open",
		Note:           req.Note,
	}
	stmt.CompanyID = companyID

	if err := s.repo.CreateStatement(stmt); err != nil {
		return nil, fmt.Errorf("gagal membuat bank statement: %w", err)
	}

	if s.chatterSvc != nil {
		msg := fmt.Sprintf("Rekening koran <b>%s</b> berhasil dibuat (Saldo Awal: Rp %.2f, Saldo Akhir Target: Rp %.2f).",
			stmt.Name, stmt.BalanceStart, stmt.BalanceEndReal)
		_ = s.chatterSvc.LogTracking("account.bank.statement", stmt.ID, userID, "Bank Reconcile Engine", msg, "create", companyID)
	}

	return stmt, nil
}

// GetStatement retrieves a statement by ID with all lines and journal
func (s *Service) GetStatement(id uint, companyID uint) (*AccountBankStatement, error) {
	stmt, err := s.repo.GetStatementByID(id, companyID)
	if err != nil {
		return nil, errors.New("rekening koran tidak ditemukan")
	}
	return stmt, nil
}

// ListStatements lists statements with filters
func (s *Service) ListStatements(companyID uint, state string, journalID uint) ([]AccountBankStatement, error) {
	return s.repo.ListStatements(companyID, state, journalID)
}

// AddStatementLine adds a manual line to a statement and updates balances
func (s *Service) AddStatementLine(statementID uint, req CreateStatementLineRequest, companyID uint) (*AccountBankStatementLine, error) {
	stmt, err := s.repo.GetStatementByID(statementID, companyID)
	if err != nil {
		return nil, errors.New("rekening koran tidak ditemukan")
	}
	if stmt.State != "open" {
		return nil, errors.New("tidak dapat menambah mutasi pada rekening koran yang sudah dikonfirmasi")
	}

	dateStr := req.Date
	if dateStr == "" {
		dateStr = stmt.Date
	}

	line := &AccountBankStatementLine{
		StatementID: statementID,
		Date:        dateStr,
		Name:        strings.TrimSpace(req.Name),
		PartnerID:   req.PartnerID,
		Amount:      req.Amount,
		Ref:         strings.TrimSpace(req.Ref),
	}
	line.CompanyID = companyID

	if err := s.repo.CreateStatementLine(line); err != nil {
		return nil, err
	}

	// Recalculate statement balance
	s.recalculateStatementBalances(stmt)
	_ = s.repo.UpdateStatement(stmt)

	return line, nil
}

// DeleteStatementLine deletes a statement line and updates balances
func (s *Service) DeleteStatementLine(lineID uint, companyID uint) error {
	line, err := s.repo.GetStatementLineByID(lineID, companyID)
	if err != nil {
		return errors.New("baris mutasi tidak ditemukan")
	}
	if line.IsReconciled {
		return errors.New("baris mutasi yang sudah direkonsiliasi tidak dapat dihapus secara langsung")
	}

	statementID := line.StatementID
	if err := s.repo.DeleteStatementLine(lineID, companyID); err != nil {
		return err
	}

	stmt, err := s.repo.GetStatementByID(statementID, companyID)
	if err == nil {
		s.recalculateStatementBalances(stmt)
		_ = s.repo.UpdateStatement(stmt)
	}

	return nil
}

// ParseAndImportCSV processes multi-format bank statement CSVs (BCA, Mandiri, BRI, Generic)
func (s *Service) ParseAndImportCSV(req ImportStatementCSVRequest, companyID uint, userID uint) (*AccountBankStatement, int, error) {
	if strings.TrimSpace(req.CSVContent) == "" {
		return nil, 0, errors.New("konten CSV mutasi rekening bank tidak boleh kosong")
	}

	var stmt *AccountBankStatement
	var err error

	if req.StatementID != nil && *req.StatementID > 0 {
		stmt, err = s.repo.GetStatementByID(*req.StatementID, companyID)
		if err != nil {
			return nil, 0, errors.New("rekening koran target tidak ditemukan")
		}
		if stmt.State != "open" {
			return nil, 0, errors.New("rekening koran target sudah berstatus confirmed")
		}
	} else {
		createReq := CreateStatementRequest{
			Name:           req.Name,
			JournalID:      req.JournalID,
			Date:           req.Date,
			BalanceStart:   req.BalanceStart,
			BalanceEndReal: req.BalanceEndReal,
			Note:           fmt.Sprintf("Diimpor via CSV Mutasi (%s)", req.Format),
		}
		stmt, err = s.CreateStatement(createReq, companyID, userID)
		if err != nil {
			return nil, 0, err
		}
	}

	// Parse CSV Lines
	lines, err := s.parseCSVRecords(req.CSVContent, req.Format, stmt.ID, companyID)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal membaca format file CSV: %w", err)
	}
	if len(lines) == 0 {
		return nil, 0, errors.New("tidak ada baris transaksi yang valid ditemukan dalam CSV")
	}

	if err := s.repo.CreateStatementLines(lines); err != nil {
		return nil, 0, fmt.Errorf("gagal menyimpan baris mutasi bank: %w", err)
	}

	// Recalculate statement balances & stats
	s.recalculateStatementBalances(stmt)
	_ = s.repo.UpdateStatement(stmt)

	// Run auto match analysis on newly imported lines
	_, _ = s.GetAutoMatchSuggestions(stmt.ID, companyID)

	if s.chatterSvc != nil {
		msg := fmt.Sprintf("Berhasil mengimpor <b>%d baris mutasi</b> ke rekening koran <b>%s</b> dari format <i>%s</i>.",
			len(lines), stmt.Name, req.Format)
		_ = s.chatterSvc.LogTracking("account.bank.statement", stmt.ID, userID, "CSV Statement Importer", msg, "import_csv", companyID)
	}

	return stmt, len(lines), nil
}

// parseCSVRecords parses various bank formats into standard AccountBankStatementLine
func (s *Service) parseCSVRecords(csvContent string, format string, statementID uint, companyID uint) ([]AccountBankStatementLine, error) {
	reader := csv.NewReader(strings.NewReader(csvContent))
	reader.TrimLeadingSpace = true
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	// Check if semicolon delimited (common in Indonesian banking exports)
	firstLine := csvContent
	if idx := strings.Index(csvContent, "\n"); idx != -1 {
		firstLine = csvContent[:idx]
	}
	if strings.Count(firstLine, ";") > strings.Count(firstLine, ",") {
		reader.Comma = ';'
	}

	var parsedLines []AccountBankStatementLine
	var headers []string
	isFirstRow := true

	dateCol := -1
	descCol := -1
	amountCol := -1
	typeCol := -1
	debitCol := -1
	creditCol := -1
	refCol := -1

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // skip malformed row
		}
		if len(row) == 0 {
			continue
		}

		if isFirstRow {
			isFirstRow = false
			// Check if header row
			isHeader := false
			for i, val := range row {
				cleanVal := strings.ToLower(strings.TrimSpace(val))
				if strings.Contains(cleanVal, "tgl") || strings.Contains(cleanVal, "tanggal") || strings.Contains(cleanVal, "date") {
					dateCol = i
					isHeader = true
				} else if strings.Contains(cleanVal, "keterangan") || strings.Contains(cleanVal, "deskripsi") || strings.Contains(cleanVal, "desc") || strings.Contains(cleanVal, "memo") || strings.Contains(cleanVal, "uraian") {
					descCol = i
					isHeader = true
				} else if strings.Contains(cleanVal, "jumlah") || strings.Contains(cleanVal, "amount") || strings.Contains(cleanVal, "nominal") || strings.Contains(cleanVal, "mutasi") {
					amountCol = i
					isHeader = true
				} else if strings.Contains(cleanVal, "tipe") || strings.Contains(cleanVal, "type") || strings.Contains(cleanVal, "cb") || strings.Contains(cleanVal, "d/c") || strings.Contains(cleanVal, "cr/db") {
					typeCol = i
					isHeader = true
				} else if strings.Contains(cleanVal, "debet") || strings.Contains(cleanVal, "debit") || strings.Contains(cleanVal, "keluar") || strings.Contains(cleanVal, "db") {
					debitCol = i
					isHeader = true
				} else if strings.Contains(cleanVal, "kredit") || strings.Contains(cleanVal, "credit") || strings.Contains(cleanVal, "masuk") || strings.Contains(cleanVal, "cr") {
					creditCol = i
					isHeader = true
				} else if strings.Contains(cleanVal, "ref") || strings.Contains(cleanVal, "no.") || strings.Contains(cleanVal, "referensi") {
					refCol = i
				}
			}

			if isHeader {
				headers = row
				continue
			}
		}

		// Fallback column positions if no header recognized
		if dateCol == -1 {
			dateCol = 0
		}
		if descCol == -1 && len(row) > 1 {
			descCol = 1
		}

		// Extract fields based on detected columns or layout
		var dateStr, descStr, refStr string
		var amount float64

		if dateCol >= 0 && dateCol < len(row) {
			dateStr = s.normalizeDate(strings.TrimSpace(row[dateCol]))
		}
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		if descCol >= 0 && descCol < len(row) {
			descStr = strings.TrimSpace(row[descCol])
		}
		if descStr == "" {
			descStr = "Mutasi Bank"
		}

		if refCol >= 0 && refCol < len(row) {
			refStr = strings.TrimSpace(row[refCol])
		}

		// Determine Amount
		if debitCol >= 0 && creditCol >= 0 && debitCol < len(row) && creditCol < len(row) {
			// Separate Debit/Credit columns (Mandiri/BRI style)
			debitVal := s.parseIndonesianNumber(row[debitCol])
			creditVal := s.parseIndonesianNumber(row[creditCol])
			if creditVal > 0 {
				amount = creditVal // Inflow
			} else if debitVal > 0 {
				amount = -debitVal // Outflow
			}
		} else if amountCol >= 0 && amountCol < len(row) {
			rawAmt := row[amountCol]
			parsedAmt := s.parseIndonesianNumber(rawAmt)

			// Check Type column if present (CR/DB)
			direction := ""
			if typeCol >= 0 && typeCol < len(row) {
				direction = strings.ToUpper(strings.TrimSpace(row[typeCol]))
			}

			// Check if CR/DB suffix is embedded inside amount text (BCA style)
			upperRawAmt := strings.ToUpper(rawAmt)
			if strings.Contains(upperRawAmt, "CR") {
				direction = "CR"
			} else if strings.Contains(upperRawAmt, "DB") {
				direction = "DB"
			}

			if direction == "DB" || direction == "DEBIT" || direction == "D" {
				amount = -math.Abs(parsedAmt)
			} else {
				// Default or CR is positive
				if parsedAmt < 0 {
					amount = parsedAmt
				} else {
					amount = math.Abs(parsedAmt)
				}
			}
		} else if len(row) >= 3 {
			// Raw fallback: 3rd column as amount
			amount = s.parseIndonesianNumber(row[2])
		}

		if amount == 0 {
			continue // skip 0 amount lines
		}

		line := AccountBankStatementLine{
			StatementID:  statementID,
			Date:         dateStr,
			Name:         descStr,
			Amount:       amount,
			Ref:          refStr,
			IsReconciled: false,
		}
		line.CompanyID = companyID
		parsedLines = append(parsedLines, line)
	}

	_ = headers
	return parsedLines, nil
}

// parseIndonesianNumber converts Indonesian format strings (e.g. "1.500.000,50", "25,000", "-50.000,00") to float64
func (s *Service) parseIndonesianNumber(str string) float64 {
	clean := strings.TrimSpace(str)
	clean = strings.ReplaceAll(clean, "Rp", "")
	clean = strings.ReplaceAll(clean, "IDR", "")
	clean = strings.ReplaceAll(clean, "cr", "")
	clean = strings.ReplaceAll(clean, "CR", "")
	clean = strings.ReplaceAll(clean, "db", "")
	clean = strings.ReplaceAll(clean, "DB", "")
	clean = strings.TrimSpace(clean)

	isNegative := false
	if strings.HasPrefix(clean, "-") || (strings.HasPrefix(clean, "(") && strings.HasSuffix(clean, ")")) {
		isNegative = true
		clean = strings.Trim(clean, "-()")
	}

	// Detect decimal separator
	hasDot := strings.Contains(clean, ".")
	hasComma := strings.Contains(clean, ",")

	if hasDot && hasComma {
		lastDot := strings.LastIndex(clean, ".")
		lastComma := strings.LastIndex(clean, ",")
		if lastComma > lastDot {
			// Indonesian style: 1.500.000,00 -> dot is thousand, comma is decimal
			clean = strings.ReplaceAll(clean, ".", "")
			clean = strings.ReplaceAll(clean, ",", ".")
		} else {
			// US style: 1,500,000.00 -> comma is thousand, dot is decimal
			clean = strings.ReplaceAll(clean, ",", "")
		}
	} else if hasComma && !hasDot {
		// e.g. 1500000,00 or 1,500
		parts := strings.Split(clean, ",")
		if len(parts) == 2 && len(parts[1]) <= 2 {
			// Decimal comma
			clean = strings.ReplaceAll(clean, ",", ".")
		} else {
			// Thousand separator
			clean = strings.ReplaceAll(clean, ",", "")
		}
	} else if hasDot && !hasComma {
		// e.g. 1.500.000 or 1500.50
		parts := strings.Split(clean, ".")
		if len(parts) == 2 && len(parts[1]) <= 2 {
			// Decimal dot
		} else {
			// Thousand dot
			clean = strings.ReplaceAll(clean, ".", "")
		}
	}

	val, err := strconv.ParseFloat(strings.TrimSpace(clean), 64)
	if err != nil {
		return 0
	}
	if isNegative {
		return -val
	}
	return val
}

// normalizeDate formats dates like DD/MM/YYYY, DD-MM-YYYY to YYYY-MM-DD
func (s *Service) normalizeDate(dateStr string) string {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Now().Format("2006-01-02")
	}

	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"02-01-2006",
		"02/01/06",
		"02/01",
		"2006/01/02",
	}

	for _, f := range formats {
		if t, err := time.Parse(f, dateStr); err == nil {
			if f == "02/01" {
				// add current year
				return fmt.Sprintf("%d-%02d-%02d", time.Now().Year(), t.Month(), t.Day())
			}
			return t.Format("2006-01-02")
		}
	}
	return dateStr
}

// GetAutoMatchSuggestions computes smart matching for unreconciled lines against open invoices & GL rules
func (s *Service) GetAutoMatchSuggestions(statementID uint, companyID uint) ([]AutoMatchSuggestion, error) {
	stmt, err := s.repo.GetStatementByID(statementID, companyID)
	if err != nil {
		return nil, errors.New("rekening koran tidak ditemukan")
	}

	openInvoices, err := s.repo.FindOpenInvoices(companyID)
	if err != nil {
		return nil, err
	}

	rules, _ := s.repo.ListReconcileModels(companyID)

	var suggestions []AutoMatchSuggestion
	invRegex := regexp.MustCompile(`(?i)\b((?:INV|BILL)/[0-9]{4}/[0-9]{5}|INV-[0-9]+)\b`)

	for _, line := range stmt.LineIDs {
		if line.IsReconciled {
			continue
		}

		sugg := AutoMatchSuggestion{
			LineID:     line.ID,
			LineMemo:   line.Name,
			LineAmount: line.Amount,
			LineDate:   line.Date,
		}

		matched := false

		// Rule 1: Regex Invoice / Bill Reference in Memo
		if matches := invRegex.FindStringSubmatch(line.Name + " " + line.Ref); len(matches) > 1 {
			targetRef := strings.ToUpper(matches[1])
			for _, inv := range openInvoices {
				if strings.EqualFold(inv.Name, targetRef) {
					absAmt := math.Abs(line.Amount)
					sugg.SuggestedMoveID = &inv.ID
					sugg.SuggestedMoveName = inv.Name
					sugg.SuggestedMoveType = inv.MoveType
					sugg.SuggestedResidual = inv.AmountResidual
					sugg.SuggestedPartnerID = &inv.PartnerID
					if inv.Partner.Name != "" {
						sugg.SuggestedPartner = inv.Partner.Name
					}

					if math.Abs(absAmt-inv.AmountResidual) < 0.01 {
						sugg.MatchType = "exact_ref"
						sugg.Confidence = 0.98
						sugg.Reason = fmt.Sprintf("Nomor faktur %s terdeteksi dalam memo bank dengan nilai residual pas", inv.Name)
					} else {
						sugg.MatchType = "partial_ref"
						sugg.Confidence = 0.88
						sugg.Reason = fmt.Sprintf("Nomor faktur %s terdeteksi dalam memo bank (pembayaran sebagian)", inv.Name)
					}
					matched = true
					break
				}
			}
		}

		// Rule 2: Bank Charge / Admin Fee Model Rules
		if !matched && len(rules) > 0 {
			for _, r := range rules {
				textMatch := false
				lowerMemo := strings.ToLower(line.Name)
				lowerTarget := strings.ToLower(r.MatchText)

				if r.MatchType == "contains" && strings.Contains(lowerMemo, lowerTarget) {
					textMatch = true
				} else if r.MatchType == "regex" {
					if re, err := regexp.Compile("(?i)" + r.MatchText); err == nil && re.MatchString(line.Name) {
						textMatch = true
					}
				} else if r.MatchType == "exact" && lowerMemo == lowerTarget {
					textMatch = true
				}

				if textMatch {
					sugg.SuggestedAccountID = &r.AccountID
					sugg.SuggestedAccount = r.RuleName
					sugg.MatchType = "bank_charge_rule"
					sugg.Confidence = 0.95
					sugg.Reason = fmt.Sprintf("Sesuai aturan rekonsiliasi: %s (%s)", r.RuleName, r.MatchText)
					matched = true
					break
				}
			}
		}

		// Rule 3: Partner Name match + Exact Amount
		if !matched {
			absAmt := math.Abs(line.Amount)
			for _, inv := range openInvoices {
				if inv.Partner.Name != "" && len(inv.Partner.Name) > 3 {
					if strings.Contains(strings.ToLower(line.Name), strings.ToLower(inv.Partner.Name)) {
						if math.Abs(absAmt-inv.AmountResidual) < 0.01 {
							sugg.SuggestedMoveID = &inv.ID
							sugg.SuggestedMoveName = inv.Name
							sugg.SuggestedMoveType = inv.MoveType
							sugg.SuggestedResidual = inv.AmountResidual
							sugg.SuggestedPartnerID = &inv.PartnerID
							sugg.SuggestedPartner = inv.Partner.Name
							sugg.MatchType = "amount_partner"
							sugg.Confidence = 0.90
							sugg.Reason = fmt.Sprintf("Nama partner '%s' cocok dan nilai pas dengan sisa piutang/hutang", inv.Partner.Name)
							matched = true
							break
						}
					}
				}
			}
		}

		// Rule 4: Single exact amount match
		if !matched {
			absAmt := math.Abs(line.Amount)
			var candidates []account.AccountMove
			for _, inv := range openInvoices {
				if math.Abs(absAmt-inv.AmountResidual) < 0.01 {
					// Check direction: Inflow matches customer invoice (out_invoice), outflow matches vendor bill (in_invoice)
					if (line.Amount > 0 && inv.MoveType == "out_invoice") || (line.Amount < 0 && inv.MoveType == "in_invoice") {
						candidates = append(candidates, inv)
					}
				}
			}

			if len(candidates) == 1 {
				inv := candidates[0]
				sugg.SuggestedMoveID = &inv.ID
				sugg.SuggestedMoveName = inv.Name
				sugg.SuggestedMoveType = inv.MoveType
				sugg.SuggestedResidual = inv.AmountResidual
				sugg.SuggestedPartnerID = &inv.PartnerID
				if inv.Partner.Name != "" {
					sugg.SuggestedPartner = inv.Partner.Name
				}
				sugg.MatchType = "fuzzy_amount"
				sugg.Confidence = 0.78
				sugg.Reason = fmt.Sprintf("Nilai transaksi pas dengan faktur %s", inv.Name)
				matched = true
			}
		}

		if !matched {
			sugg.MatchType = "unmatched"
			sugg.Confidence = 0.0
			sugg.Reason = "Belum ada faktur atau aturan yang cocok otomatis"
		}

		// Persist suggestion confidence on the line
		if sugg.Confidence > 0 {
			line.AutoMatchConfidence = sugg.Confidence
			line.AutoMatchReason = sugg.Reason
			_ = s.repo.UpdateStatementLine(&line)
		}

		suggestions = append(suggestions, sugg)
	}

	return suggestions, nil
}

// ReconcileLine reconciles a statement line with an invoice or GL write-off
func (s *Service) ReconcileLine(req ReconcileLineRequest, companyID uint, userID uint) error {
	line, err := s.repo.GetStatementLineByID(req.StatementLineID, companyID)
	if err != nil {
		return errors.New("baris mutasi bank tidak ditemukan")
	}
	if line.IsReconciled {
		return errors.New("baris mutasi bank ini sudah direkonsiliasi sebelumnya")
	}

	stmt := line.Statement
	if stmt == nil {
		stmt, _ = s.repo.GetStatementByID(line.StatementID, companyID)
	}

	// 1. Reconcile against Invoice
	if req.MoveID != nil && *req.MoveID > 0 {
		move, err := s.accountSvc.GetMove(*req.MoveID, companyID)
		if err != nil {
			return errors.New("faktur/dokumen akuntansi tidak ditemukan")
		}
		if move.State != "posted" {
			return errors.New("hanya faktur yang sudah diposting yang dapat direkonsiliasi")
		}

		payAmt := math.Abs(line.Amount)
		if payAmt > move.AmountResidual && move.AmountResidual > 0 {
			payAmt = move.AmountResidual
		}

		memo := fmt.Sprintf("Rekonsiliasi Bank %s: %s", stmt.Name, line.Name)
		payment, updatedMove, err := s.accountSvc.RegisterPayment(move.ID, payAmt, stmt.JournalID, memo, companyID)
		if err != nil {
			return fmt.Errorf("gagal mendaftarkan pembayaran faktur: %w", err)
		}

		line.IsReconciled = true
		line.MoveID = &updatedMove.ID
		if payment != nil {
			line.PaymentID = &payment.ID
		}
		if req.PartnerID != nil {
			line.PartnerID = req.PartnerID
		} else if move.PartnerID > 0 {
			line.PartnerID = &move.PartnerID
		}

		if err := s.repo.UpdateStatementLine(line); err != nil {
			return err
		}

		s.recalculateStatementBalances(stmt)
		_ = s.repo.UpdateStatement(stmt)

		if s.chatterSvc != nil {
			msg := fmt.Sprintf("Baris mutasi <b>%s</b> (Rp %.2f) berhasil direkonsiliasi dengan faktur <b>%s</b>.",
				line.Name, line.Amount, move.Name)
			_ = s.chatterSvc.LogTracking("account.bank.statement", stmt.ID, userID, "Bank Reconcile Engine", msg, "reconcile", companyID)
		}

		return nil
	}

	// 2. Direct Write-off / Bank Charge / Interest Income
	if req.WriteOffAccountID != nil && *req.WriteOffAccountID > 0 {
		var writeOffAcc account.AccountAccount
		if err := s.db.Where("id = ? AND company_id = ?", *req.WriteOffAccountID, companyID).First(&writeOffAcc).Error; err != nil {
			return errors.New("akun write-off/biaya bank tidak ditemukan")
		}

		bankJournalAccID := stmt.Journal.DefaultAccountID
		if bankJournalAccID == 0 {
			// Fallback to Bank BCA account 110200 or first asset_cash account
			var bnkAcc account.AccountAccount
			if err := s.db.Where("code = ? AND company_id = ?", "110200", companyID).First(&bnkAcc).Error; err == nil {
				bankJournalAccID = bnkAcc.ID
			} else {
				bankJournalAccID = *req.WriteOffAccountID
			}
		}

		absAmt := math.Abs(line.Amount)
		moveName := fmt.Sprintf("BNK-WO/%s/%d", time.Now().Format("2006"), line.ID)

		// Create balanced journal entry
		var lines []account.AccountMoveLine
		if line.Amount < 0 {
			// Outflow (Bank Fee / Expense): Debit Expense, Credit Bank
			lines = []account.AccountMoveLine{
				{
					AccountID: *req.WriteOffAccountID,
					Name:      line.Name,
					Debit:     absAmt,
					Credit:    0,
					Balance:   absAmt,
				},
				{
					AccountID: bankJournalAccID,
					Name:      fmt.Sprintf("Biaya Bank - %s", line.Name),
					Debit:     0,
					Credit:    absAmt,
					Balance:   -absAmt,
				},
			}
		} else {
			// Inflow (Interest / Income): Debit Bank, Credit Income
			lines = []account.AccountMoveLine{
				{
					AccountID: bankJournalAccID,
					Name:      fmt.Sprintf("Penerimaan Bank - %s", line.Name),
					Debit:     absAmt,
					Credit:    0,
					Balance:   absAmt,
				},
				{
					AccountID: *req.WriteOffAccountID,
					Name:      line.Name,
					Debit:     0,
					Credit:    absAmt,
					Balance:   -absAmt,
				},
			}
		}

		move := &account.AccountMove{
			Name:        moveName,
			MoveType:    "entry",
			JournalID:   stmt.JournalID,
			InvoiceDate: line.Date,
			State:       "draft",
			Ref:         fmt.Sprintf("Write-off %s (%s)", stmt.Name, line.Ref),
			Note:        line.Name,
			LineIDs:     lines,
		}
		move.CompanyID = companyID

		if err := s.accountSvc.CreateMove(move); err != nil {
			return fmt.Errorf("gagal membuat jurnal penyesuaian bank: %w", err)
		}

		if _, err := s.accountSvc.ActionPost(move.ID, companyID); err != nil {
			return fmt.Errorf("gagal posting jurnal penyesuaian bank: %w", err)
		}

		line.IsReconciled = true
		line.MoveID = &move.ID
		if req.PartnerID != nil {
			line.PartnerID = req.PartnerID
		}

		if err := s.repo.UpdateStatementLine(line); err != nil {
			return err
		}

		s.recalculateStatementBalances(stmt)
		_ = s.repo.UpdateStatement(stmt)

		if s.chatterSvc != nil {
			msg := fmt.Sprintf("Baris mutasi <b>%s</b> (Rp %.2f) berhasil dibukukan ke akun <b>%s</b>.",
				line.Name, line.Amount, writeOffAcc.Name)
			_ = s.chatterSvc.LogTracking("account.bank.statement", stmt.ID, userID, "Bank Reconcile Engine", msg, "write_off", companyID)
		}

		return nil
	}

	return errors.New("pilih faktur atau akun penyesuaian untuk rekonsiliasi")
}

// BatchReconcile reconciles multiple items
func (s *Service) BatchReconcile(req BatchReconcileRequest, companyID uint, userID uint) (int, error) {
	successCount := 0
	for _, item := range req.Items {
		if err := s.ReconcileLine(item, companyID, userID); err == nil {
			successCount++
		}
	}
	return successCount, nil
}

// AutoReconcileHighConfidence automatically reconciles lines having confidence >= minConfidence
func (s *Service) AutoReconcileHighConfidence(statementID uint, minConfidence float64, companyID uint, userID uint) (int, error) {
	if minConfidence <= 0 {
		minConfidence = 0.90
	}

	suggestions, err := s.GetAutoMatchSuggestions(statementID, companyID)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, sugg := range suggestions {
		if sugg.Confidence >= minConfidence {
			req := ReconcileLineRequest{
				StatementLineID:   sugg.LineID,
				MoveID:            sugg.SuggestedMoveID,
				WriteOffAccountID: sugg.SuggestedAccountID,
				PartnerID:         sugg.SuggestedPartnerID,
			}
			if err := s.ReconcileLine(req, companyID, userID); err == nil {
				count++
			}
		}
	}

	return count, nil
}

// ConfirmStatement closes and locks the statement if reconciled or difference is resolved
func (s *Service) ConfirmStatement(id uint, companyID uint, userID uint) (*AccountBankStatement, error) {
	stmt, err := s.repo.GetStatementByID(id, companyID)
	if err != nil {
		return nil, errors.New("rekening koran tidak ditemukan")
	}
	if stmt.State == "confirm" {
		return nil, errors.New("rekening koran sudah berstatus confirmed")
	}

	s.recalculateStatementBalances(stmt)

	if math.Abs(stmt.Difference) > 0.01 {
		return nil, fmt.Errorf("rekening koran belum seimbang. Masih terdapat selisih Rp %.2f antara saldo buku (Rp %.2f) dan saldo fisik bank (Rp %.2f)",
			stmt.Difference, stmt.BalanceEnd, stmt.BalanceEndReal)
	}

	stmt.State = "confirm"
	if err := s.repo.UpdateStatement(stmt); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := fmt.Sprintf("Rekening koran <b>%s</b> telah dikonfirmasi dan ditutup dengan saldo akhir Rp %.2f (Selisih: Rp 0.00).",
			stmt.Name, stmt.BalanceEnd)
		_ = s.chatterSvc.LogTracking("account.bank.statement", stmt.ID, userID, "Bank Reconcile Engine", msg, "confirm", companyID)
	}

	return stmt, nil
}

// recalculateStatementBalances recalculates total, ending balance, difference, and line counts
func (s *Service) recalculateStatementBalances(stmt *AccountBankStatement) {
	lines, err := s.repo.ListStatementLines(stmt.ID, stmt.CompanyID)
	if err != nil {
		return
	}

	var totalMovement float64
	reconciled := 0
	for _, l := range lines {
		totalMovement += l.Amount
		if l.IsReconciled {
			reconciled++
		}
	}

	stmt.TotalLines = len(lines)
	stmt.ReconciledLines = reconciled
	stmt.BalanceEnd = stmt.BalanceStart + totalMovement
	stmt.Difference = stmt.BalanceEndReal - stmt.BalanceEnd
}

// AIBankCopilot generates intelligent statement audit & reconciliation recommendations
func (s *Service) AIBankCopilot(statementID uint, companyID uint) (*AIBankMatchReport, error) {
	stmt, err := s.repo.GetStatementByID(statementID, companyID)
	if err != nil {
		return nil, errors.New("rekening koran tidak ditemukan")
	}

	suggestions, _ := s.GetAutoMatchSuggestions(statementID, companyID)

	var highConfidenceCount, unmatchedCount int
	var discrepancyNotes []string

	if math.Abs(stmt.Difference) > 0.01 {
		discrepancyNotes = append(discrepancyNotes,
			fmt.Sprintf("Terdapat selisih saldo buku vs saldo cetak bank sebesar Rp %.2f", stmt.Difference))
	}

	for _, sugg := range suggestions {
		if sugg.Confidence >= 0.85 {
			highConfidenceCount++
		} else {
			unmatchedCount++
			if sugg.LineAmount < -10000000 {
				discrepancyNotes = append(discrepancyNotes,
					fmt.Sprintf("Transaksi pengeluaran besar tanpa faktur: Rp %.2f (%s)", sugg.LineAmount, sugg.LineMemo))
			}
		}
	}

	report := &AIBankMatchReport{
		StatementID:        stmt.ID,
		StatementName:      stmt.Name,
		RiskAssessment:     "Low Risk",
		TotalDiscrepancies: len(discrepancyNotes),
		DiscrepancyNotes:   discrepancyNotes,
		Engine:             "local_heuristic_engine",
	}

	if len(discrepancyNotes) > 2 || math.Abs(stmt.Difference) > 1000000 {
		report.RiskAssessment = "High Risk"
	} else if len(discrepancyNotes) > 0 {
		report.RiskAssessment = "Medium Risk"
	}

	// Try Cloudflare Workers AI Edge Inference if configured
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		prompt := fmt.Sprintf(`Auditi bank statement ini:
Nama: %s, Tanggal: %s, Saldo Awal: %.2f, Saldo Target: %.2f, Selisih: %.2f
Total Mutasi: %d baris, Cocok Tinggi: %d, Belum Cocok: %d.
Catatan Diskrepansi: %s
Berikan JSON dengan fields: "risk_assessment" ("Low Risk"|"Medium Risk"|"High Risk"), "executive_summary", "action_recommendations" (array of string).`,
			stmt.Name, stmt.Date, stmt.BalanceStart, stmt.BalanceEndReal, stmt.Difference,
			stmt.TotalLines, highConfidenceCount, unmatchedCount, strings.Join(discrepancyNotes, "; "))

		systemPrompt := "You are a professional CFO and senior bank reconciliation auditor. Return ONLY a valid JSON object."
		if aiResp, err := s.aiClient.CallRaw(prompt, systemPrompt); err == nil && aiResp != "" {
			var parsed struct {
				RiskAssessment   string   `json:"risk_assessment"`
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
				if parsed.RiskAssessment != "" {
					report.RiskAssessment = parsed.RiskAssessment
				}
				report.Engine = "@cf/meta/llama-3.1-8b-instruct"
			}
		}
	}

	// Fallback Heuristic
	if report.ExecutiveSummary == "" {
		report.ExecutiveSummary = fmt.Sprintf("Rekening koran %s memiliki %d transaksi. %d transaksi memiliki keyakinan pencocokan tinggi (>85%%), dan %d transaksi masih perlu verifikasi manual.",
			stmt.Name, stmt.TotalLines, highConfidenceCount, unmatchedCount)
		report.ActionRecommendations = []string{
			"Jalankan 'Auto Reconcile' untuk mencocokkan otomatis transaksi dengan tingkat keyakinan >90%.",
			"Periksa bukti transfer untuk transaksi tanpa nomor faktur.",
			"Pastikan saldo akhir mutasi bank sesuai dengan rekening koran resmi sebelum melakukan konfirmasi penutupan.",
		}
	}

	return report, nil
}

// Reconcile Models (Rules) CRUD & Seeding

func (s *Service) ListReconcileModels(companyID uint) ([]AccountReconcileModel, error) {
	return s.repo.ListReconcileModels(companyID)
}

func (s *Service) CreateReconcileModel(model *AccountReconcileModel) error {
	return s.repo.CreateReconcileModel(model)
}

func (s *Service) SeedDefaultRules(companyID uint) {
	rules, _ := s.repo.ListReconcileModels(companyID)
	if len(rules) > 0 {
		return
	}

	// Find standard Expense & Income Accounts
	var admExpenseAcc account.AccountAccount
	if err := s.db.Where("code = ? AND company_id = ?", "620100", companyID).First(&admExpenseAcc).Error; err != nil {
		// Fallback to COGS or any expense account
		s.db.Where("type = 'expense' AND company_id = ?", companyID).First(&admExpenseAcc)
	}

	var interestIncomeAcc account.AccountAccount
	if err := s.db.Where("code = ? AND company_id = ?", "710100", companyID).First(&interestIncomeAcc).Error; err != nil {
		// Fallback to income
		s.db.Where("type = 'income' AND company_id = ?", companyID).First(&interestIncomeAcc)
	}

	if admExpenseAcc.ID > 0 {
		_ = s.repo.CreateReconcileModel(&AccountReconcileModel{
			RuleName:  "Biaya Administrasi Bank (BCA / Mandiri / BRI)",
			MatchType: "contains",
			MatchText: "BIAYA ADM",
			RuleType:  "write_off",
			AccountID: admExpenseAcc.ID,
			Sequence:  10,
		})
		_ = s.repo.CreateReconcileModel(&AccountReconcileModel{
			RuleName:  "Biaya Transfer Antar Bank / BI-FAST",
			MatchType: "contains",
			MatchText: "BIAYA TRSF",
			RuleType:  "write_off",
			AccountID: admExpenseAcc.ID,
			Sequence:  20,
		})
	}

	if interestIncomeAcc.ID > 0 {
		_ = s.repo.CreateReconcileModel(&AccountReconcileModel{
			RuleName:  "Pendapatan Bunga Rekening Koran",
			MatchType: "contains",
			MatchText: "BUNGA",
			RuleType:  "write_off",
			AccountID: interestIncomeAcc.ID,
			Sequence:  30,
		})
	}
}

// GetStats returns comprehensive reconciliation KPI metrics
func (s *Service) GetStats(companyID uint) (*BankReconciliationStats, error) {
	return s.repo.GetStats(companyID)
}

// ExportStatementCSV exports statement lines as CSV
func (s *Service) ExportStatementCSV(statementID uint, companyID uint) ([]byte, string, error) {
	stmt, err := s.repo.GetStatementByID(statementID, companyID)
	if err != nil {
		return nil, "", errors.New("rekening koran tidak ditemukan")
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	_ = writer.Write([]string{"Date", "Memo", "Amount", "Reference", "Status", "Partner"})
	for _, l := range stmt.LineIDs {
		status := "Belum Rekonsiliasi"
		if l.IsReconciled {
			status = "Rekonsiliasi Selesai"
		}
		partnerName := ""
		if l.Partner != nil {
			partnerName = l.Partner.Name
		}
		_ = writer.Write([]string{
			l.Date,
			l.Name,
			fmt.Sprintf("%.2f", l.Amount),
			l.Ref,
			status,
			partnerName,
		})
	}
	writer.Flush()

	filename := fmt.Sprintf("mutasi_bank_%s.csv", strings.ReplaceAll(stmt.Name, "/", "_"))
	return buf.Bytes(), filename, nil
}
