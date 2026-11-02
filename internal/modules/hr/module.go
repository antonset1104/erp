package hr

import (
	"log"

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

func (m *Module) Name() string {
	return "hr"
}

func (m *Module) Service() *Service {
	return m.svc
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Department{},
		&JobPosition{},
		&Employee{},
		&Payslip{},
	)
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	group := api.Group("/hr")
	group.Use(authMiddleware)
	{
		group.GET("/departments", m.handler.ListDepartments)
		group.GET("/jobs", m.handler.ListJobs)
		group.GET("/employees", m.handler.ListEmployees)
		group.POST("/employees", m.handler.CreateEmployee)
		group.GET("/payslips", m.handler.ListPayslips)
		group.POST("/payslips/generate", m.handler.GeneratePayslip)
		group.POST("/payslips/:id/confirm", m.handler.ConfirmPayslip)
		group.POST("/payslips/:id/pay", m.handler.PayPayslip)
	}
}

func (m *Module) Seed(db *gorm.DB) error {
	var count int64
	db.Model(&Department{}).Count(&count)
	if count == 0 {
		depts := []Department{
			{Name: "Engineering & IT", Description: "Software development and infrastructure"},
			{Name: "Sales & Marketing", Description: "Commercial revenue and client relations"},
			{Name: "Finance & Accounting", Description: "Corporate finance, payroll and reporting"},
			{Name: "Warehouse & Operations", Description: "Inventory logistics and supply chain"},
		}
		for i := range depts {
			depts[i].CompanyID = 1
			db.Create(&depts[i])
		}

		jobs := []JobPosition{
			{Name: "Senior Golang Developer", DepartmentID: 1},
			{Name: "ERP Solutions Architect", DepartmentID: 1},
			{Name: "Account Executive", DepartmentID: 2},
			{Name: "Finance Officer", DepartmentID: 3},
			{Name: "Logistics Specialist", DepartmentID: 4},
		}
		for i := range jobs {
			jobs[i].CompanyID = 1
			db.Create(&jobs[i])
		}

		employees := []Employee{
			{
				Name:         "Rian Pratama",
				WorkEmail:    "rian.pratama@goerplang.co.id",
				WorkPhone:    "+62 811 1122 3344",
				DepartmentID: 1,
				JobID:        1,
				BasicSalary:  18500000,
				BankName:     "BCA",
				BankAccount:  "1234567890",
				Status:       "active",
			},
			{
				Name:         "Siti Rahmawati",
				WorkEmail:    "siti.rahma@goerplang.co.id",
				WorkPhone:    "+62 811 5566 7788",
				DepartmentID: 3,
				JobID:        4,
				BasicSalary:  12000000,
				BankName:     "Mandiri",
				BankAccount:  "9876543210",
				Status:       "active",
			},
		}
		for i := range employees {
			employees[i].CompanyID = 1
			db.Create(&employees[i])
		}

		// Seed sample payslip for Rian
		slip := Payslip{
			Name:        "SLIP/2026/10/00001",
			EmployeeID:  1,
			Period:      "October 2026",
			Date:        "2026-10-04",
			BasicSalary: 18500000,
			Allowances:  1500000, // Tunjangan transport & makan
			Deductions:  500000,  // BPJS Ketenagakerjaan
			NetSalary:   19500000,
			State:       "confirmed",
			Note:        "Monthly salary transfer",
		}
		slip.CompanyID = 1
		db.Create(&slip)

		log.Println("[HR] Seeded default Departments, Jobs, Employees & Payslip")
	}
	return nil
}
