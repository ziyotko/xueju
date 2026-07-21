package contentsecurity

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	green "github.com/alibabacloud-go/green-20220302/v3/client"
	"github.com/alibabacloud-go/tea/dara"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"xueju/backend/internal/compliance"
	"xueju/backend/internal/config"
)

type ImageCheckRequest struct {
	Kind      string
	Filename  string
	Data      []byte
	AccountID string
}

type ImageCheckResult struct {
	Status       compliance.ModerationStatus
	RequestID    string
	ReviewResult string
}

type aliyunGreenClient interface {
	TextModerationPlusWithContext(context.Context, *green.TextModerationPlusRequest, *dara.RuntimeOptions) (*green.TextModerationPlusResponse, error)
	ImageModerationWithContext(context.Context, *green.ImageModerationRequest, *dara.RuntimeOptions) (*green.ImageModerationResponse, error)
	DescribeUploadToken() (*green.DescribeUploadTokenResponse, error)
}

type aliyunService struct {
	cfg    config.Config
	client aliyunGreenClient

	tokenMu sync.Mutex
	token   *green.DescribeUploadTokenResponseBodyData
}

func newAliyunService(cfg config.Config) (*aliyunService, error) {
	if strings.TrimSpace(cfg.AliyunContentAccessKeyID) == "" || strings.TrimSpace(cfg.AliyunContentAccessKeySecret) == "" {
		return nil, errors.New("aliyun content security authentication is not configured")
	}
	if strings.TrimSpace(cfg.AliyunContentEndpoint) == "" {
		return nil, errors.New("aliyun content security endpoint is not configured")
	}
	sdkConfig := &openapi.Config{
		AccessKeyId:     tea.String(cfg.AliyunContentAccessKeyID),
		AccessKeySecret: tea.String(cfg.AliyunContentAccessKeySecret),
		Endpoint:        tea.String(cfg.AliyunContentEndpoint),
		ConnectTimeout:  tea.Int(5000),
		ReadTimeout:     tea.Int(65000),
	}
	client, err := green.NewClient(sdkConfig)
	if err != nil {
		return nil, fmt.Errorf("create aliyun content security client: %w", err)
	}
	return &aliyunService{cfg: cfg, client: client}, nil
}

func (s *aliyunService) checkText(ctx context.Context, req TextCheckRequest) error {
	parameters := map[string]string{"content": req.Content}
	if req.OpenID != "" {
		parameters["accountId"] = req.OpenID
	}
	payload, err := json.Marshal(parameters)
	if err != nil {
		return fmt.Errorf("encode aliyun text moderation request: %w", err)
	}
	response, err := s.client.TextModerationPlusWithContext(ctx, &green.TextModerationPlusRequest{
		Service:           tea.String(s.cfg.AliyunTextService),
		ServiceParameters: tea.String(string(payload)),
	}, &dara.RuntimeOptions{})
	if err != nil {
		return fmt.Errorf("aliyun text moderation request failed: %w", err)
	}
	if response == nil || response.StatusCode == nil || *response.StatusCode != 200 || response.Body == nil || response.Body.Code == nil || *response.Body.Code != 200 || response.Body.Data == nil {
		return fmt.Errorf("aliyun text moderation failed: %s", textResponseDetails(response))
	}
	level := textRiskLevel(response.Body.Data)
	if level == "high" || level == "medium" {
		return errors.New(compliance.ContentRiskMessage)
	}
	if level != "low" && level != "none" {
		return fmt.Errorf("aliyun text moderation returned an unknown risk level %q", pointerValue(response.Body.Data.RiskLevel))
	}
	return nil
}

func (s *Service) CheckImage(ctx context.Context, req ImageCheckRequest) (ImageCheckResult, error) {
	result := ImageCheckResult{Status: compliance.ModerationApproved}
	if !s.enabled() {
		return result, nil
	}
	if s.Provider() != "aliyun" {
		return result, fmt.Errorf("image sync moderation requires the aliyun provider")
	}
	if s.aliyunErr != nil {
		return result, s.aliyunErr
	}
	return s.aliyun.checkImage(ctx, req)
}

func (s *aliyunService) checkImage(ctx context.Context, req ImageCheckRequest) (ImageCheckResult, error) {
	token, err := s.uploadToken()
	if err != nil {
		return ImageCheckResult{}, err
	}
	objectName := pointerValue(token.FileNamePrefix) + randomDataID() + strings.ToLower(filepath.Ext(req.Filename))
	if err := uploadTemporaryImage(token, objectName, req.Data); err != nil {
		return ImageCheckResult{}, fmt.Errorf("upload image for aliyun moderation: %w", err)
	}
	parameters := map[string]string{
		"ossBucketName": pointerValue(token.BucketName),
		"ossObjectName": objectName,
		"dataId":        randomDataID(),
	}
	if req.AccountID != "" {
		parameters["accountId"] = req.AccountID
	}
	payload, err := json.Marshal(parameters)
	if err != nil {
		return ImageCheckResult{}, fmt.Errorf("encode aliyun image moderation request: %w", err)
	}
	serviceCode := s.cfg.AliyunEventImageService
	if req.Kind == "avatars" {
		serviceCode = s.cfg.AliyunAvatarImageService
	}
	response, err := s.client.ImageModerationWithContext(ctx, &green.ImageModerationRequest{
		Service:           tea.String(serviceCode),
		ServiceParameters: tea.String(string(payload)),
	}, &dara.RuntimeOptions{})
	if err != nil {
		return ImageCheckResult{}, fmt.Errorf("aliyun image moderation request failed: %w", err)
	}
	if response == nil || response.StatusCode == nil || *response.StatusCode != 200 || response.Body == nil || response.Body.Code == nil || *response.Body.Code != 200 || response.Body.Data == nil {
		return ImageCheckResult{}, fmt.Errorf("aliyun image moderation failed: %s", imageResponseDetails(response))
	}
	level := imageRiskLevel(response.Body.Data)
	result := ImageCheckResult{
		Status:       imageStatus(level),
		RequestID:    pointerValue(response.Body.RequestId),
		ReviewResult: imageReviewSummary(level, response.Body.Data.Result),
	}
	return result, nil
}

