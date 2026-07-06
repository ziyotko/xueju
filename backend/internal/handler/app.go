package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"xueju/backend/internal/compliance"
	"xueju/backend/internal/config"
	"xueju/backend/internal/contentsecurity"
	"xueju/backend/internal/middleware"
	"xueju/backend/internal/response"
	"xueju/backend/internal/textfilter"
)

type AppHandler struct {
	cfg        config.Config
	db         *sql.DB
	security   *contentsecurity.Service
	textFilter *textfilter.Filter
	client     *http.Client
}

func NewAppHandler(cfg config.Config, db *sql.DB) *AppHandler {
	filter, _ := textfilter.New()
	return &AppHandler{
		cfg:        cfg,
		db:         db,
		security:   contentsecurity.New(cfg),
		textFilter: filter,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

func (h *AppHandler) cleanText(text string) string {
	if h == nil || h.textFilter == nil {
		return text
	}
	return h.textFilter.Clean(text)
}

func (h *AppHandler) cleanTexts(values []string) []string {
	if h == nil || h.textFilter == nil {
		return values
	}
	return h.textFilter.CleanSlice(values)
}

func (h *AppHandler) WechatLogin(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "code is required")
		return
	}

	openid, unionid, err := h.code2Session(c.Request.Context(), req.Code)
	if err != nil {
		response.Error(c, http.StatusBadGateway, response.CodeServerError, err.Error())
		return
	}

	user, err := h.upsertUserByOpenID(openid, unionid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}

	userID, _ := user["id"].(int64)
	token, err := h.issueToken(userID, openid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"token": token, "user": user})
}

func (h *AppHandler) Me(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	user, err := h.userByID(userID)
	if err != nil {
		h.sqlError(c, err)
		return
	}
	response.Success(c, user)
}

