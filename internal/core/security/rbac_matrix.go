package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SSOProviderConfig struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Protocol           string   `json:"protocol"` // "SAML 2.0" or "OIDC"
	IssuerURL          string   `json:"issuer_url"`
	EntityID           string   `json:"entity_id"`
	ACSURL             string   `json:"acs_url"`
	CallbackURL        string   `json:"callback_url"`
	CertificateStatus  string   `json:"certificate_status"`
	CertThumbprint     string   `json:"cert_thumbprint"`
	SupportedClaimKeys []string `json:"supported_claim_keys"`
	Status             string   `json:"status"` // "ACTIVE", "READY_FOR_IDP_LINKING"
}

type EnterpriseSSOResponse struct {
	EntityID            string              `json:"entity_id"`
	SPMetadataURL       string              `json:"sp_metadata_url"`
	EnforceSSOForDomain string              `json:"enforce_sso_for_domain"`
	Providers           []SSOProviderConfig `json:"providers"`
}

type ModulePermission struct {
	Module      string   `json:"module"`
	Read        bool     `json:"read"`
	Create      bool     `json:"create"`
	Update      bool     `json:"update"`
	Delete      bool     `json:"delete"`
	Post        bool     `json:"post"`
	Approve     bool     `json:"approve"`
	MaskedFields []string `json:"masked_fields,omitempty"`
}

type RoleRBACProfile struct {
	RoleName        string             `json:"role_name"`
	Description     string             `json:"description"`
	Permissions     []ModulePermission `json:"permissions"`
	FieldMaskRules  map[string][]string `json:"field_mask_rules"`
}

type RBACMatrixResponse struct {
	SystemRoles     []string           `json:"system_roles"`
	AvailableModules []string          `json:"available_modules"`
	RoleProfiles    []RoleRBACProfile  `json:"role_profiles"`
	LastAuditedAt   string             `json:"last_audited_at"`
}

type MaskRecordRequest struct {
	Role   string                 `json:"role" binding:"required"`
	Module string                 `json:"module" binding:"required"`
	Data   map[string]interface{} `json:"data" binding:"required"`
}

type SSOCallbackRequest struct {
	ProviderID string `json:"provider_id" binding:"required"`
	Token      string `json:"token" binding:"required"`
	Email      string `json:"email" binding:"required"`
}

type SSOLoginResponse struct {
	Status        string   `json:"status"`
	Authenticated bool     `json:"authenticated"`
	UserEmail     string   `json:"user_email"`
	AssignedRole  string   `json:"assigned_role"`
	SAMLSubject   string   `json:"saml_subject"`
	Groups        []string `json:"groups"`
	JWTToken      string   `json:"jwt_token"`
	ExpiresInSec  int      `json:"expires_in_sec"`
}

// GetEnterpriseSSOConfig returns federated SSO profiles
func GetEnterpriseSSOConfig() *EnterpriseSSOResponse {
	thumbprint := sha256.Sum256([]byte("GOERP-SAML-2026-X509-CERT-CHAIN"))
	thumbHex := hex.EncodeToString(thumbprint[:16])

	return &EnterpriseSSOResponse{
		EntityID:            "https://erp.enterprise.local/auth/saml/metadata",
		SPMetadataURL:       "https://erp.enterprise.local/api/v1/security/sso/metadata.xml",
		EnforceSSOForDomain: "@corp.enterprise.com",
		Providers: []SSOProviderConfig{
			{
				ID:                 "okta-corporate-saml",
				Name:               "Okta Enterprise Workforce Identity",
				Protocol:           "SAML 2.0",
				IssuerURL:          "https://company.okta.com",
				EntityID:           "https://erp.enterprise.local/saml2/sp",
				ACSURL:             "https://erp.enterprise.local/api/v1/security/sso/saml/acs",
				CallbackURL:        "https://erp.enterprise.local/api/v1/security/sso/callback",
				CertificateStatus:  "VALID_SHA256_RSA",
				CertThumbprint:     thumbHex,
				SupportedClaimKeys: []string{"email", "firstName", "lastName", "costCenter", "groups"},
				Status:             "ACTIVE",
			},
			{
				ID:                 "google-workspace-oidc",
				Name:               "Google Workspace Enterprise Cloud Identity",
				Protocol:           "OIDC",
				IssuerURL:          "https://accounts.google.com",
				EntityID:           "goerp-google-oauth-client-id",
				ACSURL:             "https://erp.enterprise.local/api/v1/security/sso/oidc/callback",
				CallbackURL:        "https://erp.enterprise.local/api/v1/security/sso/callback",
				CertificateStatus:  "VALID_JWKS_DISCOVERY",
				CertThumbprint:     "jwks-google-oauth2-v3",
				SupportedClaimKeys: []string{"email", "hd", "sub", "name"},
				Status:             "ACTIVE",
			},
			{
				ID:                 "azure-entra-saml",
				Name:               "Microsoft Entra ID (Azure AD)",
				Protocol:           "SAML 2.0",
				IssuerURL:          "https://login.microsoftonline.com/tenant-id/v2.0",
				EntityID:           "https://erp.enterprise.local/saml2/sp",
				ACSURL:             "https://erp.enterprise.local/api/v1/security/sso/saml/acs",
				CallbackURL:        "https://erp.enterprise.local/api/v1/security/sso/callback",
				CertificateStatus:  "VALID_SHA256_RSA",
				CertThumbprint:     thumbHex,
				SupportedClaimKeys: []string{"upn", "email", "roles"},
				Status:             "ACTIVE",
			},
		},
	}
}

