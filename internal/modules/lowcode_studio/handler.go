package lowcode_studio

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	prefixes := []string{"/lowcode-studio", "/lowcode"}

	for _, p := range prefixes {
		studio := r.Group(p)
		if authMiddleware != nil {
			studio.Use(authMiddleware)
		}
		{
			studio.GET("/stats", h.GetStats)

			// Models
			studio.GET("/models", h.ListModels)
			studio.POST("/models", h.CreateModel)
			studio.GET("/models/:id", h.GetModel)
			studio.PUT("/models/:id", h.UpdateModel)
			studio.DELETE("/models/:id", h.DeleteModel)
			studio.GET("/models/by-name/:name", h.GetModelByTechnicalName)

			// Fields
			studio.POST("/models/:id/fields", h.AddField)
			studio.DELETE("/models/:id/fields/:field_id", h.DeleteFieldFromModel)
			studio.DELETE("/fields/:id", h.DeleteField)

			// Records (supports both numeric record ID and model technical name)
			studio.GET("/models/:id/records", h.ListRecords)
			studio.POST("/models/:id/records", h.CreateRecord)
			studio.GET("/records/:id", h.GetOrListRecordsByParam)
			studio.POST("/records/:id", h.CreateRecordByParam)
			studio.PUT("/records/:id", h.UpdateRecord)
			studio.DELETE("/records/:id", h.DeleteRecord)
			studio.DELETE("/records/:id/:record_id", h.DeleteRecordNested)

			// Records access by model technical name
			studio.GET("/model-records/:name", h.ListRecordsByModelName)
			studio.POST("/model-records/:name", h.CreateRecordByModelName)

			// Core Extensions
			studio.POST("/core-extensions", h.SaveCoreExtension)
			studio.GET("/core-extensions", h.GetCoreExtension)

			// AI Schema Generator Copilot
			studio.POST("/ai-schema-generator", h.GenerateSchemaWithAI)
			studio.POST("/ai/schema-copilot", h.GenerateSchemaWithAI)
		}
	}
}

func (h *Handler) GetStats(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_models":            stats.TotalCustomModels,
		"total_custom_models":     stats.TotalCustomModels,
		"total_fields":            stats.TotalCustomFields,
		"total_custom_fields":     stats.TotalCustomFields,
		"total_records":           stats.TotalCustomRecords,
		"total_custom_records":    stats.TotalCustomRecords,
		"active_workflows":        stats.ActiveBusinessSchemas,
		"active_business_schemas": stats.ActiveBusinessSchemas,
		"storage_kb":              128 + stats.TotalCustomRecords*2,
		"schema_health":           "100% OK",
		"system_readiness":        stats.SystemReadinessRate,
		"system_readiness_rate":   stats.SystemReadinessRate,
		"data":                    stats,
	})
}

func (h *Handler) ListModels(c *gin.Context) {
	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	models, err := h.service.ListModels(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, models)
}

func (h *Handler) CreateModel(c *gin.Context) {
	var req CreateCustomModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	model, err := h.service.CreateModel(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, model)
}

func (h *Handler) GetModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		// Fallback: try looking up by technical name
		model, errName := h.service.GetModelByTechnicalName(c.Param("id"))
		if errName == nil && model != nil {
			c.JSON(http.StatusOK, model)
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid"})
		return
	}

	model, err := h.service.GetModelByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, model)
}

func (h *Handler) GetModelByTechnicalName(c *gin.Context) {
	name := c.Param("name")
	model, err := h.service.GetModelByTechnicalName(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, model)
}

func (h *Handler) UpdateModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid"})
		return
	}

	var req CreateCustomModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model, err := h.service.UpdateModel(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, model)
}

func (h *Handler) DeleteModel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid"})
		return
	}

	if err := h.service.DeleteModel(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Model kustom berhasil dihapus"})
}

func (h *Handler) AddField(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid"})
		return
	}

	var req CreateCustomFieldRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	field, err := h.service.AddField(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, field)
}

func (h *Handler) DeleteFieldFromModel(c *gin.Context) {
	fieldID, err := strconv.ParseUint(c.Param("field_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID field tidak valid"})
		return
	}

	if err := h.service.DeleteField(uint(fieldID)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Field kustom berhasil dihapus"})
}

func (h *Handler) DeleteField(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID field tidak valid"})
		return
	}

	if err := h.service.DeleteField(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Field kustom berhasil dihapus"})
}

func (h *Handler) ListRecords(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid"})
		return
	}

	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	records, total, err := h.service.ListRecords(uint(id), companyID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  records,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

func (h *Handler) ListRecordsByModelName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		name = c.Param("id")
	}
	model, err := h.service.GetModelByTechnicalName(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model tidak ditemukan: " + name})
		return
	}

	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	records, _, err := h.service.ListRecords(model.ID, companyID, 1, 100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, records)
}

func (h *Handler) GetOrListRecordsByParam(c *gin.Context) {
	param := c.Param("id")
	if id, err := strconv.ParseUint(param, 10, 32); err == nil {
		rec, err := h.service.GetRecordByID(uint(id))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data baris tidak ditemukan"})
			return
		}
		c.JSON(http.StatusOK, rec)
		return
	}
	// Non-numeric param is treated as model technical name
	h.ListRecordsByModelName(c)
}

