package config

import (
	"os"
)

type Config struct {
	Port         string
	DBType       string // "sqlite" or "postgres"
	DBDSN        string
	JWTSecret    string
	AppName      string
	Environment  string

	// Cloudflare Workers AI Configuration
	CloudflareAccountID string
	CloudflareAPIToken  string
	CloudflareAIModel   string
	CloudflareWorkerURL string

	// Cloudflare R2 Storage Configuration
	CloudflareR2Bucket          string
	CloudflareR2AccessKeyID     string
	CloudflareR2SecretAccessKey string
	CloudflareR2Endpoint        string
	UploadDir                   string
}

func LoadConfig() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbType := os.Getenv("DB_TYPE")
	if dbType == "" {
		dbType = "sqlite"
	}

	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		if dbType == "postgres" {
			dbDSN = "host=localhost user=odoo password=odoo dbname=odoo_erp port=5432 sslmode=disable"
		} else {
			dbDSN = "odoo_erp.db"
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "odoo-golang-erp-super-secret-key-2026"
	}

	cfModel := os.Getenv("CLOUDFLARE_AI_MODEL")
	if cfModel == "" {
		cfModel = "@cf/meta/llama-3.1-8b-instruct"
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads/attachments"
	}

	return &Config{
		Port:                        port,
		DBType:                      dbType,
		DBDSN:                       dbDSN,
		JWTSecret:                   jwtSecret,
		AppName:                     "Odoo-Go ERP",
		Environment:                 "development",
		CloudflareAccountID:         os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		CloudflareAPIToken:          os.Getenv("CLOUDFLARE_API_TOKEN"),
		CloudflareAIModel:           cfModel,
		CloudflareWorkerURL:         os.Getenv("CLOUDFLARE_WORKER_AI_URL"),
		CloudflareR2Bucket:          os.Getenv("CLOUDFLARE_R2_BUCKET"),
		CloudflareR2AccessKeyID:     os.Getenv("CLOUDFLARE_R2_ACCESS_KEY_ID"),
		CloudflareR2SecretAccessKey: os.Getenv("CLOUDFLARE_R2_SECRET_ACCESS_KEY"),
		CloudflareR2Endpoint:        os.Getenv("CLOUDFLARE_R2_ENDPOINT"),
		UploadDir:                   uploadDir,
	}
}
