package config

import (
	"strings"
	"testing"
)

func validProductionConfig() Config {
	return Config{
		AppEnv: "production", MySQLDSN: "user:pass@tcp(db:3306)/xueju", JWTSecret: strings.Repeat("s", 32),
		WechatAppID: "wx-app", WechatAppSecret: "secret", ContentSecurity: true, ContentSecurityProvider: "wechat",
		MediaCallbackToken: strings.Repeat("t", 32),
		AdminUsername:      "operator", AdminPassword: "a-long-admin-password", PublicBaseURL: "https://api.example.com",
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

func TestValidateRejectsAliyunAsProductionContentSecurityProvider(t *testing.T) {
	cfg := validProductionConfig()
	cfg.ContentSecurityProvider = "aliyun"
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "must be wechat") {
		t.Fatalf("expected production to require WeChat content security, got %v", err)
	}
}

func TestValidateAcceptsProductionLocalStorageWithAbsolutePath(t *testing.T) {
	cfg := validProductionConfig()
	cfg.StorageDriver = "local"
	cfg.UploadDir = t.TempDir()
	cfg.S3Endpoint = ""
	cfg.S3Bucket = ""
	cfg.S3AccessKey = ""
	cfg.S3SecretKey = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestValidateRejectsProductionLocalStorageWithRelativePath(t *testing.T) {
	cfg := validProductionConfig()
	cfg.StorageDriver = "local"
	cfg.UploadDir = "uploads"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected relative production upload directory to be rejected")
	}
}

func TestValidateRejectsUnknownTimezone(t *testing.T) {
	cfg := Config{AppEnv: "development", Timezone: "Mars/Olympus_Mons"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid timezone to be rejected")
	}
}

func TestLocationDefaultsToShanghai(t *testing.T) {
	location, err := (Config{}).Location()
	if err != nil {
		t.Fatal(err)
	}
	if location.String() != "Asia/Shanghai" {
		t.Fatalf("unexpected default timezone %q", location.String())
	}
}
