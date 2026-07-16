package phoneverification

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strings"
)

var mainlandPhone = regexp.MustCompile(`^1[3-9]\d{9}$`)

func Normalize(phone string) (string, error) {
	phone = strings.NewReplacer(" ", "", "-", "", "+86", "").Replace(strings.TrimSpace(phone))
	if !mainlandPhone.MatchString(phone) {
		return "", errors.New("invalid mainland China mobile number")
	}
	return phone, nil
}

func Mask(phone string) string {
	if len(phone) != 11 {
		return ""
	}
	return phone[:3] + "****" + phone[7:]
}

func Hash(phone, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(phone))
	return hex.EncodeToString(mac.Sum(nil))
}

func Encrypt(phone, secret string) (string, error) {
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(phone), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}
