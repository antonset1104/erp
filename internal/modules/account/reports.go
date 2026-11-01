package account

import (
	"math"
	"strings"
	"time"
)

type ReportAccountLine struct {
	Code    string  `json:"code"`
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	Debit   float64 `json:"debit"`
	Credit  float64 `json:"credit"`
	Balance float64 `json:"balance"` // Net balance
}

type ProfitAndLossReport struct {
	CompanyID          uint                `json:"company_id"`
	DateFrom           string              `json:"date_from"`
	DateTo             string              `json:"date_to"`
	RevenueLines       []ReportAccountLine `json:"revenue_lines"`
	TotalRevenue       float64             `json:"total_revenue"`
	CogsLines          []ReportAccountLine `json:"cogs_lines"`
	TotalCogs          float64             `json:"total_cogs"`
	GrossProfit        float64             `json:"gross_profit"`
	OperatingExpenses  []ReportAccountLine `json:"operating_expenses"`
	TotalOperatingExp  float64             `json:"total_operating_expenses"`
	NetProfit          float64             `json:"net_profit"`
}

type BalanceSheetReport struct {
	CompanyID         uint                `json:"company_id"`
	AsOfDate          string              `json:"as_of_date"`
	CurrentAssetLines []ReportAccountLine `json:"current_asset_lines"`
	TotalCurrentAsset float64             `json:"total_current_assets"`
	FixedAssetLines   []ReportAccountLine `json:"fixed_asset_lines"`
	TotalFixedAssets  float64             `json:"total_fixed_assets"`
	TotalAssets       float64             `json:"total_assets"`
	LiabilityLines    []ReportAccountLine `json:"liability_lines"`
	TotalLiabilities  float64             `json:"total_liabilities"`
	EquityLines       []ReportAccountLine `json:"equity_lines"`
	CurrentYearProfit float64             `json:"current_year_profit"`
	TotalEquity       float64             `json:"total_equity"`
	TotalLiabEquity   float64             `json:"total_liabilities_and_equity"`
	IsBalanced        bool                `json:"is_balanced"`
}

type TrialBalanceReport struct {
	CompanyID   uint                `json:"company_id"`
	GeneratedAt string              `json:"generated_at"`
	Lines       []ReportAccountLine `json:"lines"`
	TotalDebit  float64             `json:"total_debit"`
	TotalCredit float64             `json:"total_credit"`
	Difference  float64             `json:"difference"`
	IsBalanced  bool                `json:"is_balanced"`
}

// GetProfitAndLoss computes the Income Statement based on posted journal items
func (s *Service) GetProfitAndLoss(companyID uint, dateFrom, dateTo string) (*ProfitAndLossReport, error) {
	if dateTo == "" {
		dateTo = time.Now().Format("2006-01-02")
	}
	if dateFrom == "" {
		dateFrom = time.Now().Format("2006") + "-01-01"
	}

	report := &ProfitAndLossReport{
		CompanyID:         companyID,
		DateFrom:          dateFrom,
		DateTo:            dateTo,
		RevenueLines:      []ReportAccountLine{},
		CogsLines:         []ReportAccountLine{},
		OperatingExpenses: []ReportAccountLine{},
	}

	// Fetch all posted move lines
	var lines []AccountMoveLine
	err := s.db.Joins("JOIN account_moves ON account_moves.id = account_move_lines.move_id").
		Joins("JOIN account_accounts ON account_accounts.id = account_move_lines.account_id").
		Preload("Account").
		Where("account_moves.company_id = ? AND account_moves.state = 'posted'", companyID).
		Where("account_moves.invoice_date >= ? AND account_moves.invoice_date <= ?", dateFrom, dateTo).
		Find(&lines).Error
	if err != nil {
		return nil, err
	}

	// Aggregate balances per account
	accMap := make(map[uint]*ReportAccountLine)
	for _, l := range lines {
		if l.Account.ID == 0 {
			continue
		}
		if _, exists := accMap[l.Account.ID]; !exists {
			accMap[l.Account.ID] = &ReportAccountLine{
				Code: l.Account.Code,
				Name: l.Account.Name,
				Type: l.Account.Type,
			}
		}
		item := accMap[l.Account.ID]
		item.Debit += l.Debit
		item.Credit += l.Credit
		item.Balance = item.Debit - item.Credit
	}

	// Group into Revenue, COGS, Expenses
	for _, item := range accMap {
		if item.Type == "income" || strings.HasPrefix(item.Code, "4") {
			// In accounting, income normal balance is credit
			netIncome := item.Credit - item.Debit
			item.Balance = netIncome
			report.RevenueLines = append(report.RevenueLines, *item)
			report.TotalRevenue += netIncome
		} else if strings.HasPrefix(item.Code, "5") {
			// Cost of Goods Sold (COGS)
			netCogs := item.Debit - item.Credit
			item.Balance = netCogs
			report.CogsLines = append(report.CogsLines, *item)
			report.TotalCogs += netCogs
		} else if item.Type == "expense" || strings.HasPrefix(item.Code, "6") {
			// Operating Expenses
			netExp := item.Debit - item.Credit
			item.Balance = netExp
			report.OperatingExpenses = append(report.OperatingExpenses, *item)
			report.TotalOperatingExp += netExp
		}
	}

	report.GrossProfit = report.TotalRevenue - report.TotalCogs
	report.NetProfit = report.GrossProfit - report.TotalOperatingExp

	return report, nil
}

