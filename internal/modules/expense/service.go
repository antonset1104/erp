package expense

import (
	"errors"
	"fmt"
	"math"
	"time"

	"erp-golang/internal/modules/account"
	"erp-golang/internal/modules/chatter"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	accountSvc *account.Service
	chatterSvc *chatter.Service
}

func NewService(db *gorm.DB, accountSvc *account.Service, chatterSvc *chatter.Service) *Service {
	return &Service{
		db:         db,
		accountSvc: accountSvc,
		chatterSvc: chatterSvc,
	}
}

// ---------------------------------------------------------------------------
// Expense Categories (hr.expense.category)
// ---------------------------------------------------------------------------

func (s *Service) ListCategories(companyID uint) ([]ExpenseCategory, error) {
	var categories []ExpenseCategory
	err := s.db.Preload("Account").
		Where("company_id = ? OR company_id IS NULL OR company_id = 0", companyID).
		Order("id asc").
		Find(&categories).Error
	return categories, err
}

func (s *Service) GetCategory(id, companyID uint) (*ExpenseCategory, error) {
	var category ExpenseCategory
	err := s.db.Preload("Account").
		Where("id = ? AND (company_id = ? OR company_id IS NULL OR company_id = 0)", id, companyID).
		First(&category).Error
	if err != nil {
		return nil, errors.New("expense category not found")
	}
	return &category, nil
}

func (s *Service) CreateCategory(cat *ExpenseCategory) error {
	if cat.Name == "" || cat.Code == "" {
		return errors.New("category name and code are required")
	}
	if cat.AccountCode == "" {
		cat.AccountCode = "620100"
	}
	return s.db.Create(cat).Error
}

// ---------------------------------------------------------------------------
// Individual Expense Items (hr.expense)
// ---------------------------------------------------------------------------

func (s *Service) ListExpenses(companyID uint, employeeID uint, sheetID *uint, state string) ([]Expense, error) {
	var expenses []Expense
	query := s.db.Preload("Employee").
		Preload("Category").
		Preload("Project").
		Preload("Vehicle.Model").
		Where("company_id = ?", companyID)

	if employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}
	if sheetID != nil {
		query = query.Where("sheet_id = ?", *sheetID)
	}
	if state != "" {
		query = query.Where("state = ?", state)
	}

	err := query.Order("id desc").Find(&expenses).Error
	return expenses, err
}

func (s *Service) GetExpense(id, companyID uint) (*Expense, error) {
	var exp Expense
	err := s.db.Preload("Employee").
		Preload("Category").
		Preload("Project").
		Preload("Vehicle.Model").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&exp).Error
	if err != nil {
		return nil, errors.New("expense item not found")
	}
	return &exp, nil
}

func (s *Service) CreateExpense(exp *Expense, companyID uint, userID uint) error {
	if exp.Name == "" {
		return errors.New("expense description is required")
	}
	if exp.EmployeeID == 0 {
		return errors.New("employee is required")
	}
	if exp.CategoryID == 0 {
		return errors.New("expense category is required")
	}
	if exp.UnitPrice <= 0 {
		return errors.New("unit price must be greater than zero")
	}
	if exp.Quantity <= 0 {
		exp.Quantity = 1.0
	}
	exp.TotalAmount = math.Round((exp.UnitPrice*exp.Quantity)*100) / 100

	if exp.Date == "" {
		exp.Date = time.Now().Format("2006-01-02")
	}
	if exp.PaymentMode == "" {
		exp.PaymentMode = "own_account"
	}
	if exp.State == "" {
		exp.State = "draft"
	}
	exp.CompanyID = companyID

	// Generate Sequence
	if exp.ExpenseNumber == "" {
		year := time.Now().Format("2006")
		var count int64
		s.db.Model(&Expense{}).Where("company_id = ?", companyID).Count(&count)
		exp.ExpenseNumber = fmt.Sprintf("EXP-ITM/%s/%05d", year, count+1)
	}

	if err := s.db.Create(exp).Error; err != nil {
		return err
	}

	// Update sheet total if sheet is already attached
	if exp.SheetID != nil && *exp.SheetID > 0 {
		s.recomputeSheetTotal(*exp.SheetID, companyID)
	}

	return nil
}

