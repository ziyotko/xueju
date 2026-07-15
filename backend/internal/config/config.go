package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName            string
	AppEnv             string
	Port               string
	MySQLDSN           string
	JWTSecret          string
	JWTExpiresHours    int
	WechatAppID        string
	WechatAppSecret    string
	ContentSecurity    bool
	MediaCallbackToken string
	AdminUsername      string
	AdminPassword      string
	PublicBaseURL      string
	UploadDir          string
	StorageDriver      string
	S3Endpoint         string
	S3Bucket           string
	S3Region           string
	S3AccessKey        string
	S3SecretKey        string
}

func Load() Config {
	loadEnv(".env", "../.env", "../../.env")

	return Config{
		AppName:            getEnv("APP_NAME", "xueju-api"),
		AppEnv:             getEnv("APP_ENV", "development"),
		Port:               getEnv("APP_PORT", "8080"),
		MySQLDSN:           getEnv("MYSQL_DSN", ""),
		JWTSecret:          getEnv("JWT_SECRET", "xueju-dev-secret"),
		JWTExpiresHours:    getEnvAsInt("JWT_EXPIRES_HOURS", 168),
		WechatAppID:        getEnv("WECHAT_APP_ID", ""),
		WechatAppSecret:    getEnv("WECHAT_APP_SECRET", ""),
		ContentSecurity:    getEnvAsBool("WECHAT_CONTENT_SECURITY_ENABLED", false),
		MediaCallbackToken: getEnv("WECHAT_MEDIA_CALLBACK_TOKEN", ""),
		AdminUsername:      getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:      getEnv("ADMIN_PASSWORD", "xueju-admin"),
		PublicBaseURL:      strings.TrimRight(getEnv("PUBLIC_BASE_URL", ""), "/"),
		UploadDir:          getEnv("UPLOAD_DIR", "uploads"),
		StorageDriver:      getEnv("STORAGE_DRIVER", "local"),
		S3Endpoint:         strings.TrimRight(getEnv("S3_ENDPOINT", ""), "/"),
		S3Bucket:           getEnv("S3_BUCKET", ""),
		S3Region:           getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:        getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:        getEnv("S3_SECRET_KEY", ""),
	}
}

// Validate rejects development fallbacks in production. This is intentionally
// strict: a misconfigured process must fail before it can accept real traffic.
func (c Config) Validate() error {
	if c.AppEnv != "production" {
		return nil
	}
	var problems []string
	if strings.TrimSpace(c.MySQLDSN) == "" {
		problems = append(problems, "MYSQL_DSN is required")
	}
	if len(c.JWTSecret) < 32 || c.JWTSecret == "xueju-dev-secret" {
		problems = append(problems, "JWT_SECRET must be a non-default value of at least 32 characters")
	}
	if c.WechatAppID == "" || c.WechatAppSecret == "" {
		problems = append(problems, "WECHAT_APP_ID and WECHAT_APP_SECRET are required")
	}
	if !c.ContentSecurity {
		problems = append(problems, "WECHAT_CONTENT_SECURITY_ENABLED must be true")
	}
	if len(c.MediaCallbackToken) < 24 {
		problems = append(problems, "WECHAT_MEDIA_CALLBACK_TOKEN must be at least 24 characters")
	}
	if c.AdminUsername == "admin" || c.AdminPassword == "xueju-admin" || len(c.AdminPassword) < 12 {
		problems = append(problems, "ADMIN_USERNAME and ADMIN_PASSWORD must not use development defaults")
	}
	if c.PublicBaseURL == "" || !strings.HasPrefix(c.PublicBaseURL, "https://") {
		problems = append(problems, "PUBLIC_BASE_URL must be an https URL")
	}
	if c.StorageDriver != "s3" || !strings.HasPrefix(c.S3Endpoint, "https://") || c.S3Bucket == "" || c.S3AccessKey == "" || c.S3SecretKey == "" {
		problems = append(problems, "production requires STORAGE_DRIVER=s3 and complete S3-compatible storage settings")
	}
	if len(problems) > 0 {
		return errors.New(fmt.Sprintf("invalid production configuration: %s", strings.Join(problems, "; ")))
	}
	return nil
}

func loadEnv(paths ...string) {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
			return
		}
	}
}

func getEnv(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvAsBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
