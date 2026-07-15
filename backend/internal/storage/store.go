package storage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"xueju/backend/internal/config"
)

type Store interface {
	Put(context.Context, string, []byte, string) error
	Get(context.Context, string) ([]byte, string, error)
	Delete(context.Context, string) error
}

func New(cfg config.Config) Store {
	if cfg.StorageDriver == "s3" {
		return &s3Store{endpoint: cfg.S3Endpoint, bucket: cfg.S3Bucket, region: cfg.S3Region, accessKey: cfg.S3AccessKey, secretKey: cfg.S3SecretKey, client: &http.Client{Timeout: 30 * time.Second}}
	}
	return &localStore{root: cfg.UploadDir}
}

type localStore struct{ root string }

func (s *localStore) path(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || clean == ".." {
		return "", errors.New("invalid storage key")
	}
	return filepath.Join(s.root, clean), nil
}

func (s *localStore) Put(_ context.Context, key string, data []byte, _ string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (s *localStore) Get(_ context.Context, key string) ([]byte, string, error) {
	path, err := s.path(key)
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	return data, http.DetectContentType(data), nil
}

func (s *localStore) Delete(_ context.Context, key string) error {
	path, err := s.path(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

type s3Store struct {
	endpoint, bucket, region, accessKey, secretKey string
	client                                         *http.Client
}

func (s *s3Store) objectURL(key string) (string, error) {
	if s.endpoint == "" || s.bucket == "" {
		return "", errors.New("incomplete s3 configuration")
	}
	segments := strings.Split(strings.TrimLeft(filepath.ToSlash(key), "/"), "/")
	for index, segment := range segments {
		segments[index] = url.PathEscape(segment)
	}
	return s.endpoint + "/" + url.PathEscape(s.bucket) + "/" + strings.Join(segments, "/"), nil
}

func (s *s3Store) Put(ctx context.Context, key string, data []byte, mime string) error {
	request, err := s.request(ctx, http.MethodPut, key, data)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", mime)
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return fmt.Errorf("s3 put failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}
	return nil
}

func (s *s3Store) Get(ctx context.Context, key string) ([]byte, string, error) {
	request, err := s.request(ctx, http.MethodGet, key, nil)
	if err != nil {
		return nil, "", err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("s3 get failed: %s", response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 6*1024*1024))
	return data, response.Header.Get("Content-Type"), err
}

func (s *s3Store) Delete(ctx context.Context, key string) error {
	request, err := s.request(ctx, http.MethodDelete, key, nil)
	if err != nil {
		return err
	}
	response, err := s.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("s3 delete failed: %s", response.Status)
	}
	return nil
}

func (s *s3Store) request(ctx context.Context, method, key string, data []byte) (*http.Request, error) {
	objectURL, err := s.objectURL(key)
	if err != nil {
		return nil, err
	}
	payloadHash := sha256Hex(data)
	request, err := http.NewRequestWithContext(ctx, method, objectURL, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	amzDate, dateStamp := now.Format("20060102T150405Z"), now.Format("20060102")
	request.Header.Set("x-amz-date", amzDate)
	request.Header.Set("x-amz-content-sha256", payloadHash)
	parsed, _ := url.Parse(objectURL)
	canonicalHeaders := "host:" + parsed.Host + "\n" + "x-amz-content-sha256:" + payloadHash + "\n" + "x-amz-date:" + amzDate + "\n"
	signedHeaders := "host;x-amz-content-sha256;x-amz-date"
	canonicalRequest := method + "\n" + parsed.EscapedPath() + "\n\n" + canonicalHeaders + "\n" + signedHeaders + "\n" + payloadHash
	scope := dateStamp + "/" + s.region + "/s3/aws4_request"
	stringToSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256Hex([]byte(canonicalRequest))
	signature := hex.EncodeToString(hmacSHA256(signingKey(s.secretKey, dateStamp, s.region), []byte(stringToSign)))
	request.Header.Set("Authorization", "AWS4-HMAC-SHA256 Credential="+s.accessKey+"/"+scope+", SignedHeaders="+signedHeaders+", Signature="+signature)
	return request, nil
}

func sha256Hex(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func hmacSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}
func signingKey(secret, date, region string) []byte {
	dateKey := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	regionKey := hmacSHA256(dateKey, []byte(region))
	serviceKey := hmacSHA256(regionKey, []byte("s3"))
	return hmacSHA256(serviceKey, []byte("aws4_request"))
}
