package recruitment

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/hr"

	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	repo       *Repository
	hrSvc      *hr.Service
	chatterSvc *chatter.Service
	aiClient   *ai.Client
}

func NewService(db *gorm.DB, repo *Repository, hrSvc *hr.Service, chatterSvc *chatter.Service, aiClient *ai.Client) *Service {
	return &Service{
		db:         db,
		repo:       repo,
		hrSvc:      hrSvc,
		chatterSvc: chatterSvc,
		aiClient:   aiClient,
	}
}

// ---------------------------------------------------------------------------
// Stages & Openings
// ---------------------------------------------------------------------------

func (s *Service) ListStages(companyID uint) ([]RecruitmentStage, error) {
	return s.repo.ListStages(companyID)
}

func (s *Service) CreateStage(stage *RecruitmentStage) error {
	if stage.Name == "" {
		return errors.New("nama tahapan rekrutmen wajib diisi")
	}
	return s.repo.CreateStage(stage)
}

func (s *Service) ListJobOpenings(companyID uint, state string) ([]JobOpening, error) {
	return s.repo.ListJobOpenings(companyID, state)
}

func (s *Service) GetJobOpening(id, companyID uint) (*JobOpening, error) {
	return s.repo.GetJobOpening(id, companyID)
}

func (s *Service) CreateJobOpening(req *CreateJobOpeningRequest, companyID, userID uint) (*JobOpening, error) {
	if req.Title == "" {
		return nil, errors.New("judul lowongan kerja wajib diisi")
	}
	if req.DepartmentID == 0 {
		return nil, errors.New("departemen wajib dipilih")
	}
	if req.ExpectedEmployees <= 0 {
		req.ExpectedEmployees = 1
	}

	opening := JobOpening{
		Title:             req.Title,
		DepartmentID:      req.DepartmentID,
		JobID:             req.JobID,
		ExpectedEmployees: req.ExpectedEmployees,
		CurrentHired:      0,
		Requirements:      req.Requirements,
		Description:       req.Description,
		SalaryMin:         req.SalaryMin,
		SalaryMax:         req.SalaryMax,
		State:             "open",
	}
	opening.CompanyID = companyID
	opening.CreateUID = userID
	opening.WriteUID = userID

	if err := s.repo.CreateJobOpening(&opening); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.job.opening",
			ResID:       opening.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Lowongan kerja baru '%s' dibuka dengan target %d karyawan.", opening.Title, opening.ExpectedEmployees),
			MessageType: "notification",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.repo.GetJobOpening(opening.ID, companyID)
}

func (s *Service) UpdateJobOpeningState(id uint, state string, companyID uint) error {
	opening, err := s.repo.GetJobOpening(id, companyID)
	if err != nil {
		return err
	}
	opening.State = state
	return s.repo.UpdateJobOpening(opening)
}

// ---------------------------------------------------------------------------
// Applicants
// ---------------------------------------------------------------------------

func (s *Service) ListApplicants(companyID uint, filter ApplicantFilter) ([]Applicant, int64, error) {
	return s.repo.ListApplicants(companyID, filter)
}

func (s *Service) GetApplicant(id, companyID uint) (*Applicant, error) {
	return s.repo.GetApplicant(id, companyID)
}

func (s *Service) CreateApplicant(req *CreateApplicantRequest, companyID, userID uint) (*Applicant, error) {
	if req.Name == "" || req.Email == "" {
		return nil, errors.New("nama dan email pelamar wajib diisi")
	}
	if req.JobOpeningID == 0 {
		return nil, errors.New("lowongan pekerjaan wajib dipilih")
	}

	opening, err := s.repo.GetJobOpening(req.JobOpeningID, companyID)
	if err != nil {
		return nil, errors.New("lowongan pekerjaan tidak ditemukan")
	}

	// Resolve initial stage (lowest sequence or not folded)
	stages, err := s.repo.ListStages(companyID)
	if err != nil || len(stages) == 0 {
		return nil, errors.New("tahapan rekrutmen belum dikonfigurasi")
	}
	initialStageID := stages[0].ID

	applicant := Applicant{
		Name:           req.Name,
		Email:          req.Email,
		Phone:          req.Phone,
		JobOpeningID:   req.JobOpeningID,
		DepartmentID:   opening.DepartmentID,
		StageID:        initialStageID,
		SalaryExpected: req.SalaryExpected,
		ResumeText:     req.ResumeText,
		State:          "new",
	}
	applicant.CompanyID = companyID
	applicant.CreateUID = userID
	applicant.WriteUID = userID

	if err := s.repo.CreateApplicant(&applicant); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.applicant",
			ResID:       applicant.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Pelamar baru '%s' mendaftar untuk posisi '%s'.", applicant.Name, opening.Title),
			MessageType: "notification",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.repo.GetApplicant(applicant.ID, companyID)
}

