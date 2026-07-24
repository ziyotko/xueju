package contentsecurity

import (
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"sort"
	"strings"
)

type MediaCallback struct {
	AppID      string `json:"appid" xml:"appid"`
	TraceID    string `json:"trace_id" xml:"trace_id"`
	MsgType    string `json:"MsgType" xml:"MsgType"`
	Event      string `json:"Event" xml:"Event"`
	ErrCode    int    `json:"errcode" xml:"errcode"`
	StatusCode int    `json:"status_code" xml:"status_code"`
	IsRisky    *int   `json:"isrisky" xml:"isrisky"`
	Suggest    string `json:"suggest" xml:"suggest"`
	Result     struct {
		Suggest string `json:"suggest" xml:"suggest"`
	} `json:"result" xml:"result"`
	Detail MediaCallbackDetails `json:"detail" xml:"detail"`
}

type MediaCallbackDetail struct {
	ErrCode int    `json:"errcode" xml:"errcode"`
	Suggest string `json:"suggest" xml:"suggest"`
}

type MediaCallbackDetails []MediaCallbackDetail

func (details *MediaCallbackDetails) UnmarshalJSON(data []byte) error {
	var list []MediaCallbackDetail
	if err := json.Unmarshal(data, &list); err == nil {
		*details = list
		return nil
	}
	var single MediaCallbackDetail
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*details = []MediaCallbackDetail{single}
	return nil
}

func VerifyCallbackSignature(token, timestamp, nonce, signature string) bool {
	if token == "" || timestamp == "" || nonce == "" || signature == "" {
		return false
	}
	values := []string{token, timestamp, nonce}
	sort.Strings(values)
	sum := sha1.Sum([]byte(strings.Join(values, "")))
	expected := hex.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(signature))) == 1
}

func ParseMediaCallback(body []byte) (MediaCallback, error) {
	var payload MediaCallback
	if len(body) == 0 {
		return payload, errors.New("empty media callback")
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.TraceID != "" {
		return payload, nil
	}
	payload = MediaCallback{}
	if err := xml.Unmarshal(body, &payload); err != nil {
		return MediaCallback{}, err
	}
	if payload.TraceID == "" {
		return MediaCallback{}, errors.New("media callback trace_id is required")
	}
	return payload, nil
}

func (c MediaCallback) ModerationSuggestion() string {
	suggestions := make([]string, 0, len(c.Detail)+2)
	appendSuggestion := func(raw string) {
		if strings.TrimSpace(raw) == "" {
			return
		}
		value := normalizeSuggestion(raw)
		if value == "" {
			value = "review"
		}
		suggestions = append(suggestions, value)
	}
	appendSuggestion(c.Suggest)
	appendSuggestion(c.Result.Suggest)
	if c.ErrCode != 0 || c.StatusCode != 0 {
		suggestions = append(suggestions, "review")
	}
	if c.IsRisky != nil {
		if *c.IsRisky == 0 {
			suggestions = append(suggestions, "pass")
		} else {
			suggestions = append(suggestions, "risky")
		}
	}
	for _, detail := range c.Detail {
		if detail.ErrCode != 0 {
			suggestions = append(suggestions, "review")
			continue
		}
		appendSuggestion(detail.Suggest)
	}
	if len(suggestions) == 0 {
		return "review"
	}
	for _, suggestion := range suggestions {
		if suggestion == "risky" {
			return "risky"
		}
	}
	for _, suggestion := range suggestions {
		if suggestion == "review" {
			return "review"
		}
	}
	return "pass"
}

func normalizeSuggestion(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pass", "risky", "review":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}