// GetBalanceSheet computes the Balance Sheet (Assets = Liabilities + Equity)
func (s *Service) GetBalanceSheet(companyID uint, asOfDate string) (*BalanceSheetReport, error) {
	if asOfDate == "" {
		asOfDate = time.Now().Format("2006-01-02")
	}

	report := &BalanceSheetReport{
		CompanyID:         companyID,
		AsOfDate:          asOfDate,
		CurrentAssetLines: []ReportAccountLine{},
		FixedAssetLines:   []ReportAccountLine{},
		LiabilityLines:    []ReportAccountLine{},
		EquityLines:       []ReportAccountLine{},
	}

	var lines []AccountMoveLine
	err := s.db.Joins("JOIN account_moves ON account_moves.id = account_move_lines.move_id").
		Joins("JOIN account_accounts ON account_accounts.id = account_move_lines.account_id").
		Preload("Account").
		Where("account_moves.company_id = ? AND account_moves.state = 'posted'", companyID).
		Where("account_moves.invoice_date <= ?", asOfDate).
		Find(&lines).Error
	if err != nil {
		return nil, err
	}

	accMap := make(map[uint]*ReportAccountLine)
	for _, l := range lines {
		if l.Account.ID == 0 {
			continue
		}
		if _, exists := accMap[l.Account.ID]; !exists {
			accMap[l.Account.ID] = &ReportAccountLine{
				Code: l.Account.Code,
				Name: l.Account.Name,
				Type: l.Account.Type,
			}
		}
		item := accMap[l.Account.ID]
		item.Debit += l.Debit
		item.Credit += l.Credit
		item.Balance = item.Debit - item.Credit
	}

	var totalAssets, totalLiab float64
	for _, item := range accMap {
		if strings.HasPrefix(item.Code, "1") {
			// Assets (Normal Debit)
			netAsset := item.Debit - item.Credit
			item.Balance = netAsset
			if strings.HasPrefix(item.Code, "15") || strings.HasPrefix(item.Code, "16") {
				report.FixedAssetLines = append(report.FixedAssetLines, *item)
				report.TotalFixedAssets += netAsset
			} else {
				report.CurrentAssetLines = append(report.CurrentAssetLines, *item)
				report.TotalCurrentAsset += netAsset
			}
			totalAssets += netAsset
		} else if strings.HasPrefix(item.Code, "2") {
			// Liabilities (Normal Credit)
			netLiab := item.Credit - item.Debit
			item.Balance = netLiab
			report.LiabilityLines = append(report.LiabilityLines, *item)
			totalLiab += netLiab
		} else if strings.HasPrefix(item.Code, "3") {
			// Equity (Normal Credit)
			netEquity := item.Credit - item.Debit
			item.Balance = netEquity
			report.EquityLines = append(report.EquityLines, *item)
		}
	}

	report.TotalAssets = totalAssets
	report.TotalLiabilities = totalLiab

	// Compute retained net income from P&L as of date
	pnl, err := s.GetProfitAndLoss(companyID, "2000-01-01", asOfDate)
	if err == nil && pnl != nil {
		report.CurrentYearProfit = pnl.NetProfit
	}

	var sumEquity float64
	for _, eq := range report.EquityLines {
		sumEquity += eq.Balance
	}
	report.TotalEquity = sumEquity + report.CurrentYearProfit
	report.TotalLiabEquity = report.TotalLiabilities + report.TotalEquity

	// Balanced check
	report.IsBalanced = math.Abs(report.TotalAssets-report.TotalLiabEquity) < 1.0

	return report, nil
}

// GetTrialBalance computes trial balance of all accounts verifying sum(Debit) == sum(Credit)
func (s *Service) GetTrialBalance(companyID uint) (*TrialBalanceReport, error) {
	report := &TrialBalanceReport{
		CompanyID:   companyID,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Lines:       []ReportAccountLine{},
	}

	accounts, err := s.ListAccounts(companyID)
	if err != nil {
		return nil, err
	}

	var lines []AccountMoveLine
	err = s.db.Joins("JOIN account_moves ON account_moves.id = account_move_lines.move_id").
		Where("account_moves.company_id = ? AND account_moves.state = 'posted'", companyID).
		Find(&lines).Error
	if err != nil {
		return nil, err
	}

	debitMap := make(map[uint]float64)
	creditMap := make(map[uint]float64)
	for _, l := range lines {
		debitMap[l.AccountID] += l.Debit
		creditMap[l.AccountID] += l.Credit
	}

	for _, acc := range accounts {
		d := debitMap[acc.ID]
		c := creditMap[acc.ID]
		report.Lines = append(report.Lines, ReportAccountLine{
			Code:    acc.Code,
			Name:    acc.Name,
			Type:    acc.Type,
			Debit:   d,
			Credit:  c,
			Balance: d - c,
		})
		report.TotalDebit += d
		report.TotalCredit += c
	}

	report.Difference = math.Abs(report.TotalDebit - report.TotalCredit)
	report.IsBalanced = report.Difference < 0.01

	return report, nil
}
