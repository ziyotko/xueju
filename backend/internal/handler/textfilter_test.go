package handler

import (
	"testing"

	"xueju/backend/internal/textfilter"
)

func TestCleanEventRequest(t *testing.T) {
	filter, err := textfilter.New()
	if err != nil {
		t.Fatalf("textfilter.New() error = %v", err)
	}
	h := &AppHandler{textFilter: filter}
	req := &eventRequest{
		Title:       "赌博行程",
		ResortName:  "万龙",
		DepartCity:  "北京",
		DepartArea:  "诈骗区",
		MeetPlace:   "赌博网站门口",
		CostDesc:    "无",
		Remark:      "不要诈骗",
		PurposeTags: []string{"刻滑", "赌博"},
	}

	h.cleanEventRequest(req)

	if req.Title != "**行程" {
		t.Fatalf("Title = %q", req.Title)
	}
	if req.DepartArea != "**区" {
		t.Fatalf("DepartArea = %q", req.DepartArea)
	}
	if req.MeetPlace != "****门口" {
		t.Fatalf("MeetPlace = %q", req.MeetPlace)
	}
	if req.Remark != "不要**" {
		t.Fatalf("Remark = %q", req.Remark)
	}
	if len(req.PurposeTags) != 2 || req.PurposeTags[0] != "刻滑" || req.PurposeTags[1] != "**" {
		t.Fatalf("PurposeTags = %#v", req.PurposeTags)
	}
}

func TestCleanEventRequestReplacesPoliticalSensitiveWords(t *testing.T) {
	filter, err := textfilter.New()
	if err != nil {
		t.Fatalf("textfilter.New() error = %v", err)
	}
	h := &AppHandler{textFilter: filter}
	req := &eventRequest{Remark: "习近平"}

	h.cleanEventRequest(req)

	if req.Remark != "***" {
		t.Fatalf("Remark = %q", req.Remark)
	}
}

func TestCleanTextFallsBackWhenFilterMissing(t *testing.T) {
	h := &AppHandler{}
	got := h.cleanText("赌博")
	if got != "赌博" {
		t.Fatalf("cleanText() = %q, want fallback text unchanged", got)
	}
}
