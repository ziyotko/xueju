package phoneverification

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"xueju/backend/internal/config"
)

type SendResult struct {
	ProviderReference string
}

type CheckResult struct {
	Passed            bool
	ProviderReference string
}

type Client interface {
	Send(ctx context.Context, phone, outID string) (SendResult, error)
	Check(ctx context.Context, phone, code, outID string) (CheckResult, error)
}

type AliyunClient struct {
	accessKeyID     string
	accessKeySecret string
	signName        string
	templateCode    string
	schemeName      string
	expires         int
	httpClient      *http.Client
	endpoint        string
}

func NewAliyunClient(cfg config.Config) *AliyunClient {
	return &AliyunClient{
		accessKeyID: cfg.AliyunAccessKeyID, accessKeySecret: cfg.AliyunAccessKeySecret,
		signName: cfg.AliyunSmsSignName, templateCode: cfg.AliyunSmsTemplateCode,
		schemeName: cfg.AliyunSmsSchemeName, expires: cfg.SmsCodeExpireSeconds,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		endpoint:   "https://dypnsapi.aliyuncs.com/",
	}
}

func (a *AliyunClient) Send(ctx context.Context, phone, outID string) (SendResult, error) {
	minutes := a.expires / 60
	if minutes < 1 {
		minutes = 5
	}
	templateParam, _ := json.Marshal(map[string]string{"code": "##code##", "min": fmt.Sprint(minutes)})
	params := map[string]string{
		"PhoneNumber": phone, "CountryCode": "86", "SignName": a.signName,
		"TemplateCode": a.templateCode, "TemplateParam": string(templateParam), "OutId": outID,
	}
	if a.schemeName != "" {
		params["SchemeName"] = a.schemeName
	}
	var result aliyunResponse
	if err := a.call(ctx, "SendSmsVerifyCode", params, &result); err != nil {
		return SendResult{}, err
	}
	if !result.Success || result.Code != "OK" {
		return SendResult{}, fmt.Errorf("aliyun sms send failed: %s", result.Code)
	}
	reference := firstNonEmpty(result.Model.BizID, result.Model.RequestID, result.RequestID)
	return SendResult{ProviderReference: reference}, nil
}

func (a *AliyunClient) Check(ctx context.Context, phone, code, outID string) (CheckResult, error) {
	params := map[string]string{"PhoneNumber": phone, "CountryCode": "86", "VerifyCode": code, "OutId": outID, "CaseAuthPolicy": "1"}
	if a.schemeName != "" {
		params["SchemeName"] = a.schemeName
	}
	var result aliyunResponse
	if err := a.call(ctx, "CheckSmsVerifyCode", params, &result); err != nil {
		return CheckResult{}, err
	}
	if !result.Success || result.Code != "OK" {
		return CheckResult{}, fmt.Errorf("aliyun sms check failed: %s", result.Code)
	}
	return CheckResult{Passed: result.Model.VerifyResult == "PASS", ProviderReference: result.RequestID}, nil
}

type aliyunResponse struct {
	Success   bool   `json:"Success"`
	Code      string `json:"Code"`
	Message   string `json:"Message"`
	RequestID string `json:"RequestId"`
	Model     struct {
		RequestID    string `json:"RequestId"`
		BizID        string `json:"BizId"`
		VerifyResult string `json:"VerifyResult"`
	} `json:"Model"`
}

func (a *AliyunClient) call(ctx context.Context, action string, params map[string]string, target interface{}) error {
	if a.accessKeyID == "" || a.accessKeySecret == "" {
		return errors.New("aliyun sms authentication is not configured")
	}
	common := map[string]string{
		"AccessKeyId": a.accessKeyID, "Action": action, "Format": "JSON", "SignatureMethod": "HMAC-SHA1",
		"SignatureNonce": nonce(), "SignatureVersion": "1.0", "Timestamp": time.Now().UTC().Format("2006-01-02T15:04:05Z"), "Version": "2017-05-25",
	}
	for key, value := range params {
		common[key] = value
	}
	canonical := canonicalQuery(common)
	stringToSign := "POST&%2F&" + percentEncode(canonical)
	mac := hmac.New(sha1.New, []byte(a.accessKeySecret+"&"))
	_, _ = mac.Write([]byte(stringToSign))
	common["Signature"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, strings.NewReader(url.Values(toValues(common)).Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := a.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var failure aliyunResponse
		if err := json.NewDecoder(response.Body).Decode(&failure); err == nil && failure.Code != "" {
			return fmt.Errorf("aliyun sms http status %d code %s request %s", response.StatusCode, failure.Code, failure.RequestID)
		}
		return fmt.Errorf("aliyun sms http status %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func canonicalQuery(values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, percentEncode(key)+"="+percentEncode(values[key]))
	}
	return strings.Join(parts, "&")
}

func percentEncode(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(url.QueryEscape(value), "+", "%20"), "*", "%2A"), "%7E", "~")
}

func toValues(values map[string]string) map[string][]string {
	result := make(map[string][]string, len(values))
	for key, value := range values {
		result[key] = []string{value}
	}
	return result
}

func nonce() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
