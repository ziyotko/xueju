package handler

import (
	"crypto/subtle"
	"database/sql"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"xueju/backend/internal/compliance"
	"xueju/backend/internal/config"
	"xueju/backend/internal/response"
	"xueju/backend/internal/storage"
)

type AdminHandler struct {
	cfg     config.Config
	db      *sql.DB
	storage storage.Store
}

func NewAdminHandler(cfg config.Config, db *sql.DB) *AdminHandler {
	return &AdminHandler{cfg: cfg, db: db, storage: storage.New(cfg)}
}

func (h *AdminHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "登录信息格式不正确")
		return
	}
	usernameOK := subtle.ConstantTimeCompare([]byte(req.Username), []byte(h.cfg.AdminUsername)) == 1
	passwordOK := subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.cfg.AdminPassword)) == 1
	if !usernameOK || !passwordOK {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "账号或密码错误")
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
	selects := map[string]string{
		"user":        `SELECT CONCAT('user:',id) item_id, nickname target, '用户资料' item_type, CONCAT(nickname, ' · ', bio) summary, status, updated_at FROM users WHERE deleted_at IS NULL`,
		"event":       `SELECT CONCAT('event:',id) item_id, title target, '滑雪局' item_type, CONCAT(resort_name, ' · ', COALESCE(remark,'')) summary, status, updated_at FROM ski_events WHERE deleted_at IS NULL`,
		"application": `SELECT CONCAT('application:',r.id) item_id, u.nickname target, '加入申请' item_type, r.message summary, r.status status, r.updated_at updated_at FROM join_requests r JOIN users u ON u.id=r.applicant_id`,
		"message":     `SELECT CONCAT('message:',m.id) item_id, u.nickname target, '群聊消息' item_type, m.content summary, m.status status, m.created_at updated_at FROM chat_messages m JOIN users u ON u.id=m.sender_id`,
		"review":      `SELECT CONCAT('review:',r.id) item_id, u.nickname target, '滑后评价' item_type, r.content summary, r.status status, r.created_at updated_at FROM reviews r JOIN users u ON u.id=r.reviewer_id`,
		"report":      `SELECT CONCAT('report:',id) item_id, CONCAT(target_type,' #',target_id) target, '举报内容' item_type, CONCAT(reason, ' · ', content) summary, status, updated_at FROM reports`,
	}
	itemTypes := []string{"user", "event", "application", "review"}
	if strings.TrimSpace(c.Query("scope")) != "content" {
		itemTypes = append(itemTypes, "message", "report")
	}
	if itemType := strings.TrimSpace(c.Query("itemType")); itemType != "" && selects[itemType] != "" {
		allowed := false
		for _, candidate := range itemTypes {
			if candidate == itemType {
				allowed = true
				break
			}
		}
		if allowed {
			itemTypes = []string{itemType}
		}
	}
	unionParts := make([]string, 0, len(itemTypes))
	for _, itemType := range itemTypes {
		unionParts = append(unionParts, selects[itemType])
	}
	union := strings.Join(unionParts, " UNION ALL ")
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
	if reviewState := strings.TrimSpace(c.Query("reviewState")); reviewState == "active" {
		where = append(where, "status NOT IN ('disabled','removed','rejected','hidden','resolved')")
	} else if reviewState == "handled" {
		where = append(where, "status IN ('disabled','removed','rejected','hidden','resolved')")
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
		if err := rows.Scan(&id, &target, &itemType, &summary, &rowStatus, &updated); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		prefix := strings.SplitN(id, ":", 2)[0]
		list = append(list, adminRow(id, target, itemType, summary, riskByStatus(rowStatus), rowStatus, updated, gin.H{"itemType": prefix}))
	}
	var total int64
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM (`+union+`) content WHERE `+strings.Join(where, " AND "), args...).Scan(&total)
	response.Success(c, pageData(list, page, pageSize, total))
}

func (h *AdminHandler) ModerationSummary(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	contentUnion := `SELECT status FROM users WHERE deleted_at IS NULL
		UNION ALL SELECT status FROM ski_events WHERE deleted_at IS NULL
		UNION ALL SELECT status FROM join_requests
		UNION ALL SELECT status FROM reviews`
	var contentTotal, contentActive, contentHandled int64
	var messageTotal, messageNormal, messageHidden int64
	var mediaTotal, mediaPending, mediaApproved, mediaRejected int64
	queries := []struct {
		query string
		dest  *int64
	}{
		{`SELECT COUNT(*) FROM (` + contentUnion + `) moderation_content`, &contentTotal},
		{`SELECT COUNT(*) FROM (` + contentUnion + `) moderation_content WHERE status NOT IN ('disabled','removed','rejected','hidden','resolved')`, &contentActive},
		{`SELECT COUNT(*) FROM (` + contentUnion + `) moderation_content WHERE status IN ('disabled','removed','rejected','hidden','resolved')`, &contentHandled},
		{`SELECT COUNT(*) FROM chat_messages`, &messageTotal},
		{`SELECT COUNT(*) FROM chat_messages WHERE status='normal'`, &messageNormal},
		{`SELECT COUNT(*) FROM chat_messages WHERE status='hidden'`, &messageHidden},
		{`SELECT COUNT(*) FROM media_uploads`, &mediaTotal},
		{`SELECT COUNT(*) FROM media_uploads WHERE status='pending'`, &mediaPending},
		{`SELECT COUNT(*) FROM media_uploads WHERE status='approved'`, &mediaApproved},
		{`SELECT COUNT(*) FROM media_uploads WHERE status='rejected'`, &mediaRejected},
	}
	for _, item := range queries {
		if err := h.db.QueryRow(item.query).Scan(item.dest); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
	}
	response.Success(c, gin.H{
		"content":  gin.H{"total": contentTotal, "active": contentActive, "handled": contentHandled},
		"messages": gin.H{"total": messageTotal, "normal": messageNormal, "hidden": messageHidden},
		"media":    gin.H{"total": mediaTotal, "pending": mediaPending, "approved": mediaApproved, "rejected": mediaRejected},
	})
}

func (h *AdminHandler) Users(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where, args := adminWhere(c, []string{"deleted_at IS NULL"}, "nickname", "city")
	rows, err := h.db.Query(`SELECT id, nickname, city, ski_type, ski_level, credit_score, verification_status, verification_method, phone_masked, verified_at, status, updated_at FROM users WHERE `+where+` ORDER BY id DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var nickname, city, skiType, skiLevel, verificationStatus, verificationMethod, phoneMasked, status string
		var credit float64
		var verifiedAt sql.NullTime
		var updated time.Time
		_ = rows.Scan(&id, &nickname, &city, &skiType, &skiLevel, &credit, &verificationStatus, &verificationMethod, &phoneMasked, &verifiedAt, &status, &updated)
		list = append(list, adminRow(id, nickname, "用户", city+" "+skiType+" "+skiLevel, riskByStatus(status), status, updated, gin.H{"creditScore": credit, "verificationStatus": verificationStatus, "verificationMethod": verificationMethod, "phoneMasked": phoneMasked, "verifiedAt": nullableAdminTime(verifiedAt)}))
	}
	response.Success(c, pageData(list, page, pageSize, h.count("users", where, args)))
}

