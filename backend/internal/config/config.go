package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppName         string
	AppEnv          string
	Port            string
	MySQLDSN        string
	JWTSecret       string
	JWTExpiresHours int
	WechatAppID     string
	WechatAppSecret string
	ContentSecurity bool
	AdminUsername   string
	AdminPassword   string
}

func Load() Config {
	loadEnv(".env", "../.env", "../../.env")

	return Config{
		AppName:         getEnv("APP_NAME", "xueju-api"),
		AppEnv:          getEnv("APP_ENV", "development"),
		Port:            getEnv("APP_PORT", "8080"),
		MySQLDSN:        getEnv("MYSQL_DSN", ""),
		JWTSecret:       getEnv("JWT_SECRET", "xueju-dev-secret"),
		JWTExpiresHours: getEnvAsInt("JWT_EXPIRES_HOURS", 168),
		WechatAppID:     getEnv("WECHAT_APP_ID", ""),
		WechatAppSecret: getEnv("WECHAT_APP_SECRET", ""),
		ContentSecurity: getEnvAsBool("WECHAT_CONTENT_SECURITY_ENABLED", false),
		AdminUsername:   getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword:   getEnv("ADMIN_PASSWORD", "xueju-admin"),
	}
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