func (h *AppHandler) UpdateMe(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}

	var req struct {
		Nickname        string   `json:"nickname"`
		AvatarURL       string   `json:"avatarUrl"`
		Gender          int      `json:"gender"`
		GenderVisible   bool     `json:"genderVisible"`
		Phone           string   `json:"phone"`
		City            string   `json:"city"`
		SkiType         string   `json:"skiType"`
		SkiLevel        string   `json:"skiLevel"`
		StyleTags       []string `json:"styleTags"`
		FavoriteResorts []string `json:"favoriteResorts"`
		HasCar          bool     `json:"hasCar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid profile")
		return
	}
	req.Nickname = h.cleanText(req.Nickname)
	req.City = h.cleanText(req.City)
	req.SkiType = h.cleanText(req.SkiType)
	req.SkiLevel = h.cleanText(req.SkiLevel)
	req.StyleTags = h.cleanTexts(req.StyleTags)
	req.FavoriteResorts = h.cleanTexts(req.FavoriteResorts)
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldUserNickname, req.Nickname); err != nil {
		h.contentError(c, err)
		return
	}
	styleTags, _ := json.Marshal(req.StyleTags)
	favoriteResorts, _ := json.Marshal(req.FavoriteResorts)
	_, err := h.db.Exec(`UPDATE users SET nickname=?, avatar_url=?, gender=?, gender_visible=?, phone=?, city=?, ski_type=?, ski_level=?, style_tags=?, favorite_resorts=?, has_car=? WHERE id=?`,
		defaultString(req.Nickname, "雪友"), req.AvatarURL, req.Gender, boolInt(req.GenderVisible), req.Phone, req.City, req.SkiType, req.SkiLevel, string(styleTags), string(favoriteResorts), boolInt(req.HasCar), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	user, _ := h.userByID(userID)
	response.Success(c, user)
}

func (h *AppHandler) Events(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	page, pageSize := pagination(c)
	where := []string{"e.deleted_at IS NULL", "e.status <> 'removed'"}
	args := []interface{}{}

	filters := map[string]string{
		"city":           "e.depart_city",
		"resortId":       "e.resort_id",
		"skiType":        "e.ski_type_req",
		"level":          "e.level_req",
		"trafficType":    "e.traffic_type",
		"allowCarPool":   "e.allow_car_pool",
		"allowRoomShare": "e.allow_room_share",
		"sameGenderOnly": "e.same_gender_only",
	}
	for key, column := range filters {
		if value := strings.TrimSpace(c.Query(key)); value != "" {
			if strings.HasPrefix(key, "allow") || key == "sameGenderOnly" {
				where = append(where, column+" = ?")
				args = append(args, boolQuery(value))
				continue
			}
			where = append(where, column+" = ?")
			args = append(args, value)
		}
	}
	if value := strings.TrimSpace(c.Query("date")); value != "" {
		where = append(where, "e.event_date = ?")
		args = append(args, value)
	}
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		where = append(where, "(e.title LIKE ? OR e.resort_name LIKE ? OR e.depart_area LIKE ? OR e.remark LIKE ?)")
		like := "%" + keyword + "%"
		args = append(args, like, like, like, like)
	}

	order := "e.created_at DESC"
	if c.Query("sort") == "recommend" {
		order = "e.current_members DESC, e.created_at DESC"
	}

	data, total, err := h.eventPage(where, args, order, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, pageData(data, page, pageSize, total))
}

func (h *AppHandler) EventDetail(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	id := c.Param("id")
	event, err := h.eventByID(id)
	if err != nil {
		h.sqlError(c, err)
		return
	}
	members, _ := h.eventMembers(id)
	_, _ = h.db.Exec(`UPDATE ski_events SET view_count=view_count+1 WHERE id=?`, id)
	event["members"] = members
	response.Success(c, event)
}

func (h *AppHandler) CreateEvent(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var req eventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid event")
		return
	}
	h.cleanEventRequest(&req)
	if req.Title == "" {
		req.Title = req.ResortName
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventTitle, req.Title); err != nil {
		h.contentError(c, err)
		return
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, req.Remark); err != nil {
		h.contentError(c, err)
		return
	}
	tags, _ := json.Marshal(req.PurposeTags)
	resortName := req.ResortName
	if resortName == "" && req.ResortID > 0 {
		_ = h.db.QueryRow(`SELECT name FROM ski_resorts WHERE id=?`, req.ResortID).Scan(&resortName)
	}
	if resortName == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "resort is required")
		return
	}
	result, err := h.db.Exec(`INSERT INTO ski_events
		(title, creator_id, resort_id, resort_name, event_date, start_time, depart_city, depart_area, meet_place, traffic_type, max_members, current_members, ski_type_req, level_req, purpose_tags, allow_beginner, same_gender_only, allow_car_pool, allow_room_share, allow_photo, cost_desc, remark, image_url, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'recruiting')`,
		req.Title, userID, req.ResortID, resortName, nullString(req.EventDate), nullString(req.StartTime), req.DepartCity, req.DepartArea, req.MeetPlace, req.TrafficType, maxInt(req.MaxMembers, 1), req.SkiTypeReq, req.LevelReq, string(tags), boolInt(req.AllowBeginner), boolInt(req.SameGenderOnly), boolInt(req.AllowCarPool), boolInt(req.AllowRoomShare), boolInt(req.AllowPhoto), req.CostDesc, req.Remark, req.ImageURL)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	eventID, _ := result.LastInsertId()
	_, _ = h.db.Exec(`INSERT INTO event_members (event_id, user_id, role, status) VALUES (?, ?, 'creator', 'active')`, eventID, userID)
	_, _ = h.db.Exec(`UPDATE users SET event_count=event_count+1 WHERE id=?`, userID)
	event, _ := h.eventByID(strconv.FormatInt(eventID, 10))
	response.Success(c, event)
}

func (h *AppHandler) UploadEventImage(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	h.uploadImage(c, userID, "events")
}

func (h *AppHandler) UploadAvatar(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	h.uploadImage(c, userID, "avatars")
}

func (h *AppHandler) uploadImage(c *gin.Context, userID int64, folder string) {
	file, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "image is required")
		return
	}
	if file.Size > 5*1024*1024 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "image must be smaller than 5MB")
		return
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "unsupported image type")
		return
	}

	dir := filepath.Join("uploads", folder)
	if err := os.MkdirAll(dir, 0755); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	filename := fmt.Sprintf("%d-%d%s", userID, time.Now().UnixNano(), ext)
	if err := c.SaveUploadedFile(file, filepath.Join(dir, filename)); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}

	path := "/uploads/" + folder + "/" + filename
	scheme := "http"
	if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	} else if c.Request.TLS != nil {
		scheme = "https"
	}
	response.Success(c, gin.H{"url": fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, path), "path": path})
}

func (h *AppHandler) UpdateEvent(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var creatorID int64
	if err := h.db.QueryRow(`SELECT creator_id FROM ski_events WHERE id=? AND deleted_at IS NULL`, c.Param("id")).Scan(&creatorID); err != nil {
		h.sqlError(c, err)
		return
	}
	if creatorID != userID {
		response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only creator can update event")
		return
	}
	var req eventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid event")
		return
	}
	h.cleanEventRequest(&req)
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventTitle, req.Title); err != nil {
		h.contentError(c, err)
		return
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, req.Remark); err != nil {
		h.contentError(c, err)
		return
	}
	tags, _ := json.Marshal(req.PurposeTags)
	_, err := h.db.Exec(`UPDATE ski_events SET title=?, resort_id=?, resort_name=?, event_date=?, start_time=?, depart_city=?, depart_area=?, meet_place=?, traffic_type=?, max_members=?, ski_type_req=?, level_req=?, purpose_tags=?, allow_beginner=?, same_gender_only=?, allow_car_pool=?, allow_room_share=?, allow_photo=?, cost_desc=?, remark=?, image_url=? WHERE id=?`,
		req.Title, req.ResortID, req.ResortName, nullString(req.EventDate), nullString(req.StartTime), req.DepartCity, req.DepartArea, req.MeetPlace, req.TrafficType, maxInt(req.MaxMembers, 1), req.SkiTypeReq, req.LevelReq, string(tags), boolInt(req.AllowBeginner), boolInt(req.SameGenderOnly), boolInt(req.AllowCarPool), boolInt(req.AllowRoomShare), boolInt(req.AllowPhoto), req.CostDesc, req.Remark, req.ImageURL, c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	event, _ := h.eventByID(c.Param("id"))
	response.Success(c, event)
}

func (h *AppHandler) DeleteEvent(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer tx.Rollback()

	result, err := tx.Exec(`UPDATE ski_events SET deleted_at=NOW(), status='cancelled' WHERE id=? AND creator_id=? AND deleted_at IS NULL`, c.Param("id"), userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only creator can delete event")
		return
	}
	if _, err := tx.Exec(`UPDATE event_members SET status='removed' WHERE event_id=?`, c.Param("id")); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE users SET event_count=CASE WHEN event_count > 0 THEN event_count-1 ELSE 0 END WHERE id=?`, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *AppHandler) SetEventStatus(status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireDB(c) {
			return
		}
		userID, ok := currentUserID(c)
		if !ok || !h.ensureActive(c, userID) {
			return
		}
		result, err := h.db.Exec(`UPDATE ski_events SET status=? WHERE id=? AND creator_id=?`, status, c.Param("id"), userID)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		if rows, _ := result.RowsAffected(); rows == 0 {
			response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only creator can update event")
			return
		}
		response.Success(c, gin.H{"status": status})
	}
}

