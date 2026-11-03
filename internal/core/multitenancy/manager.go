package multitenancy

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Tenant represents an isolated corporate tenant in the GoERP SaaS ecosystem
type Tenant struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	TenantID    string    `json:"tenant_id" gorm:"uniqueIndex;size:64;not null"`
	Name        string    `json:"name" gorm:"size:128;not null"`
	Domain      string    `json:"domain" gorm:"size:128"`
	Plan        string    `json:"plan" gorm:"size:32;default:'professional'"` // "starter", "professional", "enterprise"
	Status      string    `json:"status" gorm:"size:32;default:'active'"`    // "active", "trial", "suspended"
	StorageQuotaMB int64  `json:"storage_quota_mb" gorm:"default:10240"`
	UserLimit   int       `json:"user_limit" gorm:"default:50"`
	DBType      string    `json:"db_type" gorm:"size:32;default:'sqlite'"`
	DBSchema    string    `json:"db_schema" gorm:"size:64"`
	DBPath      string    `json:"db_path" gorm:"size:255"`
	AdminEmail  string    `json:"admin_email" gorm:"size:128"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TenantRegisterRequest is the payload for POST /api/v1/saas/tenants/register
type TenantRegisterRequest struct {
	TenantID   string `json:"tenant_id" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Domain     string `json:"domain"`
	Plan       string `json:"plan"` // "starter", "professional", "enterprise"
	AdminEmail string `json:"admin_email" binding:"required"`
	AdminName  string `json:"admin_name"`
}

// TenantManager handles multi-tenancy resolution, tenant isolation, and provisioning
type TenantManager struct {
	db       *gorm.DB
	mu       sync.RWMutex
	cache    map[string]*Tenant
}

var globalManager *TenantManager
var managerOnce sync.Once

// InitManager initializes the global SaaS Tenant Manager
func InitManager(db *gorm.DB) *TenantManager {
	managerOnce.Do(func() {
		// Auto-migrate Tenant model
		_ = db.AutoMigrate(&Tenant{})

		globalManager = &TenantManager{
			db:    db,
			cache: make(map[string]*Tenant),
		}

		// Seed primary default tenant if not exists
		var cnt int64
		db.Model(&Tenant{}).Where("tenant_id = ?", "default").Count(&cnt)
		if cnt == 0 {
			defaultTenant := Tenant{
				TenantID:       "default",
				Name:           "GoERP Primary Organization",
				Domain:         "app.goerp.local",
				Plan:           "enterprise",
				Status:         "active",
				StorageQuotaMB: 51200,
				UserLimit:      500,
				DBType:         "sqlite",
				DBSchema:       "public",
				AdminEmail:     "admin@erp.local",
			}
			db.Create(&defaultTenant)
			globalManager.cache["default"] = &defaultTenant
		}
	})
	return globalManager
}

// GetManager returns the active TenantManager instance
func GetManager() *TenantManager {
	return globalManager
}

// ResolveTenant extracts tenant identifier from Host, Headers, or Query params
func (tm *TenantManager) ResolveTenant(c *gin.Context) *Tenant {
	// 1. Explicit Header X-Tenant-ID
	tID := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))

	// 2. Query param ?tenant=...
	if tID == "" {
		tID = strings.TrimSpace(c.Query("tenant"))
	}

	// 3. Subdomain parsing from Host header (e.g. "acme.goerp.io" -> "acme")
	if tID == "" {
		host := c.Request.Host
		if colon := strings.Index(host, ":"); colon != -1 {
			host = host[:colon]
		}
		parts := strings.Split(host, ".")
		if len(parts) >= 3 && parts[0] != "www" && parts[0] != "api" && parts[0] != "app" {
			tID = parts[0]
		}
	}

	if tID == "" {
		tID = "default"
	}

	tID = strings.ToLower(tID)

	tm.mu.RLock()
	if cached, ok := tm.cache[tID]; ok {
		tm.mu.RUnlock()
		return cached
	}
	tm.mu.RUnlock()

	var tenant Tenant
	if err := tm.db.Where("tenant_id = ? OR domain = ?", tID, tID).First(&tenant).Error; err == nil {
		tm.mu.Lock()
		tm.cache[tID] = &tenant
		tm.mu.Unlock()
		return &tenant
	}

	// Fallback to default
	var defTenant Tenant
	if err := tm.db.Where("tenant_id = ?", "default").First(&defTenant).Error; err == nil {
		return &defTenant
	}

	return &Tenant{TenantID: "default", Name: "GoERP Primary", Plan: "enterprise", Status: "active"}
}

// RegisterTenant provisions a new tenant in the SaaS ecosystem
func (tm *TenantManager) RegisterTenant(req *TenantRegisterRequest) (*Tenant, error) {
	tID := strings.ToLower(strings.TrimSpace(req.TenantID))
	if tID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	var existing Tenant
	if err := tm.db.Where("tenant_id = ?", tID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("tenant_id '%s' is already registered", tID)
	}

	plan := strings.ToLower(strings.TrimSpace(req.Plan))
	if plan == "" {
		plan = "professional"
	}

	domain := req.Domain
	if domain == "" {
		domain = fmt.Sprintf("%s.goerp.local", tID)
	}

	tenant := Tenant{
		TenantID:       tID,
		Name:           req.Name,
		Domain:         domain,
		Plan:           plan,
		Status:         "active",
		StorageQuotaMB: 20480,
		UserLimit:      100,
		DBType:         "sqlite",
		DBSchema:       fmt.Sprintf("tenant_%s", tID),
		DBPath:         fmt.Sprintf("./data/tenants/%s/erp.db", tID),
		AdminEmail:     req.AdminEmail,
	}

	if err := tm.db.Create(&tenant).Error; err != nil {
		return nil, err
	}

	tm.mu.Lock()
	tm.cache[tID] = &tenant
	tm.mu.Unlock()

	return &tenant, nil
}

// ListTenants returns all registered tenants
func (tm *TenantManager) ListTenants() ([]Tenant, error) {
	var list []Tenant
	err := tm.db.Order("id desc").Find(&list).Error
	return list, err
}

// TenantMiddleware injects the resolved Tenant into gin.Context
func TenantMiddleware(tm *TenantManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenant := tm.ResolveTenant(c)
		c.Set("tenant", tenant)
		c.Set("tenant_id", tenant.TenantID)
		c.Header("X-Tenant-ID", tenant.TenantID)
		c.Next()
	}
}

// RegisterSaaSRoutes mounts tenant provisioning & listing endpoints on Gin
func RegisterSaaSRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc, tm *TenantManager) {
	group := router.Group("/saas/tenants")
	{
		// Public registration endpoint for new corporate customers
		group.POST("/register", func(c *gin.Context) {
			var req TenantRegisterRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			tenant, err := tm.RegisterTenant(&req)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{
				"success": true,
				"message": fmt.Sprintf("Tenant '%s' provisioned successfully on %s plan.", tenant.TenantID, tenant.Plan),
				"tenant":  tenant,
			})
		})

		// Authenticated tenant management endpoints
		authed := group.Group("")
		authed.Use(authMiddleware)
		{
			authed.GET("", func(c *gin.Context) {
				tenants, err := tm.ListTenants()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"success":       true,
					"total_tenants": len(tenants),
					"tenants":       tenants,
				})
			})

			authed.GET("/current", func(c *gin.Context) {
				tenant := tm.ResolveTenant(c)
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"tenant":  tenant,
				})
			})
		}
	}
}