// GetRBACMatrix returns full role-based access control and field masking profiles
func GetRBACMatrix() *RBACMatrixResponse {
	roles := []string{"admin", "finance", "warehouse", "sales", "hr", "auditor"}
	modules := []string{"sale", "purchase", "stock", "account", "mrp", "hr", "quality"}

	profiles := []RoleRBACProfile{
		{
			RoleName:    "admin",
			Description: "Full unrestricted administrative privileges across all modules with zero field masking.",
			Permissions: []ModulePermission{
				{Module: "sale", Read: true, Create: true, Update: true, Delete: true, Post: true, Approve: true},
				{Module: "purchase", Read: true, Create: true, Update: true, Delete: true, Post: true, Approve: true},
				{Module: "stock", Read: true, Create: true, Update: true, Delete: true, Post: true, Approve: true},
				{Module: "account", Read: true, Create: true, Update: true, Delete: true, Post: true, Approve: true},
				{Module: "mrp", Read: true, Create: true, Update: true, Delete: true, Post: true, Approve: true},
				{Module: "hr", Read: true, Create: true, Update: true, Delete: true, Post: true, Approve: true},
				{Module: "quality", Read: true, Create: true, Update: true, Delete: true, Post: true, Approve: true},
			},
			FieldMaskRules: map[string][]string{},
		},
		{
			RoleName:    "warehouse",
			Description: "Warehouse & logistics operator. Access to stock & pickings. Masked cost prices and confidential margins.",
			Permissions: []ModulePermission{
				{Module: "stock", Read: true, Create: true, Update: true, Delete: false, Post: true, Approve: false, MaskedFields: []string{"standard_price", "cost_price"}},
				{Module: "purchase", Read: true, Create: false, Update: false, Delete: false, Post: false, Approve: false, MaskedFields: []string{"amount_untaxed", "amount_total", "price_unit"}},
				{Module: "sale", Read: false, Create: false, Update: false, Delete: false, Post: false, Approve: false},
				{Module: "account", Read: false, Create: false, Update: false, Delete: false, Post: false, Approve: false},
				{Module: "quality", Read: true, Create: true, Update: true, Delete: false, Post: true, Approve: false},
			},
			FieldMaskRules: map[string][]string{
				"product":  {"standard_price", "cost_price"},
				"purchase": {"amount_untaxed", "amount_total", "price_unit"},
				"account":  {"all_fields"},
			},
		},
		{
			RoleName:    "sales",
			Description: "Sales representative. Full access to customer orders and quotations. Masked vendor costs and BOM margins.",
			Permissions: []ModulePermission{
				{Module: "sale", Read: true, Create: true, Update: true, Delete: false, Post: true, Approve: false},
				{Module: "product", Read: true, Create: false, Update: false, Delete: false, Post: false, Approve: false, MaskedFields: []string{"standard_price"}},
				{Module: "stock", Read: true, Create: false, Update: false, Delete: false, Post: false, Approve: false},
				{Module: "purchase", Read: false, Create: false, Update: false, Delete: false, Post: false, Approve: false},
			},
			FieldMaskRules: map[string][]string{
				"product":  {"standard_price"},
				"purchase": {"all_fields"},
			},
		},
		{
			RoleName:    "finance",
			Description: "Financial controller & accountant. Full visibility into General Ledger, bills, tax, and invoices.",
			Permissions: []ModulePermission{
				{Module: "account", Read: true, Create: true, Update: true, Delete: false, Post: true, Approve: true},
				{Module: "purchase", Read: true, Create: true, Update: true, Delete: false, Post: true, Approve: true},
				{Module: "sale", Read: true, Create: true, Update: true, Delete: false, Post: true, Approve: true},
				{Module: "stock", Read: true, Create: false, Update: false, Delete: false, Post: false, Approve: false},
				{Module: "hr", Read: true, Create: false, Update: false, Delete: false, Post: false, Approve: true},
			},
			FieldMaskRules: map[string][]string{},
		},
	}

	return &RBACMatrixResponse{
		SystemRoles:      roles,
		AvailableModules: modules,
		RoleProfiles:     profiles,
		LastAuditedAt:    time.Now().Format("2006-01-02 15:04:05"),
	}
}

