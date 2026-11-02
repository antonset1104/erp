package recruitment

import (
	"erp-golang/internal/core/models"
	"erp-golang/internal/modules/hr"
)

// RecruitmentStage corresponds to standard enterprise hr.recruitment.stage
type RecruitmentStage struct {
	models.ModelBase
	Name      string `gorm:"size:128;not null" json:"name"`
	Sequence  int    `gorm:"default:10" json:"sequence"`
	Folded    bool   `gorm:"default:false" json:"folded"`
	IsHired   bool   `gorm:"default:false" json:"is_hired"`   // Marks stage as hired
	IsRefused bool   `gorm:"default:false" json:"is_refused"` // Marks stage as rejected
}

func (RecruitmentStage) TableName() string {
	return "hr_recruitment_stages"
}

// JobOpening corresponds to standard enterprise hr.job (open positions for recruitment)
type JobOpening struct {
	models.ModelBase
	Title             string         `gorm:"size:128;not null" json:"title"`
	DepartmentID      uint           `gorm:"not null;index" json:"department_id"`
	Department        *hr.Department `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	JobID             *uint          `gorm:"index" json:"job_id,omitempty"` // Linked hr.job position
	JobPosition       *hr.JobPosition `gorm:"foreignKey:JobID" json:"job_position,omitempty"`
	ExpectedEmployees int            `gorm:"default:1" json:"expected_employees"`
	CurrentHired      int            `gorm:"default:0" json:"current_hired"`
	Requirements      string         `gorm:"type:text" json:"requirements"`
	Description       string         `gorm:"type:text" json:"description"`
	SalaryMin         float64        `gorm:"default:0" json:"salary_min"`
	SalaryMax         float64        `gorm:"default:0" json:"salary_max"`
	State             string         `gorm:"size:32;default:'open';index" json:"state"` // "open", "closed"
}

func (JobOpening) TableName() string {
	return "hr_job_openings"
}

// Applicant corresponds to standard enterprise hr.applicant
type Applicant struct {
	models.ModelBase
	Name               string            `gorm:"size:128;not null" json:"name"`
	Email              string            `gorm:"size:128;not null;index" json:"email"`
	Phone              string            `gorm:"size:64" json:"phone"`
	JobOpeningID       uint              `gorm:"not null;index" json:"job_opening_id"`
	JobOpening         *JobOpening       `gorm:"foreignKey:JobOpeningID" json:"job_opening,omitempty"`
	DepartmentID       uint              `gorm:"not null;index" json:"department_id"`
	Department         *hr.Department    `gorm:"foreignKey:DepartmentID" json:"department,omitempty"`
	StageID            uint              `gorm:"not null;index" json:"stage_id"`
	Stage              *RecruitmentStage `gorm:"foreignKey:StageID" json:"stage,omitempty"`
	SalaryExpected     float64           `gorm:"default:0" json:"salary_expected"`
	ResumeText         string            `gorm:"type:text" json:"resume_text"`
	
	// Cloudflare Workers AI Talent Screening Metrics
	AIScore            float64           `gorm:"default:0" json:"ai_score"` // 0.0 - 100.0%
	AISummary          string            `gorm:"type:text" json:"ai_summary"`
	AIRecommendation   string            `gorm:"size:64" json:"ai_recommendation"` // "Strong Hire", "Hire", "Consider / Hold", "Reject"
	AIFitAnalysis      string            `gorm:"type:text" json:"ai_fit_analysis"`
	KeyStrengths       string            `gorm:"type:text" json:"key_strengths"`
	KeyWeaknesses      string            `gorm:"type:text" json:"key_weaknesses"`
	SuggestedQuestions string            `gorm:"type:text" json:"suggested_questions"`
	AIScreenedAt       string            `gorm:"size:32" json:"ai_screened_at,omitempty"`
	AIEngineSource     string            `gorm:"size:64" json:"ai_engine_source,omitempty"`

	// Hired or Refused
	HiredEmployeeID    *uint             `gorm:"index" json:"hired_employee_id,omitempty"`
	HiredEmployee      *hr.Employee      `gorm:"foreignKey:HiredEmployeeID" json:"hired_employee,omitempty"`
	State              string            `gorm:"size:32;default:'new';index" json:"state"` // "new", "in_progress", "hired", "refused"
	RefusalReason      string            `gorm:"size:255" json:"refusal_reason,omitempty"`
}

func (Applicant) TableName() string {
	return "hr_applicants"
}

// Request and Filter DTOs
type CreateJobOpeningRequest struct {
	Title             string  `json:"title" binding:"required"`
	DepartmentID      uint    `json:"department_id" binding:"required"`
	JobID             *uint   `json:"job_id"`
	ExpectedEmployees int     `json:"expected_employees"`
	Requirements      string  `json:"requirements"`
	Description       string  `json:"description"`
	SalaryMin         float64 `json:"salary_min"`
	SalaryMax         float64 `json:"salary_max"`
}

type CreateApplicantRequest struct {
	Name           string  `json:"name" binding:"required"`
	Email          string  `json:"email" binding:"required,email"`
	Phone          string  `json:"phone"`
	JobOpeningID   uint    `json:"job_opening_id" binding:"required"`
	SalaryExpected float64 `json:"salary_expected"`
	ResumeText     string  `json:"resume_text" binding:"required"`
}

type UpdateStageRequest struct {
	StageID uint `json:"stage_id" binding:"required"`
}

type HireApplicantRequest struct {
	BasicSalary float64 `json:"basic_salary"`
	BankName    string  `json:"bank_name"`
	BankAccount string  `json:"bank_account"`
}

type RefuseApplicantRequest struct {
	Reason string `json:"reason" binding:"required"`
}

type ApplicantFilter struct {
	JobOpeningID uint
	DepartmentID uint
	StageID      uint
	State        string
	MinAIScore   float64
	Search       string
	Limit        int
	Offset       int
}

type AIScreeningResult struct {
	ApplicantID        uint     `json:"applicant_id"`
	ApplicantName      string   `json:"applicant_name"`
	JobTitle           string   `json:"job_title"`
	AIScore            float64  `json:"ai_score"`
	AIRecommendation   string   `json:"ai_recommendation"`
	AISummary          string   `json:"ai_summary"`
	AIFitAnalysis      string   `json:"ai_fit_analysis"`
	KeyStrengths       []string `json:"key_strengths"`
	KeyWeaknesses      []string `json:"key_weaknesses"`
	SuggestedQuestions []string `json:"suggested_questions"`
	EngineSource       string   `json:"engine_source"`
}

type RecruitmentStats struct {
	TotalOpenings     int64   `json:"total_openings"`
	TotalApplicants   int64   `json:"total_applicants"`
	InProgressCount   int64   `json:"in_progress_count"`
	HiredCount        int64   `json:"hired_count"`
	RefusedCount      int64   `json:"refused_count"`
	AverageAIScore    float64 `json:"average_ai_score"`
	TopJobOpening     string  `json:"top_job_opening"`
}
