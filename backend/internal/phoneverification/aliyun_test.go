package phoneverification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"xueju/backend/internal/config"
)

func TestAliyunSendAndCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("AccessKeyId") == "" || r.Form.Get("Signature") == "" {
			t.Fatal("missing signed RPC parameters")
		}
		switch r.Form.Get("Action") {
		case "SendSmsVerifyCode":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Success": true, "Code": "OK", "RequestId": "send-request", "Model": map[string]string{"BizId": "biz-1"}})
		case "CheckSmsVerifyCode":
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"Success": true, "Code": "OK", "RequestId": "check-request", "Model": map[string]string{"VerifyResult": "PASS"}})
		default:
			t.Fatalf("unexpected action %q", r.Form.Get("Action"))
		}
	}))
	defer server.Close()

	client := &AliyunClient{accessKeyID: "ram-key", accessKeySecret: "secret", signName: "sign", templateCode: "100001", expires: 300, httpClient: server.Client(), endpoint: server.URL}
	sent, err := client.Send(context.Background(), "13800138000", "internal-request")
	if err != nil || sent.ProviderReference != "biz-1" {
		t.Fatalf("send failed: %#v %v", sent, err)
	}
	checked, err := client.Check(context.Background(), "13800138000", "123456", "internal-request")
	if err != nil || !checked.Passed || checked.ProviderReference != "check-request" {
		t.Fatalf("check failed: %#v %v", checked, err)
	}
}

func TestLiveAliyunConfiguration(t *testing.T) {
	if os.Getenv("XUEJU_LIVE_ALIYUN_CHECK") != "1" {
		t.Skip("set XUEJU_LIVE_ALIYUN_CHECK=1 to run the no-SMS live connectivity check")
	}
	client := NewAliyunClient(config.Load())
	result, err := client.Check(context.Background(), "13000000000", "000000", "xueju-connectivity-check")
	if err != nil {
		t.Fatalf("Aliyun connectivity check failed: %v", err)
	}
	if result.Passed {
		t.Fatal("unexpected verification pass for diagnostic code")
	}
}
