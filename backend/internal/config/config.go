package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName                string
	AppEnv                 string
	Port                   string
	Timezone               string
	MySQLDSN               string
	JWTSecret              string
	JWTExpiresHours        int
	WechatAppID            string
	WechatAppSecret        string
	ContentSecurity        bool
	MediaCallbackToken     string
	AdminUsername          string
	AdminPassword          string
	PublicBaseURL          string
	UploadDir              string
	StorageDriver          string
	S3Endpoint             string
	S3Bucket               string
	S3Region               string
	S3AccessKey            string
	S3SecretKey            string
	AliyunAccessKeyID      string
	AliyunAccessKeySecret  string
	AliyunSmsSignName      string
	AliyunSmsTemplateCode  string
	AliyunSmsSchemeName    string
	PhoneEncryptionKey     string
	PhoneHashSecret        string
	SmsCodeExpireSeconds   int
	SmsSendIntervalSeconds int
	SmsDailyLimitPerPhone  int
	SmsDailyLimitPerUser   int
	SmsHourlyLimitPerIP    int
}

func Load() Config {
	loadEnv(".env", "../.env", "../../.env")

	return Config{
		AppName:                getEnv("APP_NAME", "xueju-api"),
		AppEnv:                 getEnv("APP_ENV", "development"),
		Port:                   getEnv("APP_PORT", "8080"),
		Timezone:               getEnv("APP_TIMEZONE", "Asia/Shanghai"),
		MySQLDSN:               getEnv("MYSQL_DSN", ""),
		JWTSecret:              getEnv("JWT_SECRET", "xueju-dev-secret"),
		JWTExpiresHours:        getEnvAsInt("JWT_EXPIRES_HOURS", 168),
		WechatAppID:            getEnv("WECHAT_APP_ID", ""),
		WechatAppSecret:        getEnv("WECHAT_APP_SECRET", ""),
		ContentSecurity:        getEnvAsBool("WECHAT_CONTENT_SECURITY_ENABLED", false),
		MediaCallbackToken:     getEnv("WECHAT_MEDIA_CALLBACK_TOKEN", ""),
		AdminUsername:          getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:          getEnv("ADMIN_PASSWORD", "xueju-admin"),
		PublicBaseURL:          strings.TrimRight(getEnv("PUBLIC_BASE_URL", ""), "/"),
		UploadDir:              getEnv("UPLOAD_DIR", "uploads"),
		StorageDriver:          getEnv("STORAGE_DRIVER", "local"),
		S3Endpoint:             strings.TrimRight(getEnv("S3_ENDPOINT", ""), "/"),
		S3Bucket:               getEnv("S3_BUCKET", ""),
		S3Region:               getEnv("S3_REGION", "us-east-1"),
		S3AccessKey:            getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:            getEnv("S3_SECRET_KEY", ""),
		AliyunAccessKeyID:      getEnv("ALIYUN_ACCESS_KEY_ID", ""),
		AliyunAccessKeySecret:  getEnv("ALIYUN_ACCESS_KEY_SECRET", ""),
		AliyunSmsSignName:      getEnv("ALIYUN_SMS_VERIFY_SIGN_NAME", ""),
		AliyunSmsTemplateCode:  getEnv("ALIYUN_SMS_VERIFY_TEMPLATE_CODE", ""),
		AliyunSmsSchemeName:    getEnv("ALIYUN_SMS_VERIFY_SCHEME_NAME", ""),
		PhoneEncryptionKey:     getEnv("PHONE_ENCRYPTION_KEY", ""),
		PhoneHashSecret:        getEnv("PHONE_HASH_SECRET", ""),
		SmsCodeExpireSeconds:   getEnvAsInt("SMS_CODE_EXPIRE_SECONDS", 300),
		SmsSendIntervalSeconds: getEnvAsInt("SMS_SEND_INTERVAL_SECONDS", 60),
		SmsDailyLimitPerPhone:  getEnvAsInt("SMS_DAILY_LIMIT_PER_PHONE", 10),
		SmsDailyLimitPerUser:   getEnvAsInt("SMS_DAILY_LIMIT_PER_USER", 10),
		SmsHourlyLimitPerIP:    getEnvAsInt("SMS_HOURLY_LIMIT_PER_IP", 30),
	}
}

// Validate rejects development fallbacks in production. This is intentionally
// strict: a misconfigured process must fail before it can accept real traffic.
func (c Config) Validate() error {
	if _, err := c.Location(); err != nil {
		return err
	}
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
	if c.AliyunAccessKeyID == "" || c.AliyunAccessKeySecret == "" || c.AliyunSmsSignName == "" || c.AliyunSmsTemplateCode == "" {
		problems = append(problems, "Aliyun SMS verification credentials, sign name and template code are required")
	}
	if len(c.PhoneEncryptionKey) < 32 || len(c.PhoneHashSecret) < 32 {
		problems = append(problems, "PHONE_ENCRYPTION_KEY and PHONE_HASH_SECRET must each be at least 32 characters")
	}
	if len(problems) > 0 {
		return errors.New(fmt.Sprintf("invalid production configuration: %s", strings.Join(problems, "; ")))
	}
	return nil
}

func (c Config) Location() (*time.Location, error) {
	name := strings.TrimSpace(c.Timezone)
	if name == "" {
		name = "Asia/Shanghai"
	}
	location, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid APP_TIMEZONE %q: %w", name, err)
	}
	return location, nil
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
