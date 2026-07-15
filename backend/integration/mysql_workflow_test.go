package integration

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"xueju/backend/internal/config"
	"xueju/backend/internal/database"
	"xueju/backend/internal/router"
)

func TestMySQLJoinWorkflowAndRemovedVisibility(t *testing.T) {
	dsn := os.Getenv("XUEJU_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set XUEJU_TEST_MYSQL_DSN to run MySQL integration tests")
	}
	db, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}

	suffix := time.Now().UnixNano()
	creatorID := insertUser(t, db, fmt.Sprintf("integration-creator-%d", suffix))
	applicantID := insertUser(t, db, fmt.Sprintf("integration-applicant-%d", suffix))
	eventID := insertEvent(t, db, creatorID)
	defer cleanup(t, db, eventID, creatorID, applicantID)

	cfg := config.Config{AppEnv: "development", JWTSecret: "integration-test-secret", JWTExpiresHours: 1, UploadDir: t.TempDir()}
	engine := router.New(cfg, db)
	applicantToken := userToken(t, cfg.JWTSecret, applicantID)
	creatorToken := userToken(t, cfg.JWTSecret, creatorID)

	status, body := call(t, engine, http.MethodPost, fmt.Sprintf("/api/events/%d/apply", eventID), applicantToken, map[string]interface{}{"message": "申请加入", "skiLevel": "intermediate", "skiType": "snowboard"})
	if status != http.StatusOK {
		t.Fatalf("first apply failed: %d %s", status, body)
	}
	var requestID int64
	if err := db.QueryRow(`SELECT id FROM join_requests WHERE event_id=? AND applicant_id=?`, eventID, applicantID).Scan(&requestID); err != nil {
		t.Fatal(err)
	}

	status, _ = call(t, engine, http.MethodPost, fmt.Sprintf("/api/events/%d/apply", eventID), applicantToken, map[string]interface{}{"message": "重复申请"})
	if status != http.StatusBadRequest {
		t.Fatalf("duplicate apply status=%d", status)
	}
	status, body = call(t, engine, http.MethodPost, fmt.Sprintf("/api/join-requests/%d/reject", requestID), creatorToken, map[string]interface{}{"reason": "本次不合适"})
	if status != http.StatusOK {
		t.Fatalf("reject failed: %d %s", status, body)
	}
	status, body = call(t, engine, http.MethodPost, fmt.Sprintf("/api/events/%d/apply", eventID), applicantToken, map[string]interface{}{"message": "重新申请"})
	if status != http.StatusOK {
		t.Fatalf("reapply failed: %d %s", status, body)
	}
	status, body = call(t, engine, http.MethodPost, fmt.Sprintf("/api/join-requests/%d/approve", requestID), creatorToken, map[string]interface{}{})
	if status != http.StatusOK {
		t.Fatalf("approve failed: %d %s", status, body)
	}
	var currentMembers int
	var eventStatus string
	if err := db.QueryRow(`SELECT current_members, status FROM ski_events WHERE id=?`, eventID).Scan(&currentMembers, &eventStatus); err != nil {
		t.Fatal(err)
	}
	if currentMembers != 2 || eventStatus != "full" {
		t.Fatalf("unexpected event state members=%d status=%s", currentMembers, eventStatus)
	}
	status, body = call(t, engine, http.MethodDelete, fmt.Sprintf("/api/events/%d/members/%d", eventID, applicantID), creatorToken, nil)
	if status != http.StatusOK {
		t.Fatalf("remove member failed: %d %s", status, body)
	}
	status, _ = call(t, engine, http.MethodGet, fmt.Sprintf("/api/events/%d/messages", eventID), applicantToken, nil)
	if status != http.StatusForbidden {
		t.Fatalf("removed member retained chat access: status=%d", status)
	}
	status, _ = call(t, engine, http.MethodPost, fmt.Sprintf("/api/events/%d/apply", eventID), applicantToken, map[string]interface{}{"message": "被移出后再次申请"})
	if status != http.StatusBadRequest {
		t.Fatalf("removed approved member reapplied: status=%d", status)
	}
	if err := db.QueryRow(`SELECT current_members, status FROM ski_events WHERE id=?`, eventID).Scan(&currentMembers, &eventStatus); err != nil {
		t.Fatal(err)
	}
	if currentMembers != 1 || eventStatus != "recruiting" {
		t.Fatalf("unexpected state after member removal members=%d status=%s", currentMembers, eventStatus)
	}

	if _, err := db.Exec(`UPDATE ski_events SET status='removed' WHERE id=?`, eventID); err != nil {
		t.Fatal(err)
	}
	status, _ = call(t, engine, http.MethodGet, fmt.Sprintf("/api/events/%d", eventID), "", nil)
	if status != http.StatusNotFound {
		t.Fatalf("removed event remained public: status=%d", status)
	}

	expiredEventID := insertEvent(t, db, creatorID)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM event_members WHERE event_id=?`, expiredEventID)
		_, _ = db.Exec(`DELETE FROM ski_events WHERE id=?`, expiredEventID)
	})
	if _, err := db.Exec(`UPDATE ski_events SET event_date=DATE_SUB(CURDATE(), INTERVAL 1 DAY), start_time=DATE_SUB(NOW(), INTERVAL 1 DAY) WHERE id=?`, expiredEventID); err != nil {
		t.Fatal(err)
	}
	status, body = call(t, engine, http.MethodGet, "/api/events?page=1&pageSize=50", "", nil)
	if status != http.StatusOK {
		t.Fatalf("event discovery failed: %d %s", status, body)
	}
	if err := db.QueryRow(`SELECT status FROM ski_events WHERE id=?`, expiredEventID).Scan(&eventStatus); err != nil {
		t.Fatal(err)
	}
	if eventStatus != "finished" {
		t.Fatalf("expired event status=%s", eventStatus)
	}

	status, body = call(t, engine, http.MethodPost, "/api/reports", applicantToken, map[string]interface{}{"targetType": "app", "targetId": 0, "reason": "产品建议", "content": "希望增加更多雪场"})
	if status != http.StatusOK {
		t.Fatalf("app feedback failed: %d %s", status, body)
	}
	status, body = call(t, engine, http.MethodDelete, "/api/user/me", applicantToken, nil)
	if status != http.StatusOK {
		t.Fatalf("account deletion failed: %d %s", status, body)
	}
	var accountStatus string
	var deletedAt sql.NullTime
	if err := db.QueryRow(`SELECT status, deleted_at FROM users WHERE id=?`, applicantID).Scan(&accountStatus, &deletedAt); err != nil {
		t.Fatal(err)
	}
	if accountStatus != "disabled" || !deletedAt.Valid {
		t.Fatalf("unexpected deleted account state status=%s deleted=%v", accountStatus, deletedAt.Valid)
	}
}

func TestConcurrentApprovalsDoNotOverbook(t *testing.T) {
	dsn := os.Getenv("XUEJU_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set XUEJU_TEST_MYSQL_DSN to run MySQL integration tests")
	}
	db, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(db); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().UnixNano()
	creatorID := insertUser(t, db, fmt.Sprintf("concurrent-creator-%d", suffix))
	firstApplicantID := insertUser(t, db, fmt.Sprintf("concurrent-first-%d", suffix))
	secondApplicantID := insertUser(t, db, fmt.Sprintf("concurrent-second-%d", suffix))
	eventID := insertEvent(t, db, creatorID)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM join_request_history WHERE event_id=?`, eventID)
		_, _ = db.Exec(`DELETE FROM join_requests WHERE event_id=?`, eventID)
		_, _ = db.Exec(`DELETE FROM event_members WHERE event_id=?`, eventID)
		_, _ = db.Exec(`DELETE FROM ski_events WHERE id=?`, eventID)
		_, _ = db.Exec(`DELETE FROM notifications WHERE user_id IN (?,?,?)`, creatorID, firstApplicantID, secondApplicantID)
		_, _ = db.Exec(`DELETE FROM users WHERE id IN (?,?,?)`, creatorID, firstApplicantID, secondApplicantID)
	})

	cfg := config.Config{AppEnv: "development", JWTSecret: "integration-test-secret", JWTExpiresHours: 1, UploadDir: t.TempDir()}
	engine := router.New(cfg, db)
	creatorToken := userToken(t, cfg.JWTSecret, creatorID)
	applicants := []struct {
		id    int64
		token string
	}{
		{firstApplicantID, userToken(t, cfg.JWTSecret, firstApplicantID)},
		{secondApplicantID, userToken(t, cfg.JWTSecret, secondApplicantID)},
	}
	requestIDs := make([]int64, 0, len(applicants))
	for _, applicant := range applicants {
		status, body := call(t, engine, http.MethodPost, fmt.Sprintf("/api/events/%d/apply", eventID), applicant.token, map[string]interface{}{"message": "并发名额申请"})
		if status != http.StatusOK {
			t.Fatalf("apply failed: %d %s", status, body)
		}
		var requestID int64
		if err := db.QueryRow(`SELECT id FROM join_requests WHERE event_id=? AND applicant_id=?`, eventID, applicant.id).Scan(&requestID); err != nil {
			t.Fatal(err)
		}
		requestIDs = append(requestIDs, requestID)
	}

	start := make(chan struct{})
	statuses := make(chan int, len(requestIDs))
	var wait sync.WaitGroup
	for _, requestID := range requestIDs {
		wait.Add(1)
		go func(id int64) {
			defer wait.Done()
			<-start
			status, _ := call(t, engine, http.MethodPost, fmt.Sprintf("/api/join-requests/%d/approve", id), creatorToken, nil)
			statuses <- status
		}(requestID)
	}
	close(start)
	wait.Wait()
	close(statuses)
	successes, rejected := 0, 0
	for status := range statuses {
		if status == http.StatusOK {
			successes++
		} else if status == http.StatusBadRequest {
			rejected++
		}
	}
	if successes != 1 || rejected != 1 {
		t.Fatalf("unexpected concurrent results success=%d rejected=%d", successes, rejected)
	}
	var currentMembers int
	var eventStatus string
	if err := db.QueryRow(`SELECT current_members, status FROM ski_events WHERE id=?`, eventID).Scan(&currentMembers, &eventStatus); err != nil {
		t.Fatal(err)
	}
	if currentMembers != 2 || eventStatus != "full" {
		t.Fatalf("event overbooked or inconsistent members=%d status=%s", currentMembers, eventStatus)
	}
}