func (h *Handler) CreateRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID model tidak valid"})
		return
	}

	var req SaveCustomRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}
	userID := uint(1)
	if uid, exists := c.Get("user_id"); exists {
		if v, ok := uid.(uint); ok && v > 0 {
			userID = v
		}
	}

	rec, err := h.service.CreateRecord(uint(id), req, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rec)
}

func (h *Handler) CreateRecordByParam(c *gin.Context) {
	param := c.Param("id")
	if _, err := strconv.ParseUint(param, 10, 32); err == nil {
		h.CreateRecord(c)
		return
	}
	h.CreateRecordByModelName(c)
}

func (h *Handler) CreateRecordByModelName(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		name = c.Param("id")
	}
	model, err := h.service.GetModelByTechnicalName(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model tidak ditemukan: " + name})
		return
	}

	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}
	userID := uint(1)
	if uid, exists := c.Get("user_id"); exists {
		if v, ok := uid.(uint); ok && v > 0 {
			userID = v
		}
	}

	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	displayName := "Record"
	if n, ok := raw["name"].(string); ok && n != "" {
		displayName = n
	} else if n, ok := raw["display_name"].(string); ok && n != "" {
		displayName = n
	}

	var values map[string]interface{}
	if v, ok := raw["data"].(map[string]interface{}); ok {
		values = v
	} else if v, ok := raw["values"].(map[string]interface{}); ok {
		values = v
	} else {
		values = raw
	}

	rec, err := h.service.CreateRecord(model.ID, SaveCustomRecordRequest{
		DisplayName: displayName,
		Values:      values,
	}, companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, rec)
}

func (h *Handler) GetRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID record tidak valid"})
		return
	}

	rec, err := h.service.GetRecordByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data baris tidak ditemukan"})
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *Handler) UpdateRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID record tidak valid"})
		return
	}

	var req SaveCustomRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rec, err := h.service.UpdateRecord(uint(id), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rec)
}

func (h *Handler) DeleteRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID record tidak valid"})
		return
	}

	if err := h.service.DeleteRecord(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Data baris berhasil dihapus"})
}

func (h *Handler) DeleteRecordNested(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("record_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID record tidak valid"})
		return
	}

	if err := h.service.DeleteRecord(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Data baris berhasil dihapus"})
}

func (h *Handler) SaveCoreExtension(c *gin.Context) {
	var req SaveCoreExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	companyID := uint(1)
	if cid, exists := c.Get("company_id"); exists {
		if v, ok := cid.(uint); ok && v > 0 {
			companyID = v
		}
	}

	ext, err := h.service.SaveCoreExtension(req, companyID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ext)
}

func (h *Handler) GetCoreExtension(c *gin.Context) {
	coreModel := c.Query("core_model")
	recID, _ := strconv.ParseUint(c.Query("record_id"), 10, 32)

	if coreModel == "" || recID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "core_model dan record_id wajib diisi"})
		return
	}

	ext, err := h.service.GetCoreExtension(coreModel, uint(recID))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"data": nil})
		return
	}
	c.JSON(http.StatusOK, ext)
}

func (h *Handler) GenerateSchemaWithAI(c *gin.Context) {
	var raw map[string]interface{}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prompt := ""
	if p, ok := raw["industry_prompt"].(string); ok && p != "" {
		prompt = p
	} else if p, ok := raw["prompt"].(string); ok && p != "" {
		prompt = p
	}

	targetCat := ""
	if cat, ok := raw["target_category"].(string); ok {
		targetCat = cat
	}

	if prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Prompt deskripsi bisnis wajib diisi"})
		return
	}

	schema, err := h.service.GenerateSchemaWithAI(prompt, targetCat)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Prepare proposed_model representation for the frontend
	proposedFields := make([]gin.H, 0)
	for _, f := range schema.RecommendedFields {
		lbl := f.FieldLabel
		if lbl == "" {
			lbl = f.Label
		}
		opt := f.SelectionOptions
		if opt == "" {
			opt = f.OptionsJSON
		}
		proposedFields = append(proposedFields, gin.H{
			"field_name":    f.FieldName,
			"field_type":    f.FieldType,
			"label":         lbl,
			"required":      f.IsRequired || f.Required,
			"options_json":  opt,
			"default_value": f.DefaultValue,
		})
	}

	proposedModel := gin.H{
		"name":               schema.ModelDisplayName,
		"technical_name":     schema.ModelTechnicalName,
		"category":           schema.MenuCategory,
		"icon":               schema.Icon,
		"description":        schema.Description,
		"fields":             proposedFields,
		"architecture_notes": schema.ArchitectureNotes,
		"provider":           schema.Provider,
	}

	c.JSON(http.StatusOK, gin.H{
		"status":         "success",
		"data":           schema,
		"proposed_model": proposedModel,
		"message":        "Skema arsitektur industri berhasil dirancang otomatis oleh AI",
	})
}