func (h *AppHandler) ApplyEvent(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var req struct {
		SkiLevel       string `json:"skiLevel"`
		SkiType        string `json:"skiType"`
		HasCar         bool   `json:"hasCar"`
		CanCarryPeople bool   `json:"canCarryPeople"`
		DepartArea     string `json:"departArea"`
		Message        string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Message) == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "message is required")
		return
	}
	req.DepartArea = h.cleanText(req.DepartArea)
	req.Message = h.cleanText(req.Message)
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, req.Message); err != nil {
		h.contentError(c, err)
		return
	}
	var creatorID int64
	var status string
	if err := h.db.QueryRow(`SELECT creator_id, status FROM ski_events WHERE id=? AND deleted_at IS NULL`, c.Param("id")).Scan(&creatorID, &status); err != nil {
		h.sqlError(c, err)
		return
	}
	if creatorID == userID {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "creator cannot apply")
		return
	}
	if status != "recruiting" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "event is not recruiting")
		return
	}
	result, err := h.db.Exec(`INSERT INTO join_requests (event_id, applicant_id, creator_id, ski_level, ski_type, has_car, can_carry_people, depart_area, message, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
		c.Param("id"), userID, creatorID, req.SkiLevel, req.SkiType, boolInt(req.HasCar), boolInt(req.CanCarryPeople), req.DepartArea, req.Message)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "application already exists or cannot be created")
		return
	}
	id, _ := result.LastInsertId()
	response.Success(c, gin.H{"id": id, "status": "pending"})
}

func (h *AppHandler) MyJoinRequests(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	rows, err := h.db.Query(`SELECT r.id, r.event_id, e.title, e.resort_name, r.ski_level, r.ski_type, r.depart_area, r.message, r.status, r.reject_reason, r.created_at
		FROM join_requests r JOIN ski_events e ON e.id=r.event_id WHERE r.applicant_id=? AND e.deleted_at IS NULL ORDER BY r.created_at DESC`, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, eventID int64
		var title, resortName, skiLevel, skiType, departArea, message, status, rejectReason string
		var created time.Time
		_ = rows.Scan(&id, &eventID, &title, &resortName, &skiLevel, &skiType, &departArea, &message, &status, &rejectReason, &created)
		list = append(list, gin.H{"id": id, "eventId": eventID, "eventTitle": title, "resort": resortName, "level": skiLevel, "skiType": skiType, "departArea": departArea, "message": message, "status": status, "rejectReason": rejectReason, "createdAt": created})
	}
	response.Success(c, list)
}

func (h *AppHandler) EventApplications(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	var creatorID int64
	if err := h.db.QueryRow(`SELECT creator_id FROM ski_events WHERE id=?`, c.Param("id")).Scan(&creatorID); err != nil {
		h.sqlError(c, err)
		return
	}
	if creatorID != userID {
		response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only creator can view applications")
		return
	}
	list, err := h.joinRequests("r.event_id=?", []interface{}{c.Param("id")}, 1, 200)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, list)
}

func (h *AppHandler) ReviewJoinRequest(status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireDB(c) {
			return
		}
		userID, ok := currentUserID(c)
		if !ok || !h.ensureActive(c, userID) {
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&req)
		req.Reason = h.cleanText(req.Reason)
		var eventID, applicantID, creatorID int64
		var currentStatus string
		if err := h.db.QueryRow(`SELECT event_id, applicant_id, creator_id, status FROM join_requests WHERE id=?`, c.Param("id")).Scan(&eventID, &applicantID, &creatorID, &currentStatus); err != nil {
			h.sqlError(c, err)
			return
		}
		if creatorID != userID {
			response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only creator can review application")
			return
		}
		if currentStatus != "pending" {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "application is not pending")
			return
		}
		tx, err := h.db.Begin()
		if err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		defer tx.Rollback()
		if _, err = tx.Exec(`UPDATE join_requests SET status=?, reject_reason=? WHERE id=?`, status, req.Reason, c.Param("id")); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		if status == "approved" {
			if _, err = tx.Exec(`INSERT INTO event_members (event_id, user_id, role, status) VALUES (?, ?, 'member', 'active') ON DUPLICATE KEY UPDATE status='active'`, eventID, applicantID); err != nil {
				response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
				return
			}
			if _, err = tx.Exec(`UPDATE ski_events SET current_members=current_members+1, status=IF(current_members+1 >= max_members, 'full', status) WHERE id=?`, eventID); err != nil {
				response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
				return
			}
			_, _ = tx.Exec(`UPDATE users SET join_count=join_count+1 WHERE id=?`, applicantID)
		}
		if err = tx.Commit(); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		response.Success(c, gin.H{"status": status})
	}
}

func (h *AppHandler) Trips(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireDB(c) {
			return
		}
		userID, ok := currentUserID(c)
		if !ok {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
			return
		}
		where := []string{"e.deleted_at IS NULL", "e.status <> 'removed'"}
		args := []interface{}{}
		switch kind {
		case "created":
			where = append(where, "e.creator_id=?", "e.status NOT IN ('finished', 'cancelled')")
			args = append(args, userID)
		case "joined":
			where = append(where, "m.user_id=? AND m.role='member' AND m.status='active'", "e.status NOT IN ('finished', 'cancelled')")
			args = append(args, userID)
		case "pending":
			where = append(where, "r.applicant_id=? AND r.status='pending'", "e.status NOT IN ('finished', 'cancelled')")
			args = append(args, userID)
		case "finished":
			where = append(where, "(e.status='finished' AND (e.creator_id=? OR m.user_id=?))")
			args = append(args, userID, userID)
		}
		list, _, err := h.eventPage(where, args, "e.event_date DESC, e.created_at DESC", 1, 200)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		response.Success(c, list)
	}
}

func (h *AppHandler) ChatConversations(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	rows, err := h.db.Query(`SELECT e.id, e.title, e.resort_name, e.event_date, e.start_time, e.current_members, e.max_members, e.image_url, e.status,
			COALESCE(lm.id, 0), COALESCE(lm.content, ''), COALESCE(lm.message_type, ''), lm.created_at, COALESCE(sender.nickname, ''),
			(SELECT COUNT(*) FROM chat_messages unread WHERE unread.event_id=e.id AND unread.status='normal' AND unread.id > COALESCE(r.last_read_message_id, 0) AND unread.sender_id<>?)
		FROM event_members em
		JOIN ski_events e ON e.id=em.event_id
		LEFT JOIN chat_reads r ON r.event_id=e.id AND r.user_id=?
		LEFT JOIN chat_messages lm ON lm.id=(SELECT MAX(id) FROM chat_messages last WHERE last.event_id=e.id AND last.status='normal')
		LEFT JOIN users sender ON sender.id=lm.sender_id
		WHERE em.user_id=? AND em.status='active' AND e.deleted_at IS NULL AND e.status<>'removed'
		ORDER BY COALESCE(lm.created_at, e.updated_at) DESC
		LIMIT 200`, userID, userID, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	totalUnread := int64(0)
	for rows.Next() {
		var eventID, lastID int64
		var title, resortName, imageURL, status, lastContent, messageType, senderName string
		var eventDate, startTime, lastCreated sql.NullTime
		var currentMembers, maxMembers int
		var unread int64
		if err := rows.Scan(&eventID, &title, &resortName, &eventDate, &startTime, &currentMembers, &maxMembers, &imageURL, &status, &lastID, &lastContent, &messageType, &lastCreated, &senderName, &unread); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		totalUnread += unread
		lastMessage := "还没有消息，先打个招呼吧"
		if lastID > 0 {
			if senderName == "" {
				senderName = "雪友"
			}
			lastMessage = senderName + "：" + lastContent
			if messageType == "system" {
				lastMessage = lastContent
			}
		}
		conversationTime := ""
		if lastCreated.Valid {
			conversationTime = lastCreated.Time.Format("15:04")
		} else if startTime.Valid {
			conversationTime = startTime.Time.Format("01-02")
		} else if eventDate.Valid {
			conversationTime = eventDate.Time.Format("01-02")
		}
		list = append(list, gin.H{
			"id":            fmt.Sprintf("conv-%d", eventID),
			"eventId":       eventID,
			"title":         defaultString(resortName, title),
			"image":         imageURL,
			"time":          conversationTime,
			"lastMessage":   lastMessage,
			"lastMessageId": lastID,
			"status":        status,
			"memberText":    fmt.Sprintf("已%d人，缺%d人", currentMembers, maxInt(maxMembers-currentMembers, 0)),
			"unread":        unread,
		})
	}
	response.Success(c, gin.H{"list": list, "unreadCount": totalUnread})
}

func (h *AppHandler) Messages(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	if !h.canAccessEvent(c, c.Param("id")) {
		return
	}
	rows, err := h.db.Query(`SELECT m.id, m.sender_id, u.nickname, u.avatar_url, m.message_type, m.content, m.created_at
		FROM chat_messages m JOIN users u ON u.id=m.sender_id
		WHERE m.event_id=? AND m.status='normal' ORDER BY m.id ASC LIMIT 200`, c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, senderID int64
		var nickname, avatarURL, messageType, content string
		var created time.Time
		_ = rows.Scan(&id, &senderID, &nickname, &avatarURL, &messageType, &content, &created)
		list = append(list, gin.H{"id": id, "senderId": senderID, "nickname": nickname, "avatarUrl": avatarURL, "messageType": messageType, "content": content, "createdAt": created, "time": created.Format("15:04")})
	}
	response.Success(c, list)
}

func (h *AppHandler) MarkChatRead(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.canAccessEvent(c, c.Param("id")) {
		return
	}
	if err := h.markChatRead(c.Param("id"), userID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"read": true})
}

func (h *AppHandler) MarkAllChatsRead(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	_, err := h.db.Exec(`INSERT INTO chat_reads (event_id, user_id, last_read_message_id)
		SELECT em.event_id, ?, COALESCE(MAX(m.id), 0)
		FROM event_members em
		JOIN ski_events e ON e.id=em.event_id
		LEFT JOIN chat_messages m ON m.event_id=em.event_id AND m.status='normal'
		WHERE em.user_id=? AND em.status='active' AND e.deleted_at IS NULL AND e.status<>'removed'
		GROUP BY em.event_id
		ON DUPLICATE KEY UPDATE last_read_message_id=VALUES(last_read_message_id)`, userID, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"read": true})
}

func (h *AppHandler) SendMessage(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) || !h.canAccessEvent(c, c.Param("id")) {
		return
	}
	var req struct {
		MessageType string `json:"messageType"`
		Content     string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.Content) == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "content is required")
		return
	}
	req.Content = h.cleanText(req.Content)
	if req.MessageType == "" {
		req.MessageType = "text"
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldChatMessage, req.Content); err != nil {
		h.contentError(c, err)
		return
	}
	result, err := h.db.Exec(`INSERT INTO chat_messages (event_id, sender_id, message_type, content) VALUES (?, ?, ?, ?)`, c.Param("id"), userID, req.MessageType, req.Content)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	id, _ := result.LastInsertId()
	_ = h.markChatRead(c.Param("id"), userID)
	response.Success(c, gin.H{"id": id, "messageType": req.MessageType, "content": req.Content})
}

func (h *AppHandler) CreateReview(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var req struct {
		EventID      int64    `json:"eventId"`
		RevieweeID   int64    `json:"revieweeId"`
		Score        int      `json:"score"`
		PositiveTags []string `json:"positiveTags"`
		NegativeTags []string `json:"negativeTags"`
		Content      string   `json:"content"`
		IsAnonymous  bool     `json:"isAnonymous"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.EventID == 0 || req.RevieweeID == 0 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid review")
		return
	}
	req.PositiveTags = h.cleanTexts(req.PositiveTags)
	req.NegativeTags = h.cleanTexts(req.NegativeTags)
	req.Content = h.cleanText(req.Content)
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldReview, req.Content); err != nil {
		h.contentError(c, err)
		return
	}
	if req.Score < 1 || req.Score > 5 {
		req.Score = 5
	}
	positive, _ := json.Marshal(req.PositiveTags)
	negative, _ := json.Marshal(req.NegativeTags)
	result, err := h.db.Exec(`INSERT INTO reviews (event_id, reviewer_id, reviewee_id, score, positive_tags, negative_tags, content, is_anonymous) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.EventID, userID, req.RevieweeID, req.Score, string(positive), string(negative), req.Content, boolInt(req.IsAnonymous))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	id, _ := result.LastInsertId()
	response.Success(c, gin.H{"id": id})
}

func (h *AppHandler) UserReviews(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	rows, err := h.db.Query(`SELECT r.id, r.event_id, e.title, r.reviewer_id, u.nickname, r.score, r.positive_tags, r.negative_tags, r.content, r.is_anonymous, r.created_at
		FROM reviews r JOIN ski_events e ON e.id=r.event_id JOIN users u ON u.id=r.reviewer_id
		WHERE r.reviewee_id=? AND r.status='normal' ORDER BY r.created_at DESC`, c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, eventID, reviewerID int64
		var title, nickname, positive, negative, content string
		var score, anonymous int
		var created time.Time
		_ = rows.Scan(&id, &eventID, &title, &reviewerID, &nickname, &score, &positive, &negative, &content, &anonymous, &created)
		if anonymous == 1 {
			nickname = "匿名雪友"
			reviewerID = 0
		}
		list = append(list, gin.H{"id": id, "eventId": eventID, "eventTitle": title, "reviewerId": reviewerID, "reviewerName": nickname, "score": score, "positiveTags": jsonList(positive), "negativeTags": jsonList(negative), "content": content, "createdAt": created})
	}
	response.Success(c, list)
}

func (h *AppHandler) CreateReport(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var req struct {
		TargetType string `json:"targetType"`
		TargetID   int64  `json:"targetId"`
		Reason     string `json:"reason"`
		Content    string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TargetType == "" || strings.TrimSpace(req.Content) == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid report")
		return
	}
	req.Reason = h.cleanText(req.Reason)
	req.Content = h.cleanText(req.Content)
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldReport, req.Content); err != nil {
		h.contentError(c, err)
		return
	}
	result, err := h.db.Exec(`INSERT INTO reports (reporter_id, target_type, target_id, reason, content) VALUES (?, ?, ?, ?, ?)`, userID, req.TargetType, req.TargetID, req.Reason, req.Content)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	id, _ := result.LastInsertId()
	response.Success(c, gin.H{"id": id, "status": "pending"})
}

func (h *AppHandler) Resorts(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	rows, err := h.db.Query(`SELECT id, name, city, province, image_url, status, sort FROM ski_resorts WHERE status='normal' ORDER BY sort ASC, id ASC`)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, sort int64
		var name, city, province, imageURL, status string
		_ = rows.Scan(&id, &name, &city, &province, &imageURL, &status, &sort)
		list = append(list, gin.H{"id": id, "name": name, "city": city, "province": province, "imageUrl": imageURL, "status": status, "sort": sort})
	}
	response.Success(c, list)
}

func (h *AppHandler) Tags(c *gin.Context) {
	response.Success(c, gin.H{
		"skiTypes":      []string{"snowboard", "ski", "both"},
		"levels":        []string{"beginner", "primary", "intermediate", "advanced"},
		"purposeTags":   []string{"刷道", "练习", "平花", "公园", "拍照", "休闲滑"},
		"trafficTypes":  []string{"self_drive", "high_speed_rail", "bus", "other"},
		"positiveTags":  []string{"准时", "友好", "水平真实", "沟通顺畅", "安全意识好", "愿意再次同滑"},
		"negativeTags":  []string{"爽约", "迟到严重", "水平虚假", "临时改计划", "言语不适", "危险行为"},
		"reportReasons": []string{"骚扰或不适内容", "虚假行程", "爽约风险", "危险行为", "其他问题"},
	})
}

func (h *AppHandler) Cities(c *gin.Context) {
	response.Success(c, []string{"北京", "张家口", "崇礼", "吉林", "哈尔滨", "乌鲁木齐"})
}

type eventRequest struct {
	Title          string   `json:"title"`
	ResortID       int64    `json:"resortId"`
	ResortName     string   `json:"resortName"`
	EventDate      string   `json:"eventDate"`
	StartTime      string   `json:"startTime"`
	DepartCity     string   `json:"departCity"`
	DepartArea     string   `json:"departArea"`
	MeetPlace      string   `json:"meetPlace"`
	TrafficType    string   `json:"trafficType"`
	MaxMembers     int      `json:"maxMembers"`
	SkiTypeReq     string   `json:"skiTypeReq"`
	LevelReq       string   `json:"levelReq"`
	PurposeTags    []string `json:"purposeTags"`
	AllowBeginner  bool     `json:"allowBeginner"`
	SameGenderOnly bool     `json:"sameGenderOnly"`
	AllowCarPool   bool     `json:"allowCarPool"`
	AllowRoomShare bool     `json:"allowRoomShare"`
	AllowPhoto     bool     `json:"allowPhoto"`
	CostDesc       string   `json:"costDesc"`
	Remark         string   `json:"remark"`
	ImageURL       string   `json:"imageUrl"`
}

func (h *AppHandler) cleanEventRequest(req *eventRequest) {
	if req == nil {
		return
	}
	req.Title = h.cleanText(req.Title)
	req.ResortName = h.cleanText(req.ResortName)
	req.DepartCity = h.cleanText(req.DepartCity)
	req.DepartArea = h.cleanText(req.DepartArea)
	req.MeetPlace = h.cleanText(req.MeetPlace)
	req.CostDesc = h.cleanText(req.CostDesc)
	req.Remark = h.cleanText(req.Remark)
	req.PurposeTags = h.cleanTexts(req.PurposeTags)
}

func (h *AppHandler) code2Session(ctx context.Context, code string) (string, string, error) {
	if h.cfg.AppEnv == "development" && strings.HasPrefix(code, "dev-") {
		return "dev-openid-" + strings.TrimPrefix(code, "dev-"), "", nil
	}
	if h.cfg.WechatAppID == "" || h.cfg.WechatAppSecret == "" {
		return "", "", errors.New("WECHAT_APP_ID and WECHAT_APP_SECRET are required")
	}
	url := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code", h.cfg.WechatAppID, h.cfg.WechatAppSecret, code)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", "", err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	var body struct {
		OpenID  string `json:"openid"`
		UnionID string `json:"unionid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", "", err
	}
	if body.ErrCode != 0 || body.OpenID == "" {
		return "", "", fmt.Errorf("wechat login failed %d: %s", body.ErrCode, body.ErrMsg)
	}
	return body.OpenID, body.UnionID, nil
}