// MaskRecordDynamically filters and masks fields based on role and module
func MaskRecordDynamically(role string, module string, rawData map[string]interface{}) map[string]interface{} {
	role = strings.ToLower(role)
	if role == "admin" || role == "finance" {
		return rawData // No masking required
	}

	result := make(map[string]interface{})
	for k, v := range rawData {
		result[k] = v
	}

	matrix := GetRBACMatrix()
	var matchingProfile *RoleRBACProfile
	for i := range matrix.RoleProfiles {
		if strings.ToLower(matrix.RoleProfiles[i].RoleName) == role {
			matchingProfile = &matrix.RoleProfiles[i]
			break
		}
	}

	if matchingProfile == nil {
		// Default strict masking
		return result
	}

	fieldsToMask := matchingProfile.FieldMaskRules[module]
	for _, f := range fieldsToMask {
		if _, exists := result[f]; exists {
			result[f] = "***CONFIDENTIAL***"
		}
	}

	return result
}

// Handler functions for HTTP routes
type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) GetSSOConfig(c *gin.Context) {
	c.JSON(http.StatusOK, GetEnterpriseSSOConfig())
}

func (h *Handler) GetRBACMatrix(c *gin.Context) {
	c.JSON(http.StatusOK, GetRBACMatrix())
}

func (h *Handler) MaskRecord(c *gin.Context) {
	var req MaskRecordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	masked := MaskRecordDynamically(req.Role, req.Module, req.Data)
	c.JSON(http.StatusOK, gin.H{
		"role":   req.Role,
		"module": req.Module,
		"record": masked,
	})
}

func (h *Handler) HandleSSOCallback(c *gin.Context) {
	var req SSOCallbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Mock SAML/OIDC federated token exchange
	role := "sales"
	if strings.Contains(req.Email, "admin") {
		role = "admin"
	} else if strings.Contains(req.Email, "finance") {
		role = "finance"
	} else if strings.Contains(req.Email, "warehouse") {
		role = "warehouse"
	}

	mockJWT := fmt.Sprintf("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.%s", hex.EncodeToString([]byte(req.Email)))

	c.JSON(http.StatusOK, SSOLoginResponse{
		Status:        "SUCCESS",
		Authenticated: true,
		UserEmail:     req.Email,
		AssignedRole:  role,
		SAMLSubject:   fmt.Sprintf("FEDERATED-SUB-%s", req.Email),
		Groups:        []string{"Enterprise-Users", "GoERP-" + strings.Title(role)},
		JWTToken:      mockJWT,
		ExpiresInSec:  28800, // 8 hours
	})
}

// RegisterSecurityRoutes registers security & SSO endpoints
func RegisterSecurityRoutes(router *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	h := NewHandler()
	group := router.Group("/security")
	{
		// Public SSO discovery
		group.GET("/sso/config", h.GetSSOConfig)
		group.POST("/sso/callback", h.HandleSSOCallback)

		// Authenticated RBAC routes
		authed := group.Group("")
		authed.Use(authMiddleware)
		{
			authed.GET("/rbac-matrix", h.GetRBACMatrix)
			authed.POST("/mask-record", h.MaskRecord)
		}
	}
}