func (s *aliyunService) uploadToken() (*green.DescribeUploadTokenResponseBodyData, error) {
	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()
	if validUploadToken(s.token, time.Now()) {
		return s.token, nil
	}
	response, err := s.client.DescribeUploadToken()
	if err != nil {
		return nil, fmt.Errorf("get aliyun image upload token: %w", err)
	}
	if response == nil || response.StatusCode == nil || *response.StatusCode != 200 || response.Body == nil || response.Body.Code == nil || *response.Body.Code != 200 || !validUploadToken(response.Body.Data, time.Now()) {
		return nil, errors.New("aliyun image upload token response is invalid")
	}
	s.token = response.Body.Data
	return s.token, nil
}

func uploadTemporaryImage(token *green.DescribeUploadTokenResponseBodyData, objectName string, data []byte) error {
	client, err := oss.New(
		pointerValue(token.OssInternetEndPoint),
		pointerValue(token.AccessKeyId),
		pointerValue(token.AccessKeySecret),
		oss.SecurityToken(pointerValue(token.SecurityToken)),
	)
	if err != nil {
		return err
	}
	bucket, err := client.Bucket(pointerValue(token.BucketName))
	if err != nil {
		return err
	}
	return bucket.PutObject(objectName, bytes.NewReader(data))
}

func validUploadToken(token *green.DescribeUploadTokenResponseBodyData, now time.Time) bool {
	return token != nil && token.Expiration != nil && int64(*token.Expiration) > now.Unix()+60 &&
		pointerValue(token.AccessKeyId) != "" && pointerValue(token.AccessKeySecret) != "" &&
		pointerValue(token.SecurityToken) != "" && pointerValue(token.BucketName) != "" &&
		pointerValue(token.FileNamePrefix) != "" && pointerValue(token.OssInternetEndPoint) != ""
}

func textRiskLevel(data *green.TextModerationPlusResponseBodyData) string {
	if data == nil {
		return "unknown"
	}
	level := normalizeRiskLevel(pointerValue(data.RiskLevel))
	if level == "" && len(data.Result) > 0 {
		return "unknown"
	}
	return level
}

func imageRiskLevel(data *green.ImageModerationResponseBodyData) string {
	if data == nil {
		return "unknown"
	}
	level := normalizeRiskLevel(pointerValue(data.RiskLevel))
	for _, item := range data.Result {
		if item == nil {
			continue
		}
		candidate := normalizeRiskLevel(pointerValue(item.RiskLevel))
		if riskRank(candidate) > riskRank(level) {
			level = candidate
		}
	}
	if level == "" && len(data.Result) > 0 {
		return "unknown"
	}
	return level
}

func imageStatus(level string) compliance.ModerationStatus {
	switch level {
	case "high":
		return compliance.ModerationRejected
	case "medium", "unknown":
		return compliance.ModerationPending
	default:
		return compliance.ModerationApproved
	}
}

func imageReviewSummary(level string, results []*green.ImageModerationResponseBodyDataResult) string {
	labels := make([]string, 0, len(results))
	for _, item := range results {
		if item == nil || pointerValue(item.Label) == "" {
			continue
		}
		labels = append(labels, pointerValue(item.Label))
	}
	sort.Strings(labels)
	if len(labels) > 4 {
		labels = labels[:4]
	}
	summary := "aliyun:" + level
	if len(labels) > 0 {
		summary += ":" + strings.Join(labels, ",")
	}
	if len(summary) > 255 {
		summary = summary[:255]
	}
	return summary
}

func normalizeRiskLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "high", "medium", "low", "none":
		return strings.ToLower(strings.TrimSpace(level))
	default:
		return ""
	}
}

func riskRank(level string) int {
	switch level {
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	case "none":
		return 1
	default:
		return 0
	}
}

func randomDataID() string {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func textResponseDetails(response *green.TextModerationPlusResponse) string {
	if response == nil {
		return "empty response"
	}
	status := int32(0)
	if response.StatusCode != nil {
		status = *response.StatusCode
	}
	if response.Body == nil {
		return fmt.Sprintf("http_status=%d empty_body", status)
	}
	code := int32(0)
	if response.Body.Code != nil {
		code = *response.Body.Code
	}
	return fmt.Sprintf("http_status=%d code=%d message=%q request_id=%q", status, code, pointerValue(response.Body.Message), pointerValue(response.Body.RequestId))
}

func imageResponseDetails(response *green.ImageModerationResponse) string {
	if response == nil {
		return "empty response"
	}
	status := int32(0)
	if response.StatusCode != nil {
		status = *response.StatusCode
	}
	if response.Body == nil {
		return fmt.Sprintf("http_status=%d empty_body", status)
	}
	code := int32(0)
	if response.Body.Code != nil {
		code = *response.Body.Code
	}
	return fmt.Sprintf("http_status=%d code=%d message=%q request_id=%q", status, code, pointerValue(response.Body.Msg), pointerValue(response.Body.RequestId))
}