func (h *AppHandler) issueToken(userID int64, openid string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"openid":  openid,
		"exp":     time.Now().Add(time.Duration(h.cfg.JWTExpiresHours) * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(h.cfg.JWTSecret))
}

func (h *AppHandler) upsertUserByOpenID(openid, unionid string) (gin.H, error) {
	_, err := h.db.Exec(`INSERT INTO users (openid, unionid, nickname) VALUES (?, ?, '雪友') ON DUPLICATE KEY UPDATE unionid=IF(VALUES(unionid)='', unionid, VALUES(unionid))`, openid, unionid)
	if err != nil {
		return nil, err
	}
	var id int64
	if err := h.db.QueryRow(`SELECT id FROM users WHERE openid=?`, openid).Scan(&id); err != nil {
		return nil, err
	}
	return h.userByID(id)
}

func (h *AppHandler) userByID(id int64) (gin.H, error) {
	var styleTags, favoriteResorts sql.NullString
	var user userRow
	var phone sql.NullString
	err := h.db.QueryRow(`SELECT id, openid, phone, nickname, avatar_url, gender, gender_visible, city, ski_type, ski_level, style_tags, favorite_resorts, has_car, credit_score, event_count, join_count, good_rate, status, created_at, updated_at FROM users WHERE id=? AND deleted_at IS NULL`, id).
		Scan(&user.ID, &user.OpenID, &phone, &user.Nickname, &user.AvatarURL, &user.Gender, &user.GenderVisible, &user.City, &user.SkiType, &user.SkiLevel, &styleTags, &favoriteResorts, &user.HasCar, &user.CreditScore, &user.EventCount, &user.JoinCount, &user.GoodRate, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return gin.H{"id": user.ID, "openid": user.OpenID, "phone": phone.String, "nickname": user.Nickname, "avatarUrl": user.AvatarURL, "gender": user.Gender, "genderVisible": user.GenderVisible == 1, "city": user.City, "skiType": user.SkiType, "skiLevel": user.SkiLevel, "styleTags": jsonList(styleTags.String), "favoriteResorts": jsonList(favoriteResorts.String), "hasCar": user.HasCar == 1, "creditScore": user.CreditScore, "eventCount": user.EventCount, "joinCount": user.JoinCount, "goodRate": user.GoodRate, "status": user.Status, "createdAt": user.CreatedAt, "updatedAt": user.UpdatedAt}, nil
}

func (h *AppHandler) eventPage(where []string, args []interface{}, order string, page, pageSize int) ([]gin.H, int64, error) {
	join := ""
	if strings.Contains(strings.Join(where, " "), "m.") {
		join += " LEFT JOIN event_members m ON m.event_id=e.id"
	}
	if strings.Contains(strings.Join(where, " "), "r.") {
		join += " LEFT JOIN join_requests r ON r.event_id=e.id"
	}
	whereSQL := " WHERE " + strings.Join(where, " AND ")
	var total int64
	if err := h.db.QueryRow(`SELECT COUNT(DISTINCT e.id) FROM ski_events e`+join+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, (page-1)*pageSize, pageSize)
	rows, err := h.db.Query(`SELECT DISTINCT e.id, e.title, e.creator_id, u.nickname, e.resort_id, e.resort_name, e.event_date, e.start_time, e.depart_city, e.depart_area, e.meet_place, e.traffic_type, e.max_members, e.current_members, e.ski_type_req, e.level_req, e.purpose_tags, e.allow_beginner, e.same_gender_only, e.allow_car_pool, e.allow_room_share, e.allow_photo, e.cost_desc, e.remark, e.image_url, e.status, e.view_count, e.created_at, e.updated_at
		FROM ski_events e JOIN users u ON u.id=e.creator_id`+join+whereSQL+` ORDER BY `+order+` LIMIT ?, ?`, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, 0, err
		}
		list = append(list, event)
	}
	return list, total, nil
}