func insertUser(t *testing.T, db *sql.DB, openid string) int64 {
	result, err := db.Exec(`INSERT INTO users (openid, nickname) VALUES (?, '集成测试用户')`, openid)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	return id
}

func insertEvent(t *testing.T, db *sql.DB, creatorID int64) int64 {
	result, err := db.Exec(`INSERT INTO ski_events (title, creator_id, resort_name, event_date, max_members, current_members, status) VALUES ('集成测试雪局', ?, '测试雪场', CURDATE(), 2, 1, 'recruiting')`, creatorID)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	if _, err := db.Exec(`INSERT INTO event_members (event_id, user_id, role, status) VALUES (?, ?, 'creator', 'active')`, id, creatorID); err != nil {
		t.Fatal(err)
	}
	return id
}

func userToken(t *testing.T, secret string, userID int64) string {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": userID, "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func call(t *testing.T, engine http.Handler, method, path, token string, payload interface{}) (int, string) {
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, &body)
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, request)
	return recorder.Code, recorder.Body.String()
}

func cleanup(t *testing.T, db *sql.DB, eventID, creatorID, applicantID int64) {
	_, _ = db.Exec(`DELETE FROM join_request_history WHERE event_id=?`, eventID)
	_, _ = db.Exec(`DELETE FROM join_requests WHERE event_id=?`, eventID)
	_, _ = db.Exec(`DELETE FROM event_members WHERE event_id=?`, eventID)
	_, _ = db.Exec(`DELETE FROM ski_events WHERE id=?`, eventID)
	_, _ = db.Exec(`DELETE FROM notifications WHERE user_id IN (?,?)`, creatorID, applicantID)
	_, _ = db.Exec(`DELETE FROM reports WHERE reporter_id IN (?,?)`, creatorID, applicantID)
	_, _ = db.Exec(`DELETE FROM users WHERE id IN (?,?)`, creatorID, applicantID)
}