func (h *AdminHandler) UserVerifications(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	rows, err := h.db.Query(`SELECT id, status, method, provider, provider_reference, phone_masked, verified_at, revoked_at, revoked_by, revoke_reason, created_at FROM user_verifications WHERE user_id=? ORDER BY id DESC`, c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id int64
		var status, method, provider, reference, masked, revokedBy, reason string
		var verifiedAt, revokedAt sql.NullTime
		var created time.Time
		if err := rows.Scan(&id, &status, &method, &provider, &reference, &masked, &verifiedAt, &revokedAt, &revokedBy, &reason, &created); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		list = append(list, gin.H{"id": id, "status": status, "method": method, "provider": provider, "providerReference": reference, "phoneMasked": masked, "verifiedAt": nullableAdminTime(verifiedAt), "revokedAt": nullableAdminTime(revokedAt), "revokedBy": revokedBy, "revokeReason": reason, "createdAt": created})
	}
	response.Success(c, list)
}

func (h *AdminHandler) changePhoneVerificationStatus(c *gin.Context, id, action, reason string) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "必须填写原因")
		return
	}
	userID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "用户编号无效")
		return
	}
	newStatus := "reverify_required"
	verificationStatus := "reverify_required"
	if action == "revoke_phone_verification" {
		newStatus = "revoked"
		verificationStatus = "revoked"
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer tx.Rollback()
	var oldStatus string
	if err := tx.QueryRow(`SELECT verification_status FROM users WHERE id=? FOR UPDATE`, userID).Scan(&oldStatus); err != nil {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "用户不存在")
		return
	}
	if _, err := tx.Exec(`UPDATE users SET verification_status=? WHERE id=?`, newStatus, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	adminUsername, _ := c.Get("adminUsername")
	adminID := fmt.Sprint(adminUsername)
	if action == "revoke_phone_verification" {
		_, err = tx.Exec(`UPDATE user_verifications SET status=?, active_phone_hash=NULL, revoked_at=NOW(), revoked_by=?, revoke_reason=? WHERE user_id=? AND active_phone_hash IS NOT NULL`, verificationStatus, adminID, reason, userID)
	} else {
		_, err = tx.Exec(`UPDATE user_verifications SET status=? WHERE user_id=? AND active_phone_hash IS NOT NULL`, verificationStatus, userID)
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`INSERT INTO verification_audit_logs (user_id, action, operator_type, operator_id, old_status, new_status, ip_address, user_agent, remark) VALUES (?, ?, 'admin', ?, ?, ?, ?, ?, ?)`, userID, action, adminID, oldStatus, newStatus, c.ClientIP(), limitedUserAgent(c), reason); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"status": newStatus})
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
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where := []string{"1=1"}
	args := []interface{}{}
	if status := strings.TrimSpace(c.Query("status")); status != "" && status != "all" {
		where = append(where, "m.status=?")
		args = append(args, status)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		where = append(where, "(m.content LIKE ? OR u.nickname LIKE ? OR e.title LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like)
	}
	from := ` FROM chat_messages m JOIN users u ON u.id=m.sender_id JOIN ski_events e ON e.id=m.event_id WHERE ` + strings.Join(where, " AND ")
	rows, err := h.db.Query(`SELECT m.id, m.event_id, m.sender_id, u.nickname, e.title, m.message_type, m.content, m.status, m.created_at`+from+` ORDER BY m.created_at DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, eventID, senderID int64
		var nickname, eventTitle, messageType, content, status string
		var created time.Time
		if err := rows.Scan(&id, &eventID, &senderID, &nickname, &eventTitle, &messageType, &content, &status, &created); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		list = append(list, adminRow(id, nickname, "群聊消息", eventTitle+" · "+content, riskByStatus(status), status, created, gin.H{"eventId": eventID, "eventTitle": eventTitle, "senderId": senderID, "messageType": messageType}))
	}
	var total int64
	if err := h.db.QueryRow(`SELECT COUNT(*)`+from, args...).Scan(&total); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, pageData(list, page, pageSize, total))
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
		list = append(list, adminRow(id, kind+" #"+strconv.FormatInt(id, 10), "媒体审核", publicURL, riskByStatus(status), status, updated, gin.H{"userId": userID, "kind": kind, "mimeType": mimeType, "result": result, "previewPath": "/admin/uploads/" + strconv.FormatInt(id, 10) + "/preview"}))
	}
	response.Success(c, pageData(list, page, pageSize, h.count("media_uploads", where, args)))
}

func (h *AdminHandler) UploadPreview(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "媒体编号无效")
		return
	}
	var path, storedMime string
	if err := h.db.QueryRow(`SELECT path, mime_type FROM media_uploads WHERE id=?`, id).Scan(&path, &storedMime); err != nil {
		if err == sql.ErrNoRows {
			response.Error(c, http.StatusNotFound, response.CodeNotFound, "媒体记录不存在")
			return
		}
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	data, detectedMime, err := h.storage.Get(c.Request.Context(), path)
	if err != nil {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "媒体文件不存在")
		return
	}
	if detectedMime == "" {
		detectedMime = storedMime
	}
	if detectedMime == "" {
		detectedMime = http.DetectContentType(data)
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("Content-Disposition", "inline")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Data(http.StatusOK, detectedMime, data)
}

func (h *AdminHandler) UploadResortImage(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "请选择雪场图片")
		return
	}
	if file.Size > 5*1024*1024 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "图片不能超过 5MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "仅支持 JPG、JPEG、PNG 图片")
		return
	}
	source, err := file.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "无法读取图片")
		return
	}
	defer source.Close()
	header := make([]byte, 512)
	n, _ := source.Read(header)
	mimeType := http.DetectContentType(header[:n])
	validMime := (ext == ".png" && mimeType == "image/png") || ((ext == ".jpg" || ext == ".jpeg") && mimeType == "image/jpeg")
	if !validMime {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "图片扩展名与内容不匹配")
		return
	}
	if _, err := source.Seek(0, 0); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "无法检查图片")
		return
	}
	imageConfig, _, err := image.DecodeConfig(source)
	if err != nil || imageConfig.Width < 1 || imageConfig.Height < 1 || imageConfig.Width > 12000 || imageConfig.Height > 12000 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "图片内容无效")
		return
	}
	if _, err := source.Seek(0, 0); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "无法读取图片")
		return
	}
	data, err := io.ReadAll(io.LimitReader(source, 5*1024*1024+1))
	if err != nil || len(data) > 5*1024*1024 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "无法读取图片")
		return
	}

	filename := fmt.Sprintf("resort-%d%s", time.Now().UnixNano(), ext)
	storageKey := filepath.ToSlash(filepath.Join("resorts", filename))
	if err := h.storage.Put(c.Request.Context(), storageKey, data, mimeType); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	path := "/uploads/resorts/" + filename
	publicURL := h.cfg.PublicBaseURL + path
	if h.cfg.PublicBaseURL == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, path)
	}
	result, err := h.db.Exec(`INSERT INTO media_uploads (user_id, kind, path, public_url, mime_type, status, review_result) VALUES (0, 'resorts', ?, ?, ?, 'approved', '管理员上传')`, storageKey, publicURL, mimeType)
	if err != nil {
		_ = h.storage.Delete(c.Request.Context(), storageKey)
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	uploadID, _ := result.LastInsertId()
	response.Success(c, gin.H{"id": uploadID, "url": publicURL})
}

func (h *AdminHandler) AuditLogs(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where := []string{"1=1"}
	args := []interface{}{}
	if resource := strings.TrimSpace(c.Query("resource")); resource != "" {
		where = append(where, "resource=?")
		args = append(args, resource)
	}
	if targetID := strings.TrimSpace(c.Query("targetId")); targetID != "" {
		where = append(where, "target_id=?")
		args = append(args, targetID)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		where = append(where, "(admin_username LIKE ? OR action LIKE ? OR detail LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like)
	}
	whereSQL := strings.Join(where, " AND ")
	rows, err := h.db.Query(`SELECT id, admin_username, resource, target_id, action, before_status, after_status, detail, created_at FROM admin_action_logs WHERE `+whereSQL+` ORDER BY created_at DESC LIMIT ?, ?`, append(args, (page-1)*pageSize, pageSize)...)
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
		list = append(list, adminRow(id, username, resource+" #"+targetID, action+" · "+detail, "低", afterStatus, created, gin.H{
			"resource": resource, "targetId": targetID, "action": action, "beforeStatus": beforeStatus, "afterStatus": afterStatus, "detail": detail,
		}))
	}
	var total int64
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM admin_action_logs WHERE `+whereSQL, args...).Scan(&total); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
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
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "雪场名称和城市不能为空")
		return
	}
	req.Status = defaultString(req.Status, "normal")
	id := c.Param("id")
	if id == "" {
		if strings.TrimSpace(req.ImageURL) == "" {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "请上传雪场图片")
			return
		}
		var imageCount int64
		if err := h.db.QueryRow(`SELECT COUNT(*) FROM media_uploads WHERE kind='resorts' AND public_url=? AND status='approved'`, req.ImageURL).Scan(&imageCount); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		if imageCount == 0 {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "请先上传有效的雪场图片")
			return
		}
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
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "雪场记录不存在")
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
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "操作参数格式不正确")
		return
	}
	if resource != "content-reviews" && strings.Contains(id, ":") {
		parts := strings.SplitN(id, ":", 2)
		expectedPrefix := map[string]string{"users": "user", "events": "event", "applications": "application", "messages": "message", "reviews": "review", "reports": "report", "dicts": "dict", "uploads": "upload"}[resource]
		if len(parts) != 2 || parts[0] != expectedPrefix {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "操作对象无效")
			return
		}
		id = parts[1]
	}
	if resource != "content-reviews" {
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "操作对象无效")
			return
		}
	}
	if resource == "users" && (req.Action == "require_phone_reverification" || req.Action == "revoke_phone_verification") {
		h.changePhoneVerificationStatus(c, id, req.Action, req.Reason)
		return
	}
	if resource == "content-reviews" {
		parts := strings.SplitN(id, ":", 2)
		if len(parts) != 2 {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "巡检对象无效")
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
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "不支持处置该类内容")
			return
		}
	}
	allowedActions := map[string]map[string]bool{
		"users":        {"disable_user": true, "enable_user": true},
		"events":       {"delist_event": true, "restore_event": true},
		"applications": {"reject_application": true},
		"messages":     {"hide_message": true, "restore_message": true},
		"reviews":      {"hide_review": true, "restore_review": true},
		"reports":      {"resolve_report": true},
		"dicts":        {"disable_dict": true, "enable_dict": true},
		"uploads":      {"approve_upload": true, "reject_upload": true},
	}
	if !allowedActions[resource][req.Action] {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "不支持该操作")
		return
	}
	reasonRequired := map[string]bool{
		"disable_user": true, "delist_event": true, "reject_application": true,
		"hide_message": true, "hide_review": true, "reject_upload": true,
		"disable_dict": true,
	}
	if reasonRequired[req.Action] && strings.TrimSpace(defaultString(req.Reason, req.Result)) == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "必须填写操作原因")
		return
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
		status := "disabled"
		if req.Action == "enable_user" {
			status = "normal"
		}
		query = `UPDATE users SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "events":
		status := "removed"
		if req.Action == "restore_event" {
			status = "recruiting"
		}
		query = `UPDATE ski_events SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "applications":
		query = `UPDATE join_requests SET status='rejected', reject_reason=? WHERE id=?`
		args = []interface{}{defaultString(req.Reason, "运营审核未通过"), id}
	case "messages":
		status := "hidden"
		if req.Action == "restore_message" {
			status = "normal"
		}
		query = `UPDATE chat_messages SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "reviews":
		status := "hidden"
		if req.Action == "restore_review" {
			status = "normal"
		}
		query = `UPDATE reviews SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "reports":
		query = `UPDATE reports SET status='resolved', result=? WHERE id=?`
		args = []interface{}{defaultString(req.Result, req.Reason), id}
	case "dicts":
		status := "disabled"
		if req.Action == "enable_dict" {
			status = "normal"
		}
		query = `UPDATE ski_resorts SET status=? WHERE id=?`
		args = []interface{}{status, id}
	case "uploads":
		status := "approved"
		if req.Action == "reject_upload" {
			status = "rejected"
		}
		query = `UPDATE media_uploads SET status=?, review_result=? WHERE id=?`
		args = []interface{}{status, defaultString(req.Result, req.Reason), id}
	default:
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "不支持该业务模块")
		return
	}

	result, err := h.db.Exec(query, args...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		if beforeStatus != "" {
			response.Success(c, gin.H{"status": beforeStatus, "unchanged": true})
			return
		}
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "记录不存在")
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
	afterStatus := ""
	_ = h.db.QueryRow(`SELECT status FROM `+statusTable[resource]+` WHERE id=?`, id).Scan(&afterStatus)
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
	response.Error(c, http.StatusServiceUnavailable, response.CodeServerError, "数据库未配置")
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
		"updatedAt": updated.Format(time.RFC3339),
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

func nullableAdminTime(value sql.NullTime) interface{} {
	if value.Valid {
		return value.Time
	}
	return nil
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
