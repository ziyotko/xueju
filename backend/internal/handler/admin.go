package handler

import (
	"crypto/subtle"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"xueju/backend/internal/compliance"
	"xueju/backend/internal/config"
	"xueju/backend/internal/response"
)

type AdminHandler struct {
	cfg config.Config
	db  *sql.DB
}

func NewAdminHandler(cfg config.Config, db *sql.DB) *AdminHandler {
	return &AdminHandler{cfg: cfg, db: db}
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid login payload")
		return
	}
	usernameOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(h.cfg.AdminUsername)) == 1
	passwordOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.cfg.AdminPassword)) == 1
	if !usernameOK || !passwordOK {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid username or password")
		return
	}
	expiresAt := time.Now().Add(time.Duration(h.cfg.JWTExpiresHours) * time.Hour)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"role":     "admin",
		"username": req.Username,
		"exp":      expiresAt.Unix(),
	})
	tokenText, err := token.SignedString([]byte(h.cfg.JWTSecret))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"token": tokenText, "expiresAt": expiresAt, "user": gin.H{"username": req.Username, "role": "admin"}})
}

func (h *AdminHandler) Dashboard(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	var pendingReports, handledReports, users, events, applications, reviews int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM reports WHERE status IN ('pending','processing')`).Scan(&pendingReports)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM reports WHERE status IN ('resolved','rejected')`).Scan(&handledReports)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM users WHERE deleted_at IS NULL`).Scan(&users)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM ski_events WHERE deleted_at IS NULL`).Scan(&events)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM join_requests WHERE status='pending'`).Scan(&applications)
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE status='normal'`).Scan(&reviews)
	response.Success(c, gin.H{
		"contentReview": gin.H{"pending": applications + pendingReports, "approved": reviews, "rejected": 0},
		"reports":       gin.H{"pending": pendingReports, "handled": handledReports},
		"totals":        gin.H{"users": users, "events": events, "applications": applications, "reviews": reviews},
		"actions": []compliance.AdminAction{
			compliance.ActionDisableUser,
			compliance.ActionDelistEvent,
			compliance.ActionHideMessage,
			compliance.ActionHideReview,
			compliance.ActionResolveReport,
		},
	})
}

func (h *AdminHandler) ContentReviews(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	union := `
		SELECT CONCAT('user:',id) item_id, nickname target, '用户资料' item_type, CONCAT(nickname, ' · ', bio) summary, status, updated_at FROM users WHERE deleted_at IS NULL
		UNION ALL SELECT CONCAT('event:',id), title, '滑雪局', CONCAT(resort_name, ' · ', COALESCE(remark,'')), status, updated_at FROM ski_events WHERE deleted_at IS NULL
		UNION ALL SELECT CONCAT('application:',r.id), u.nickname, '加入申请', r.message, r.status, r.updated_at FROM join_requests r JOIN users u ON u.id=r.applicant_id
		UNION ALL SELECT CONCAT('message:',m.id), u.nickname, '群聊消息', m.content, m.status, m.created_at FROM chat_messages m JOIN users u ON u.id=m.sender_id
		UNION ALL SELECT CONCAT('review:',r.id), u.nickname, '滑后评价', r.content, r.status, r.created_at FROM reviews r JOIN users u ON u.id=r.reviewer_id
		UNION ALL SELECT CONCAT('report:',id), CONCAT(target_type,' #',target_id), '举报内容', CONCAT(reason, ' · ', content), status, updated_at FROM reports`
	keyword := strings.TrimSpace(c.Query("keyword"))
	status := strings.TrimSpace(c.Query("status"))
	where := []string{"1=1"}
	args := []interface{}{}
	if keyword != "" {
		where = append(where, "(target LIKE ? OR summary LIKE ? OR item_type LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like)
	}
	if status != "" && status != "all" {
		where = append(where, "status=?")
		args = append(args, status)
	}
	query := `SELECT item_id, target, item_type, summary, status, updated_at FROM (` + union + `) content WHERE ` + strings.Join(where, " AND ") + ` ORDER BY updated_at DESC LIMIT ?, ?`
	rows, err := h.db.Query(query, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, target, itemType, summary, rowStatus string
		var updated time.Time
		_ = rows.Scan(&id, &target, &itemType, &summary, &rowStatus, &updated)
		list = append(list, adminRow(id, target, itemType, summary, riskByStatus(rowStatus), rowStatus, updated, nil))
	}
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM (`+union+`) content WHERE `+strings.Join(where, " AND "), args...).Scan(&total)
	response.Success(c, pageData(list, page, pageSize, total))
}

func (h *AdminHandler) Users(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where, args := adminWhere(c, []string{"deleted_at IS NULL"}, "nickname", "city")
	rows, err := h.db.Query(`SELECT id, nickname, city, ski_type, ski_level, credit_score, status, updated_at FROM users WHERE `+where+` ORDER BY id DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var nickname, city, skiType, skiLevel, status string
		var credit float64
		var updated time.Time
		_ = rows.Scan(&id, &nickname, &city, &skiType, &skiLevel, &credit, &status, &updated)
		list = append(list, adminRow(id, nickname, "用户", city+" "+skiType+" "+skiLevel, riskByStatus(status), status, updated, gin.H{"creditScore": credit}))
	}
	response.Success(c, pageData(list, page, pageSize, h.count("users", where, args)))
}

