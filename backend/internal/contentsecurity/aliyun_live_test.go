package contentsecurity

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"xueju/backend/internal/config"
)

func TestLiveAliyunTextAndImageModeration(t *testing.T) {
	if os.Getenv("XUEJU_LIVE_ALIYUN_CONTENT_CHECK") != "1" {
		t.Skip("set XUEJU_LIVE_ALIYUN_CONTENT_CHECK=1 to run paid Aliyun content moderation checks")
	}
	cfg := config.Load()
	cfg.ContentSecurity = true
	cfg.ContentSecurityProvider = "aliyun"
	service := New(cfg)

	if err := service.CheckText(context.Background(), TextCheckRequest{Content: "今天一起去滑雪，注意安全。"}); err != nil {
		t.Fatalf("safe text moderation failed: %v", err)
	}

	canvas := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			canvas.Set(x, y, color.RGBA{R: 230, G: 242, B: 255, A: 255})
		}
	}
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, canvas); err != nil {
		t.Fatal(err)
	}
	result, err := service.CheckImage(context.Background(), ImageCheckRequest{
		Kind: "avatars", Filename: "safe-test.png", Data: imageData.Bytes(), AccountID: "codex-live-test",
	})
	if err != nil {
		t.Fatalf("safe image moderation failed: %v", err)
	}
	if result.Status != "approved" {
		t.Fatalf("safe image was not approved: status=%s review=%s", result.Status, result.ReviewResult)
	}
	t.Logf("live Aliyun checks passed: image_status=%s review=%s", result.Status, result.ReviewResult)
}
