package middleware

import "testing"

func TestClientKindDoesNotReturnRawUserAgent(t *testing.T) {
	userAgent := "Mozilla/5.0 wechatdevtools MicroMessenger sid/secret token/sensitive"
	if got := clientKind(userAgent); got != "wechat-devtools" {
		t.Fatalf("clientKind()=%q", got)
	}
	if got := clientKind("Mozilla/5.0 Chrome/150"); got != "browser" {
		t.Fatalf("browser clientKind()=%q", got)
	}
}
