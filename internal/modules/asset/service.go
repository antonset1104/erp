package asset

import (
	"errors"
	"fmt"
	"math"
	"time"

	"erp-golang/internal/modules/account"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	accountSvc *account.Service
}

func NewService(db *gorm.DB, accountSvc *account.Service) *Service {
	return &Service{
		db:         db,
		accountSvc: accountSvc,
	}
}

func (s *Service) CreateAsset(a *Asset) error {
	if a.Name == "" {
		return errors.New("asset name is required")
	}
	if a.OriginalValue <= 0 {
		return errors.New("original value must be greater than 0")
	}
	if a.MethodNumber <= 0 {
		a.MethodNumber = 36 // default 3 years / 36 months
	}
	if a.Method == "" {
		a.Method = "linear"
	}
	if a.MethodPeriod == "" {
		a.MethodPeriod = "months"
	}
	if a.State == "" {
		a.State = "draft"
	}

	a.SalvableValue = a.OriginalValue - a.SalvageValue
	if a.SalvableValue < 0 {
		return errors.New("salvage value cannot exceed original value")
	}
	a.BookValue = a.OriginalValue
	a.ValueDepreciated = 0

	// Generate Code if empty
	if a.Code == "" {
		var count int64
		s.db.Model(&Asset{}).Count(&count)
		a.Code = fmt.Sprintf("AST/%05d", count+1)
	}

	if err := s.db.Create(a).Error; err != nil {
		return err
	}

	// Compute depreciation schedule immediately
	_, err := s.ComputeDepreciationBoard(a.ID, a.CompanyID)
	return err
}

func (s *Service) ListAssets(companyID uint, state string) ([]Asset, error) {
	var assets []Asset
	query := s.db.Where("company_id = ?", companyID)
	if state != "" && state != "all" {
		query = query.Where("state = ?", state)
	}
	err := query.
		Preload("Partner").
		Preload("AssetAccount").
		Preload("DepreciationAccount").
		Preload("AccumulatedAccount").
		Preload("Journal").
		Order("id desc").
		Find(&assets).Error
	return assets, err
}

func (s *Service) GetAsset(id, companyID uint) (*Asset, error) {
	var a Asset
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("Partner").
		Preload("AssetAccount").
		Preload("DepreciationAccount").
		Preload("AccumulatedAccount").
		Preload("Journal").
		Preload("DepreciationLines.Move").
		First(&a).Error
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Service) ComputeDepreciationBoard(assetID, companyID uint) (*Asset, error) {
	var asset Asset
	if err := s.db.Where("id = ? AND company_id = ?", assetID, companyID).First(&asset).Error; err != nil {
		return nil, errors.New("asset not found")
	}

	// Delete unposted lines
	s.db.Where("asset_id = ? AND is_posted = ?", asset.ID, false).Delete(&AssetDepreciationLine{})

	// Check if already any posted lines
	var postedCount int64
	var postedSum float64
	s.db.Model(&AssetDepreciationLine{}).
		Where("asset_id = ? AND is_posted = ?", asset.ID, true).
		Count(&postedCount)
	s.db.Model(&AssetDepreciationLine{}).
		Where("asset_id = ? AND is_posted = ?", asset.ID, true).
		Select("COALESCE(SUM(amount), 0)").Scan(&postedSum)

	remainingPeriods := asset.MethodNumber - int(postedCount)
	if remainingPeriods <= 0 {
		return &asset, nil
	}

	remainingAmountToDepreciate := asset.SalvableValue - postedSum
	monthlyAmount := math.Round((remainingAmountToDepreciate/float64(remainingPeriods))*100) / 100

	startDate, err := time.Parse("2006-01-02", asset.PurchaseDate)
	if err != nil {
		startDate = time.Now()
	}

	currentDepreciated := postedSum
	var lines []AssetDepreciationLine

	for i := 1; i <= remainingPeriods; i++ {
		seq := int(postedCount) + i
		date := startDate.AddDate(0, seq, 0).Format("2006-01-02")

		amount := monthlyAmount
		if i == remainingPeriods {
			// Adjust exact rounding on final period
			amount = math.Round((asset.SalvableValue-currentDepreciated)*100) / 100
		}

		currentDepreciated += amount
		rem := math.Max(asset.SalvageValue, asset.OriginalValue-currentDepreciated)

		line := AssetDepreciationLine{
			AssetID:          asset.ID,
			Sequence:         seq,
			DepreciationDate: date,
			Amount:           amount,
			DepreciatedValue: currentDepreciated,
			RemainingValue:   rem,
			IsPosted:         false,
		}
		line.CompanyID = companyID
		lines = append(lines, line)
	}

	if len(lines) > 0 {
		if err := s.db.Create(&lines).Error; err != nil {
			return nil, err
		}
	}

	return s.GetAsset(asset.ID, companyID)
}

