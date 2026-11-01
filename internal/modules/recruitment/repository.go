package recruitment

import (
	"strings"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// ---------------------------------------------------------------------------
// Recruitment Stages
// ---------------------------------------------------------------------------

func (r *Repository) ListStages(companyID uint) ([]RecruitmentStage, error) {
	var stages []RecruitmentStage
	err := r.db.Where("company_id = ? OR company_id = 0", companyID).
		Order("sequence asc, id asc").
		Find(&stages).Error
	return stages, err
}

func (r *Repository) GetStage(id uint, companyID uint) (*RecruitmentStage, error) {
	var stage RecruitmentStage
	err := r.db.Where("id = ? AND (company_id = ? OR company_id = 0)", id, companyID).First(&stage).Error
	if err != nil {
		return nil, err
	}
	return &stage, nil
}

func (r *Repository) CreateStage(stage *RecruitmentStage) error {
	return r.db.Create(stage).Error
}

// ---------------------------------------------------------------------------
// Job Openings
// ---------------------------------------------------------------------------

func (r *Repository) ListJobOpenings(companyID uint, state string) ([]JobOpening, error) {
	var openings []JobOpening
	query := r.db.Preload("Department").Preload("JobPosition").
		Where("company_id = ?", companyID)

	if state != "" {
		query = query.Where("state = ?", state)
	}

	err := query.Order("id desc").Find(&openings).Error
	return openings, err
}

func (r *Repository) GetJobOpening(id uint, companyID uint) (*JobOpening, error) {
	var opening JobOpening
	err := r.db.Preload("Department").Preload("JobPosition").
		Where("id = ? AND company_id = ?", id, companyID).First(&opening).Error
	if err != nil {
		return nil, err
	}
	return &opening, nil
}

func (r *Repository) CreateJobOpening(opening *JobOpening) error {
	return r.db.Create(opening).Error
}

func (r *Repository) UpdateJobOpening(opening *JobOpening) error {
	return r.db.Save(opening).Error
}

// ---------------------------------------------------------------------------
// Applicants
// ---------------------------------------------------------------------------

func (r *Repository) ListApplicants(companyID uint, filter ApplicantFilter) ([]Applicant, int64, error) {
	var applicants []Applicant
	var total int64

	query := r.db.Model(&Applicant{}).
		Preload("JobOpening.Department").
		Preload("Department").
		Preload("Stage").
		Preload("HiredEmployee").
		Where("hr_applicants.company_id = ?", companyID)

	if filter.JobOpeningID > 0 {
		query = query.Where("hr_applicants.job_opening_id = ?", filter.JobOpeningID)
	}
	if filter.DepartmentID > 0 {
		query = query.Where("hr_applicants.department_id = ?", filter.DepartmentID)
	}
	if filter.StageID > 0 {
		query = query.Where("hr_applicants.stage_id = ?", filter.StageID)
	}
	if filter.State != "" {
		query = query.Where("hr_applicants.state = ?", filter.State)
	}
	if filter.MinAIScore > 0 {
		query = query.Where("hr_applicants.ai_score >= ?", filter.MinAIScore)
	}
	if filter.Search != "" {
		s := "%" + strings.ToLower(filter.Search) + "%"
		query = query.Where("(LOWER(hr_applicants.name) LIKE ? OR LOWER(hr_applicants.email) LIKE ? OR LOWER(hr_applicants.resume_text) LIKE ?)", s, s, s)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	err := query.Order("hr_applicants.ai_score desc, hr_applicants.id desc").Find(&applicants).Error
	return applicants, total, err
}

func (r *Repository) GetApplicant(id uint, companyID uint) (*Applicant, error) {
	var applicant Applicant
	err := r.db.Preload("JobOpening.Department").
		Preload("Department").
		Preload("Stage").
		Preload("HiredEmployee").
		Where("id = ? AND company_id = ?", id, companyID).First(&applicant).Error
	if err != nil {
		return nil, err
	}
	return &applicant, nil
}

func (r *Repository) CreateApplicant(applicant *Applicant) error {
	return r.db.Create(applicant).Error
}

func (r *Repository) UpdateApplicant(applicant *Applicant) error {
	return r.db.Save(applicant).Error
}

func (r *Repository) DeleteApplicant(id uint, companyID uint) error {
	return r.db.Where("id = ? AND company_id = ?", id, companyID).Delete(&Applicant{}).Error
}

// ---------------------------------------------------------------------------
// KPI Analytics
// ---------------------------------------------------------------------------

func (r *Repository) GetRecruitmentStats(companyID uint) (*RecruitmentStats, error) {
	var stats RecruitmentStats

	// Total Job Openings
	r.db.Model(&JobOpening{}).Where("company_id = ? AND state = 'open'", companyID).Count(&stats.TotalOpenings)

	// Total Applicants
	r.db.Model(&Applicant{}).Where("company_id = ?", companyID).Count(&stats.TotalApplicants)

	// In Progress Count
	r.db.Model(&Applicant{}).Where("company_id = ? AND state IN ('new', 'in_progress')", companyID).Count(&stats.InProgressCount)

	// Hired Count
	r.db.Model(&Applicant{}).Where("company_id = ? AND state = 'hired'", companyID).Count(&stats.HiredCount)

	// Refused Count
	r.db.Model(&Applicant{}).Where("company_id = ? AND state = 'refused'", companyID).Count(&stats.RefusedCount)

	// Average AI Score
	var avgScore struct {
		Avg float64
	}
	r.db.Model(&Applicant{}).
		Where("company_id = ? AND ai_score > 0", companyID).
		Select("COALESCE(AVG(ai_score), 0) as avg").
		Scan(&avgScore)
	stats.AverageAIScore = avgScore.Avg

	// Top Job Opening by applicant count
	var topJob struct {
		Title string
	}
	r.db.Model(&Applicant{}).
		Select("hr_job_openings.title, COUNT(*) as cnt").
		Joins("JOIN hr_job_openings ON hr_job_openings.id = hr_applicants.job_opening_id").
		Where("hr_applicants.company_id = ?", companyID).
		Group("hr_job_openings.title").
		Order("cnt desc").
		Limit(1).
		Scan(&topJob)
	stats.TopJobOpening = topJob.Title
	if stats.TopJobOpening == "" {
		stats.TopJobOpening = "-"
	}

	return &stats, nil
}
