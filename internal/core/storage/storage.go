package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"erp-golang/internal/config"
)

// Provider is the universal storage provider interface (ir.attachment backend)
type Provider interface {
	Save(ctx context.Context, key string, contentType string, data []byte) (string, error)
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
	Type() string
}

// LocalProvider implements file storage on local disk
type LocalProvider struct {
	baseDir string
}

func NewLocalProvider(baseDir string) (*LocalProvider, error) {
	if baseDir == "" {
		baseDir = "./uploads/attachments"
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create upload directory %s: %w", baseDir, err)
	}
	return &LocalProvider{baseDir: baseDir}, nil
}

func (l *LocalProvider) Save(ctx context.Context, key string, contentType string, data []byte) (string, error) {
	cleanKey := filepath.Clean(key)
	fullPath := filepath.Join(l.baseDir, cleanKey)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write file %s: %w", fullPath, err)
	}

	return cleanKey, nil
}

func (l *LocalProvider) Get(ctx context.Context, key string) ([]byte, error) {
	cleanKey := filepath.Clean(key)
	fullPath := filepath.Join(l.baseDir, cleanKey)
	return os.ReadFile(fullPath)
}

func (l *LocalProvider) Delete(ctx context.Context, key string) error {
	cleanKey := filepath.Clean(key)
	fullPath := filepath.Join(l.baseDir, cleanKey)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (l *LocalProvider) Type() string {
	return "local"
}

// R2Provider implements Cloudflare R2 S3-compatible cloud object storage
type R2Provider struct {
	accountID string
	bucket    string
	accessKey string
	secretKey string
	endpoint  string
	client    *http.Client
	fallback  *LocalProvider
}

func NewR2Provider(cfg *config.Config, fallback *LocalProvider) *R2Provider {
	endpoint := cfg.CloudflareR2Endpoint
	if endpoint == "" && cfg.CloudflareAccountID != "" {
		endpoint = fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.CloudflareAccountID)
	}

	return &R2Provider{
		accountID: cfg.CloudflareAccountID,
		bucket:    cfg.CloudflareR2Bucket,
		accessKey: cfg.CloudflareR2AccessKeyID,
		secretKey: cfg.CloudflareR2SecretAccessKey,
		endpoint:  endpoint,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		fallback: fallback,
	}
}

func (r *R2Provider) isConfigured() bool {
	return r.bucket != "" && r.accessKey != "" && r.secretKey != "" && r.endpoint != ""
}

func (r *R2Provider) Save(ctx context.Context, key string, contentType string, data []byte) (string, error) {
	if !r.isConfigured() {
		if r.fallback != nil {
			return r.fallback.Save(ctx, key, contentType, data)
		}
		return "", errors.New("neither R2 nor local storage provider is ready")
	}

	url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(r.endpoint, "/"), r.bucket, strings.TrimLeft(key, "/"))
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))

	now := time.Now().UTC()
	signAWS4(req, data, r.accessKey, r.secretKey, "auto", "s3", now)

	resp, err := r.client.Do(req)
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Save(ctx, key, contentType, data)
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return key, nil
	}

	body, _ := io.ReadAll(resp.Body)
	if r.fallback != nil {
		return r.fallback.Save(ctx, key, contentType, data)
	}
	return "", fmt.Errorf("R2 upload returned status %d: %s", resp.StatusCode, string(body))
}

func (r *R2Provider) Get(ctx context.Context, key string) ([]byte, error) {
	if !r.isConfigured() {
		if r.fallback != nil {
			return r.fallback.Get(ctx, key)
		}
		return nil, errors.New("storage provider not configured")
	}

	url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(r.endpoint, "/"), r.bucket, strings.TrimLeft(key, "/"))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	signAWS4(req, nil, r.accessKey, r.secretKey, "auto", "s3", now)

	resp, err := r.client.Do(req)
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Get(ctx, key)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return io.ReadAll(resp.Body)
	}

	if r.fallback != nil {
		return r.fallback.Get(ctx, key)
	}
	return nil, fmt.Errorf("R2 download returned status %d", resp.StatusCode)
}

func (r *R2Provider) Delete(ctx context.Context, key string) error {
	if !r.isConfigured() {
		if r.fallback != nil {
			return r.fallback.Delete(ctx, key)
		}
		return nil
	}

	url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(r.endpoint, "/"), r.bucket, strings.TrimLeft(key, "/"))
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	signAWS4(req, nil, r.accessKey, r.secretKey, "auto", "s3", now)

	resp, err := r.client.Do(req)
	if err != nil {
		if r.fallback != nil {
			_ = r.fallback.Delete(ctx, key)
		}
		return err
	}
	defer resp.Body.Close()

	if r.fallback != nil {
		_ = r.fallback.Delete(ctx, key)
	}
	return nil
}

func (r *R2Provider) Type() string {
	if r.isConfigured() {
		return "r2"
	}
	return "local"
}

// GCSProvider implements Google Cloud Storage (GCS) S3-interoperable object storage
type GCSProvider struct {
	projectID string
	bucket    string
	accessKey string
	secretKey string
	endpoint  string
	region    string
	client    *http.Client
	fallback  *LocalProvider
}

