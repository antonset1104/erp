package intercompany

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Sequence Number Generators
// ---------------------------------------------------------------------------

func (r *Repository) GenerateRunCode(fiscalYear int) (string, error) {
	if fiscalYear <= 0 {
		fiscalYear = time.Now().Year()
	}
	var count int64
	if err := r.db.Model(&ConsolidationRun{}).Where("fiscal_year = ?", fiscalYear).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("CNS/%d/%05d", fiscalYear, count+1), nil
}

func (r *Repository) GenerateTransactionCode(year int) (string, error) {
	if year <= 0 {
		year = time.Now().Year()
	}
	var count int64
	if err := r.db.Model(&IntercompanyTransaction{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("ICT/%d/%05d", year, count+1), nil
}

func (r *Repository) GenerateGroupCode() (string, error) {
	var count int64
	if err := r.db.Model(&IntercompanyGroup{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("GRP/%d/%03d", time.Now().Year(), count+1), nil
}

func (r *Repository) GenerateRelationCode() (string, error) {
	var count int64
	if err := r.db.Model(&IntercompanyRelation{}).Count(&count).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("REL/%d/%03d", time.Now().Year(), count+1), nil
}

// ---------------------------------------------------------------------------
// Group CRUD
// ---------------------------------------------------------------------------

func (r *Repository) CreateGroup(group *IntercompanyGroup) error {
	return r.db.Create(group).Error
}

func (r *Repository) GetGroup(id uint) (*IntercompanyGroup, error) {
	var group IntercompanyGroup
	if err := r.db.First(&group, id).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *Repository) GetGroupByHolding(holdingID uint) (*IntercompanyGroup, error) {
	var group IntercompanyGroup
	if err := r.db.Where("holding_company_id = ?", holdingID).First(&group).Error; err != nil {
		return nil, err
	}
	return &group, nil
}

func (r *Repository) ListGroups() ([]IntercompanyGroup, error) {
	var groups []IntercompanyGroup
	err := r.db.Order("id DESC").Find(&groups).Error
	return groups, err
}

func (r *Repository) UpdateGroup(group *IntercompanyGroup) error {
	return r.db.Save(group).Error
}

// ---------------------------------------------------------------------------
// Relation CRUD
// ---------------------------------------------------------------------------

func (r *Repository) CreateRelation(rel *IntercompanyRelation) error {
	return r.db.Create(rel).Error
}

func (r *Repository) GetRelation(id uint) (*IntercompanyRelation, error) {
	var rel IntercompanyRelation
	if err := r.db.First(&rel, id).Error; err != nil {
		return nil, err
	}
	return &rel, nil
}

func (r *Repository) GetRelationByCompanies(fromCompanyID, toCompanyID uint) (*IntercompanyRelation, error) {
	var rel IntercompanyRelation
	err := r.db.Where("(from_company_id = ? AND to_company_id = ?) OR (from_company_id = ? AND to_company_id = ?)",
		fromCompanyID, toCompanyID, toCompanyID, fromCompanyID).First(&rel).Error
	if err != nil {
		return nil, err
	}
	return &rel, nil
}

func (r *Repository) ListRelations(groupID uint) ([]IntercompanyRelation, error) {
	var rels []IntercompanyRelation
	q := r.db.Model(&IntercompanyRelation{})
	if groupID > 0 {
		q = q.Where("group_id = ?", groupID)
	}
	err := q.Order("id ASC").Find(&rels).Error
	return rels, err
}

// ---------------------------------------------------------------------------
// Transactions CRUD
// ---------------------------------------------------------------------------

func (r *Repository) CreateTransaction(tx *IntercompanyTransaction) error {
	return r.db.Create(tx).Error
}

func (r *Repository) GetTransaction(id uint) (*IntercompanyTransaction, error) {
	var tx IntercompanyTransaction
	if err := r.db.First(&tx, id).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}

func (r *Repository) ListTransactions(groupID uint) ([]IntercompanyTransaction, error) {
	var txs []IntercompanyTransaction
	q := r.db.Model(&IntercompanyTransaction{})
	if groupID > 0 {
		q = q.Where("group_id = ?", groupID)
	}
	err := q.Order("id DESC").Limit(100).Find(&txs).Error
	return txs, err
}

// ---------------------------------------------------------------------------
// ConsolidationRun CRUD
// ---------------------------------------------------------------------------

func (r *Repository) CreateConsolidationRun(run *ConsolidationRun) error {
	return r.db.Create(run).Error
}

func (r *Repository) GetConsolidationRun(id uint) (*ConsolidationRun, error) {
	var run ConsolidationRun
	if err := r.db.Preload("Lines").First(&run, id).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func (r *Repository) ListConsolidationRuns(groupID uint) ([]ConsolidationRun, error) {
	var runs []ConsolidationRun
	q := r.db.Model(&ConsolidationRun{})
	if groupID > 0 {
		q = q.Where("group_id = ?", groupID)
	}
	err := q.Order("id DESC").Find(&runs).Error
	return runs, err
}

// ---------------------------------------------------------------------------
// Stats & Analytics Query
// ---------------------------------------------------------------------------

func (r *Repository) GetStats(holdingCompanyID uint) (*IntercompanyStats, error) {
	var totalGroups, totalRelations, totalTxs, totalRuns int64
	var totalElim float64

	r.db.Model(&IntercompanyGroup{}).Count(&totalGroups)
	r.db.Model(&IntercompanyRelation{}).Count(&totalRelations)
	r.db.Model(&IntercompanyTransaction{}).Count(&totalTxs)
	r.db.Model(&ConsolidationRun{}).Count(&totalRuns)

	r.db.Model(&ConsolidationRun{}).Select("COALESCE(SUM(eliminated_receivables + eliminated_revenue), 0)").Scan(&totalElim)

	var lastRun ConsolidationRun
	r.db.Order("id DESC").First(&lastRun)

	return &IntercompanyStats{
		TotalGroups:               totalGroups,
		TotalRelations:            totalRelations,
		TotalMirroredTransactions: totalTxs,
		TotalConsolidationRuns:    totalRuns,
		TotalEliminatedVolume:     totalElim,
		LastConsolidatedRevenue:   lastRun.ConsolidatedRevenue,
		LastConsolidatedNetIncome: lastRun.ConsolidatedNetIncome,
	}, nil
}
