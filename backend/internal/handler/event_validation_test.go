package handler

import (
	"strings"
	"testing"
	"time"
)

func TestValidateEventRequestGroupLimit(t *testing.T) {
	request := eventRequest{
		ResortName:  "test resort",
		EventDate:   time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
		DepartCity:  "Beijing",
		DepartArea:  "Chaoyang",
		MeetPlace:   "Station",
		MaxMembers:  20,
		SkiTypeReq:  "both",
		LevelReq:    "intermediate",
		TrafficType: "other",
	}
	if message := validateEventRequest(request, 1); message != "" {
		t.Fatalf("20-member group should be valid: %s", message)
	}

	request.MaxMembers = 21
	if message := validateEventRequest(request, 1); !strings.Contains(message, "between 1 and 20") {
		t.Fatalf("21-member group should be rejected, got: %s", message)
	}
}

func TestValidateEventRequestRejectsOversizedText(t *testing.T) {
	request := eventRequest{
		Title:       strings.Repeat("雪", 129),
		ResortName:  "test resort",
		EventDate:   time.Now().AddDate(0, 0, 1).Format("2006-01-02"),
		DepartCity:  "Beijing",
		DepartArea:  "Chaoyang",
		MeetPlace:   "Station",
		MaxMembers:  4,
		SkiTypeReq:  "both",
		LevelReq:    "intermediate",
		TrafficType: "other",
		PurposeTags: []string{"刷道"},
	}
	if message := validateEventRequest(request, 1); message != "event field is too long" {
		t.Fatalf("oversized event title should be rejected, got: %s", message)
	}
}

func TestTextListWithinLimitsItemsAndRunes(t *testing.T) {
	if !textListWithin([]string{"刻滑", "公园"}, 2, 2) {
		t.Fatal("valid tag list was rejected")
	}
	if textListWithin([]string{"刻滑", "公园", "休闲"}, 2, 2) {
		t.Fatal("too many tags were accepted")
	}
	if textListWithin([]string{"新手友好"}, 2, 2) {
		t.Fatal("oversized tag was accepted")
	}
}

func TestValidateEventScheduleAcceptsFutureDateWithMorningTime(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 17, 11, 25, 0, 0, location)
	if message := validateEventSchedule("2026-07-20", "2026-07-20 07:20:00", now, location); message != "" {
		t.Fatalf("future event was rejected: %s", message)
	}
}

func TestValidateEventScheduleRejectsPastTimeOnSameDay(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	now := time.Date(2026, 7, 17, 11, 25, 0, 0, location)
	if message := validateEventSchedule("2026-07-17", "2026-07-17 07:20:00", now, location); message != "集合时间已过，请选择未来的日期或时间" {
		t.Fatalf("unexpected validation message: %s", message)
	}
}
