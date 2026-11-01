package project

import (
	"errors"
	"math"

	"gorm.io/gorm"
)

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) CreateProject(p *Project) error {
	if p.Name == "" {
		return errors.New("project name is required")
	}
	if p.Status == "" {
		p.Status = "active"
	}
	if p.Color == "" {
		p.Color = "#6366f1"
	}
	return s.db.Create(p).Error
}

func (s *Service) ListProjects(companyID uint) ([]Project, error) {
	var projects []Project
	err := s.db.Where("company_id = ?", companyID).
		Preload("Partner").
		Preload("User").
		Preload("Tasks").
		Order("id desc").
		Find(&projects).Error
	return projects, err
}

func (s *Service) GetProject(id, companyID uint) (*Project, error) {
	var p Project
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("Partner").
		Preload("User").
		Preload("Tasks.User").
		First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Service) CreateTask(t *Task) error {
	if t.Name == "" {
		return errors.New("task name is required")
	}
	if t.ProjectID == 0 {
		return errors.New("project_id is required")
	}
	if t.Stage == "" {
		t.Stage = "todo"
	}
	if t.Priority == "" {
		t.Priority = "1"
	}
	return s.db.Create(t).Error
}

func (s *Service) ListTasks(companyID uint, projectID uint, stage string) ([]Task, error) {
	query := s.db.Where("company_id = ?", companyID)
	if projectID > 0 {
		query = query.Where("project_id = ?", projectID)
	}
	if stage != "" && stage != "all" {
		query = query.Where("stage = ?", stage)
	}

	var tasks []Task
	err := query.
		Preload("Project").
		Preload("Partner").
		Preload("User").
		Order("priority desc, id desc").
		Find(&tasks).Error
	return tasks, err
}

func (s *Service) GetTask(id, companyID uint) (*Task, error) {
	var t Task
	err := s.db.Where("id = ? AND company_id = ?", id, companyID).
		Preload("Project").
		Preload("Partner").
		Preload("User").
		Preload("Timesheets.User").
		First(&t).Error
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Service) UpdateTaskStage(id uint, stage string, companyID uint) (*Task, error) {
	var t Task
	if err := s.db.Where("id = ? AND company_id = ?", id, companyID).First(&t).Error; err != nil {
		return nil, err
	}

	t.Stage = stage
	if stage == "done" && t.Progress < 100 {
		t.Progress = 100
	}
	if err := s.db.Save(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Service) LogTimesheet(ts *TimesheetLine, companyID uint) error {
	if ts.UnitAmount <= 0 {
		return errors.New("timesheet hours must be greater than 0")
	}
	if ts.TaskID == 0 {
		return errors.New("task_id is required")
	}
	if ts.Name == "" {
		return errors.New("description of work is required")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		var task Task
		if err := tx.Where("id = ? AND company_id = ?", ts.TaskID, companyID).First(&task).Error; err != nil {
			return errors.New("task not found")
		}

		ts.ProjectID = task.ProjectID
		ts.CompanyID = companyID

		if err := tx.Create(ts).Error; err != nil {
			return err
		}

		// Recalculate EffectiveHours on task
		var sumHours float64
		tx.Model(&TimesheetLine{}).Where("task_id = ?", task.ID).Select("COALESCE(SUM(unit_amount), 0)").Scan(&sumHours)
		task.EffectiveHours = sumHours

		if task.PlannedHours > 0 {
			prog := (task.EffectiveHours / task.PlannedHours) * 100.0
			task.Progress = math.Min(100.0, math.Round(prog*10)/10)
		}

		return tx.Save(&task).Error
	})
}

func (s *Service) ListTimesheets(taskID uint, companyID uint) ([]TimesheetLine, error) {
	var lines []TimesheetLine
	query := s.db.Where("company_id = ?", companyID)
	if taskID > 0 {
		query = query.Where("task_id = ?", taskID)
	}
	err := query.Preload("User").Order("id desc").Find(&lines).Error
	return lines, err
}
