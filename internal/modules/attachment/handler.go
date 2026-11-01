package attachment

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"erp-golang/internal/core/middleware"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Upload(c *gin.Context) {
	userID, companyID, userName := middleware.GetUserContext(c)

	var req UploadAttachmentRequest
	var fileBytes []byte
	var mimeType string

	contentType := c.GetHeader("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		// Multipart Form Upload
		name := c.PostForm("name")
		resModel := c.PostForm("res_model")
		resIDStr := c.PostForm("res_id")
		resName := c.PostForm("res_name")
		desc := c.PostForm("description")
		autoOCRStr := c.PostForm("auto_ocr")
		textHint := c.PostForm("text_hint")

		var resID uint
		if resIDStr != "" {
			if parsed, err := strconv.ParseUint(resIDStr, 10, 32); err == nil {
				resID = uint(parsed)
			}
		}

		autoOCR := autoOCRStr == "true" || autoOCRStr == "1"

		fileHeader, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file field is required in form-data"})
			return
		}

		if name == "" {
			name = fileHeader.Filename
		}
		mimeType = fileHeader.Header.Get("Content-Type")

		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
			return
		}
		defer file.Close()

		fileBytes, err = io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
			return
		}

		req = UploadAttachmentRequest{
			Name:        name,
			ResModel:    resModel,
			ResID:       resID,
			ResName:     resName,
			Description: desc,
			AutoOCR:     autoOCR,
			TextHint:    textHint,
		}
	} else {
		// JSON base64 upload
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.FileBase64 == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "file_base64 is required"})
			return
		}

		parts := strings.Split(req.FileBase64, ",")
		rawB64 := req.FileBase64
		if len(parts) > 1 {
			// Extract mime type if data URI e.g. "data:image/png;base64,..."
			header := parts[0]
			if strings.HasPrefix(header, "data:") && strings.Contains(header, ";") {
				mimeType = strings.TrimPrefix(strings.Split(header, ";")[0], "data:")
			}
			rawB64 = parts[1]
		}

		decoded, err := base64.StdEncoding.DecodeString(rawB64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid base64 content"})
			return
		}
		fileBytes = decoded
	}

	att, err := h.service.UploadAttachment(companyID, userID, userName, &req, fileBytes, mimeType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, att)
}

func (h *Handler) List(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	resModel := c.Query("res_model")
	ocrStatus := c.Query("ocr_status")
	search := c.Query("search")

	var resID uint
	if idStr := c.Query("res_id"); idStr != "" {
		if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			resID = uint(id)
		}
	}

	limit := 50
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	offset := 0
	if oStr := c.Query("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	filter := AttachmentFilter{
		ResModel:  resModel,
		ResID:     resID,
		OCRStatus: ocrStatus,
		Search:    search,
		Limit:     limit,
		Offset:    offset,
	}

	attachments, total, err := h.service.ListAttachments(companyID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  attachments,
		"total": total,
	})
}

func (h *Handler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachment id"})
		return
	}

	att, _, err := h.service.GetAttachment(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, att)
}

func (h *Handler) Download(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachment id"})
		return
	}

	att, data, err := h.service.GetAttachment(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	contentType := att.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, att.Name))
	c.Header("Content-Length", fmt.Sprintf("%d", len(data)))
	c.Data(http.StatusOK, contentType, data)
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachment id"})
		return
	}

	if err := h.service.DeleteAttachment(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "attachment deleted successfully"})
}

type RunOCRRequest struct {
	TextHint string `json:"text_hint"`
}

func (h *Handler) RunOCR(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachment id"})
		return
	}

	var req RunOCRRequest
	_ = c.ShouldBindJSON(&req)

	result, err := h.service.RunOCR(uint(id), req.TextHint)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

type CreateExpenseFromReceiptRequest struct {
	EmployeeID uint `json:"employee_id" binding:"required"`
}

func (h *Handler) CreateExpenseFromReceipt(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachment id"})
		return
	}

	var req CreateExpenseFromReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exp, err := h.service.CreateExpenseFromReceipt(uint(id), req.EmployeeID, companyID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, exp)
}

func (h *Handler) GetStats(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)

	stats, err := h.service.GetStats(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