func (s *Service) UpdateApplicantStage(id, stageID, companyID, userID uint) (*Applicant, error) {
	applicant, err := s.repo.GetApplicant(id, companyID)
	if err != nil {
		return nil, err
	}
	newStage, err := s.repo.GetStage(stageID, companyID)
	if err != nil {
		return nil, errors.New("tahapan baru tidak ditemukan")
	}

	oldStageName := ""
	if applicant.Stage != nil {
		oldStageName = applicant.Stage.Name
	}

	applicant.StageID = stageID
	applicant.Stage = nil
	applicant.JobOpening = nil
	applicant.Department = nil
	applicant.HiredEmployee = nil

	if applicant.State == "new" {
		applicant.State = "in_progress"
	}
	if newStage.IsRefused {
		applicant.State = "refused"
	}
	applicant.WriteUID = userID

	if err := s.repo.UpdateApplicant(applicant); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.applicant",
			ResID:       applicant.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Tahapan pelamar diubah dari '%s' ke '%s'.", oldStageName, newStage.Name),
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.repo.GetApplicant(id, companyID)
}

func (s *Service) DeleteApplicant(id, companyID uint) error {
	applicant, err := s.repo.GetApplicant(id, companyID)
	if err != nil {
		return err
	}
	if applicant.State == "hired" {
		return errors.New("tidak dapat menghapus pelamar yang sudah resmi direkrut sebagai karyawan")
	}
	return s.repo.DeleteApplicant(id, companyID)
}

// ---------------------------------------------------------------------------
// Cloudflare Workers AI Talent Screening & Resume Evaluation
// ---------------------------------------------------------------------------

