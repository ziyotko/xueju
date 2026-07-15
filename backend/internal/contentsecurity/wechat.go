package contentsecurity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"xueju/backend/internal/compliance"
	"xueju/backend/internal/config"
)

const (
	wechatTokenURL       = "https://api.weixin.qq.com/cgi-bin/token"
	wechatMsgSecCheckURL = "https://api.weixin.qq.com/wxa/msg_sec_check"
	wechatMediaCheckURL  = "https://api.weixin.qq.com/wxa/media_check_async"
)

type Service struct {
	cfg        config.Config
	httpClient *http.Client
	mu         sync.Mutex
	token      string
	expiresAt  time.Time
}

type TextCheckRequest struct {
	OpenID  string
	Scene   int
	Field   compliance.TextField
	Content string
}

type MediaCheckRequest struct {
	OpenID    string
	Scene     int
	MediaURL  string
	MediaType int
}

func New(cfg config.Config) *Service {
	return &Service{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (s *Service) CheckText(ctx context.Context, req TextCheckRequest) error {
	if req.Content == "" || !s.enabled() {
		return nil
	}

	accessToken, err := s.accessToken(ctx)
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"version": 2,
		"openid":  req.OpenID,
		"scene":   req.Scene,
		"content": req.Content,
	}

	var result secCheckResponse
	if err := s.postJSON(ctx, wechatMsgSecCheckURL, accessToken, payload, &result); err != nil {
		return err
	}
	return result.toError()
}

func (s *Service) CheckMediaAsync(ctx context.Context, req MediaCheckRequest) (string, error) {
	if req.MediaURL == "" || !s.enabled() {
		return "", nil
	}

	accessToken, err := s.accessToken(ctx)
	if err != nil {
		return "", err
	}

	payload := map[string]interface{}{
		"version":    2,
		"openid":     req.OpenID,
		"scene":      req.Scene,
		"media_url":  req.MediaURL,
		"media_type": req.MediaType,
	}

	var result secCheckResponse
	if err := s.postJSON(ctx, wechatMediaCheckURL, accessToken, payload, &result); err != nil {
		return "", err
	}
	if err := result.toError(); err != nil {
		return "", err
	}
	return result.TraceID, nil
}

func (s *Service) enabled() bool {
	return s.cfg.ContentSecurity && s.cfg.WechatAppID != "" && s.cfg.WechatAppSecret != ""
}

func (s *Service) accessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	if s.token != "" && time.Now().Before(s.expiresAt) {
		token := s.token
		s.mu.Unlock()
		return token, nil
	}
	s.mu.Unlock()

	params := url.Values{}
	params.Set("grant_type", "client_credential")
	params.Set("appid", s.cfg.WechatAppID)
	params.Set("secret", s.cfg.WechatAppSecret)

	requestURL := wechatTokenURL + "?" + params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return "", err
	}

	response, err := s.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	var body tokenResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.ErrCode != 0 {
		return "", fmt.Errorf("wechat token error %d: %s", body.ErrCode, body.ErrMsg)
	}

	s.mu.Lock()
	s.token = body.AccessToken
	s.expiresAt = time.Now().Add(time.Duration(body.ExpiresIn-300) * time.Second)
	s.mu.Unlock()

	return body.AccessToken, nil
}

func (s *Service) postJSON(ctx context.Context, endpoint string, token string, payload interface{}, target interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	requestURL := endpoint + "?access_token=" + url.QueryEscape(token)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := s.httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return json.NewDecoder(response.Body).Decode(target)
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type secCheckResponse struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	TraceID string `json:"trace_id"`
	Result  struct {
		Suggest string `json:"suggest"`
		Label   int    `json:"label"`
	} `json:"result"`
}

func (r secCheckResponse) toError() error {
	if r.ErrCode != 0 {
		return fmt.Errorf("wechat content security error %d: %s", r.ErrCode, r.ErrMsg)
	}
	if r.Result.Suggest == "risky" {
		return errors.New(compliance.ContentRiskMessage)
	}
	return nil
}
