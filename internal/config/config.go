package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port        string
	DBType      string // "sqlite" or "postgres"
	DBDSN       string
	JWTSecret   string
	AppName     string
	Environment string

	// Multi-Provider AI Router ("auto", "gemini", "vertex_ai", "openai", "anthropic", "deepseek", "cloudflare", "local")
	AIProvider string

	// Google Gemini & Google Cloud Platform (GCP) Configuration
	GeminiAPIKey       string
	GeminiModel        string
	GCPProjectID       string
	GCPLocation        string
	GCPVertexModel     string
	GCPBigQueryDataset string
	GCPPubSubTopic     string
	GCSBucket          string
	GCSAccessKeyID     string
	GCSSecretAccessKey string
	GCSEndpoint        string

	// OpenAI API / Agents SDK Configuration
	OpenAIAPIKey  string
	OpenAIModel   string
	OpenAIBaseURL string

	// Anthropic Claude API & MCP Configuration
	AnthropicAPIKey string
	AnthropicModel  string

	// DeepSeek / OpenRouter Reasoning Agent Configuration
	DeepSeekAPIKey  string
	DeepSeekModel   string
	DeepSeekBaseURL string

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

	aiProvider := os.Getenv("AI_PROVIDER")
	if aiProvider == "" {
		aiProvider = "auto"
	}

	geminiModel := os.Getenv("GEMINI_MODEL")
	if geminiModel == "" {
		geminiModel = "gemini-2.5-flash"
	}

	gcpLocation := os.Getenv("GCP_LOCATION")
	if gcpLocation == "" {
		gcpLocation = "asia-southeast2"
	}

	gcpVertexModel := os.Getenv("GCP_VERTEX_MODEL")
	if gcpVertexModel == "" {
		gcpVertexModel = "gemini-2.5-pro"
	}

	gcpBQDataset := os.Getenv("GCP_BIGQUERY_DATASET")
	if gcpBQDataset == "" {
		gcpBQDataset = "goerp_analytics"
	}

	gcpPubSubTopic := os.Getenv("GCP_PUBSUB_TOPIC")
	if gcpPubSubTopic == "" {
		gcpPubSubTopic = "goerp-domain-events"
	}

	gcsEndpoint := os.Getenv("GCS_ENDPOINT")
	if gcsEndpoint == "" {
		gcsEndpoint = "https://storage.googleapis.com"
	}

	openAIModel := os.Getenv("OPENAI_MODEL")
	if openAIModel == "" {
		openAIModel = "gpt-4o"
	}

	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	if openAIBaseURL == "" {
		openAIBaseURL = "https://api.openai.com/v1"
	}

	anthropicModel := os.Getenv("ANTHROPIC_MODEL")
	if anthropicModel == "" {
		anthropicModel = "claude-3-7-sonnet-latest"
	}

	deepSeekModel := os.Getenv("DEEPSEEK_MODEL")
	if deepSeekModel == "" {
		deepSeekModel = "deepseek-chat"
	}

	deepSeekBaseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if deepSeekBaseURL == "" {
		deepSeekBaseURL = "https://api.deepseek.com/v1"
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
		AIProvider:                  aiProvider,
		GeminiAPIKey:                os.Getenv("GEMINI_API_KEY"),
		GeminiModel:                 geminiModel,
		GCPProjectID:                os.Getenv("GCP_PROJECT_ID"),
		GCPLocation:                 gcpLocation,
		GCPVertexModel:              gcpVertexModel,
		GCPBigQueryDataset:          gcpBQDataset,
		GCPPubSubTopic:              gcpPubSubTopic,
		GCSBucket:                   os.Getenv("GCS_BUCKET"),
		GCSAccessKeyID:              os.Getenv("GCS_ACCESS_KEY_ID"),
		GCSSecretAccessKey:          os.Getenv("GCS_SECRET_ACCESS_KEY"),
		GCSEndpoint:                 gcsEndpoint,
		OpenAIAPIKey:                os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:                 openAIModel,
		OpenAIBaseURL:               openAIBaseURL,
		AnthropicAPIKey:             os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:              anthropicModel,
		DeepSeekAPIKey:              os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekModel:               deepSeekModel,
		DeepSeekBaseURL:             deepSeekBaseURL,
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