func (s *Service) RunAIScreening(applicantID, companyID, userID uint) (*AIScreeningResult, error) {
	applicant, err := s.repo.GetApplicant(applicantID, companyID)
	if err != nil {
		return nil, err
	}
	opening := applicant.JobOpening
	if opening == nil {
		opening, _ = s.repo.GetJobOpening(applicant.JobOpeningID, companyID)
	}
	if opening == nil {
		return nil, errors.New("lowongan pekerjaan tidak ditemukan")
	}

	jobTitle := opening.Title
	jobReqs := opening.Requirements
	if jobReqs == "" {
		jobReqs = opening.Description
	}

	resume := applicant.ResumeText
	if strings.TrimSpace(resume) == "" {
		return nil, errors.New("resume atau CV pelamar masih kosong, harap lengkapi resume terlebih dahulu")
	}

	var result AIScreeningResult
	result.ApplicantID = applicant.ID
	result.ApplicantName = applicant.Name
	result.JobTitle = jobTitle

	// 1. Attempt Cloudflare Workers AI Inference
	var aiSuccess bool
	if s.aiClient != nil && s.aiClient.IsCloudflareConfigured() {
		systemPrompt := `You are an expert HR Talent Acquisition Specialist and Technical Recruiter.
Analyze the candidate's resume/CV against the job requirements.
Output STRICT VALID JSON only with the following exact keys:
{
  "ai_score": (number between 0 and 100 representing percentage match),
  "ai_recommendation": ("Strong Hire" | "Hire" | "Consider / Hold" | "Reject"),
  "ai_summary": (concise 2-sentence executive assessment in Indonesian),
  "ai_fit_analysis": (detailed 2-3 sentence technical & experience fit in Indonesian),
  "key_strengths": [(array of 3-4 bullet strings highlighting candidate strengths in Indonesian)],
  "key_weaknesses": [(array of 1-3 bullet strings highlighting skill gaps or risks in Indonesian)],
  "suggested_questions": [(array of 3-4 specific technical/behavioral interview questions tailored to this candidate in Indonesian)]
}`

		userPrompt := fmt.Sprintf(`Job Title: %s
Requirements & Responsibilities:
%s

Candidate Resume:
%s`, jobTitle, jobReqs, resume)

		rawResp, err := s.aiClient.CallRaw(userPrompt, systemPrompt)
		if err == nil && len(rawResp) > 0 {
			// Extract and parse JSON
			jsonStr := extractJSON(rawResp)
			type LLMResp struct {
				AIScore            float64  `json:"ai_score"`
				AIRecommendation   string   `json:"ai_recommendation"`
				AISummary          string   `json:"ai_summary"`
				AIFitAnalysis      string   `json:"ai_fit_analysis"`
				KeyStrengths       []string `json:"key_strengths"`
				KeyWeaknesses      []string `json:"key_weaknesses"`
				SuggestedQuestions []string `json:"suggested_questions"`
			}
			var parsed LLMResp
			if jsonErr := json.Unmarshal([]byte(jsonStr), &parsed); jsonErr == nil && parsed.AIScore > 0 {
				result.AIScore = parsed.AIScore
				result.AIRecommendation = parsed.AIRecommendation
				result.AISummary = parsed.AISummary
				result.AIFitAnalysis = parsed.AIFitAnalysis
				result.KeyStrengths = parsed.KeyStrengths
				result.KeyWeaknesses = parsed.KeyWeaknesses
				result.SuggestedQuestions = parsed.SuggestedQuestions
				result.EngineSource = "cloudflare_workers_ai"
				aiSuccess = true
			}
		}
	}

	// 2. Fallback to Smart Local Heuristic Screening Engine
	if !aiSuccess {
		result = s.runLocalHeuristicScreening(applicant, opening)
	}

	// 3. Persist AI metrics to Applicant Record
	applicant.AIScore = result.AIScore
	applicant.AIRecommendation = result.AIRecommendation
	applicant.AISummary = result.AISummary
	applicant.AIFitAnalysis = result.AIFitAnalysis
	applicant.KeyStrengths = strings.Join(result.KeyStrengths, "\n")
	applicant.KeyWeaknesses = strings.Join(result.KeyWeaknesses, "\n")
	applicant.SuggestedQuestions = strings.Join(result.SuggestedQuestions, "\n")
	applicant.AIScreenedAt = time.Now().Format("2006-01-02 15:04:05")
	applicant.AIEngineSource = result.EngineSource
	applicant.WriteUID = userID
	applicant.Stage = nil
	applicant.JobOpening = nil
	applicant.Department = nil
	applicant.HiredEmployee = nil

	if err := s.repo.UpdateApplicant(applicant); err != nil {
		return nil, err
	}

	// 4. Log to Chatter
	if s.chatterSvc != nil {
		chatterBody := fmt.Sprintf("🤖 **AI Talent Screening (%s)**:\n- **Skor Kecocokan**: %.1f%%\n- **Rekomendasi**: %s\n- **Ringkasan**: %s\n- **Kekuatan Utama**: %s",
			result.EngineSource, result.AIScore, result.AIRecommendation, result.AISummary, strings.Join(result.KeyStrengths, ", "))
		msg := chatter.MailMessage{
			ModelName:   "hr.applicant",
			ResID:       applicant.ID,
			AuthorID:    userID,
			Body:        chatterBody,
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return &result, nil
}

func (s *Service) runLocalHeuristicScreening(applicant *Applicant, opening *JobOpening) AIScreeningResult {
	resumeLower := strings.ToLower(applicant.ResumeText)
	reqLower := strings.ToLower(opening.Requirements + " " + opening.Title)

	// Keywords check
	techKeywords := []string{
		"golang", "go", "gin", "gorm", "postgresql", "postgres", "sql", "sqlite",
		"cloudflare", "docker", "microservices", "modular", "rest api", "git",
		"redis", "linux", "aws", "kubernetes", "erp", "accounting", "finance",
		"payroll", "crm", "supply chain", "react", "javascript", "typescript",
	}

	matchedCount := 0
	var matchedTechs []string
	for _, kw := range techKeywords {
		if strings.Contains(reqLower, kw) && strings.Contains(resumeLower, kw) {
			matchedCount++
			matchedTechs = append(matchedTechs, strings.Title(kw))
		}
	}

	// Experience years detection
	hasExperience := strings.Contains(resumeLower, "tahun") || strings.Contains(resumeLower, "years") || strings.Contains(resumeLower, "pengalaman")

	// Base score calculation
	score := 50.0
	if len(matchedTechs) > 0 {
		score += float64(len(matchedTechs)) * 8.5
	}
	if hasExperience {
		score += 10.0
	}
	if len(applicant.ResumeText) > 250 {
		score += 5.0
	}
	if score > 98.0 {
		score = 98.0
	}
	score = math.Round(score*10) / 10

	var recommendation string
	if score >= 85 {
		recommendation = "Strong Hire"
	} else if score >= 70 {
		recommendation = "Hire"
	} else if score >= 55 {
		recommendation = "Consider / Hold"
	} else {
		recommendation = "Reject"
	}

	strengths := []string{}
	if len(matchedTechs) > 0 {
		strengths = append(strengths, fmt.Sprintf("Menguasai teknologi kunci yang disyaratkan: %s", strings.Join(matchedTechs, ", ")))
	} else {
		strengths = append(strengths, "Memiliki latar belakang dan minat pada bidang terkait")
	}
	if hasExperience {
		strengths = append(strengths, "Memiliki rekam jejak pengalaman kerja profesional terbukti")
	}
	strengths = append(strengths, "Profil komunikasi dan format resume tersusun terstruktur")

	weaknesses := []string{}
	if len(matchedTechs) < 3 {
		weaknesses = append(weaknesses, "Perlu pendalaman lebih lanjut pada beberapa stack teknologi pendukung")
	}
	if applicant.SalaryExpected > opening.SalaryMax && opening.SalaryMax > 0 {
		weaknesses = append(weaknesses, fmt.Sprintf("Ekspektasi gaji (Rp %.0f) melebihi batas anggaran maksimal (Rp %.0f)", applicant.SalaryExpected, opening.SalaryMax))
	} else {
		weaknesses = append(weaknesses, "Belum menyertakan portofolio kode publik / sertifikasi resmi terkait")
	}

	questions := []string{
		fmt.Sprintf("Jelaskan proyek paling menantang yang pernah Anda kerjakan menggunakan %s dan bagaimana Anda mengatasi hambatan arsitekturnya?", opening.Title),
		"Bagaimana pendekatan Anda dalam merancang sistem dengan reliabilitas tinggi dan penanganan konkurensi data?",
		"Ceritakan pengalaman Anda dalam berkolaborasi lintas tim (Product, QA, dan Business) dalam lingkungan kerja yang dinamis.",
	}

	summary := fmt.Sprintf("Pelamar '%s' menunjukkan kecocokan kualifikasi yang menjanjikan untuk posisi '%s' dengan skor relevansi %.1f%%. Sangat direkomendasikan untuk tahapan wawancara teknis mendalam.", applicant.Name, opening.Title, score)
	fitAnalysis := fmt.Sprintf("Pengalaman dan keahlian yang tercantum dalam CV selaras dengan kebutuhan kompetensi posisi ini. Rekomendasi: %s.", recommendation)

	return AIScreeningResult{
		ApplicantID:        applicant.ID,
		ApplicantName:      applicant.Name,
		JobTitle:           opening.Title,
		AIScore:            score,
		AIRecommendation:   recommendation,
		AISummary:          summary,
		AIFitAnalysis:      fitAnalysis,
		KeyStrengths:       strengths,
		KeyWeaknesses:      weaknesses,
		SuggestedQuestions: questions,
		EngineSource:       "local_heuristic_engine",
	}
}

// ---------------------------------------------------------------------------
// 1-Click Action Hire (Convert to HR Employee)
// ---------------------------------------------------------------------------

func (s *Service) ActionHire(applicantID uint, req HireApplicantRequest, companyID, userID uint) (*Applicant, *hr.Employee, error) {
	applicant, err := s.repo.GetApplicant(applicantID, companyID)
	if err != nil {
		return nil, nil, err
	}
	if applicant.State == "hired" {
		return nil, nil, errors.New("pelamar ini sudah resmi direkrut sebagai karyawan sebelumnya")
	}

	opening := applicant.JobOpening
	if opening == nil {
		opening, _ = s.repo.GetJobOpening(applicant.JobOpeningID, companyID)
	}
	if opening == nil {
		return nil, nil, errors.New("lowongan pekerjaan tidak ditemukan")
	}

	salary := req.BasicSalary
	if salary <= 0 {
		salary = applicant.SalaryExpected
	}
	if salary <= 0 && opening.SalaryMin > 0 {
		salary = opening.SalaryMin
	}
	if salary <= 0 {
		salary = 5000000 // default UMR / standard
	}

	// 1. Resolve or Create JobPosition in HR module
	var jobID uint
	if opening.JobID != nil && *opening.JobID > 0 {
		jobID = *opening.JobID
	} else {
		// Find or create in hr_job_positions
		var existingJob hr.JobPosition
		if err := s.db.Where("name = ? AND company_id = ?", opening.Title, companyID).First(&existingJob).Error; err == nil {
			jobID = existingJob.ID
		} else {
			newJob := hr.JobPosition{
				Name:         opening.Title,
				DepartmentID: opening.DepartmentID,
			}
			newJob.CompanyID = companyID
			newJob.CreateUID = userID
			newJob.WriteUID = userID
			if err := s.hrSvc.CreateJob(&newJob); err == nil {
				jobID = newJob.ID
				opening.JobID = &jobID
				_ = s.repo.UpdateJobOpening(opening)
			}
		}
	}

	if jobID == 0 {
		jobID = 1 // default fallback
	}

	// 2. Create hr.Employee record
	employee := hr.Employee{
		Name:         applicant.Name,
		WorkEmail:    applicant.Email,
		WorkPhone:    applicant.Phone,
		DepartmentID: opening.DepartmentID,
		JobID:        jobID,
		BasicSalary:  salary,
		BankName:     req.BankName,
		BankAccount:  req.BankAccount,
		Status:       "active",
	}
	employee.CompanyID = companyID
	employee.CreateUID = userID
	employee.WriteUID = userID

	if err := s.hrSvc.CreateEmployee(&employee); err != nil {
		return nil, nil, fmt.Errorf("gagal membuat data karyawan di modul HR: %w", err)
	}

	// 3. Advance Applicant stage to "Hired"
	stages, _ := s.repo.ListStages(companyID)
	var hiredStageID uint
	for _, st := range stages {
		if st.IsHired || strings.ToLower(st.Name) == "hired" || strings.Contains(strings.ToLower(st.Name), "diterima") {
			hiredStageID = st.ID
			break
		}
	}
	if hiredStageID > 0 {
		applicant.StageID = hiredStageID
	}

	applicant.State = "hired"
	applicant.HiredEmployeeID = &employee.ID
	applicant.WriteUID = userID
	applicant.Stage = nil
	applicant.JobOpening = nil
	applicant.Department = nil
	applicant.HiredEmployee = nil
	if err := s.repo.UpdateApplicant(applicant); err != nil {
		return nil, nil, err
	}

	// 4. Increment JobOpening.CurrentHired
	opening.CurrentHired++
	if opening.CurrentHired >= opening.ExpectedEmployees && opening.ExpectedEmployees > 0 {
		opening.State = "closed"
	}
	_ = s.repo.UpdateJobOpening(opening)

	// 5. Post Chatter Messages
	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.applicant",
			ResID:       applicant.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("🎉 **RESMI DIREKRUT**: Pelamar '%s' telah resmi diterima menjadi karyawan baru (Employee ID #%d) di Departemen %s dengan gaji pokok Rp %.2f.", applicant.Name, employee.ID, opening.Department.Name, salary),
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)

		empMsg := chatter.MailMessage{
			ModelName:   "hr.employee",
			ResID:       employee.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Profil karyawan dibuat dari hasil proses rekrutmen pelamar '%s' (Applicant ID #%d) pada lowongan '%s'.", applicant.Name, applicant.ID, opening.Title),
			MessageType: "notification",
		}
		empMsg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&empMsg)
	}

	updatedApplicant, err := s.repo.GetApplicant(applicantID, companyID)
	if err != nil {
		return nil, nil, err
	}
	return updatedApplicant, &employee, nil
}

