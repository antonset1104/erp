package leave

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	svc     *Service
	handler *Handler
}

func NewModule(db *gorm.DB) *Module {
	svc := NewService(db)
	handler := NewHandler(svc)
	return &Module{
		db:      db,
		svc:     svc,
		handler: handler,
	}
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) Name() string {
	return "leave"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&LeaveType{},
		&LeaveAllocation{},
		&LeaveRequest{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/leave")
	group.Use(authMiddleware)
	{
		// Types
		group.GET("/types", m.handler.ListLeaveTypes)
		group.POST("/types", m.handler.CreateLeaveType)

		// Allocations
		group.GET("/allocations", m.handler.ListAllocations)
		group.POST("/allocations", m.handler.CreateAllocation)
		group.PUT("/allocations/:id/approve", m.handler.ApproveAllocation)

		// Requests
		group.GET("/requests", m.handler.ListLeaveRequests)
		group.GET("/requests/:id", m.handler.GetLeaveRequest)
		group.POST("/requests", m.handler.CreateLeaveRequest)
		group.PUT("/requests/:id/approve", m.handler.ApproveLeaveRequest)
		group.PUT("/requests/:id/refuse", m.handler.RefuseLeaveRequest)

		// Summary
		group.GET("/summary", m.handler.GetLeaveSummary)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var typeCount int64
	db.Model(&LeaveType{}).Count(&typeCount)
	if typeCount == 0 {
		tAnnual := LeaveType{
			Name:               "Cuti Tahunan (Paid Annual Leave)",
			Code:               "ANNUAL",
			IsUnpaid:           false,
			RequiresAllocation: true,
			DefaultDays:        12,
			Color:              "#10b981",
			Description:        "Hak cuti tahunan berbayar karyawan 12 hari kerja per tahun kalender",
		}
		tAnnual.CompanyID = 1
		tAnnual.CreateUID = 1
		tAnnual.WriteUID = 1
		db.Create(&tAnnual)

		tSick := LeaveType{
			Name:               "Cuti Sakit (Sick Leave)",
			Code:               "SICK",
			IsUnpaid:           false,
			RequiresAllocation: false,
			DefaultDays:        14,
			Color:              "#f59e0b",
			Description:        "Izin sakit dengan surat keterangan dokter klinik / rumah sakit",
		}
		tSick.CompanyID = 1
		tSick.CreateUID = 1
		tSick.WriteUID = 1
		db.Create(&tSick)

		tUnpaid := LeaveType{
			Name:               "Cuti Tanpa Gaji (Unpaid Leave)",
			Code:               "UNPAID",
			IsUnpaid:           true,
			RequiresAllocation: false,
			DefaultDays:        0,
			Color:              "#ef4444",
			Description:        "Cuti di luar tanggungan perusahaan yang memotong prorata gaji pada payslip",
		}
		tUnpaid.CompanyID = 1
		tUnpaid.CreateUID = 1
		tUnpaid.WriteUID = 1
		db.Create(&tUnpaid)

		tMaternity := LeaveType{
			Name:               "Cuti Melahirkan & Bersalin",
			Code:               "MATERNITY",
			IsUnpaid:           false,
			RequiresAllocation: false,
			DefaultDays:        90,
			Color:              "#8b5cf6",
			Description:        "Hak cuti melahirkan 3 bulan (90 hari) bagi karyawan wanita",
		}
		tMaternity.CompanyID = 1
		tMaternity.CreateUID = 1
		tMaternity.WriteUID = 1
		db.Create(&tMaternity)

		tSpecial := LeaveType{
			Name:               "Izin Khusus / Keperluan Mendesak",
			Code:               "SPECIAL",
			IsUnpaid:           false,
			RequiresAllocation: false,
			DefaultDays:        3,
			Color:              "#06b6d4",
			Description:        "Izin pernikahan anak, duka cita keluarga, atau musibah",
		}
		tSpecial.CompanyID = 1
		tSpecial.CreateUID = 1
		tSpecial.WriteUID = 1
		db.Create(&tSpecial)

		// Seed initial allocation for Employee #1
		var allocCount int64
		db.Model(&LeaveAllocation{}).Count(&allocCount)
		if allocCount == 0 {
			currentYear := time.Now().Year()

			alloc1 := LeaveAllocation{
				Name:          "Alokasi Cuti Tahunan 2026 - Bambang Wijaya",
				EmployeeID:    1,
				LeaveTypeID:   tAnnual.ID,
				Year:          currentYear,
				AllocatedDays: 12,
				UsedDays:      2,
				RemainingDays: 10,
				State:         "approved",
				Notes:         "Alokasi jatah cuti reguler tahun berjalan disetujui HRD",
			}
			alloc1.CompanyID = 1
			alloc1.CreateUID = 1
			alloc1.WriteUID = 1
			db.Create(&alloc1)

			// Sample approved leave request
			approverID := uint(1)
			req1 := LeaveRequest{
				RequestNumber: "LV/2026/00001",
				EmployeeID:    1,
				LeaveTypeID:   tAnnual.ID,
				DateFrom:      "2026-10-14",
				DateTo:        "2026-10-15",
				NumberOfDays:  2,
				Description:   "Cuti tahunan keperluan acara wisuda keluarga di Yogyakarta",
				State:         "validate",
				ApproverID:    &approverID,
				ApprovalDate:  "2026-10-02 09:15:00",
			}
			req1.CompanyID = 1
			req1.CreateUID = 1
			req1.WriteUID = 1
			db.Create(&req1)
		}

		log.Println("[Leave] Seeded default Leave Types, Allocations & Approved Requests")
	}
	return nil
}
