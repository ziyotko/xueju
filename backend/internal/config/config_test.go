package config

import (
	"strings"
	"testing"
)

func validProductionConfig() Config {
	return Config{
		AppEnv: "production", MySQLDSN: "user:pass@tcp(db:3306)/xueju", JWTSecret: strings.Repeat("s", 32),
		WechatAppID: "wx-app", WechatAppSecret: "secret", ContentSecurity: true, MediaCallbackToken: "media-callback-token-long-enough",
		AdminUsername: "operator", AdminPassword: "a-long-admin-password", PublicBaseURL: "https://api.example.com",
		StorageDriver: "s3", S3Endpoint: "https://storage.example.com", S3Bucket: "xueju", S3Region: "cn-north-1", S3AccessKey: "key", S3SecretKey: "secret",
		AliyunAccessKeyID: "ram-key", AliyunAccessKeySecret: "ram-secret", AliyunSmsSignName: "sign", AliyunSmsTemplateCode: "100001",
		PhoneEncryptionKey: strings.Repeat("e", 32), PhoneHashSecret: strings.Repeat("h", 32),
	}
}

func TestValidateRejectsProductionDefaults(t *testing.T) {
	cfg := validProductionConfig()
	cfg.JWTSecret = "xueju-dev-secret"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production defaults to be rejected")
	}
}

func TestValidateAcceptsHardenedProductionConfig(t *testing.T) {
	if err := validProductionConfig().Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