func (s *Service) ActionRefuse(applicantID uint, reason string, companyID, userID uint) (*Applicant, error) {
	applicant, err := s.repo.GetApplicant(applicantID, companyID)
	if err != nil {
		return nil, err
	}
	if applicant.State == "hired" {
		return nil, errors.New("tidak dapat menolak pelamar yang sudah resmi direkrut sebagai karyawan")
	}

	stages, _ := s.repo.ListStages(companyID)
	var refusedStageID uint
	for _, st := range stages {
		if st.IsRefused || strings.ToLower(st.Name) == "refused" || strings.Contains(strings.ToLower(st.Name), "ditolak") {
			refusedStageID = st.ID
			break
		}
	}
	if refusedStageID > 0 {
		applicant.StageID = refusedStageID
	}

	applicant.State = "refused"
	applicant.RefusalReason = reason
	applicant.WriteUID = userID
	applicant.Stage = nil
	applicant.JobOpening = nil
	applicant.Department = nil
	applicant.HiredEmployee = nil

	if err := s.repo.UpdateApplicant(applicant); err != nil {
		return nil, err
	}

	if s.chatterSvc != nil {
		msg := chatter.MailMessage{
			ModelName:   "hr.applicant",
			ResID:       applicant.ID,
			AuthorID:    userID,
			Body:        fmt.Sprintf("Pelamar '%s' ditolak. Alasan: %s", applicant.Name, reason),
			MessageType: "comment",
		}
		msg.CompanyID = companyID
		_ = s.chatterSvc.PostMessage(&msg)
	}

	return s.repo.GetApplicant(applicantID, companyID)
}

func (s *Service) GetStats(companyID uint) (*RecruitmentStats, error) {
	return s.repo.GetRecruitmentStats(companyID)
}

func extractJSON(raw string) string {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start != -1 && end != -1 && end > start {
		return raw[start : end+1]
	}
	return raw
}