func (h *AdminHandler) Events(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where, args := adminWhere(c, []string{"deleted_at IS NULL"}, "title", "resort_name", "depart_city")
	rows, err := h.db.Query(`SELECT id, title, resort_name, depart_city, status, current_members, max_members, updated_at FROM ski_events WHERE `+where+` ORDER BY created_at DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var title, resortName, departCity, status string
		var currentMembers, maxMembers int
		var updated time.Time
		_ = rows.Scan(&id, &title, &resortName, &departCity, &status, &currentMembers, &maxMembers, &updated)
		list = append(list, adminRow(id, title, "滑雪局", resortName+" · "+departCity+" · "+strconv.Itoa(currentMembers)+"/"+strconv.Itoa(maxMembers), riskByStatus(status), status, updated, nil))
	}
	response.Success(c, pageData(list, page, pageSize, h.count("ski_events", where, args)))
}

func (h *AdminHandler) Applications(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where := []string{"1=1"}
	args := []interface{}{}
	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		where = append(where, "r.status=?")
		args = append(args, status)
	}
	rows, err := h.db.Query(`SELECT r.id, e.title, u.nickname, r.message, r.status, r.created_at
		FROM join_requests r JOIN ski_events e ON e.id=r.event_id JOIN users u ON u.id=r.applicant_id
		WHERE `+strings.Join(where, " AND ")+` ORDER BY r.created_at DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var title, nickname, message, status string
		var created time.Time
		_ = rows.Scan(&id, &title, &nickname, &message, &status, &created)
		list = append(list, adminRow(id, nickname, "加入申请", title+" · "+message, riskByStatus(status), status, created, nil))
	}
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM join_requests r WHERE `+strings.Join(where, " AND "), args...).Scan(&total)
	response.Success(c, pageData(list, page, pageSize, total))
}

func (h *AdminHandler) Reports(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where, args := adminWhere(c, []string{"1=1"}, "target_type", "reason", "content", "status")
	rows, err := h.db.Query(`SELECT id, target_type, target_id, reason, content, status, result, updated_at FROM reports WHERE `+where+` ORDER BY created_at DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, targetID int64
		var targetType, reason, content, status, result string
		var updated time.Time
		_ = rows.Scan(&id, &targetType, &targetID, &reason, &content, &status, &result, &updated)
		list = append(list, adminRow(id, targetType+" #"+strconv.FormatInt(targetID, 10), "举报", reason+" · "+content, reportRisk(status), status, updated, gin.H{"result": result, "targetType": targetType, "targetId": targetID}))
	}
	response.Success(c, pageData(list, page, pageSize, h.count("reports", where, args)))
}

func (h *AdminHandler) Messages(c *gin.Context) {
	h.ContentReviews(c)
}