func (h *AppHandler) eventByID(id string) (gin.H, error) {
	row := h.db.QueryRow(`SELECT e.id, e.title, e.creator_id, u.nickname, e.resort_id, e.resort_name, e.event_date, e.start_time, e.depart_city, e.depart_area, e.meet_place, e.traffic_type, e.max_members, e.current_members, e.ski_type_req, e.level_req, e.purpose_tags, e.allow_beginner, e.same_gender_only, e.allow_car_pool, e.allow_room_share, e.allow_photo, e.cost_desc, e.remark, e.image_url, e.status, e.view_count, e.created_at, e.updated_at
		FROM ski_events e JOIN users u ON u.id=e.creator_id WHERE e.id=? AND e.deleted_at IS NULL`, id)
	return scanEvent(row)
}

func (h *AppHandler) eventMembers(id string) ([]gin.H, error) {
	rows, err := h.db.Query(`SELECT m.user_id, u.nickname, u.avatar_url, u.ski_type, u.ski_level, u.credit_score, m.role, m.status, m.joined_at
		FROM event_members m JOIN users u ON u.id=m.user_id WHERE m.event_id=? AND m.status='active' ORDER BY m.role='creator' DESC, m.joined_at ASC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var userID int64
		var nickname, avatarURL, skiType, skiLevel, role, status string
		var credit float64
		var joined time.Time
		_ = rows.Scan(&userID, &nickname, &avatarURL, &skiType, &skiLevel, &credit, &role, &status, &joined)
		list = append(list, gin.H{"id": userID, "userId": userID, "nickname": nickname, "avatarUrl": avatarURL, "skiType": skiType, "skiLevel": skiLevel, "creditScore": credit, "role": role, "status": status, "joinedAt": joined})
	}
	return list, nil
}

func scanEvent(scanner interface{ Scan(...interface{}) error }) (gin.H, error) {
	var id, creatorID, resortID int64
	var title, creatorName, resortName, departCity, departArea, meetPlace, trafficType, skiTypeReq, levelReq, tags, costDesc, imageURL, status string
	var eventDate, startTime sql.NullTime
	var remark sql.NullString
	var maxMembers, currentMembers, allowBeginner, sameGenderOnly, allowCarPool, allowRoomShare, allowPhoto, viewCount int
	var createdAt, updatedAt time.Time
	err := scanner.Scan(&id, &title, &creatorID, &creatorName, &resortID, &resortName, &eventDate, &startTime, &departCity, &departArea, &meetPlace, &trafficType, &maxMembers, &currentMembers, &skiTypeReq, &levelReq, &tags, &allowBeginner, &sameGenderOnly, &allowCarPool, &allowRoomShare, &allowPhoto, &costDesc, &remark, &imageURL, &status, &viewCount, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return gin.H{"id": id, "title": title, "creatorId": creatorID, "creatorName": creatorName, "resortId": resortID, "resortName": resortName, "eventDate": timeString(eventDate, "2006-01-02"), "startTime": timeString(startTime, "2006-01-02 15:04:05"), "departCity": departCity, "departArea": departArea, "meetPlace": meetPlace, "trafficType": trafficType, "maxMembers": maxMembers, "currentMembers": currentMembers, "skiTypeReq": skiTypeReq, "levelReq": levelReq, "purposeTags": jsonList(tags), "allowBeginner": allowBeginner == 1, "sameGenderOnly": sameGenderOnly == 1, "allowCarPool": allowCarPool == 1, "allowRoomShare": allowRoomShare == 1, "allowPhoto": allowPhoto == 1, "costDesc": costDesc, "remark": remark.String, "imageUrl": imageURL, "status": status, "viewCount": viewCount, "createdAt": createdAt, "updatedAt": updatedAt}, nil
}

type userRow struct {
	ID            int64
	OpenID        string
	Nickname      string
	AvatarURL     string
	Gender        int
	GenderVisible int
	City          string
	SkiType       string
	SkiLevel      string
	HasCar        int
	CreditScore   float64
	EventCount    int
	JoinCount     int
	GoodRate      float64
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (h *AppHandler) joinRequests(where string, args []interface{}, page, pageSize int) ([]gin.H, error) {
	args = append(args, (page-1)*pageSize, pageSize)
	rows, err := h.db.Query(`SELECT r.id, r.event_id, e.title, e.resort_name, r.applicant_id, u.nickname, u.avatar_url, u.credit_score, r.ski_level, r.ski_type, r.has_car, r.can_carry_people, r.depart_area, r.message, r.status, r.reject_reason, r.created_at
		FROM join_requests r JOIN ski_events e ON e.id=r.event_id JOIN users u ON u.id=r.applicant_id WHERE `+where+` ORDER BY r.created_at DESC LIMIT ?, ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []gin.H{}
	for rows.Next() {
		var id, eventID, applicantID int64
		var title, resortName, nickname, avatarURL, skiLevel, skiType, departArea, message, status, rejectReason string
		var hasCar, canCarryPeople int
		var credit float64
		var created time.Time
		_ = rows.Scan(&id, &eventID, &title, &resortName, &applicantID, &nickname, &avatarURL, &credit, &skiLevel, &skiType, &hasCar, &canCarryPeople, &departArea, &message, &status, &rejectReason, &created)
		list = append(list, gin.H{"id": id, "eventId": eventID, "eventTitle": title, "resortName": resortName, "applicantId": applicantID, "name": nickname, "nickname": nickname, "avatarUrl": avatarURL, "creditScore": credit, "level": skiLevel, "skiLevel": skiLevel, "skiType": skiType, "hasCar": hasCar == 1, "canCarryPeople": canCarryPeople == 1, "departArea": departArea, "message": message, "status": status, "rejectReason": rejectReason, "createdAt": created})
	}
	return list, nil
}

func (h *AppHandler) requireDB(c *gin.Context) bool {
	if h.db != nil {
		return true
	}
	response.Error(c, http.StatusServiceUnavailable, response.CodeServerError, "database is not configured")
	return false
}

func (h *AppHandler) ensureActive(c *gin.Context, userID int64) bool {
	var status string
	if err := h.db.QueryRow(`SELECT status FROM users WHERE id=?`, userID).Scan(&status); err != nil {
		h.sqlError(c, err)
		return false
	}
	if status == "disabled" {
		response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "user is disabled")
		return false
	}
	return true
}

