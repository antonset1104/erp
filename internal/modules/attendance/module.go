package attendance

import (
	"log"
	"time"

	"erp-golang/internal/core/ai"
	"erp-golang/internal/modules/chatter"
	"erp-golang/internal/modules/hr"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	db      *gorm.DB
	svc     *Service
	handler *Handler
}

func NewModule(db *gorm.DB, chatterSvc *chatter.Service, aiClient *ai.Client) *Module {
	svc := NewService(db, chatterSvc, aiClient)
	handler := NewHandler(svc)
	return &Module{
		db:      db,
		svc:     svc,
		handler: handler,
	}
}

func (m *Module) Name() string {
	return "attendance"
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Shift{},
		&Attendance{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/attendance")
	group.Use(authMiddleware)
	{
		// Clock In & Clock Out
		group.POST("/check-in", m.handler.CheckIn)
		group.POST("/check-out", m.handler.CheckOut)
		group.GET("/active/:employee_id", m.handler.GetActiveAttendance)

		// Listing & Queries
		group.GET("", m.handler.ListAttendances)
		group.GET("/:id", m.handler.GetAttendance)
		group.DELETE("/:id", m.handler.DeleteAttendance)
		group.GET("/stats", m.handler.GetStats)

		// Shifts
		group.GET("/shifts", m.handler.ListShifts)
		group.POST("/shifts", m.handler.CreateShift)

		// Cloudflare Workers AI & Payroll Integration
		group.POST("/ai-anomaly-check", m.handler.RunAIAnomalyCheck)
		group.POST("/sync-payroll", m.handler.SyncPayroll)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&Shift{}).Count(&count)
	if count == 0 {
		shifts := []Shift{
			{
				Name:               "Shift Reguler Kantor (08:30 - 17:30)",
				Code:               "REGULAR",
				StartTime:          "08:30",
				EndTime:            "17:30",
				StandardHours:      8.0,
				GracePeriodMinutes: 15,
				IsFlexible:         false,
				Color:              "#3b82f6",
				Description:        "Jadwal kerja standar kantor pusat Jakarta dengan toleransi terlambat 15 menit",
			},
			{
				Name:               "Shift Malam / Pabrik (20:00 - 05:00)",
				Code:               "NIGHT",
				StartTime:          "20:00",
				EndTime:            "05:00",
				StandardHours:      8.0,
				GracePeriodMinutes: 10,
				IsFlexible:         false,
				Color:              "#8b5cf6",
				Description:        "Jadwal operasional perakitan lini produksi pabrikasi malam",
			},
			{
				Name:               "Shift Fleksibel / WFH Remote",
				Code:               "WFH",
				StartTime:          "09:00",
				EndTime:            "18:00",
				StandardHours:      8.0,
				GracePeriodMinutes: 60,
				IsFlexible:         true,
				Color:              "#10b981",
				Description:        "Jadwal kerja jarak jauh / remote tanpa batasan radius geofence kantor",
			},
		}

		for i := range shifts {
			shifts[i].CompanyID = 1
			shifts[i].CreateUID = 1
			shifts[i].WriteUID = 1
			if err := db.Create(&shifts[i]).Error; err != nil {
				log.Printf("[Attendance] Error seeding shift %s: %v", shifts[i].Name, err)
			}
		}
		log.Println("[Attendance] Seeded 3 default work shifts (REGULAR, NIGHT, WFH)")
	}

	// Seed sample historical attendances if empty
	var attCount int64
	db.Model(&Attendance{}).Count(&attCount)
	if attCount == 0 {
		var emps []hr.Employee
		db.Limit(3).Find(&emps)

		var regShift Shift
		db.Where("code = ?", "REGULAR").First(&regShift)

		shiftID := regShift.ID
		now := time.Now()

		for idx, emp := range emps {
			// Yesterdays completed attendance with overtime
			yesterday := now.AddDate(0, 0, -1)
			checkInTime := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 8, 25, 0, 0, time.Local)
			checkOutTime := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 18, 55, 0, 0, time.Local)
			worked := 9.5
			overtime := 1.5

			attHist := Attendance{
				EmployeeID:    emp.ID,
				ShiftID:       &shiftID,
				Date:          yesterday.Format("2006-01-02"),
				CheckIn:       checkInTime,
				CheckOut:      &checkOutTime,
				WorkedHours:   worked,
				OvertimeHours: overtime,
				LateMinutes:   0,
				Status:        "overtime",
				InLatitude:    -6.200100,
				InLongitude:   106.816650,
				InAddress:     "Menara Sudirman Lantai 18, Jakarta Pusat",
				OutLatitude:   -6.200120,
				OutLongitude:  106.816640,
				OutAddress:    "Menara Sudirman Lantai 18, Jakarta Pusat",
				Device:        "Mobile Chrome / Android 14",
				IPAddress:     "182.253.110.15",
				Country:       "ID",
				CFRay:         "8c8942b08a123f12",
				Note:          "Penyelesaian sprint delivery order akhir kuartal",
			}
			attHist.CompanyID = 1
			attHist.CreateUID = 1
			attHist.WriteUID = 1
			db.Create(&attHist)

			// Today's attendance: Employee 0 checked in on-time, Employee 1 checked in late (40m late)
			if idx == 0 {
				todayIn := time.Date(now.Year(), now.Month(), now.Day(), 8, 28, 0, 0, time.Local)
				attToday := Attendance{
					EmployeeID:    emp.ID,
					ShiftID:       &shiftID,
					Date:          now.Format("2006-01-02"),
					CheckIn:       todayIn,
					LateMinutes:   0,
					Status:        "checked_in",
					InLatitude:    -6.200050,
					InLongitude:   106.816680,
					InAddress:     "Lobi Utama Gedung Kantor Pusat",
					Device:        "PWA Edge Client / iOS 17.5",
					IPAddress:     "114.124.200.80",
					Country:       "ID",
					CFRay:         "8c8965f90a9821a0",
					Note:          "Check-in pagi hari tepat waktu",
				}
				attToday.CompanyID = 1
				attToday.CreateUID = 1
				attToday.WriteUID = 1
				db.Create(&attToday)
			} else if idx == 1 {
				todayLateIn := time.Date(now.Year(), now.Month(), now.Day(), 9, 25, 0, 0, time.Local)
				attTodayLate := Attendance{
					EmployeeID:    emp.ID,
					ShiftID:       &shiftID,
					Date:          now.Format("2006-01-02"),
					CheckIn:       todayLateIn,
					LateMinutes:   40,
					Status:        "late",
					InLatitude:    -6.200150,
					InLongitude:   106.816720,
					InAddress:     "Area Parkir Timur Gedung Kantor",
					Device:        "Mobile Safari / iOS",
					IPAddress:     "180.252.88.19",
					Country:       "ID",
					CFRay:         "8c8971f11c7710bc",
					Note:          "Terlambat karena kendala kemacetan jalan tol",
				}
				attTodayLate.CompanyID = 1
				attTodayLate.CreateUID = 1
				attTodayLate.WriteUID = 1
				db.Create(&attTodayLate)
			}
		}
		log.Println("[Attendance] Seeded initial demo employee attendance records")
	}

	return nil
}