func (h *AdminHandler) Reviews(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where := []string{"1=1"}
	args := []interface{}{}
	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		where = append(where, "r.status=?")
		args = append(args, status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		where = append(where, "(r.content LIKE ? OR e.title LIKE ? OR reviewer.nickname LIKE ? OR reviewee.nickname LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like)
	}
	rows, err := h.db.Query(`SELECT r.id, e.title, reviewer.nickname, reviewee.nickname, r.score, r.content, r.status, r.created_at
		FROM reviews r JOIN ski_events e ON e.id=r.event_id JOIN users reviewer ON reviewer.id=r.reviewer_id JOIN users reviewee ON reviewee.id=r.reviewee_id
		WHERE `+strings.Join(where, " AND ")+` ORDER BY r.created_at DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var title, reviewer, reviewee, content, status string
		var score int
		var created time.Time
		_ = rows.Scan(&id, &title, &reviewer, &reviewee, &score, &content, &status, &created)
		list = append(list, adminRow(id, reviewer+" → "+reviewee, "评价", title+" · "+content, riskByStatus(status), status, created, gin.H{"score": score}))
	}
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM reviews r JOIN ski_events e ON e.id=r.event_id JOIN users reviewer ON reviewer.id=r.reviewer_id JOIN users reviewee ON reviewee.id=r.reviewee_id WHERE `+strings.Join(where, " AND "), args...).Scan(&total)
	response.Success(c, pageData(list, page, pageSize, total))
}

func (h *AdminHandler) Uploads(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where, args := adminWhere(c, []string{"1=1"}, "kind", "mime_type", "status")
	rows, err := h.db.Query(`SELECT id, user_id, kind, public_url, mime_type, status, review_result, updated_at FROM media_uploads WHERE `+where+` ORDER BY created_at DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, userID int64
		var kind, publicURL, mimeType, status, result string
		var updated time.Time
		if err := rows.Scan(&id, &userID, &kind, &publicURL, &mimeType, &status, &result, &updated); err != nil {
			continue
		}
		list = append(list, adminRow(id, kind+" #"+strconv.FormatInt(id, 10), "媒体审核", publicURL, riskByStatus(status), status, updated, gin.H{"userId": userID, "mimeType": mimeType, "result": result}))
	}
	response.Success(c, pageData(list, page, pageSize, h.count("media_uploads", where, args)))
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	rows, err := h.db.Query(`SELECT id, admin_username, resource, target_id, action, before_status, after_status, detail, created_at FROM admin_action_logs ORDER BY created_at DESC LIMIT ?, ?`, (page-1)*pageSize, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var username, resource, targetID, action, beforeStatus, afterStatus, detail string
		var created time.Time
		if err := rows.Scan(&id, &username, &resource, &targetID, &action, &beforeStatus, &afterStatus, &detail, &created); err != nil {
			continue
		}
		list = append(list, adminRow(id, username, resource+" #"+targetID, action+" · "+detail, "低", afterStatus, created, gin.H{"beforeStatus": beforeStatus}))
	}
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM admin_action_logs`).Scan(&total)
	response.Success(c, pageData(list, page, pageSize, total))
}

func (h *AdminHandler) Dicts(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	rows, err := h.db.Query(`SELECT id, name, city, province, image_url, status, sort, updated_at FROM ski_resorts ORDER BY sort ASC, id ASC LIMIT ?, ?`, (page-1)*pageSize, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, sort int64
		var name, city, province, imageURL, status string
		var updated time.Time
		_ = rows.Scan(&id, &name, &city, &province, &imageURL, &status, &sort, &updated)
		list = append(list, adminRow(id, name, "雪场字典", city+" · "+province+" · 排序 "+strconv.FormatInt(sort, 10), riskByStatus(status), status, updated, gin.H{"city": city, "province": province, "imageUrl": imageURL, "sort": sort}))
	}
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM ski_resorts`).Scan(&total)
	response.Success(c, pageData(list, page, pageSize, total))
}

func (h *AdminHandler) SaveDict(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	var req struct {
		Name     string `json:"name"`
		City     string `json:"city"`
		Province string `json:"province"`
		ImageURL string `json:"imageUrl"`
		Sort     int64  `json:"sort"`
		Status   string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.City) == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "name and city are required")
		return
	}
	req.Status = defaultString(req.Status, "normal")
	id := c.Param("id")
	if id == "" {
		result, err := h.db.Exec(`INSERT INTO ski_resorts (name, city, province, image_url, sort, status) VALUES (?, ?, ?, ?, ?, ?)`, req.Name, req.City, req.Province, req.ImageURL, req.Sort, req.Status)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		newID, _ := result.LastInsertId()
		h.logAdminAction(c, "dicts", strconv.FormatInt(newID, 10), "create_dict", "", req.Status, req.Name)
		response.Success(c, gin.H{"id": newID})
		return
	}
	result, err := h.db.Exec(`UPDATE ski_resorts SET name=?, city=?, province=?, image_url=?, sort=?, status=? WHERE id=?`, req.Name, req.City, req.Province, req.ImageURL, req.Sort, req.Status, id)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "not found")
		return
	}
	h.logAdminAction(c, "dicts", id, "update_dict", "", req.Status, req.Name)
	response.Success(c, gin.H{"id": id})
}

func (h *AdminHandler) Action(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	resource := c.Param("resource")
	id := c.Param("id")
	var req struct {
		Action       string `json:"action"`
		Status       string `json:"status"`
		Result       string `json:"result"`
		Reason       string `json:"reason"`
		LinkedAction bool   `json:"linkedAction"`
	}
	_ = c.ShouldBindJSON(&req)
	if resource == "content-reviews" {
		parts := strings.SplitN(id, ":", 2)
		if len(parts) != 2 {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid content review target")
			return
		}
		id = parts[1]
		switch parts[0] {
		case "user":
			resource, req.Action, req.Status = "users", "disable_user", "disabled"
		case "event":
			resource, req.Action, req.Status = "events", "delist_event", "removed"
		case "application":
			resource, req.Action, req.Status = "applications", "reject_application", "rejected"
		case "message":
			resource, req.Action, req.Status = "messages", "hide_message", "hidden"
		case "review":
			resource, req.Action, req.Status = "reviews", "hide_review", "hidden"
		case "report":
			resource, req.Action, req.Status = "reports", "resolve_report", "resolved"
		default:
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "unsupported content review target")
			return
		}
	}
	beforeStatus := ""
	statusTable := map[string]string{"users": "users", "events": "ski_events", "applications": "join_requests", "messages": "chat_messages", "reviews": "reviews", "reports": "reports", "dicts": "ski_resorts", "uploads": "media_uploads"}
	if table := statusTable[resource]; table != "" {
		_ = h.db.QueryRow(`SELECT status FROM `+table+` WHERE id=?`, id).Scan(&beforeStatus)
	}

	var query string
	args := []interface{}{}
	switch resource {
	case "users":
		status := defaultString(req.Status, "disabled")
		if req.Action == "enable_user" {
			status = "normal"
		}
		query = `UPDATE users SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "events":
		status := defaultString(req.Status, "removed")
		if req.Action == "restore_event" {
			status = "recruiting"
		}
		query = `UPDATE ski_events SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "applications":
		query = `UPDATE join_requests SET status='rejected', reject_reason=? WHERE id=?`
		args = []interface{}{defaultString(req.Reason, "运营审核未通过"), id}
	case "messages":
		status := defaultString(req.Status, "hidden")
		query = `UPDATE chat_messages SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "reviews":
		status := defaultString(req.Status, "hidden")
		if req.Action == "restore_review" {
			status = "normal"
		}
		query = `UPDATE reviews SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "reports":
		status := defaultString(req.Status, "resolved")
		query = `UPDATE reports SET status=?, result=? WHERE id=?`
		args = []interface{}{status, defaultString(req.Result, req.Reason), id}
	case "dicts":
		status := defaultString(req.Status, "disabled")
		if req.Action == "enable_dict" {
			status = "normal"
		}
		query = `UPDATE ski_resorts SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "uploads":
		status := defaultString(req.Status, "approved")
		if req.Action == "reject_upload" {
			status = "rejected"
		}
		query = `UPDATE media_uploads SET status=?, review_result=? WHERE id=?`
		args = []interface{}{status, defaultString(req.Result, req.Reason), id}
	default:
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "unsupported resource")
		return
	}

	result, err := h.db.Exec(query, args...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "not found")
		return
	}
	if resource == "reviews" {
		var revieweeID int64
		if err := h.db.QueryRow(`SELECT reviewee_id FROM reviews WHERE id=?`, id).Scan(&revieweeID); err == nil {
			_ = h.refreshRevieweeStats(revieweeID)
		}
	}
	if resource == "reports" && req.LinkedAction {
		var targetType string
		var targetID int64
		if err := h.db.QueryRow(`SELECT target_type, target_id FROM reports WHERE id=?`, id).Scan(&targetType, &targetID); err == nil {
			switch targetType {
			case "user":
				_, _ = h.db.Exec(`UPDATE users SET status='disabled' WHERE id=?`, targetID)
			case "event":
				_, _ = h.db.Exec(`UPDATE ski_events SET status='removed' WHERE id=?`, targetID)
			case "message":
				_, _ = h.db.Exec(`UPDATE chat_messages SET status='hidden' WHERE id=?`, targetID)
			case "review":
				var revieweeID int64
				_ = h.db.QueryRow(`SELECT reviewee_id FROM reviews WHERE id=?`, targetID).Scan(&revieweeID)
				_, _ = h.db.Exec(`UPDATE reviews SET status='hidden' WHERE id=?`, targetID)
				if revieweeID > 0 {
					_ = h.refreshRevieweeStats(revieweeID)
				}
			}
		}
	}
	afterStatus := req.Status
	if afterStatus == "" {
		_ = h.db.QueryRow(`SELECT status FROM `+statusTable[resource]+` WHERE id=?`, id).Scan(&afterStatus)
	}
	h.logAdminAction(c, resource, id, req.Action, beforeStatus, afterStatus, defaultString(req.Result, req.Reason))
	response.Success(c, gin.H{"status": "accepted"})
}