func (h *AppHandler) canAccessEvent(c *gin.Context, eventID string) bool {
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return false
	}
	var count int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM event_members WHERE event_id=? AND user_id=? AND status='active'`, eventID, userID).Scan(&count); err != nil {
		h.sqlError(c, err)
		return false
	}
	if count == 0 {
		response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only members can access event chat")
		return false
	}
	return true
}

func (h *AppHandler) markChatRead(eventID string, userID int64) error {
	var lastID int64
	if err := h.db.QueryRow(`SELECT COALESCE(MAX(id), 0) FROM chat_messages WHERE event_id=? AND status='normal'`, eventID).Scan(&lastID); err != nil {
		return err
	}
	_, err := h.db.Exec(`INSERT INTO chat_reads (event_id, user_id, last_read_message_id) VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE last_read_message_id=GREATEST(last_read_message_id, VALUES(last_read_message_id))`, eventID, userID, lastID)
	return err
}

func (h *AppHandler) checkText(ctx context.Context, userID int64, field compliance.TextField, content string) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	var openid string
	_ = h.db.QueryRow(`SELECT openid FROM users WHERE id=?`, userID).Scan(&openid)
	return h.security.CheckText(ctx, contentsecurity.TextCheckRequest{OpenID: openid, Scene: 2, Field: field, Content: content})
}

func (h *AppHandler) contentError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	if err.Error() == compliance.ContentRiskMessage {
		response.ContentRisk(c)
		return
	}
	response.Error(c, http.StatusBadGateway, response.CodeServerError, err.Error())
}

func (h *AppHandler) sqlError(c *gin.Context, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "not found")
		return
	}
	response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
}

func currentUserID(c *gin.Context) (int64, bool) {
	value, ok := c.Get(middleware.ContextUserID)
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case string:
		id, err := strconv.ParseInt(v, 10, 64)
		return id, err == nil
	default:
		return 0, false
	}
}

func pagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 10
	}
	return page, pageSize
}

func pageData(list interface{}, page, pageSize int, total int64) gin.H {
	return gin.H{"list": list, "page": page, "pageSize": pageSize, "total": total}
}

func jsonList(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []string{}
	}
	return list
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func boolQuery(value string) int {
	value = strings.ToLower(value)
	if value == "true" || value == "1" || value == "yes" {
		return 1
	}
	return 0
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func maxInt(value, min int) int {
	if value < min {
		return min
	}
	return value
}

func nullString(value string) interface{} {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func timeString(value sql.NullTime, layout string) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(layout)
}
