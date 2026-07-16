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
