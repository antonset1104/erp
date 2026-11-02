package config

import (
	"fmt"
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
			dbHost := os.Getenv("DB_HOST")
			if dbHost == "" {
				dbHost = "localhost"
			}
			dbPort := os.Getenv("DB_PORT")
			if dbPort == "" {
				dbPort = "5432"
			}
			dbUser := os.Getenv("DB_USER")
			if dbUser == "" {
				dbUser = "erp"
			}
			dbPassword := os.Getenv("DB_PASSWORD")
			if dbPassword == "" {
				dbPassword = "erp"
			}
			dbName := os.Getenv("DB_NAME")
			if dbName == "" {
				dbName = "erp_db"
			}
			dbSSLMode := os.Getenv("DB_SSLMODE")
			if dbSSLMode == "" {
				dbSSLMode = "disable"
			}
			dbDSN = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
				dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode)
		} else {
			dbPath := os.Getenv("DB_PATH")
			if dbPath != "" {
				dbDSN = dbPath
			} else {
				dbDSN = "erp.db"
			}
		}
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "go-erplang-erp-super-secret-key-2026"
	}

	cfModel := os.Getenv("CLOUDFLARE_AI_MODEL")
	if cfModel == "" {
		cfModel = "@cf/meta/llama-3.1-8b-instruct"
	}

	uploadDir := os.Getenv("UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./uploads/attachments"
	}

	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("APP_ENV")
	}
	if env == "" {
		env = "development"
	}

	return &Config{
		Port:                        port,
		DBType:                      dbType,
		DBDSN:                       dbDSN,
		JWTSecret:                   jwtSecret,
		AppName:                     "GoERP",
		Environment:                 env,
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