func (s *Service) ActionConfirm(assetID, companyID uint) (*Asset, error) {
	var asset Asset
	if err := s.db.Where("id = ? AND company_id = ?", assetID, companyID).First(&asset).Error; err != nil {
		return nil, errors.New("asset not found")
	}

	if asset.State != "draft" {
		return nil, errors.New("only draft assets can be confirmed")
	}

	asset.State = "running"
	if err := s.db.Save(&asset).Error; err != nil {
		return nil, err
	}
	return s.GetAsset(asset.ID, companyID)
}

func (s *Service) PostDepreciationLine(lineID, companyID, userID uint) (*AssetDepreciationLine, error) {
	var line AssetDepreciationLine
	if err := s.db.Where("id = ? AND company_id = ?", lineID, companyID).First(&line).Error; err != nil {
		return nil, errors.New("depreciation line not found")
	}

	if line.IsPosted {
		return nil, errors.New("depreciation line is already posted to general ledger")
	}

	var asset Asset
	if err := s.db.Where("id = ? AND company_id = ?", line.AssetID, companyID).First(&asset).Error; err != nil {
		return nil, errors.New("parent asset not found")
	}

	if asset.State != "running" {
		return nil, errors.New("depreciation can only be posted for running assets")
	}

	// Create Balanced AccountMove entry in General Ledger
	partnerID := uint(1)
	if asset.PartnerID != nil && *asset.PartnerID > 0 {
		partnerID = *asset.PartnerID
	}

	move := account.AccountMove{
		MoveType:       "entry",
		JournalID:      asset.JournalID,
		PartnerID:      partnerID,
		InvoiceDate:    line.DepreciationDate,
		Ref:            fmt.Sprintf("%s (Penyusutan #%d)", asset.Code, line.Sequence),
		State:          "draft",
		AmountUntaxed:  line.Amount,
		AmountTotal:    line.Amount,
		AmountResidual: 0,
		Note:           fmt.Sprintf("Penyusutan otomatis aset %s untuk periode %s", asset.Name, line.DepreciationDate),
		LineIDs: []account.AccountMoveLine{
			{
				AccountID: asset.DepreciationAccountID,
				Name:      fmt.Sprintf("Beban Penyusutan %s (#%d)", asset.Name, line.Sequence),
				Debit:     line.Amount,
				Credit:    0,
				Balance:   line.Amount,
			},
			{
				AccountID: asset.AccumulatedAccountID,
				Name:      fmt.Sprintf("Akumulasi Penyusutan %s (#%d)", asset.Name, line.Sequence),
				Debit:     0,
				Credit:    line.Amount,
				Balance:   -line.Amount,
			},
		},
	}
	move.CompanyID = companyID
	move.CreateUID = userID
	move.WriteUID = userID

	if err := s.accountSvc.CreateMove(&move); err != nil {
		return nil, fmt.Errorf("failed to create GL journal entry: %w", err)
	}

	// Post the entry to finalize in General Ledger
	postedMove, err := s.accountSvc.ActionPost(move.ID, companyID)
	if err != nil {
		return nil, fmt.Errorf("failed to post GL entry: %w", err)
	}

	// Update Line status
	line.IsPosted = true
	line.MoveID = &postedMove.ID
	if err := s.db.Save(&line).Error; err != nil {
		return nil, err
	}

	// Recalculate asset values
	asset.ValueDepreciated += line.Amount
	asset.BookValue = math.Max(asset.SalvageValue, asset.OriginalValue-asset.ValueDepreciated)

	// If fully depreciated, mark asset as closed
	if asset.BookValue <= asset.SalvageValue {
		asset.State = "close"
	}
	s.db.Save(&asset)

	line.Move = postedMove
	return &line, nil
}

func (s *Service) AutoPostDueDepreciations(companyID, userID uint, asOfDate string) (int, error) {
	if asOfDate == "" {
		asOfDate = time.Now().Format("2006-01-02")
	}

	var lines []AssetDepreciationLine
	err := s.db.Joins("JOIN assets ON assets.id = asset_depreciation_lines.asset_id").
		Where("asset_depreciation_lines.company_id = ? AND asset_depreciation_lines.is_posted = ? AND asset_depreciation_lines.depreciation_date <= ? AND assets.state = ?",
			companyID, false, asOfDate, "running").
		Order("asset_depreciation_lines.sequence asc").
		Find(&lines).Error
	if err != nil {
		return 0, err
	}

	postedCount := 0
	for _, l := range lines {
		if _, err := s.PostDepreciationLine(l.ID, companyID, userID); err == nil {
			postedCount++
		}
	}

	return postedCount, nil
}