func (h *AdminHandler) logAdminAction(c *gin.Context, resource, targetID, action, beforeStatus, afterStatus, detail string) {
	adminUsername, _ := c.Get("adminUsername")
	requestID, _ := c.Get("requestID")
	auditDetail := strings.TrimSpace(detail + " requestId=" + fmt.Sprint(requestID) + " ip=" + c.ClientIP())
	_, _ = h.db.Exec(`INSERT INTO admin_action_logs (admin_username, resource, target_id, action, before_status, after_status, detail) VALUES (?, ?, ?, ?, ?, ?, ?)`, fmt.Sprint(adminUsername), resource, targetID, action, beforeStatus, afterStatus, auditDetail)
}

func (h *AdminHandler) refreshRevieweeStats(userID int64) error {
	var avg sql.NullFloat64
	var total, good int64
	if err := h.db.QueryRow(`SELECT AVG(score), COUNT(*), COALESCE(SUM(CASE WHEN score >= 4 THEN 1 ELSE 0 END),0) FROM reviews WHERE reviewee_id=? AND status='normal'`, userID).Scan(&avg, &total, &good); err != nil {
		return err
	}
	credit, goodRate := 5.0, 100.0
	if avg.Valid {
		credit = avg.Float64
	}
	if total > 0 {
		goodRate = float64(good) * 100 / float64(total)
	}
	_, err := h.db.Exec(`UPDATE users SET credit_score=?, good_rate=? WHERE id=?`, credit, goodRate, userID)
	return err
}

