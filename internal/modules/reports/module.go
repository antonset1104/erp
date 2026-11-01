package reports

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Module struct {
	service Service
	handler *Handler
}

func NewModule(db *gorm.DB) *Module {
	svc := NewService(db)
	h := NewHandler(svc)
	return &Module{
		service: svc,
		handler: h,
	}
}

func (m *Module) Name() string {
	return "reports"
}

func (m *Module) AutoMigrate(db *gorm.DB) error {
	// Reports is a view and rendering engine layer, no separate DB tables needed
	return nil
}

func (m *Module) Seed(db *gorm.DB) error {
	log.Println("📑 [Reports] Universal Enterprise Report & Printout Engine initialized successfully")
	return nil
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	m.handler.RegisterRoutes(api, authMiddleware)
}

func (m *Module) Service() Service {
	return m.service
}