func NewGCSProvider(cfg *config.Config, fallback *LocalProvider) *GCSProvider {
	endpoint := cfg.GCSEndpoint
	if endpoint == "" {
		endpoint = "https://storage.googleapis.com"
	}
	region := cfg.GCPLocation
	if region == "" {
		region = "asia-southeast2"
	}
	return &GCSProvider{
		projectID: cfg.GCPProjectID,
		bucket:    cfg.GCSBucket,
		accessKey: cfg.GCSAccessKeyID,
		secretKey: cfg.GCSSecretAccessKey,
		endpoint:  endpoint,
		region:    region,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		fallback: fallback,
	}
}

func (g *GCSProvider) isConfigured() bool {
	return g.bucket != "" && g.accessKey != "" && g.secretKey != ""
}

func (g *GCSProvider) Save(ctx context.Context, key string, contentType string, data []byte) (string, error) {
	if !g.isConfigured() {
		if g.fallback != nil {
			return g.fallback.Save(ctx, key, contentType, data)
		}
		return "", errors.New("neither Google Cloud Storage nor local storage provider is ready")
	}

	url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(g.endpoint, "/"), g.bucket, strings.TrimLeft(key, "/"))
	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(data)))

	now := time.Now().UTC()
	signAWS4(req, data, g.accessKey, g.secretKey, g.region, "s3", now)

	resp, err := g.client.Do(req)
	if err != nil {
		if g.fallback != nil {
			return g.fallback.Save(ctx, key, contentType, data)
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return key, nil
	}

	if g.fallback != nil {
		return g.fallback.Save(ctx, key, contentType, data)
	}
	body, _ := io.ReadAll(resp.Body)
	return "", fmt.Errorf("GCS upload returned status %d: %s", resp.StatusCode, string(body))
}

func (g *GCSProvider) Get(ctx context.Context, key string) ([]byte, error) {
	if !g.isConfigured() {
		if g.fallback != nil {
			return g.fallback.Get(ctx, key)
		}
		return nil, errors.New("GCS storage provider not configured")
	}

	url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(g.endpoint, "/"), g.bucket, strings.TrimLeft(key, "/"))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	signAWS4(req, nil, g.accessKey, g.secretKey, g.region, "s3", now)

	resp, err := g.client.Do(req)
	if err != nil {
		if g.fallback != nil {
			return g.fallback.Get(ctx, key)
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return io.ReadAll(resp.Body)
	}

	if g.fallback != nil {
		return g.fallback.Get(ctx, key)
	}
	return nil, fmt.Errorf("GCS download returned status %d", resp.StatusCode)
}

func (g *GCSProvider) Delete(ctx context.Context, key string) error {
	if !g.isConfigured() {
		if g.fallback != nil {
			return g.fallback.Delete(ctx, key)
		}
		return nil
	}

	url := fmt.Sprintf("%s/%s/%s", strings.TrimRight(g.endpoint, "/"), g.bucket, strings.TrimLeft(key, "/"))
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	signAWS4(req, nil, g.accessKey, g.secretKey, g.region, "s3", now)

	resp, err := g.client.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}

	if g.fallback != nil {
		_ = g.fallback.Delete(ctx, key)
	}
	return nil
}

func (g *GCSProvider) Type() string {
	if g.isConfigured() {
		return "gcs"
	}
	return "local"
}

// InitStorageProvider returns the optimal storage provider based on configuration
func InitStorageProvider(cfg *config.Config) (Provider, error) {
	local, err := NewLocalProvider(cfg.UploadDir)
	if err != nil {
		return nil, err
	}

	if cfg.GCSBucket != "" && cfg.GCSAccessKeyID != "" && cfg.GCSSecretAccessKey != "" {
		return NewGCSProvider(cfg, local), nil
	}

	if cfg.CloudflareR2Bucket != "" && cfg.CloudflareR2AccessKeyID != "" && cfg.CloudflareR2SecretAccessKey != "" {
		return NewR2Provider(cfg, local), nil
	}

	return local, nil
}

// Helper: Lightweight AWS SigV4 signer for Cloudflare R2 & Google Cloud Storage S3-Interoperability
func signAWS4(req *http.Request, payload []byte, accessKey, secretKey, region, service string, t time.Time) {
	dateStamp := t.Format("20060102")
	amzDate := t.Format("20060102T150405Z")

	var payloadHash string
	if len(payload) > 0 {
		hash := sha256.Sum256(payload)
		payloadHash = hex.EncodeToString(hash[:])
	} else {
		hash := sha256.Sum256([]byte(""))
		payloadHash = hex.EncodeToString(hash[:])
	}

	req.Header.Set("x-amz-date", amzDate)
	req.Header.Set("x-amz-content-sha256", payloadHash)

	host := req.URL.Host
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", host, payloadHash, amzDate)
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"

	canonicalURI := req.URL.EscapedPath()
	if canonicalURI == "" {
		canonicalURI = "/"
	}

	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI,
		req.URL.RawQuery,
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")

	reqHash := sha256.Sum256([]byte(canonicalRequest))
	credentialScope := fmt.Sprintf("%s/%s/%s/aws4_request", dateStamp, region, service)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s", amzDate, credentialScope, hex.EncodeToString(reqHash[:]))

	kDate := hmacSHA256([]byte("AWS4"+secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))

	signature := hex.EncodeToString(hmacSHA256(kSigning, []byte(stringToSign)))
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		accessKey, credentialScope, signedHeaders, signature)

	req.Header.Set("Authorization", authHeader)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
