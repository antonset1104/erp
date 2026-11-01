package registry

import (
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module defines the contract for all ERP domain modules (addons)
type Module interface {
	Name() string
	AutoMigrate(db *gorm.DB) error
	RegisterRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc)
	Seed(db *gorm.DB) error
}

type ModuleManager struct {
	modules []Module
	db      *gorm.DB
}

func NewModuleManager(db *gorm.DB) *ModuleManager {
	return &ModuleManager{
		modules: make([]Module, 0),
		db:      db,
	}
}

func (m *ModuleManager) Register(mod Module) {
	m.modules = append(m.modules, mod)
	log.Printf("[Registry] Registered module: %s", mod.Name())
}

func (m *ModuleManager) MigrateAll() error {
	log.Println("[Registry] Running database migrations for all modules...")
	for _, mod := range m.modules {
		if err := mod.AutoMigrate(m.db); err != nil {
			return err
		}
		log.Printf("[Registry] Migrated module: %s", mod.Name())
	}
	return nil
}

func (m *ModuleManager) SeedAll() error {
	log.Println("[Registry] Seeding default master data for all modules...")
	for _, mod := range m.modules {
		if err := mod.Seed(m.db); err != nil {
			return err
		}
	}
	return nil
}

func (m *ModuleManager) RegisterAllRoutes(api *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	for _, mod := range m.modules {
		mod.RegisterRoutes(api, authMiddleware)
	}
}
