package recruitment

import (
	"net/http"
	"strconv"

	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ---------------------------------------------------------------------------
// Stages
// ---------------------------------------------------------------------------

func (h *Handler) ListStages(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stages, err := h.svc.ListStages(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stages)
}

func (h *Handler) CreateStage(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var stage RecruitmentStage
	if err := c.ShouldBindJSON(&stage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	stage.CompanyID = companyID
	if err := h.svc.CreateStage(&stage); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, stage)
}

// ---------------------------------------------------------------------------
// Job Openings
// ---------------------------------------------------------------------------

func (h *Handler) ListJobOpenings(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	state := c.Query("state")
	openings, err := h.svc.ListJobOpenings(companyID, state)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, openings)
}

func (h *Handler) GetJobOpening(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	opening, err := h.svc.GetJobOpening(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lowongan tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, opening)
}

func (h *Handler) CreateJobOpening(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req CreateJobOpeningRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	opening, err := h.svc.CreateJobOpening(&req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, opening)
}

func (h *Handler) UpdateJobOpeningState(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	var body struct {
		State string `json:"state" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateJobOpeningState(uint(id), body.State, companyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Status lowongan berhasil diperbarui"})
}

// ---------------------------------------------------------------------------
// Applicants
// ---------------------------------------------------------------------------

func (h *Handler) ListApplicants(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var filter ApplicantFilter

	if openingStr := c.Query("job_opening_id"); openingStr != "" {
		if id, err := strconv.ParseUint(openingStr, 10, 64); err == nil {
			filter.JobOpeningID = uint(id)
		}
	}
	if deptStr := c.Query("department_id"); deptStr != "" {
		if id, err := strconv.ParseUint(deptStr, 10, 64); err == nil {
			filter.DepartmentID = uint(id)
		}
	}
	if stageStr := c.Query("stage_id"); stageStr != "" {
		if id, err := strconv.ParseUint(stageStr, 10, 64); err == nil {
			filter.StageID = uint(id)
		}
	}
	filter.State = c.Query("state")
	filter.Search = c.Query("search")
	if minScoreStr := c.Query("min_ai_score"); minScoreStr != "" {
		if ms, err := strconv.ParseFloat(minScoreStr, 64); err == nil {
			filter.MinAIScore = ms
		}
	}
	if limStr := c.Query("limit"); limStr != "" {
		if l, err := strconv.Atoi(limStr); err == nil {
			filter.Limit = l
		}
	}
	if offStr := c.Query("offset"); offStr != "" {
		if o, err := strconv.Atoi(offStr); err == nil {
			filter.Offset = o
		}
	}

	applicants, total, err := h.svc.ListApplicants(companyID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":   applicants,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

func (h *Handler) GetApplicant(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	applicant, err := h.svc.GetApplicant(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pelamar tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, applicant)
}

func (h *Handler) CreateApplicant(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var req CreateApplicantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	applicant, err := h.svc.CreateApplicant(&req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, applicant)
}

func (h *Handler) UpdateApplicantStage(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	var req UpdateStageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	applicant, err := h.svc.UpdateApplicantStage(uint(id), req.StageID, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, applicant)
}

func (h *Handler) DeleteApplicant(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	if err := h.svc.DeleteApplicant(uint(id), companyID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Pelamar berhasil dihapus"})
}

// ---------------------------------------------------------------------------
// Cloudflare Workers AI Screening & 1-Click Hire / Refuse
// ---------------------------------------------------------------------------

func (h *Handler) RunAIScreening(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	result, err := h.svc.RunAIScreening(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) ActionHire(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	var req HireApplicantRequest
	_ = c.ShouldBindJSON(&req) // optional body

	applicant, employee, err := h.svc.ActionHire(uint(id), req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "Pelamar berhasil resmi direkrut sebagai karyawan baru!",
		"applicant": applicant,
		"employee":  employee,
	})
}

func (h *Handler) ActionRefuse(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}
	var req RefuseApplicantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Alasan penolakan wajib disertakan"})
		return
	}
	applicant, err := h.svc.ActionRefuse(uint(id), req.Reason, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":   "Status pelamar diperbarui menjadi ditolak",
		"applicant": applicant,
	})
}

// ---------------------------------------------------------------------------
// KPI Stats
// ---------------------------------------------------------------------------

func (h *Handler) GetStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	stats, err := h.svc.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
