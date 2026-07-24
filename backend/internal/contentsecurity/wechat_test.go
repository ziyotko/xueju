package contentsecurity

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"xueju/backend/internal/compliance"
	"xueju/backend/internal/config"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestCheckTextCallsWechatMsgSecCheck(t *testing.T) {
	cfg := config.Config{
		ContentSecurity:         true,
		ContentSecurityProvider: "wechat",
		WechatAppID:             "wx-test",
		WechatAppSecret:         "secret",
	}
	service := New(cfg)
	var checked bool
	service.httpClient = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		response := `{"access_token":"token","expires_in":7200}`
		if request.URL.Path == "/wxa/msg_sec_check" {
			checked = true
			if request.URL.Query().Get("access_token") != "token" {
				t.Fatalf("unexpected access token query")
			}
			var payload map[string]interface{}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["openid"] != "openid-1" || payload["content"] != "一起去滑雪" ||
				payload["version"] != float64(2) || payload["scene"] != float64(2) {
				t.Fatalf("unexpected msgSecCheck payload: %#v", payload)
			}
			response = `{"errcode":0,"errmsg":"ok","result":{"suggest":"pass","label":100}}`
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(response)),
		}, nil
	})}

	err := service.CheckText(context.Background(), TextCheckRequest{
		OpenID: "openid-1", Scene: 2, Field: compliance.FieldEventTitle, Content: "一起去滑雪",
	})
	if err != nil {
		t.Fatalf("CheckText returned error: %v", err)
	}
	if !checked {
		t.Fatal("msgSecCheck was not called")
	}
}

func TestSecCheckResponseFailsClosed(t *testing.T) {
	for _, suggestion := range []string{"risky", "review"} {
		result := secCheckResponse{}
		result.Result.Suggest = suggestion
		if err := result.toError(); err == nil || err.Error() != compliance.ContentRiskMessage {
			t.Fatalf("suggestion %q should return the public content-risk error, got %v", suggestion, err)
		}
	}

	result := secCheckResponse{}
	if err := result.toError(); err == nil || !strings.Contains(err.Error(), "unknown suggestion") {
		t.Fatalf("an empty suggestion must fail closed, got %v", err)
	}
}

func TestVerifyCallbackSignature(t *testing.T) {
	const signature = "94ffbd20c84766e871799062cb1bc3e3f40c5e13"
	if !VerifyCallbackSignature("abc", "123", "xyz", signature) {
		t.Fatal("valid callback signature was rejected")
	}
	if VerifyCallbackSignature("abc", "123", "changed", signature) {
		t.Fatal("invalid callback signature was accepted")
	}
}

func TestParseJSONMediaCallbackUsesHighestRisk(t *testing.T) {
	payload, err := ParseMediaCallback([]byte(`{
		"appid":"wx-test",
		"trace_id":"trace-1",
		"detail":[
			{"errcode":0,"suggest":"pass"},
			{"errcode":0,"suggest":"risky"}
		]
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if payload.TraceID != "trace-1" || payload.ModerationSuggestion() != "risky" {
		t.Fatalf("unexpected callback result: %#v", payload)
	}
}

func TestParseJSONMediaCallbackAcceptsSingleDetailObject(t *testing.T) {
	payload, err := ParseMediaCallback([]byte(`{
		"appid":"wx-test",
		"trace_id":"trace-single",
		"MsgType":"event",
		"Event":"wxa_media_check",
		"detail":{"errcode":0,"suggest":"pass"}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Detail) != 1 || payload.ModerationSuggestion() != "pass" {
		t.Fatalf("unexpected callback result: %#v", payload)
	}
}

func TestParseXMLMediaCallback(t *testing.T) {
	payload, err := ParseMediaCallback([]byte(`<xml>
		<appid>wx-test</appid>
		<trace_id>trace-2</trace_id>
		<detail><errcode>0</errcode><suggest>pass</suggest></detail>
	</xml>`))
	if err != nil {
		t.Fatal(err)
	}
	if payload.TraceID != "trace-2" || payload.ModerationSuggestion() != "pass" {
		t.Fatalf("unexpected callback result: %#v", payload)
	}
}

func TestMediaCallbackProviderErrorStaysPending(t *testing.T) {
	payload := MediaCallback{
		TraceID: "trace-3",
		Detail:  []MediaCallbackDetail{{ErrCode: -1, Suggest: "pass"}},
	}
	if got := payload.ModerationSuggestion(); got != "review" {
		t.Fatalf("provider error should stay pending, got %q", got)
	}
}