func (s *Service) UpdateExpense(exp *Expense, companyID uint) error {
	existing, err := s.GetExpense(exp.ID, companyID)
	if err != nil {
		return err
	}
	if existing.State == "approved" || existing.State == "done" {
		return errors.New("cannot edit an expense that has already been approved or reimbursed")
	}

	if exp.UnitPrice > 0 && exp.Quantity > 0 {
		exp.TotalAmount = math.Round((exp.UnitPrice*exp.Quantity)*100) / 100
	}

	if err := s.db.Model(existing).Updates(exp).Error; err != nil {
		return err
	}

	if existing.SheetID != nil && *existing.SheetID > 0 {
		s.recomputeSheetTotal(*existing.SheetID, companyID)
	}

	return nil
}

func (s *Service) DeleteExpense(id, companyID uint) error {
	exp, err := s.GetExpense(id, companyID)
	if err != nil {
		return err
	}
	if exp.State == "approved" || exp.State == "done" {
		return errors.New("cannot delete an approved or paid expense")
	}

	sheetID := exp.SheetID
	if err := s.db.Delete(&Expense{}, "id = ? AND company_id = ?", id, companyID).Error; err != nil {
		return err
	}

	if sheetID != nil && *sheetID > 0 {
		s.recomputeSheetTotal(*sheetID, companyID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// Expense Sheets / Reports (hr.expense.sheet)
// ---------------------------------------------------------------------------

func (s *Service) ListExpenseSheets(companyID uint, employeeID uint, state string) ([]ExpenseSheet, error) {
	var sheets []ExpenseSheet
	query := s.db.Preload("Employee").
		Preload("Approver").
		Preload("AccountMove").
		Preload("Expenses.Category").
		Where("company_id = ?", companyID)

	if employeeID > 0 {
		query = query.Where("employee_id = ?", employeeID)
	}
	if state != "" {
		query = query.Where("state = ?", state)
	}

	err := query.Order("id desc").Find(&sheets).Error
	return sheets, err
}

func (s *Service) GetExpenseSheet(id, companyID uint) (*ExpenseSheet, error) {
	var sheet ExpenseSheet
	err := s.db.Preload("Employee").
		Preload("Approver").
		Preload("AccountMove.LineIDs.Account").
		Preload("Expenses.Category").
		Preload("Expenses.Project").
		Preload("Expenses.Vehicle.Model").
		Where("id = ? AND company_id = ?", id, companyID).
		First(&sheet).Error
	if err != nil {
		return nil, errors.New("expense sheet not found")
	}
	return &sheet, nil
}

func (s *Service) CreateExpenseSheet(sheet *ExpenseSheet, expenseIDs []uint, companyID uint, userID uint) (*ExpenseSheet, error) {
	if sheet.Name == "" {
		return nil, errors.New("expense sheet title / subject is required")
	}
	if sheet.EmployeeID == 0 {
		return nil, errors.New("employee is required")
	}
	if sheet.State == "" {
		sheet.State = "draft"
	}
	if sheet.PaymentMode == "" {
		sheet.PaymentMode = "own_account"
	}
	sheet.CompanyID = companyID

	// Generate Sequence EXP/YYYY/00001
	if sheet.SheetNumber == "" {
		year := time.Now().Format("2006")
		var count int64
		s.db.Model(&ExpenseSheet{}).Where("company_id = ?", companyID).Count(&count)
		sheet.SheetNumber = fmt.Sprintf("EXP/%s/%05d", year, count+1)
	}

	if err := s.db.Create(sheet).Error; err != nil {
		return nil, err
	}

	// Attach expenses if provided
	if len(expenseIDs) > 0 {
		if err := s.AttachExpensesToSheet(sheet.ID, expenseIDs, companyID); err != nil {
			return nil, err
		}
	}

	// Chatter log
	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.expense.sheet",
			ResID:       sheet.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Expense Sheet '%s' (%s) dibuat dalam status Draft.", sheet.SheetNumber, sheet.Name),
			MessageType: "notification",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.GetExpenseSheet(sheet.ID, companyID)
}

func (s *Service) AttachExpensesToSheet(sheetID uint, expenseIDs []uint, companyID uint) error {
	if len(expenseIDs) == 0 {
		return nil
	}

	// Update expenses
	err := s.db.Model(&Expense{}).
		Where("id IN ? AND company_id = ?", expenseIDs, companyID).
		Updates(map[string]interface{}{
			"sheet_id": sheetID,
			"state":    "reported",
		}).Error
	if err != nil {
		return err
	}

	s.recomputeSheetTotal(sheetID, companyID)
	return nil
}

func (s *Service) RemoveExpenseFromSheet(expenseID uint, companyID uint) error {
	exp, err := s.GetExpense(expenseID, companyID)
	if err != nil {
		return err
	}
	if exp.SheetID == nil {
		return nil
	}

	sheetID := *exp.SheetID
	err = s.db.Model(&Expense{}).
		Where("id = ? AND company_id = ?", expenseID, companyID).
		Updates(map[string]interface{}{
			"sheet_id": nil,
			"state":    "draft",
		}).Error
	if err != nil {
		return err
	}

	s.recomputeSheetTotal(sheetID, companyID)
	return nil
}

func (s *Service) ActionSubmitSheet(sheetID, companyID uint, userID uint) (*ExpenseSheet, error) {
	sheet, err := s.GetExpenseSheet(sheetID, companyID)
	if err != nil {
		return nil, err
	}
	if sheet.State != "draft" {
		return nil, errors.New("only draft expense sheets can be submitted")
	}
	if len(sheet.Expenses) == 0 {
		return nil, errors.New("cannot submit an expense sheet with no expense items attached")
	}

	sheet.State = "submit"
	if err := s.db.Save(sheet).Error; err != nil {
		return nil, err
	}

	// Mark all attached lines as reported
	s.db.Model(&Expense{}).
		Where("sheet_id = ? AND company_id = ?", sheetID, companyID).
		Update("state", "reported")

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.expense.sheet",
			ResID:       sheet.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Expense Sheet '%s' sebesar Rp %.2f diajukan untuk persetujuan Manajer/Finance.", sheet.SheetNumber, sheet.TotalAmount),
			MessageType: "notification",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.GetExpenseSheet(sheetID, companyID)
}

func (s *Service) ActionApproveSheet(sheetID, companyID uint, approverID uint) (*ExpenseSheet, error) {
	sheet, err := s.GetExpenseSheet(sheetID, companyID)
	if err != nil {
		return nil, err
	}
	if sheet.State != "submit" {
		return nil, errors.New("only submitted expense sheets can be approved")
	}

	sheet.State = "approve"
	sheet.ApproverID = &approverID
	sheet.ApprovalDate = time.Now().Format("2006-01-02 15:04:05")

	if err := s.db.Save(sheet).Error; err != nil {
		return nil, err
	}

	// Mark all attached lines as approved
	s.db.Model(&Expense{}).
		Where("sheet_id = ? AND company_id = ?", sheetID, companyID).
		Update("state", "approved")

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.expense.sheet",
			ResID:       sheet.ID,
			AuthorID:    approverID,
			Body:        fmt.Sprintf("Expense Sheet '%s' telah disetujui Manajer/Finance. Siap dibukukan ke General Ledger Accounting.", sheet.SheetNumber),
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.GetExpenseSheet(sheetID, companyID)
}

// ActionPostSheet posts the approved expense sheet directly to General Ledger (account.move)
func (s *Service) ActionPostSheet(sheetID, companyID uint, userID uint) (*ExpenseSheet, *account.AccountMove, error) {
	sheet, err := s.GetExpenseSheet(sheetID, companyID)
	if err != nil {
		return nil, nil, err
	}
	if sheet.State != "approve" {
		return nil, nil, errors.New("only approved expense sheets can be posted to accounting journal entries")
	}
	if sheet.AccountMoveID != nil && *sheet.AccountMoveID > 0 {
		return nil, nil, errors.New("this expense sheet is already posted to general ledger")
	}
	if s.accountSvc == nil {
		return nil, nil, errors.New("accounting module service is not available")
	}

	// 1. Resolve Credit Account (Hutang Karyawan vs Kas/Bank)
	var creditAccountID uint
	if sheet.PaymentMode == "company_account" {
		// Cash/Bank account: 110100 (Kas Utama)
		var cashAcc account.AccountAccount
		if err := s.db.Where("code = ? AND (company_id = ? OR company_id = 1)", "110100", companyID).First(&cashAcc).Error; err == nil {
			creditAccountID = cashAcc.ID
		} else {
			// Fallback: first cash account
			s.db.Where("type = 'asset_cash'").First(&cashAcc)
			creditAccountID = cashAcc.ID
		}
	} else {
		// Own account: 210000 (Hutang Usaha / Hutang Karyawan)
		var payableAcc account.AccountAccount
		if err := s.db.Where("code = ? AND (company_id = ? OR company_id = 1)", "210000", companyID).First(&payableAcc).Error; err == nil {
			creditAccountID = payableAcc.ID
		} else {
			s.db.Where("type = 'liability_payable'").First(&payableAcc)
			creditAccountID = payableAcc.ID
		}
	}

	if creditAccountID == 0 {
		creditAccountID = 5 // Standard default fallback
	}

	// 2. Resolve Journal (BILL or MISC or CSH)
	var journal account.AccountJournal
	if err := s.db.Where("code = ? AND (company_id = ? OR company_id = 1)", "BILL", companyID).First(&journal).Error; err != nil {
		if err := s.db.Where("code = ? AND (company_id = ? OR company_id = 1)", "MISC", companyID).First(&journal).Error; err != nil {
			s.db.First(&journal)
		}
	}

	journalID := journal.ID
	if journalID == 0 {
		journalID = 2 // BILL fallback
	}

	// 3. Build AccountMoveLines (Debit lines per expense category + single balanced Credit line)
	accountingDate := sheet.AccountingDate
	if accountingDate == "" {
		accountingDate = time.Now().Format("2006-01-02")
	}

	var moveLines []account.AccountMoveLine
	var totalDebit float64

	for _, exp := range sheet.Expenses {
		debitAccID := uint(8) // Default expense fallback (COGS / General Expense)

		if exp.Category != nil {
			var expenseAcc account.AccountAccount
			// Try lookup by category's AccountCode (e.g. 620100, 620200, 620400)
			accCode := exp.Category.AccountCode
			if accCode == "" {
				accCode = "620100"
			}
			if err := s.db.Where("code = ? AND (company_id = ? OR company_id = 1)", accCode, companyID).First(&expenseAcc).Error; err == nil {
				debitAccID = expenseAcc.ID
			} else if exp.Category.AccountID != nil && *exp.Category.AccountID > 0 {
				debitAccID = *exp.Category.AccountID
			} else {
				// Fallback to any expense account
				var anyExp account.AccountAccount
				if err := s.db.Where("type = 'expense'").First(&anyExp).Error; err == nil {
					debitAccID = anyExp.ID
				}
			}
		}

		line := account.AccountMoveLine{
			AccountID: debitAccID,
			Name:      fmt.Sprintf("%s: %s", exp.Category.Name, exp.Name),
			Debit:     exp.TotalAmount,
			Credit:    0,
			Balance:   exp.TotalAmount,
			Quantity:  exp.Quantity,
			PriceUnit: exp.UnitPrice,
		}
		moveLines = append(moveLines, line)
		totalDebit += exp.TotalAmount
	}

	// Balanced Credit Line
	creditLineLabel := fmt.Sprintf("Hutang Reimbursement - %s (%s)", sheet.Employee.Name, sheet.SheetNumber)
	if sheet.PaymentMode == "company_account" {
		creditLineLabel = fmt.Sprintf("Pembayaran Kas Operasional - %s", sheet.SheetNumber)
	}

	creditLine := account.AccountMoveLine{
		AccountID: creditAccountID,
		Name:      creditLineLabel,
		Debit:     0,
		Credit:    totalDebit,
		Balance:   -totalDebit,
		Quantity:  1,
		PriceUnit: totalDebit,
	}
	moveLines = append(moveLines, creditLine)

	partnerID := uint(1) // PT Odoo-Go ERP Solusindo
	if sheet.EmployeeID > 0 {
		// Employee can use default partner
		partnerID = 1
	}

	move := account.AccountMove{
		MoveType:       "entry",
		JournalID:      journalID,
		PartnerID:      partnerID,
		InvoiceDate:    accountingDate,
		Ref:            sheet.SheetNumber,
		State:          "draft",
		AmountUntaxed:  totalDebit,
		AmountTotal:    totalDebit,
		AmountResidual: totalDebit,
		Note:           fmt.Sprintf("Expense report %s untuk %s", sheet.SheetNumber, sheet.Employee.Name),
		LineIDs:        moveLines,
	}
	move.CompanyID = companyID

	// Create Move
	if err := s.accountSvc.CreateMove(&move); err != nil {
		return nil, nil, fmt.Errorf("failed to create accounting journal entry: %w", err)
	}

	// Post Move to General Ledger
	postedMove, err := s.accountSvc.ActionPost(move.ID, companyID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to post journal entry to General Ledger: %w", err)
	}

	// Update Sheet
	sheet.AccountMoveID = &postedMove.ID
	sheet.State = "post"
	sheet.AccountingDate = accountingDate
	if err := s.db.Save(sheet).Error; err != nil {
		return nil, nil, err
	}

	// Mark expenses state = approved/posted
	s.db.Model(&Expense{}).
		Where("sheet_id = ? AND company_id = ?", sheetID, companyID).
		Update("state", "approved")

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.expense.sheet",
			ResID:       sheet.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Expense Sheet '%s' telah berhasil dibukukan ke General Ledger Accounting dengan Entri Jurnal '%s' (Total Rp %.2f).", sheet.SheetNumber, postedMove.Name, totalDebit),
			MessageType: "notification",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return sheet, postedMove, nil
}

func (s *Service) ActionPaySheet(sheetID, companyID uint, userID uint) (*ExpenseSheet, error) {
	sheet, err := s.GetExpenseSheet(sheetID, companyID)
	if err != nil {
		return nil, err
	}
	if sheet.State != "post" && sheet.State != "approve" {
		return nil, errors.New("only posted or approved expense sheets can be marked as paid / reimbursed")
	}

	sheet.State = "done"
	if err := s.db.Save(sheet).Error; err != nil {
		return nil, err
	}

	// Update all attached expense items to done
	s.db.Model(&Expense{}).
		Where("sheet_id = ? AND company_id = ?", sheetID, companyID).
		Update("state", "done")

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.expense.sheet",
			ResID:       sheet.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Dana Reimbursement Expense '%s' sebesar Rp %.2f telah ditransfer / dibayarkan lunas kepada karyawan %s.", sheet.SheetNumber, sheet.TotalAmount, sheet.Employee.Name),
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.GetExpenseSheet(sheetID, companyID)
}

func (s *Service) ActionRefuseSheet(sheetID, companyID uint, reason string, userID uint) (*ExpenseSheet, error) {
	sheet, err := s.GetExpenseSheet(sheetID, companyID)
	if err != nil {
		return nil, err
	}
	if sheet.State == "done" || sheet.State == "post" {
		return nil, errors.New("cannot refuse an expense sheet that is already posted or paid")
	}

	sheet.State = "refuse"
	sheet.RefusalReason = reason
	if err := s.db.Save(sheet).Error; err != nil {
		return nil, err
	}

	// Update attached expenses to refused
	s.db.Model(&Expense{}).
		Where("sheet_id = ? AND company_id = ?", sheetID, companyID).
		Update("state", "refused")

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.expense.sheet",
			ResID:       sheet.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Expense Sheet '%s' ditolak. Alasan penolakan: %s", sheet.SheetNumber, reason),
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.GetExpenseSheet(sheetID, companyID)
}

// ---------------------------------------------------------------------------
// Helpers & Analytics
// ---------------------------------------------------------------------------

func (s *Service) recomputeSheetTotal(sheetID, companyID uint) {
	var total float64
	s.db.Model(&Expense{}).
		Where("sheet_id = ? AND company_id = ? AND state != 'refused'", sheetID, companyID).
		Select("COALESCE(SUM(total_amount), 0)").
		Scan(&total)

	s.db.Model(&ExpenseSheet{}).
		Where("id = ? AND company_id = ?", sheetID, companyID).
		Update("total_amount", total)
}

func (s *Service) GetExpenseStats(companyID uint) (map[string]interface{}, error) {
	var totalItems int64
	var totalItemAmount float64
	s.db.Model(&Expense{}).Where("company_id = ?", companyID).Count(&totalItems)
	s.db.Model(&Expense{}).Where("company_id = ?", companyID).Select("COALESCE(SUM(total_amount), 0)").Scan(&totalItemAmount)

	var totalSheets int64
	s.db.Model(&ExpenseSheet{}).Where("company_id = ?", companyID).Count(&totalSheets)

	var pendingCount int64
	var pendingAmount float64
	s.db.Model(&ExpenseSheet{}).Where("company_id = ? AND state IN ('draft', 'submit')", companyID).Count(&pendingCount)
	s.db.Model(&ExpenseSheet{}).Where("company_id = ? AND state IN ('draft', 'submit')", companyID).Select("COALESCE(SUM(total_amount), 0)").Scan(&pendingAmount)

	var approvedCount int64
	var approvedAmount float64
	s.db.Model(&ExpenseSheet{}).Where("company_id = ? AND state IN ('approve', 'post')", companyID).Count(&approvedCount)
	s.db.Model(&ExpenseSheet{}).Where("company_id = ? AND state IN ('approve', 'post')", companyID).Select("COALESCE(SUM(total_amount), 0)").Scan(&approvedAmount)

	var paidCount int64
	var paidAmount float64
	s.db.Model(&ExpenseSheet{}).Where("company_id = ? AND state = 'done'", companyID).Count(&paidCount)
	s.db.Model(&ExpenseSheet{}).Where("company_id = ? AND state = 'done'", companyID).Select("COALESCE(SUM(total_amount), 0)").Scan(&paidAmount)

	// Category breakdown
	type CatStat struct {
		CategoryName string  `json:"category_name"`
		Code         string  `json:"code"`
		Color        string  `json:"color"`
		TotalAmount  float64 `json:"total_amount"`
		ItemCount    int64   `json:"item_count"`
	}
	var catStats []CatStat
	s.db.Table("expenses").
		Select("expense_categories.name as category_name, expense_categories.code, expense_categories.color, COALESCE(SUM(expenses.total_amount), 0) as total_amount, COUNT(expenses.id) as item_count").
		Joins("LEFT JOIN expense_categories ON expense_categories.id = expenses.category_id").
		Where("expenses.company_id = ?", companyID).
		Group("expense_categories.name, expense_categories.code, expense_categories.color").
		Order("total_amount desc").
		Scan(&catStats)

	// Fleet vehicle operational expenses summary
	var fleetTotalAmount float64
	var fleetExpenseCount int64
	s.db.Model(&Expense{}).Where("company_id = ? AND vehicle_id IS NOT NULL", companyID).Count(&fleetExpenseCount)
	s.db.Model(&Expense{}).Where("company_id = ? AND vehicle_id IS NOT NULL", companyID).Select("COALESCE(SUM(total_amount), 0)").Scan(&fleetTotalAmount)

	// Project operational expenses summary
	var projectTotalAmount float64
	var projectExpenseCount int64
	s.db.Model(&Expense{}).Where("company_id = ? AND project_id IS NOT NULL", companyID).Count(&projectExpenseCount)
	s.db.Model(&Expense{}).Where("company_id = ? AND project_id IS NOT NULL", companyID).Select("COALESCE(SUM(total_amount), 0)").Scan(&projectTotalAmount)

	return map[string]interface{}{
		"total_items":            totalItems,
		"total_item_amount":      totalItemAmount,
		"total_sheets":           totalSheets,
		"pending_count":          pendingCount,
		"pending_amount":         pendingAmount,
		"approved_count":         approvedCount,
		"approved_amount":        approvedAmount,
		"paid_count":             paidCount,
		"paid_amount":            paidAmount,
		"categories":             catStats,
		"fleet_expense_count":    fleetExpenseCount,
		"fleet_total_amount":     fleetTotalAmount,
		"project_expense_count":  projectExpenseCount,
		"project_total_amount":   projectTotalAmount,
	}, nil
}
