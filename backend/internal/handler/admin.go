package handler

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xueju/backend/internal/compliance"
	"xueju/backend/internal/response"
)

type AdminHandler struct {
	db *sql.DB
}

func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{db: db}
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
	rows, err := h.db.Query(`SELECT id, event_id, sender_id, message_type, content, status, created_at FROM chat_messages WHERE status='normal' ORDER BY created_at DESC LIMIT ?, ?`, (page-1)*pageSize, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, eventID, senderID int64
		var messageType, content, status string
		var created time.Time
		_ = rows.Scan(&id, &eventID, &senderID, &messageType, &content, &status, &created)
		list = append(list, adminRow(id, "群聊消息", messageType, content, "低", status, created, gin.H{"eventId": eventID, "senderId": senderID}))
	}
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE status='normal'`).Scan(&total)
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
		list = append(list, adminRow(id, targetType+" #"+strconv.FormatInt(targetID, 10), "举报", reason+" · "+content, reportRisk(status), status, updated, gin.H{"result": result}))
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

func (h *AdminHandler) Dicts(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	rows, err := h.db.Query(`SELECT id, name, city, province, status, sort, updated_at FROM ski_resorts ORDER BY sort ASC, id ASC LIMIT ?, ?`, (page-1)*pageSize, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, sort int64
		var name, city, province, status string
		var updated time.Time
		_ = rows.Scan(&id, &name, &city, &province, &status, &sort, &updated)
		list = append(list, adminRow(id, name, "雪场字典", city+" · "+province+" · 排序 "+strconv.FormatInt(sort, 10), riskByStatus(status), status, updated, nil))
	}
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM ski_resorts`).Scan(&total)
	response.Success(c, pageData(list, page, pageSize, total))
}

func (h *AdminHandler) Action(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	resource := c.Param("resource")
	id := c.Param("id")
	var req struct {
		Action string `json:"action"`
		Status string `json:"status"`
		Result string `json:"result"`
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

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
	case "messages", "content-reviews":
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
	response.Success(c, gin.H{"status": "accepted"})
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

func adminRow(id int64, target, typ, summary, risk, status string, updated time.Time, extra gin.H) gin.H {
	row := gin.H{
		"id":        strconv.FormatInt(id, 10),
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
