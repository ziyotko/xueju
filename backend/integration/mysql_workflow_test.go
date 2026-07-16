package integration

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestAdminModerationCenterWorkflow(t *testing.T) {
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
	cleanupStaleModerationFixtures(t, db)

	suffix := time.Now().UnixNano()
	creatorID := insertUser(t, db, fmt.Sprintf("moderation-creator-%d", suffix))
	applicantID := insertUser(t, db, fmt.Sprintf("moderation-applicant-%d", suffix))
	eventID := insertEvent(t, db, creatorID)
	messageMarker := fmt.Sprintf("moderation-message-%d", suffix)
	applicationMarker := fmt.Sprintf("moderation-application-%d", suffix)

	requestResult, err := db.Exec(`INSERT INTO join_requests (event_id, applicant_id, creator_id, message, status) VALUES (?, ?, ?, ?, 'pending')`, eventID, applicantID, creatorID, applicationMarker)
	if err != nil {
		t.Fatal(err)
	}
	requestID, _ := requestResult.LastInsertId()
	messageResult, err := db.Exec(`INSERT INTO chat_messages (event_id, sender_id, message_type, content, status) VALUES (?, ?, 'text', ?, 'normal')`, eventID, creatorID, messageMarker)
	if err != nil {
		t.Fatal(err)
	}
	messageID, _ := messageResult.LastInsertId()
	reviewResult, err := db.Exec(`INSERT INTO reviews (event_id, reviewer_id, reviewee_id, score, content, status) VALUES (?, ?, ?, 5, ?, 'normal')`, eventID, creatorID, applicantID, "moderation review")
	if err != nil {
		t.Fatal(err)
	}
	reviewID, _ := reviewResult.LastInsertId()
	reportResult, err := db.Exec(`INSERT INTO reports (reporter_id, target_type, target_id, reason, content) VALUES (?, 'message', ?, 'test', 'moderation report')`, applicantID, messageID)
	if err != nil {
		t.Fatal(err)
	}
	reportID, _ := reportResult.LastInsertId()

	uploadRoot := t.TempDir()
	storageKey := filepath.ToSlash(filepath.Join("avatars", fmt.Sprintf("moderation-%d.png", suffix)))
	filePath := filepath.Join(uploadRoot, filepath.FromSlash(storageKey))
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatal(err)
	}
	pngBytes, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNk+A8AAQUBAScY42YAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, pngBytes, 0644); err != nil {
		t.Fatal(err)
	}
	uploadResult, err := db.Exec(`INSERT INTO media_uploads (user_id, kind, path, public_url, mime_type, status) VALUES (?, 'avatars', ?, '/test.png', 'image/png', 'pending')`, creatorID, storageKey)
	if err != nil {
		t.Fatal(err)
	}
	uploadID, _ := uploadResult.LastInsertId()

	defer func() {
		_, _ = db.Exec(`DELETE FROM admin_action_logs WHERE (resource='messages' AND target_id=?) OR (resource='events' AND target_id=?) OR (resource='uploads' AND target_id=?)`, messageID, eventID, uploadID)
		_, _ = db.Exec(`DELETE FROM media_uploads WHERE id=?`, uploadID)
		_, _ = db.Exec(`DELETE FROM reports WHERE id=?`, reportID)
		_, _ = db.Exec(`DELETE FROM reviews WHERE id=?`, reviewID)
		_, _ = db.Exec(`DELETE FROM chat_messages WHERE id=?`, messageID)
		_, _ = db.Exec(`DELETE FROM join_requests WHERE id=?`, requestID)
		_, _ = db.Exec(`DELETE FROM event_members WHERE event_id=?`, eventID)
		_, _ = db.Exec(`DELETE FROM ski_events WHERE id=?`, eventID)
		_, _ = db.Exec(`DELETE FROM notifications WHERE user_id IN (?,?)`, creatorID, applicantID)
		_, _ = db.Exec(`DELETE FROM users WHERE id IN (?,?)`, creatorID, applicantID)
	}()

	cfg := config.Config{AppEnv: "development", JWTSecret: "integration-test-secret", JWTExpiresHours: 1, UploadDir: uploadRoot}
	engine := router.New(cfg, db)
	token := adminToken(t, cfg.JWTSecret)

	assertStatus := func(method, path string, payload interface{}, expected int) string {
		t.Helper()
		status, body := call(t, engine, method, path, token, payload)
		if status != expected {
			t.Fatalf("%s %s status=%d body=%s", method, path, status, body)
		}
		return body
	}

	assertStatus(http.MethodGet, "/api/admin/moderation/summary", nil, http.StatusOK)
	contentBody := assertStatus(http.MethodGet, "/api/admin/content-reviews?scope=content&keyword="+messageMarker, nil, http.StatusOK)
	if strings.Contains(contentBody, messageMarker) {
		t.Fatal("scope=content unexpectedly included chat messages")
	}
	legacyBody := assertStatus(http.MethodGet, "/api/admin/content-reviews?keyword="+messageMarker, nil, http.StatusOK)
	if !strings.Contains(legacyBody, messageMarker) {
		t.Fatal("legacy content review no longer includes chat messages")
	}
	applicationBody := assertStatus(http.MethodGet, "/api/admin/content-reviews?scope=content&itemType=application&keyword="+applicationMarker, nil, http.StatusOK)
	if !strings.Contains(applicationBody, applicationMarker) {
		t.Fatal("content scope did not include the matching application")
	}
	assertStatus(http.MethodGet, "/api/admin/messages?keyword="+messageMarker, nil, http.StatusOK)
	assertStatus(http.MethodGet, "/api/admin/uploads", nil, http.StatusOK)

	status, _ := call(t, engine, http.MethodGet, fmt.Sprintf("/api/admin/uploads/%d/preview", uploadID), "", nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("unauthenticated preview status=%d", status)
	}
	assertStatus(http.MethodGet, fmt.Sprintf("/api/admin/uploads/%d/preview", uploadID), nil, http.StatusOK)

	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/messages/%d/actions", messageID), map[string]interface{}{"action": "unknown_action", "reason": "integration test"}, http.StatusBadRequest)
	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/messages/%d/actions", messageID), map[string]interface{}{"action": "hide_message", "status": "hidden"}, http.StatusBadRequest)
	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/messages/%d/actions", messageID), map[string]interface{}{"action": "hide_message", "status": "normal", "reason": "integration test"}, http.StatusOK)
	var messageStatus string
	if err := db.QueryRow(`SELECT status FROM chat_messages WHERE id=?`, messageID).Scan(&messageStatus); err != nil || messageStatus != "hidden" {
		t.Fatalf("hide action must force hidden status, status=%q err=%v", messageStatus, err)
	}
	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/messages/%d/actions", messageID), map[string]interface{}{"action": "hide_message", "status": "hidden", "reason": "repeat integration test"}, http.StatusOK)
	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/messages/%d/actions", messageID), map[string]interface{}{"action": "restore_message", "status": "hidden"}, http.StatusOK)
	if err := db.QueryRow(`SELECT status FROM chat_messages WHERE id=?`, messageID).Scan(&messageStatus); err != nil || messageStatus != "normal" {
		t.Fatalf("restore action must force normal status, status=%q err=%v", messageStatus, err)
	}

	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/events/%d/actions", eventID), map[string]interface{}{"action": "delist_event", "status": "removed"}, http.StatusBadRequest)
	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/events/%d/actions", eventID), map[string]interface{}{"action": "delist_event", "status": "removed", "reason": "integration test"}, http.StatusOK)
	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/events/%d/actions", eventID), map[string]interface{}{"action": "restore_event", "status": "recruiting"}, http.StatusOK)

	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/uploads/%d/actions", uploadID), map[string]interface{}{"action": "reject_upload", "status": "rejected"}, http.StatusBadRequest)
	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/uploads/%d/actions", uploadID), map[string]interface{}{"action": "reject_upload", "status": "rejected", "result": "integration test"}, http.StatusOK)
	assertStatus(http.MethodPost, fmt.Sprintf("/api/admin/uploads/%d/actions", uploadID), map[string]interface{}{"action": "approve_upload", "status": "approved", "result": "integration restore"}, http.StatusOK)

	auditBody := assertStatus(http.MethodGet, fmt.Sprintf("/api/admin/audit-logs?resource=messages&targetId=%d", messageID), nil, http.StatusOK)
	if !strings.Contains(auditBody, "hide_message") {
		t.Fatal("filtered audit history did not include the message action")
	}
}

func cleanupStaleModerationFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	rows, err := db.Query(`SELECT id FROM users WHERE openid LIKE 'moderation-creator-%' OR openid LIKE 'moderation-applicant-%'`)
	if err != nil {
		t.Fatal(err)
	}
	userIDs := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		userIDs = append(userIDs, id)
	}
	rows.Close()
	for _, userID := range userIDs {
		eventRows, err := db.Query(`SELECT id FROM ski_events WHERE creator_id=?`, userID)
		if err != nil {
			t.Fatal(err)
		}
		eventIDs := []int64{}
		for eventRows.Next() {
			var eventID int64
			if err := eventRows.Scan(&eventID); err != nil {
				eventRows.Close()
				t.Fatal(err)
			}
			eventIDs = append(eventIDs, eventID)
		}
		eventRows.Close()
		for _, eventID := range eventIDs {
			_, _ = db.Exec(`DELETE FROM admin_action_logs WHERE resource='events' AND target_id=?`, eventID)
			_, _ = db.Exec(`DELETE FROM join_request_history WHERE event_id=?`, eventID)
			_, _ = db.Exec(`DELETE FROM reports WHERE target_type='event' AND target_id=?`, eventID)
			_, _ = db.Exec(`DELETE FROM reviews WHERE event_id=?`, eventID)
			_, _ = db.Exec(`DELETE FROM chat_messages WHERE event_id=?`, eventID)
			_, _ = db.Exec(`DELETE FROM join_requests WHERE event_id=?`, eventID)
			_, _ = db.Exec(`DELETE FROM event_members WHERE event_id=?`, eventID)
			_, _ = db.Exec(`DELETE FROM ski_events WHERE id=?`, eventID)
		}
		uploadRows, _ := db.Query(`SELECT id FROM media_uploads WHERE user_id=?`, userID)
		if uploadRows != nil {
			for uploadRows.Next() {
				var uploadID int64
				_ = uploadRows.Scan(&uploadID)
				_, _ = db.Exec(`DELETE FROM admin_action_logs WHERE resource='uploads' AND target_id=?`, uploadID)
			}
			uploadRows.Close()
		}
		_, _ = db.Exec(`DELETE FROM media_uploads WHERE user_id=?`, userID)
		_, _ = db.Exec(`DELETE FROM reports WHERE reporter_id=? OR (target_type='user' AND target_id=?)`, userID, userID)
		_, _ = db.Exec(`DELETE FROM reviews WHERE reviewer_id=? OR reviewee_id=?`, userID, userID)
		_, _ = db.Exec(`DELETE FROM chat_messages WHERE sender_id=?`, userID)
		_, _ = db.Exec(`DELETE FROM join_requests WHERE applicant_id=? OR creator_id=?`, userID, userID)
		_, _ = db.Exec(`DELETE FROM event_members WHERE user_id=?`, userID)
		_, _ = db.Exec(`DELETE FROM notifications WHERE user_id=?`, userID)
		_, _ = db.Exec(`DELETE FROM users WHERE id=?`, userID)
	}
}

func insertUser(t *testing.T, db *sql.DB, openid string) int64 {
	result, err := db.Exec(`INSERT INTO users (openid, nickname) VALUES (?, '集成测试用户')`, openid)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	if _, err := db.Exec(`UPDATE users SET verification_status='verified', verification_method='sms_phone', verified_at=NOW(), phone_masked='138****0000' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
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

func adminToken(t *testing.T, secret string) string {
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"role": "admin", "username": "integration-admin", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(secret))
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
