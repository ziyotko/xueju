package contentsecurity

import (
	"testing"

	green "github.com/alibabacloud-go/green-20220302/v3/client"
	"github.com/alibabacloud-go/tea/tea"
	"xueju/backend/internal/compliance"
)

func TestTextRiskLevel(t *testing.T) {
	tests := []struct {
		name string
		data *green.TextModerationPlusResponseBodyData
		want string
	}{
		{name: "high", data: &green.TextModerationPlusResponseBodyData{RiskLevel: tea.String("high")}, want: "high"},
		{name: "low", data: &green.TextModerationPlusResponseBodyData{RiskLevel: tea.String("low")}, want: "low"},
		{name: "result without level is unknown", data: &green.TextModerationPlusResponseBodyData{Result: []*green.TextModerationPlusResponseBodyDataResult{{Label: tea.String("sexual_content")}}}, want: "unknown"},
		{name: "empty level is unknown to caller", data: &green.TextModerationPlusResponseBodyData{}, want: ""},
		{name: "nil data", want: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := textRiskLevel(test.data); got != test.want {
				t.Fatalf("textRiskLevel()=%q, want %q", got, test.want)
			}
		})
	}
}

func TestImageRiskLevelUsesHighestResult(t *testing.T) {
	data := &green.ImageModerationResponseBodyData{
		RiskLevel: tea.String("low"),
		Result: []*green.ImageModerationResponseBodyDataResult{
			{Label: tea.String("normal"), RiskLevel: tea.String("low")},
			{Label: tea.String("sexual_content"), RiskLevel: tea.String("high")},
		},
	}
	if got := imageRiskLevel(data); got != "high" {
		t.Fatalf("imageRiskLevel()=%q, want high", got)
	}
	if got := imageStatus(imageRiskLevel(data)); got != compliance.ModerationRejected {
		t.Fatalf("imageStatus()=%q, want rejected", got)
	}
}

func TestImageStatusFailsClosedForUnknown(t *testing.T) {
	if got := imageStatus("unknown"); got != compliance.ModerationPending {
		t.Fatalf("imageStatus()=%q, want pending", got)
	}
	if got := imageStatus("medium"); got != compliance.ModerationPending {
		t.Fatalf("imageStatus()=%q, want pending", got)
	}
	if got := imageStatus("low"); got != compliance.ModerationApproved {
		t.Fatalf("imageStatus()=%q, want approved", got)
	}
}

func TestImageReviewSummaryIsCompact(t *testing.T) {
	results := []*green.ImageModerationResponseBodyDataResult{
		{Label: tea.String("z")}, {Label: tea.String("a")}, {Label: tea.String("b")},
		{Label: tea.String("c")}, {Label: tea.String("d")},
	}
	if got := imageReviewSummary("medium", results); got != "aliyun:medium:a,b,c,d" {
		t.Fatalf("unexpected summary %q", got)
	}
}