func (h *AdminHandler) requireDB(c *gin.Context) bool {
	if h.db != nil {
		return true
	}
	response.Error(c, http.StatusServiceUnavailable, response.CodeServerError, "database is not configured")
	return false
}

func (h *AdminHandler) count(table, where string, args []interface{}) int64 {
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE `+where, args...).Scan(&total)
	return total
}

func adminWhere(c *gin.Context, base []string, columns ...string) (string, []interface{}) {
	where := append([]string{}, base...)
	args := []interface{}{}
	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		where = append(where, "status=?")
		args = append(args, status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" && len(columns) > 0 {
		parts := []string{}
		for _, column := range columns {
			parts = append(parts, column+" LIKE ?")
			args = append(args, "%"+keyword+"%")
		}
		where = append(where, "("+strings.Join(parts, " OR ")+")")
	}
	return strings.Join(where, " AND "), args
}

func adminRow(id interface{}, target, typ, summary, risk, status string, updated time.Time, extra gin.H) gin.H {
	row := gin.H{
		"id":        fmt.Sprint(id),
		"initial":   firstRune(target),
		"target":    target,
		"type":      typ,
		"summary":   summary,
		"risk":      risk,
		"status":    status,
		"updatedAt": updated.Format("01-02 15:04"),
	}
	for key, value := range extra {
		row[key] = value
	}
	return row
}

func firstRune(value string) string {
	for _, r := range value {
		return string(r)
	}
	return "-"
}

func riskByStatus(status string) string {
	switch status {
	case "disabled", "removed", "hidden", "rejected":
		return "高"
	case "pending", "processing":
		return "中"
	default:
		return "低"
	}
}

func reportRisk(status string) string {
	if status == "pending" || status == "processing" {
		return "中"
	}
	return riskByStatus(status)
}
