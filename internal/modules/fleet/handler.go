package fleet

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

// ----------------- Models -----------------

func (h *Handler) ListModels(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	models, err := h.svc.ListModels(companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, models)
}

func (h *Handler) CreateModel(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var m FleetVehicleModel
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	m.CompanyID = companyID
	m.CreateUID = userID
	m.WriteUID = userID

	if err := h.svc.CreateModel(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, m)
}

// ----------------- Vehicles -----------------

func (h *Handler) ListVehicles(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	status := c.Query("status")
	vType := c.Query("vehicle_type")

	vehicles, err := h.svc.ListVehicles(companyID, status, vType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, vehicles)
}

func (h *Handler) GetVehicle(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid vehicle ID"})
		return
	}

	v, err := h.svc.GetVehicle(uint(id), companyID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Vehicle not found"})
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handler) CreateVehicle(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var v FleetVehicle
	if err := c.ShouldBindJSON(&v); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	v.CompanyID = companyID
	v.CreateUID = userID
	v.WriteUID = userID

	if err := h.svc.CreateVehicle(&v); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, v)
}

// ----------------- Odometer -----------------

func (h *Handler) ListOdometerLogs(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var vehicleID uint
	if vidStr := c.Query("vehicle_id"); vidStr != "" {
		if id, err := strconv.ParseUint(vidStr, 10, 32); err == nil {
			vehicleID = uint(id)
		}
	}

	logs, err := h.svc.ListOdometerLogs(vehicleID, companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) LogOdometer(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var olog FleetOdometerLog
	if err := c.ShouldBindJSON(&olog); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.LogOdometer(&olog, companyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, olog)
}

// ----------------- Service Logs -----------------

func (h *Handler) ListServiceLogs(c *gin.Context) {
	_, companyID, _ := middleware.GetUserContext(c)
	var vehicleID uint
	if vidStr := c.Query("vehicle_id"); vidStr != "" {
		if id, err := strconv.ParseUint(vidStr, 10, 32); err == nil {
			vehicleID = uint(id)
		}
	}
	sType := c.Query("service_type")
	stage := c.Query("stage")

	logs, err := h.svc.ListServiceLogs(vehicleID, companyID, sType, stage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}

func (h *Handler) CreateServiceLog(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	var srv FleetServiceLog
	if err := c.ShouldBindJSON(&srv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.CreateServiceLog(&srv, companyID, userID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, srv)
}

func (h *Handler) CompleteServiceLog(c *gin.Context) {
	userID, companyID, _ := middleware.GetUserContext(c)
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid service log ID"})
		return
	}

	srv, err := h.svc.CompleteServiceLog(uint(id), companyID, userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, srv)
}
