package handler

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net"
	"net/http"
	"net/url"
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
	"xueju/backend/internal/phoneverification"
	"xueju/backend/internal/response"
	"xueju/backend/internal/storage"
	"xueju/backend/internal/textfilter"
)

type AppHandler struct {
	cfg           config.Config
	db            *sql.DB
	security      *contentsecurity.Service
	textFilter    *textfilter.Filter
	client        *http.Client
	storage       storage.Store
	phoneVerifier phoneverification.Client
}

func NewAppHandler(cfg config.Config, db *sql.DB) *AppHandler {
	filter, _ := textfilter.New()
	return &AppHandler{
		cfg:           cfg,
		db:            db,
		security:      contentsecurity.New(cfg),
		textFilter:    filter,
		storage:       storage.New(cfg),
		phoneVerifier: phoneverification.NewAliyunClient(cfg),
		client:        &http.Client{Timeout: 10 * time.Second},
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

	delete(user, "openid")
	normalizeUserMediaForRequest(c, user)
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
	if !h.ensureActive(c, userID) {
		return
	}
	user, err := h.userByID(userID)
	if err != nil {
		h.sqlError(c, err)
		return
	}
	delete(user, "openid")
	normalizeUserMediaForRequest(c, user)
	response.Success(c, user)
}

func (h *AppHandler) PublicUser(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	user, err := h.userByIDParam(c.Param("id"))
	if err != nil {
		h.sqlError(c, err)
		return
	}
	if user["status"] == "disabled" {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "user not found")
		return
	}
	delete(user, "openid")
	normalizeUserMediaForRequest(c, user)
	if visible, ok := user["genderVisible"].(bool); !visible || !ok {
		user["gender"] = 0
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
	if !h.ensureActive(c, userID) {
		return
	}

	var req struct {
		Nickname        *string   `json:"nickname"`
		AvatarURL       *string   `json:"avatarUrl"`
		Bio             *string   `json:"bio"`
		Gender          *int      `json:"gender"`
		GenderVisible   *bool     `json:"genderVisible"`
		City            *string   `json:"city"`
		SkiType         *string   `json:"skiType"`
		SkiLevel        *string   `json:"skiLevel"`
		StyleTags       *[]string `json:"styleTags"`
		FavoriteResorts *[]string `json:"favoriteResorts"`
		HasCar          *bool     `json:"hasCar"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid profile")
		return
	}
	if (req.Nickname != nil && !textWithin(*req.Nickname, 64)) ||
		(req.AvatarURL != nil && !textWithin(*req.AvatarURL, 255)) ||
		(req.Bio != nil && !textWithin(*req.Bio, 500)) ||
		(req.City != nil && !textWithin(*req.City, 64)) ||
		(req.SkiType != nil && !textWithin(*req.SkiType, 32)) ||
		(req.SkiLevel != nil && !textWithin(*req.SkiLevel, 32)) ||
		(req.StyleTags != nil && !textListWithin(*req.StyleTags, 20, 32)) ||
		(req.FavoriteResorts != nil && !textListWithin(*req.FavoriteResorts, 20, 128)) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "profile field is too long")
		return
	}
	if req.Gender != nil && (*req.Gender < 0 || *req.Gender > 2) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid gender")
		return
	}
	if req.SkiType != nil && *req.SkiType != "" && !stringIn(*req.SkiType, "snowboard", "ski", "both") {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid skiType")
		return
	}
	if req.SkiLevel != nil && *req.SkiLevel != "" && !stringIn(*req.SkiLevel, "beginner", "primary", "intermediate", "advanced") {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid skiLevel")
		return
	}
	if h.rejectLocalRisk(c, pointerString(req.Nickname), pointerString(req.Bio), pointerString(req.City), strings.Join(pointerSlice(req.StyleTags), " ")) {
		return
	}
	updates := []string{}
	args := []interface{}{}
	add := func(column string, value interface{}) {
		updates = append(updates, column+"=?")
		args = append(args, value)
	}
	if req.Nickname != nil {
		value := h.cleanText(*req.Nickname)
		if err := h.checkText(c.Request.Context(), userID, compliance.FieldUserNickname, value); err != nil {
			h.contentError(c, err)
			return
		}
		add("nickname", defaultString(value, "雪友"))
	}
	if req.Bio != nil {
		value := h.cleanText(*req.Bio)
		if err := h.checkText(c.Request.Context(), userID, compliance.FieldUserBio, value); err != nil {
			h.contentError(c, err)
			return
		}
		add("bio", value)
	}
	if req.AvatarURL != nil {
		value := strings.TrimSpace(*req.AvatarURL)
		if value != "" && !h.isApprovedAvatar(userID, value) {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "avatar is not an approved upload")
			return
		}
		add("avatar_url", value)
	}
	if req.Gender != nil {
		add("gender", *req.Gender)
	}
	if req.GenderVisible != nil {
		add("gender_visible", boolInt(*req.GenderVisible))
	}
	if req.City != nil {
		value := h.cleanText(*req.City)
		if err := h.checkText(c.Request.Context(), userID, compliance.FieldUserBio, value); err != nil {
			h.contentError(c, err)
			return
		}
		add("city", value)
	}
	if req.SkiType != nil {
		add("ski_type", h.cleanText(*req.SkiType))
	}
	if req.SkiLevel != nil {
		add("ski_level", h.cleanText(*req.SkiLevel))
	}
	if req.StyleTags != nil {
		cleaned := h.cleanTexts(*req.StyleTags)
		if err := h.checkText(c.Request.Context(), userID, compliance.FieldUserBio, strings.Join(cleaned, " ")); err != nil {
			h.contentError(c, err)
			return
		}
		value, _ := json.Marshal(cleaned)
		add("style_tags", string(value))
	}
	if req.FavoriteResorts != nil {
		value, _ := json.Marshal(h.cleanTexts(*req.FavoriteResorts))
		add("favorite_resorts", string(value))
	}
	if req.HasCar != nil {
		add("has_car", boolInt(*req.HasCar))
	}
	if len(updates) == 0 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "no profile fields to update")
		return
	}
	args = append(args, userID)
	_, err := h.db.Exec(`UPDATE users SET `+strings.Join(updates, ", ")+` WHERE id=?`, args...)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	user, _ := h.userByID(userID)
	delete(user, "openid")
	normalizeUserMediaForRequest(c, user)
	response.Success(c, user)
}

func (h *AppHandler) DeleteMe(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer tx.Rollback()
	var currentStatus string
	if err := tx.QueryRow(`SELECT status FROM users WHERE id=? AND deleted_at IS NULL FOR UPDATE`, userID).Scan(&currentStatus); err != nil {
		h.sqlError(c, err)
		return
	}
	mediaPaths := []string{}
	rows, err := tx.Query(`SELECT path FROM media_uploads WHERE user_id=?`, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	for rows.Next() {
		var path string
		if rows.Scan(&path) == nil && path != "" {
			mediaPaths = append(mediaPaths, path)
		}
	}
	rows.Close()
	statements := []struct {
		query string
		args  []interface{}
	}{
		{`UPDATE ski_events e JOIN event_members m ON m.event_id=e.id
			SET e.current_members=GREATEST(1,e.current_members-1), e.status=IF(e.status='full','recruiting',e.status)
			WHERE m.user_id=? AND m.role='member' AND m.status='active' AND e.status IN ('recruiting','full')`, []interface{}{userID}},
		{`UPDATE ski_events SET status='cancelled' WHERE creator_id=? AND status IN ('recruiting','full')`, []interface{}{userID}},
		{`UPDATE join_requests SET status='rejected', reject_reason='账号已注销' WHERE status='pending' AND (applicant_id=? OR creator_id=?)`, []interface{}{userID, userID}},
		{`UPDATE event_members SET status='inactive' WHERE user_id=? AND status='active'`, []interface{}{userID}},
		{`DELETE FROM event_favorites WHERE user_id=?`, []interface{}{userID}},
		{`DELETE FROM user_follows WHERE follower_id=? OR followee_id=?`, []interface{}{userID, userID}},
		{`DELETE FROM chat_reads WHERE user_id=?`, []interface{}{userID}},
		{`DELETE FROM notifications WHERE user_id=?`, []interface{}{userID}},
		{`UPDATE media_uploads SET status='rejected', review_result='account_deleted' WHERE user_id=?`, []interface{}{userID}},
		{`UPDATE user_verifications SET status='revoked', active_phone_hash=NULL, phone_encrypted='', phone_hash='', phone_masked='', revoked_at=NOW(), revoke_reason='account_deleted' WHERE user_id=?`, []interface{}{userID}},
	}
	for _, statement := range statements {
		if _, err := tx.Exec(statement.query, statement.args...); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
	}
	deletedOpenID := fmt.Sprintf("deleted:%d:%d", userID, time.Now().UnixNano())
	if _, err := tx.Exec(`UPDATE users SET openid=?, unionid=NULL, nickname='已注销用户', avatar_url='', bio='', phone=NULL,
		gender=0, gender_visible=0, city='', ski_type='', ski_level='', style_tags=JSON_ARRAY(), favorite_resorts=JSON_ARRAY(),
		has_car=0, verification_status='revoked', verification_method='', verified_at=NULL, phone_masked='', status='disabled', deleted_at=NOW() WHERE id=?`, deletedOpenID, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	for _, path := range mediaPaths {
		_ = h.storage.Delete(c.Request.Context(), path)
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *AppHandler) Events(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	h.reconcileExpiredEvents()
	page, pageSize := pagination(c)
	now := time.Now().In(time.Local)
	today := now.Format("2006-01-02")
	where := []string{"e.deleted_at IS NULL", "e.status IN ('recruiting','full')", "(e.event_date>? OR (e.event_date=? AND (e.start_time IS NULL OR e.start_time>=?)))"}
	args := []interface{}{today, today, now}

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
	if value := strings.TrimSpace(c.Query("allowBeginner")); value != "" {
		where = append(where, "e.allow_beginner = ?")
		args = append(args, boolQuery(value))
	}
	if tags := c.QueryArray("purposeTags"); len(tags) > 0 {
		for _, tag := range tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			where = append(where, "JSON_CONTAINS(e.purpose_tags, JSON_QUOTE(?))")
			args = append(args, tag)
		}
	} else if tagList := strings.TrimSpace(c.Query("purposeTags")); tagList != "" {
		for _, tag := range strings.Split(tagList, ",") {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			where = append(where, "JSON_CONTAINS(e.purpose_tags, JSON_QUOTE(?))")
			args = append(args, tag)
		}
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
	h.reconcileExpiredEvents()
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
	if !h.requireVerifiedUser(c, userID) {
		return
	}
	var req eventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid event")
		return
	}
	if h.rejectLocalRisk(c, req.Title, req.DepartCity, req.DepartArea, req.MeetPlace, req.CostDesc, req.Remark, strings.Join(req.PurposeTags, " ")) {
		return
	}
	h.cleanEventRequest(&req)
	if message := validateEventRequest(req, 1); message != "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, message)
		return
	}
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
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, strings.Join([]string{req.DepartCity, req.DepartArea, req.MeetPlace, req.CostDesc, strings.Join(req.PurposeTags, " ")}, " ")); err != nil {
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
	if req.ImageURL != "" && !h.isAllowedEventImage(userID, req.ImageURL) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "event image is not approved")
		return
	}
	status := "recruiting"
	if req.MaxMembers == 1 {
		status = "full"
	}
	result, err := h.db.Exec(`INSERT INTO ski_events
		(title, creator_id, resort_id, resort_name, event_date, start_time, depart_city, depart_area, meet_place, traffic_type, max_members, current_members, ski_type_req, level_req, purpose_tags, allow_beginner, same_gender_only, allow_car_pool, allow_room_share, allow_photo, cost_desc, remark, image_url, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Title, userID, req.ResortID, resortName, nullString(req.EventDate), nullString(req.StartTime), req.DepartCity, req.DepartArea, req.MeetPlace, req.TrafficType, req.MaxMembers, req.SkiTypeReq, req.LevelReq, string(tags), boolInt(req.AllowBeginner), boolInt(req.SameGenderOnly), boolInt(req.AllowCarPool), boolInt(req.AllowRoomShare), boolInt(req.AllowPhoto), req.CostDesc, req.Remark, req.ImageURL, status)
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
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "unsupported image type")
		return
	}
	source, err := file.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "cannot read image")
		return
	}
	defer source.Close()
	buffer := make([]byte, 512)
	n, _ := source.Read(buffer)
	mimeType := http.DetectContentType(buffer[:n])
	validMime := (ext == ".png" && mimeType == "image/png") || ((ext == ".jpg" || ext == ".jpeg") && mimeType == "image/jpeg")
	if !validMime {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "image extension and content do not match")
		return
	}
	if _, err := source.Seek(0, 0); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "cannot inspect image")
		return
	}
	imageConfig, _, err := image.DecodeConfig(source)
	if err != nil || imageConfig.Width < 1 || imageConfig.Height < 1 || imageConfig.Width > 12000 || imageConfig.Height > 12000 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid image content")
		return
	}

	filename := fmt.Sprintf("%d-%d%s", userID, time.Now().UnixNano(), ext)
	if _, err := source.Seek(0, 0); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "cannot read image")
		return
	}
	data, err := io.ReadAll(io.LimitReader(source, 5*1024*1024+1))
	if err != nil || len(data) > 5*1024*1024 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "cannot read image")
		return
	}
	storageKey := filepath.ToSlash(filepath.Join(folder, filename))
	if err := h.storage.Put(c.Request.Context(), storageKey, data, mimeType); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}

	path := "/uploads/" + folder + "/" + filename
	publicURL := h.cfg.PublicBaseURL + path
	if h.cfg.PublicBaseURL == "" {
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		publicURL = fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, path)
	}
	status := "approved"
	reviewToken := ""
	if h.cfg.ContentSecurity {
		status = "pending"
		random := make([]byte, 24)
		if _, err := rand.Read(random); err != nil {
			_ = h.storage.Delete(c.Request.Context(), storageKey)
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, "cannot create media review token")
			return
		}
		reviewToken = hex.EncodeToString(random)
	}
	result, err := h.db.Exec(`INSERT INTO media_uploads (user_id, kind, path, public_url, review_token, mime_type, status) VALUES (?, ?, ?, ?, ?, ?, ?)`, userID, folder, storageKey, publicURL, reviewToken, mimeType, status)
	if err != nil {
		_ = h.storage.Delete(c.Request.Context(), storageKey)
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	uploadID, _ := result.LastInsertId()
	if h.cfg.ContentSecurity {
		var openid string
		_ = h.db.QueryRow(`SELECT openid FROM users WHERE id=?`, userID).Scan(&openid)
		reviewURL := publicURL + "?reviewToken=" + reviewToken
		traceID, checkErr := h.security.CheckMediaAsync(c.Request.Context(), contentsecurity.MediaCheckRequest{OpenID: openid, Scene: 2, MediaURL: reviewURL, MediaType: 2})
		if checkErr != nil {
			_, _ = h.db.Exec(`DELETE FROM media_uploads WHERE id=?`, uploadID)
			_ = h.storage.Delete(c.Request.Context(), storageKey)
			h.contentError(c, checkErr)
			return
		}
		_, _ = h.db.Exec(`UPDATE media_uploads SET trace_id=? WHERE id=?`, traceID, uploadID)
	}
	responseData := gin.H{"id": uploadID, "status": status}
	if status == "approved" {
		responseData["url"] = publicURL
	}
	response.Success(c, responseData)
}

func (h *AppHandler) MediaReviewCallback(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	provided := c.GetHeader("X-Callback-Token")
	if provided == "" {
		provided = c.Query("token")
	}
	if h.cfg.MediaCallbackToken == "" || provided != h.cfg.MediaCallbackToken {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "invalid callback token")
		return
	}
	var payload struct {
		TraceID string `json:"trace_id"`
		Suggest string `json:"suggest"`
		Result  struct {
			Suggest string `json:"suggest"`
			Label   int    `json:"label"`
		} `json:"result"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || payload.TraceID == "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid media callback")
		return
	}
	suggest := payload.Result.Suggest
	if suggest == "" {
		suggest = payload.Suggest
	}
	status := "rejected"
	if suggest == "pass" {
		status = "approved"
	}
	result, err := h.db.Exec(`UPDATE media_uploads SET status=?, review_result=? WHERE trace_id=? AND status='pending'`, status, suggest, payload.TraceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "pending media not found")
		return
	}
	response.Success(c, gin.H{"status": status})
}

func (h *AppHandler) UploadStatus(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var status, publicURL, result string
	if err := h.db.QueryRow(`SELECT status, public_url, review_result FROM media_uploads WHERE id=? AND user_id=?`, c.Param("id"), userID).Scan(&status, &publicURL, &result); err != nil {
		h.sqlError(c, err)
		return
	}
	data := gin.H{"id": mustInt64(c.Param("id")), "status": status, "result": result}
	if status == "approved" {
		data["url"] = publicURL
	}
	response.Success(c, data)
}

func (h *AppHandler) ServeUpload(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	relative := filepath.ToSlash(filepath.Join(c.Param("folder"), c.Param("name")))
	var storedPath string
	if err := h.db.QueryRow(`SELECT path FROM media_uploads WHERE path=? AND (status='approved' OR (status='pending' AND review_token<>'' AND review_token=?))`, relative, c.Query("reviewToken")).Scan(&storedPath); err != nil {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "image not found")
		return
	}
	data, mimeType, err := h.storage.Get(c.Request.Context(), storedPath)
	if err != nil {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "image not found")
		return
	}
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}
	c.Data(http.StatusOK, mimeType, data)
}

func (h *AppHandler) UpdateEvent(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	h.reconcileExpiredEvents()
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	if !h.requireVerifiedUser(c, userID) {
		return
	}
	var creatorID int64
	var currentMembers int
	var status string
	if err := h.db.QueryRow(`SELECT creator_id, current_members, status FROM ski_events WHERE id=? AND deleted_at IS NULL`, c.Param("id")).Scan(&creatorID, &currentMembers, &status); err != nil {
		h.sqlError(c, err)
		return
	}
	if creatorID != userID {
		response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only creator can update event")
		return
	}
	if status != "recruiting" && status != "full" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "only active events can be edited")
		return
	}
	var req eventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid event")
		return
	}
	if h.rejectLocalRisk(c, req.Title, req.DepartCity, req.DepartArea, req.MeetPlace, req.CostDesc, req.Remark, strings.Join(req.PurposeTags, " ")) {
		return
	}
	h.cleanEventRequest(&req)
	if message := validateEventRequest(req, currentMembers); message != "" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, message)
		return
	}
	if req.MaxMembers < currentMembers {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "maxMembers cannot be smaller than currentMembers")
		return
	}
	if req.ImageURL != "" && !h.isAllowedEventImage(userID, req.ImageURL) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "event image is not approved")
		return
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventTitle, req.Title); err != nil {
		h.contentError(c, err)
		return
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, req.Remark); err != nil {
		h.contentError(c, err)
		return
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, strings.Join([]string{req.DepartCity, req.DepartArea, req.MeetPlace, req.CostDesc, strings.Join(req.PurposeTags, " ")}, " ")); err != nil {
		h.contentError(c, err)
		return
	}
	tags, _ := json.Marshal(req.PurposeTags)
	nextStatus := "recruiting"
	if req.MaxMembers == currentMembers {
		nextStatus = "full"
	}
	_, err := h.db.Exec(`UPDATE ski_events SET title=?, resort_id=?, resort_name=?, event_date=?, start_time=?, depart_city=?, depart_area=?, meet_place=?, traffic_type=?, max_members=?, ski_type_req=?, level_req=?, purpose_tags=?, allow_beginner=?, same_gender_only=?, allow_car_pool=?, allow_room_share=?, allow_photo=?, cost_desc=?, remark=?, image_url=?, status=? WHERE id=?`,
		req.Title, req.ResortID, req.ResortName, nullString(req.EventDate), nullString(req.StartTime), req.DepartCity, req.DepartArea, req.MeetPlace, req.TrafficType, req.MaxMembers, req.SkiTypeReq, req.LevelReq, string(tags), boolInt(req.AllowBeginner), boolInt(req.SameGenderOnly), boolInt(req.AllowCarPool), boolInt(req.AllowRoomShare), boolInt(req.AllowPhoto), req.CostDesc, req.Remark, req.ImageURL, nextStatus, c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	event, _ := h.eventByID(c.Param("id"))
	response.Success(c, event)
}

func (h *AppHandler) DeleteEvent(c *gin.Context) {
	// Backwards-compatible DELETE: preserve the event and its audit trail by
	// translating the old operation into a normal cancellation transition.
	h.SetEventStatus("cancelled")(c)
}

func (h *AppHandler) SetEventStatus(status string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireDB(c) {
			return
		}
		h.reconcileExpiredEvents()
		userID, ok := currentUserID(c)
		if !ok || !h.ensureActive(c, userID) {
			return
		}
		if !h.requireVerifiedUser(c, userID) {
			return
		}
		if status != "cancelled" && status != "finished" {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "unsupported event status transition")
			return
		}
		tx, err := h.db.Begin()
		if err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		defer tx.Rollback()
		var creatorID int64
		var currentStatus string
		if err := tx.QueryRow(`SELECT creator_id, status FROM ski_events WHERE id=? AND deleted_at IS NULL FOR UPDATE`, c.Param("id")).Scan(&creatorID, &currentStatus); err != nil {
			h.sqlError(c, err)
			return
		}
		if creatorID != userID {
			response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only creator can update event")
			return
		}
		if currentStatus != "recruiting" && currentStatus != "full" {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "event is no longer active")
			return
		}
		if _, err := tx.Exec(`UPDATE ski_events SET status=? WHERE id=?`, status, c.Param("id")); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		if err := tx.Commit(); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		response.Success(c, gin.H{"status": status})
	}
}

func (h *AppHandler) ApplyEvent(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	h.reconcileExpiredEvents()
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	if !h.requireVerifiedUser(c, userID) {
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
	if !textWithin(req.DepartArea, 128) || !textWithin(req.Message, 500) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "application field is too long")
		return
	}
	if req.SkiLevel != "" && !stringIn(req.SkiLevel, "beginner", "primary", "intermediate", "advanced") {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid skiLevel")
		return
	}
	if req.SkiType != "" && !stringIn(req.SkiType, "snowboard", "ski", "both") {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid skiType")
		return
	}
	if h.rejectLocalRisk(c, req.DepartArea, req.Message) {
		return
	}
	req.DepartArea = h.cleanText(req.DepartArea)
	req.Message = h.cleanText(req.Message)
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, req.Message); err != nil {
		h.contentError(c, err)
		return
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, req.DepartArea); err != nil {
		h.contentError(c, err)
		return
	}
	tx, err := h.db.Begin()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer tx.Rollback()

	var creatorID int64
	var status string
	var currentMembers, maxMembers int
	if err := tx.QueryRow(`SELECT creator_id, status, current_members, max_members FROM ski_events WHERE id=? AND deleted_at IS NULL FOR UPDATE`, c.Param("id")).Scan(&creatorID, &status, &currentMembers, &maxMembers); err != nil {
		h.sqlError(c, err)
		return
	}
	if creatorID == userID {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "creator cannot apply")
		return
	}
	if status != "recruiting" || currentMembers >= maxMembers {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "event is full or not recruiting")
		return
	}
	var memberCount int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM event_members WHERE event_id=? AND user_id=? AND status='active'`, c.Param("id"), userID).Scan(&memberCount); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if memberCount > 0 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "user is already an event member")
		return
	}

	var id int64
	var oldStatus string
	err = tx.QueryRow(`SELECT id, status FROM join_requests WHERE event_id=? AND applicant_id=? FOR UPDATE`, c.Param("id"), userID).Scan(&id, &oldStatus)
	switch {
	case err == nil && oldStatus == "rejected":
		if _, err = tx.Exec(`UPDATE join_requests SET creator_id=?, ski_level=?, ski_type=?, has_car=?, can_carry_people=?, depart_area=?, message=?, status='pending', reject_reason='', updated_at=NOW() WHERE id=?`,
			creatorID, req.SkiLevel, req.SkiType, boolInt(req.HasCar), boolInt(req.CanCarryPeople), req.DepartArea, req.Message, id); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
	case err == nil && oldStatus == "removed":
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "approved member cannot apply again")
		return
	case err == nil:
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "application is already pending or approved")
		return
	case errors.Is(err, sql.ErrNoRows):
		result, insertErr := tx.Exec(`INSERT INTO join_requests (event_id, applicant_id, creator_id, ski_level, ski_type, has_car, can_carry_people, depart_area, message, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
			c.Param("id"), userID, creatorID, req.SkiLevel, req.SkiType, boolInt(req.HasCar), boolInt(req.CanCarryPeople), req.DepartArea, req.Message)
		if insertErr != nil {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "application cannot be created")
			return
		}
		id, _ = result.LastInsertId()
		oldStatus = ""
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`INSERT INTO join_request_history (join_request_id, event_id, applicant_id, from_status, to_status, operator_id) VALUES (?, ?, ?, ?, 'pending', ?)`, id, c.Param("id"), userID, oldStatus, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	_ = h.addNotification(creatorID, "有新的加入申请", "有雪友申请加入你的滑雪局，请及时审核", "join_request", "event", mustInt64(c.Param("id")))
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
		if !h.requireVerifiedUser(c, userID) {
			return
		}
		var req struct {
			Reason string `json:"reason"`
		}
		_ = c.ShouldBindJSON(&req)
		if h.rejectLocalRisk(c, req.Reason) {
			return
		}
		req.Reason = h.cleanText(req.Reason)
		if status != "approved" && status != "rejected" {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid review status")
			return
		}
		if req.Reason != "" {
			if err := h.checkText(c.Request.Context(), userID, compliance.FieldEventRemark, req.Reason); err != nil {
				h.contentError(c, err)
				return
			}
		}
		var lockedEventID int64
		if err := h.db.QueryRow(`SELECT event_id FROM join_requests WHERE id=?`, c.Param("id")).Scan(&lockedEventID); err != nil {
			h.sqlError(c, err)
			return
		}
		tx, err := h.db.Begin()
		if err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		defer tx.Rollback()
		var eventStatus string
		var currentMembers, maxMembers int
		if err := tx.QueryRow(`SELECT status, current_members, max_members FROM ski_events WHERE id=? FOR UPDATE`, lockedEventID).Scan(&eventStatus, &currentMembers, &maxMembers); err != nil {
			h.sqlError(c, err)
			return
		}
		var eventID, applicantID, creatorID int64
		var currentStatus string
		if err := tx.QueryRow(`SELECT event_id, applicant_id, creator_id, status FROM join_requests WHERE id=? FOR UPDATE`, c.Param("id")).Scan(&eventID, &applicantID, &creatorID, &currentStatus); err != nil {
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
		if eventID != lockedEventID {
			response.Error(c, http.StatusConflict, response.CodeBadRequest, "application event changed")
			return
		}
		if status == "approved" {
			var applicantVerificationStatus string
			if err := tx.QueryRow(`SELECT verification_status FROM users WHERE id=? FOR UPDATE`, applicantID).Scan(&applicantVerificationStatus); err != nil {
				h.sqlError(c, err)
				return
			}
			if applicantVerificationStatus != "verified" {
				response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "申请人尚未完成实名认证")
				return
			}
			if eventStatus != "recruiting" || maxMembers > 20 || currentMembers >= maxMembers {
				response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "event is full or no longer recruiting")
				return
			}
			var memberCount int
			if err := tx.QueryRow(`SELECT COUNT(*) FROM event_members WHERE event_id=? AND user_id=? AND status='active'`, eventID, applicantID).Scan(&memberCount); err != nil {
				response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
				return
			}
			if memberCount > 0 {
				response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "applicant is already a member")
				return
			}
			if _, err = tx.Exec(`INSERT INTO event_members (event_id, user_id, role, status) VALUES (?, ?, 'member', 'active')`, eventID, applicantID); err != nil {
				response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
				return
			}
			if _, err = tx.Exec(`UPDATE ski_events SET current_members=current_members+1, status=IF(current_members+1 >= max_members, 'full', 'recruiting') WHERE id=?`, eventID); err != nil {
				response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
				return
			}
			_, _ = tx.Exec(`UPDATE users SET join_count=join_count+1 WHERE id=?`, applicantID)
		}
		if _, err = tx.Exec(`UPDATE join_requests SET status=?, reject_reason=?, updated_at=NOW() WHERE id=?`, status, req.Reason, c.Param("id")); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		if _, err = tx.Exec(`INSERT INTO join_request_history (join_request_id, event_id, applicant_id, from_status, to_status, operator_id, reason) VALUES (?, ?, ?, ?, ?, ?, ?)`, c.Param("id"), eventID, applicantID, currentStatus, status, userID, req.Reason); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		if err = tx.Commit(); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		title := "申请已通过"
		content := "你申请的滑雪局已通过，快去群聊同步集合信息"
		if status == "rejected" {
			title = "申请已拒绝"
			content = defaultString(req.Reason, "发起人暂未通过你的加入申请")
		}
		_ = h.addNotification(applicantID, title, content, "join_request", "event", eventID)
		response.Success(c, gin.H{"status": status})
	}
}

func (h *AppHandler) RemoveEventMember(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	h.reconcileExpiredEvents()
	operatorID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, operatorID) {
		return
	}
	eventID := mustInt64(c.Param("id"))
	targetID := mustInt64(c.Param("userId"))
	if eventID <= 0 || targetID <= 0 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid event member")
		return
	}
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer tx.Rollback()
	var creatorID int64
	var eventStatus string
	var currentMembers int
	if err := tx.QueryRow(`SELECT creator_id, status, current_members FROM ski_events WHERE id=? AND deleted_at IS NULL FOR UPDATE`, eventID).Scan(&creatorID, &eventStatus, &currentMembers); err != nil {
		h.sqlError(c, err)
		return
	}
	if creatorID != operatorID {
		response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only creator can remove members")
		return
	}
	if targetID == creatorID {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "creator cannot be removed")
		return
	}
	if eventStatus != "recruiting" && eventStatus != "full" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "event no longer allows member changes")
		return
	}
	var role, memberStatus string
	if err := tx.QueryRow(`SELECT role, status FROM event_members WHERE event_id=? AND user_id=? FOR UPDATE`, eventID, targetID).Scan(&role, &memberStatus); err != nil {
		h.sqlError(c, err)
		return
	}
	if role != "member" || memberStatus != "active" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "member is not active")
		return
	}
	if _, err := tx.Exec(`UPDATE event_members SET status='inactive' WHERE event_id=? AND user_id=?`, eventID, targetID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE ski_events SET current_members=GREATEST(1,current_members-1), status='recruiting' WHERE id=?`, eventID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	var requestID int64
	var requestStatus string
	if err := tx.QueryRow(`SELECT id, status FROM join_requests WHERE event_id=? AND applicant_id=? FOR UPDATE`, eventID, targetID).Scan(&requestID, &requestStatus); err == nil {
		if _, err := tx.Exec(`UPDATE join_requests SET status='removed', reject_reason='已被发起人移出行程', updated_at=NOW() WHERE id=?`, requestID); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		if _, err := tx.Exec(`INSERT INTO join_request_history (join_request_id, event_id, applicant_id, from_status, to_status, operator_id, reason) VALUES (?, ?, ?, ?, 'removed', ?, '发起人移出成员')`, requestID, eventID, targetID, requestStatus, operatorID); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE users SET join_count=GREATEST(0,join_count-1) WHERE id=?`, targetID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	_ = h.addNotification(targetID, "已退出滑雪局", "你已被发起人移出该行程，无法继续进入群聊", "event_member", "event", eventID)
	response.Success(c, gin.H{"status": "removed", "currentMembers": maxInt(currentMembers-1, 1)})
}

func (h *AppHandler) Trips(kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireDB(c) {
			return
		}
		h.reconcileExpiredEvents()
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
			where = append(where, "(r.applicant_id=? OR r.creator_id=?) AND r.status='pending'", "e.status NOT IN ('finished', 'cancelled')")
			args = append(args, userID, userID)
		case "finished":
			where = append(where, "(e.status IN ('finished','cancelled') AND (e.creator_id=? OR m.user_id=?))")
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
	h.reconcileExpiredEvents()
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	if !h.requireVerifiedUser(c, userID) {
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
	userID, ok := currentUserID(c)
	if !ok || !h.requireVerifiedUser(c, userID) {
		return
	}
	beforeID, _ := strconv.ParseInt(c.Query("beforeId"), 10, 64)
	afterID, _ := strconv.ParseInt(c.Query("afterId"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	query := `SELECT m.id, m.sender_id, u.nickname, u.avatar_url, m.message_type, m.content, m.created_at
		FROM chat_messages m JOIN users u ON u.id=m.sender_id WHERE m.event_id=? AND m.status='normal'`
	args := []interface{}{c.Param("id")}
	reverse := true
	if afterID > 0 {
		query += ` AND m.id>? ORDER BY m.id ASC LIMIT ?`
		args = append(args, afterID, limit)
		reverse = false
	} else if beforeID > 0 {
		query += ` AND m.id<? ORDER BY m.id DESC LIMIT ?`
		args = append(args, beforeID, limit)
	} else {
		query += ` ORDER BY m.id DESC LIMIT ?`
		args = append(args, limit)
	}
	rows, err := h.db.Query(query, args...)
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
	if reverse {
		for left, right := 0, len(list)-1; left < right; left, right = left+1, right-1 {
			list[left], list[right] = list[right], list[left]
		}
	}
	nextBeforeID := int64(0)
	if reverse && len(list) == limit {
		nextBeforeID, _ = list[0]["id"].(int64)
	}
	response.Success(c, gin.H{"list": list, "nextBeforeId": nextBeforeID})
}

func (h *AppHandler) MarkChatRead(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.canAccessEvent(c, c.Param("id")) {
		return
	}
	if !h.requireVerifiedUser(c, userID) {
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
	if !h.requireVerifiedUser(c, userID) {
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
	if !h.requireVerifiedUser(c, userID) {
		return
	}
	var eventStatus string
	if err := h.db.QueryRow(`SELECT status FROM ski_events WHERE id=? AND deleted_at IS NULL`, c.Param("id")).Scan(&eventStatus); err != nil {
		h.sqlError(c, err)
		return
	}
	if eventStatus != "recruiting" && eventStatus != "full" {
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "活动已结束或取消，群聊仅可查看历史消息")
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
	if !textWithin(req.Content, 1000) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "message is too long")
		return
	}
	if h.rejectLocalRisk(c, req.Content) {
		return
	}
	req.Content = h.cleanText(req.Content)
	if req.MessageType == "" {
		req.MessageType = "text"
	}
	if req.MessageType != "text" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "unsupported message type")
		return
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
	if !h.requireVerifiedUser(c, userID) {
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
	if req.Score < 1 || req.Score > 5 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "score must be between 1 and 5")
		return
	}
	if !textWithin(req.Content, 500) || !textListWithin(req.PositiveTags, 12, 32) || !textListWithin(req.NegativeTags, 12, 32) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "review field is too long")
		return
	}
	if h.rejectLocalRisk(c, req.Content, strings.Join(req.PositiveTags, " "), strings.Join(req.NegativeTags, " ")) {
		return
	}
	if req.RevieweeID == userID {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "cannot review yourself")
		return
	}
	var eventStatus string
	if err := h.db.QueryRow(`SELECT status FROM ski_events WHERE id=? AND deleted_at IS NULL`, req.EventID).Scan(&eventStatus); err != nil {
		h.sqlError(c, err)
		return
	}
	if eventStatus != "finished" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "event is not finished")
		return
	}
	if !h.isActiveEventMember(req.EventID, userID) || !h.isActiveEventMember(req.EventID, req.RevieweeID) {
		response.Error(c, http.StatusForbidden, response.CodeUnauthorized, "only event members can review each other")
		return
	}
	var existing int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE event_id=? AND reviewer_id=? AND reviewee_id=?`, req.EventID, userID, req.RevieweeID).Scan(&existing); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if existing > 0 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "review already exists")
		return
	}
	req.PositiveTags = h.cleanTexts(req.PositiveTags)
	req.NegativeTags = h.cleanTexts(req.NegativeTags)
	req.Content = h.cleanText(req.Content)
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldReview, req.Content); err != nil {
		h.contentError(c, err)
		return
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldReview, strings.Join(append(req.PositiveTags, req.NegativeTags...), " ")); err != nil {
		h.contentError(c, err)
		return
	}
	positive, _ := json.Marshal(req.PositiveTags)
	negative, _ := json.Marshal(req.NegativeTags)
	result, err := h.db.Exec(`INSERT INTO reviews (event_id, reviewer_id, reviewee_id, score, positive_tags, negative_tags, content, is_anonymous) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		req.EventID, userID, req.RevieweeID, req.Score, string(positive), string(negative), req.Content, boolInt(req.IsAnonymous))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "review already exists or cannot be created")
		return
	}
	id, _ := result.LastInsertId()
	if err := h.refreshUserReviewStats(req.RevieweeID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"id": id})
}

func (h *AppHandler) UserReviews(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	rows, err := h.db.Query(`SELECT r.id, r.event_id, e.title, r.reviewer_id, u.nickname, u.avatar_url, r.score, r.positive_tags, r.negative_tags, r.content, r.is_anonymous, r.created_at
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
		var title, nickname, avatarURL, positive, negative, content string
		var score, anonymous int
		var created time.Time
		_ = rows.Scan(&id, &eventID, &title, &reviewerID, &nickname, &avatarURL, &score, &positive, &negative, &content, &anonymous, &created)
		if anonymous == 1 {
			nickname = "匿名雪友"
			reviewerID = 0
			avatarURL = ""
		}
		list = append(list, gin.H{"id": id, "eventId": eventID, "eventTitle": title, "reviewerId": reviewerID, "reviewerName": nickname, "reviewerAvatarUrl": avatarURL, "score": score, "positiveTags": jsonList(positive), "negativeTags": jsonList(negative), "content": content, "createdAt": created})
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
	if !textWithin(req.Reason, 255) || !textWithin(req.Content, 2000) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "report field is too long")
		return
	}
	var exists int
	switch req.TargetType {
	case "app":
		if req.TargetID != 0 {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid app feedback target")
			return
		}
		exists = 1
	case "user":
		if req.TargetID == userID {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "cannot report yourself")
			return
		}
		_ = h.db.QueryRow(`SELECT COUNT(*) FROM users WHERE id=? AND deleted_at IS NULL`, req.TargetID).Scan(&exists)
	case "event":
		_ = h.db.QueryRow(`SELECT COUNT(*) FROM ski_events WHERE id=? AND deleted_at IS NULL`, req.TargetID).Scan(&exists)
	case "message":
		_ = h.db.QueryRow(`SELECT COUNT(*) FROM chat_messages WHERE id=?`, req.TargetID).Scan(&exists)
	case "review":
		_ = h.db.QueryRow(`SELECT COUNT(*) FROM reviews WHERE id=?`, req.TargetID).Scan(&exists)
	default:
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "unsupported report target")
		return
	}
	if exists == 0 {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "report target not found")
		return
	}
	if h.rejectLocalRisk(c, req.Reason, req.Content) {
		return
	}
	req.Reason = h.cleanText(req.Reason)
	req.Content = h.cleanText(req.Content)
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldReport, req.Content); err != nil {
		h.contentError(c, err)
		return
	}
	if err := h.checkText(c.Request.Context(), userID, compliance.FieldReport, req.Reason); err != nil {
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

func (h *AppHandler) Favorites(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	list, total, err := h.eventPageWithJoin(" JOIN event_favorites f ON f.event_id=e.id", []string{"e.deleted_at IS NULL", "e.status <> 'removed'", "f.user_id=?"}, []interface{}{userID}, "e.updated_at DESC", 1, 200)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, pageData(list, 1, 200, total))
}

func (h *AppHandler) SetFavorite(favorite bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireDB(c) {
			return
		}
		userID, ok := currentUserID(c)
		if !ok || !h.ensureActive(c, userID) {
			return
		}
		if favorite {
			if _, err := h.db.Exec(`INSERT IGNORE INTO event_favorites (user_id, event_id) VALUES (?, ?)`, userID, c.Param("id")); err != nil {
				response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
				return
			}
		} else if _, err := h.db.Exec(`DELETE FROM event_favorites WHERE user_id=? AND event_id=?`, userID, c.Param("id")); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		response.Success(c, gin.H{"favorite": favorite})
	}
}

func (h *AppHandler) SetFollow(follow bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.requireDB(c) {
			return
		}
		userID, ok := currentUserID(c)
		if !ok || !h.ensureActive(c, userID) {
			return
		}
		targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil || targetID == 0 {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid user")
			return
		}
		if targetID == userID {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "cannot follow yourself")
			return
		}
		var targetCount int
		if err := h.db.QueryRow(`SELECT COUNT(*) FROM users WHERE id=? AND deleted_at IS NULL AND status='normal'`, targetID).Scan(&targetCount); err != nil || targetCount == 0 {
			response.Error(c, http.StatusNotFound, response.CodeNotFound, "user not found")
			return
		}
		if follow {
			result, err := h.db.Exec(`INSERT IGNORE INTO user_follows (follower_id, followee_id) VALUES (?, ?)`, userID, targetID)
			if err != nil {
				response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
				return
			}
			if affected, _ := result.RowsAffected(); affected > 0 {
				_ = h.addNotification(targetID, "有新的雪友关注你", "对方已关注你的雪友主页", "follow", "user", userID)
			}
		} else if _, err := h.db.Exec(`DELETE FROM user_follows WHERE follower_id=? AND followee_id=?`, userID, targetID); err != nil {
			response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
			return
		}
		response.Success(c, gin.H{"following": follow})
	}
}

func (h *AppHandler) FollowStatus(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || targetID == 0 {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "invalid user")
		return
	}
	var count int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM user_follows WHERE follower_id=? AND followee_id=?`, userID, targetID).Scan(&count); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"following": count > 0})
}

func (h *AppHandler) Notifications(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	rows, err := h.db.Query(`SELECT id, title, content, type, target_type, target_id, is_read, created_at FROM notifications WHERE user_id=? ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer rows.Close()
	list := []gin.H{}
	unread := int64(0)
	for rows.Next() {
		var id, targetID int64
		var title, content, typ, targetType string
		var read int
		var created time.Time
		_ = rows.Scan(&id, &title, &content, &typ, &targetType, &targetID, &read, &created)
		if read == 0 {
			unread++
		}
		list = append(list, gin.H{"id": id, "title": title, "content": content, "type": typ, "targetType": targetType, "targetId": targetID, "read": read == 1, "time": timeAgo(created), "createdAt": created})
	}
	response.Success(c, gin.H{"list": list, "unreadCount": unread})
}

func (h *AppHandler) MarkNotificationsRead(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	if _, err := h.db.Exec(`UPDATE notifications SET is_read=1 WHERE user_id=?`, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"read": true})
}

func (h *AppHandler) ClearNotifications(c *gin.Context) {
	if !h.requireDB(c) {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return
	}
	if _, err := h.db.Exec(`DELETE FROM notifications WHERE user_id=?`, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"cleared": true})
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
		"purposeTags":   []string{"刷道", "练习", "平花", "刻滑", "公园", "拍照", "休闲滑"},
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

func validateEventRequest(req eventRequest, currentMembers int) string {
	if !textWithin(req.Title, 128) || !textWithin(req.ResortName, 128) ||
		!textWithin(req.DepartCity, 64) || !textWithin(req.DepartArea, 128) ||
		!textWithin(req.MeetPlace, 255) || !textWithin(req.CostDesc, 255) ||
		!textWithin(req.Remark, 2000) || !textWithin(req.ImageURL, 500) ||
		!textListWithin(req.PurposeTags, 12, 32) {
		return "event field is too long"
	}
	if strings.TrimSpace(req.ResortName) == "" && req.ResortID <= 0 {
		return "resort is required"
	}
	if req.MaxMembers < currentMembers || req.MaxMembers > 20 {
		return "maxMembers must include current members and be between 1 and 20"
	}
	date, err := time.Parse("2006-01-02", req.EventDate)
	if err != nil {
		return "eventDate must use YYYY-MM-DD"
	}
	today, _ := time.ParseInLocation("2006-01-02", time.Now().Format("2006-01-02"), time.Local)
	if date.Before(today) {
		return "eventDate cannot be in the past"
	}
	if strings.TrimSpace(req.DepartCity) == "" || strings.TrimSpace(req.DepartArea) == "" || strings.TrimSpace(req.MeetPlace) == "" {
		return "departure and meeting place are required"
	}
	if req.StartTime != "" {
		startTime, err := time.ParseInLocation("2006-01-02 15:04:05", req.StartTime, time.Local)
		if err != nil {
			return "startTime must use YYYY-MM-DD HH:mm:ss"
		}
		if startTime.Format("2006-01-02") != req.EventDate {
			return "startTime must match eventDate"
		}
		if startTime.Before(time.Now()) {
			return "startTime cannot be in the past"
		}
	}
	if !stringIn(req.SkiTypeReq, "snowboard", "ski", "both") {
		return "invalid skiTypeReq"
	}
	if !stringIn(req.LevelReq, "beginner", "primary", "intermediate", "advanced") {
		return "invalid levelReq"
	}
	if !stringIn(req.TrafficType, "self_drive", "high_speed_rail", "bus", "other") {
		return "invalid trafficType"
	}
	return ""
}

func (h *AppHandler) reconcileExpiredEvents() {
	if h == nil || h.db == nil {
		return
	}
	today := time.Now().In(time.Local).Format("2006-01-02")
	_, _ = h.db.Exec(`UPDATE ski_events SET status='finished'
		WHERE deleted_at IS NULL AND status IN ('recruiting','full') AND event_date<?`, today)
}

func stringIn(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func textWithin(value string, maxRunes int) bool {
	return len([]rune(strings.TrimSpace(value))) <= maxRunes
}

func textListWithin(values []string, maxItems, maxRunes int) bool {
	if len(values) > maxItems {
		return false
	}
	for _, value := range values {
		if !textWithin(value, maxRunes) {
			return false
		}
	}
	return true
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
	var verificationStatus, verificationMethod, phoneMasked string
	var verifiedAt sql.NullTime
	err := h.db.QueryRow(`SELECT id, openid, nickname, avatar_url, bio, gender, gender_visible, city, ski_type, ski_level, style_tags, favorite_resorts, has_car, credit_score, event_count, join_count, good_rate, verification_status, verification_method, phone_masked, verified_at, status, created_at, updated_at FROM users WHERE id=? AND deleted_at IS NULL`, id).
		Scan(&user.ID, &user.OpenID, &user.Nickname, &user.AvatarURL, &user.Bio, &user.Gender, &user.GenderVisible, &user.City, &user.SkiType, &user.SkiLevel, &styleTags, &favoriteResorts, &user.HasCar, &user.CreditScore, &user.EventCount, &user.JoinCount, &user.GoodRate, &verificationStatus, &verificationMethod, &phoneMasked, &verifiedAt, &user.Status, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if user.AvatarURL == "" {
		var recovered string
		if err := h.db.QueryRow(`SELECT public_url FROM media_uploads WHERE user_id=? AND kind='avatars' AND status='approved' ORDER BY id DESC LIMIT 1`, id).Scan(&recovered); err == nil && recovered != "" {
			user.AvatarURL = recovered
			_, _ = h.db.Exec(`UPDATE users SET avatar_url=? WHERE id=? AND avatar_url=''`, recovered, id)
		}
	}
	return gin.H{"id": user.ID, "openid": user.OpenID, "nickname": user.Nickname, "avatarUrl": user.AvatarURL, "bio": user.Bio, "gender": user.Gender, "genderVisible": user.GenderVisible == 1, "city": user.City, "skiType": user.SkiType, "skiLevel": user.SkiLevel, "styleTags": jsonList(styleTags.String), "favoriteResorts": jsonList(favoriteResorts.String), "hasCar": user.HasCar == 1, "creditScore": user.CreditScore, "eventCount": user.EventCount, "joinCount": user.JoinCount, "goodRate": user.GoodRate, "verificationStatus": verificationStatus, "verificationMethod": verificationMethod, "phoneMasked": phoneMasked, "verifiedAt": nullableTime(verifiedAt), "realNameVerified": verificationStatus == "verified", "status": user.Status, "createdAt": user.CreatedAt, "updatedAt": user.UpdatedAt}, nil
}

func (h *AppHandler) userByIDParam(id string) (gin.H, error) {
	parsed, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return nil, sql.ErrNoRows
	}
	return h.userByID(parsed)
}

func (h *AppHandler) eventPage(where []string, args []interface{}, order string, page, pageSize int) ([]gin.H, int64, error) {
	join := ""
	if strings.Contains(strings.Join(where, " "), "m.") {
		join += " LEFT JOIN event_members m ON m.event_id=e.id"
	}
	if strings.Contains(strings.Join(where, " "), "r.") {
		join += " LEFT JOIN join_requests r ON r.event_id=e.id"
	}
	return h.eventPageWithJoin(join, where, args, order, page, pageSize)
}

func (h *AppHandler) eventPageWithJoin(join string, where []string, args []interface{}, order string, page, pageSize int) ([]gin.H, int64, error) {
	whereSQL := " WHERE " + strings.Join(where, " AND ")
	var total int64
	if err := h.db.QueryRow(`SELECT COUNT(DISTINCT e.id) FROM ski_events e`+join+whereSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, (page-1)*pageSize, pageSize)
	rows, err := h.db.Query(`SELECT DISTINCT e.id, e.title, e.creator_id, u.nickname, u.avatar_url, u.credit_score, u.event_count, e.resort_id, e.resort_name, e.event_date, e.start_time, e.depart_city, e.depart_area, e.meet_place, e.traffic_type, e.max_members, e.current_members, e.ski_type_req, e.level_req, e.purpose_tags, e.allow_beginner, e.same_gender_only, e.allow_car_pool, e.allow_room_share, e.allow_photo, e.cost_desc, e.remark, e.image_url, e.status, e.view_count, e.created_at, e.updated_at
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
	if err := h.attachEventMembers(list); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (h *AppHandler) attachEventMembers(events []gin.H) error {
	if len(events) == 0 {
		return nil
	}
	ids := make([]string, 0, len(events))
	byID := map[int64][]gin.H{}
	for _, event := range events {
		id, ok := event["id"].(int64)
		if !ok {
			continue
		}
		ids = append(ids, "?")
		byID[id] = []gin.H{}
	}
	if len(ids) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(events))
	for _, event := range events {
		if id, ok := event["id"].(int64); ok {
			args = append(args, id)
		}
	}
	rows, err := h.db.Query(`SELECT m.event_id, m.user_id, u.nickname, u.avatar_url, u.ski_type, u.ski_level, u.credit_score, m.role, m.status, m.joined_at
		FROM event_members m JOIN users u ON u.id=m.user_id
		WHERE m.status='active' AND m.event_id IN (`+strings.Join(ids, ",")+`)
		ORDER BY m.event_id ASC, m.role='creator' DESC, m.joined_at ASC`, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var eventID, userID int64
		var nickname, avatarURL, skiType, skiLevel, role, status string
		var credit float64
		var joined time.Time
		if err := rows.Scan(&eventID, &userID, &nickname, &avatarURL, &skiType, &skiLevel, &credit, &role, &status, &joined); err != nil {
			return err
		}
		byID[eventID] = append(byID[eventID], gin.H{"id": userID, "userId": userID, "nickname": nickname, "avatarUrl": avatarURL, "skiType": skiType, "skiLevel": skiLevel, "creditScore": credit, "role": role, "status": status, "joinedAt": joined})
	}
	for _, event := range events {
		if id, ok := event["id"].(int64); ok {
			event["members"] = byID[id]
		}
	}
	return nil
}

func (h *AppHandler) eventByID(id string) (gin.H, error) {
	row := h.db.QueryRow(`SELECT e.id, e.title, e.creator_id, u.nickname, u.avatar_url, u.credit_score, u.event_count, e.resort_id, e.resort_name, e.event_date, e.start_time, e.depart_city, e.depart_area, e.meet_place, e.traffic_type, e.max_members, e.current_members, e.ski_type_req, e.level_req, e.purpose_tags, e.allow_beginner, e.same_gender_only, e.allow_car_pool, e.allow_room_share, e.allow_photo, e.cost_desc, e.remark, e.image_url, e.status, e.view_count, e.created_at, e.updated_at
		FROM ski_events e JOIN users u ON u.id=e.creator_id WHERE e.id=? AND e.deleted_at IS NULL AND e.status<>'removed'`, id)
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
	var title, creatorName, creatorAvatarURL, resortName, departCity, departArea, meetPlace, trafficType, skiTypeReq, levelReq, tags, costDesc, imageURL, status string
	var eventDate, startTime sql.NullTime
	var remark sql.NullString
	var maxMembers, currentMembers, creatorEventCount, allowBeginner, sameGenderOnly, allowCarPool, allowRoomShare, allowPhoto, viewCount int
	var creatorCreditScore float64
	var createdAt, updatedAt time.Time
	err := scanner.Scan(&id, &title, &creatorID, &creatorName, &creatorAvatarURL, &creatorCreditScore, &creatorEventCount, &resortID, &resortName, &eventDate, &startTime, &departCity, &departArea, &meetPlace, &trafficType, &maxMembers, &currentMembers, &skiTypeReq, &levelReq, &tags, &allowBeginner, &sameGenderOnly, &allowCarPool, &allowRoomShare, &allowPhoto, &costDesc, &remark, &imageURL, &status, &viewCount, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return gin.H{"id": id, "title": title, "creatorId": creatorID, "creatorName": creatorName, "creatorAvatarUrl": creatorAvatarURL, "creatorCreditScore": creatorCreditScore, "creatorEventCount": creatorEventCount, "resortId": resortID, "resortName": resortName, "eventDate": timeString(eventDate, "2006-01-02"), "startTime": timeString(startTime, "2006-01-02 15:04:05"), "departCity": departCity, "departArea": departArea, "meetPlace": meetPlace, "trafficType": trafficType, "maxMembers": maxMembers, "currentMembers": currentMembers, "skiTypeReq": skiTypeReq, "levelReq": levelReq, "purposeTags": jsonList(tags), "allowBeginner": allowBeginner == 1, "sameGenderOnly": sameGenderOnly == 1, "allowCarPool": allowCarPool == 1, "allowRoomShare": allowRoomShare == 1, "allowPhoto": allowPhoto == 1, "costDesc": costDesc, "remark": remark.String, "imageUrl": imageURL, "status": status, "viewCount": viewCount, "createdAt": createdAt, "updatedAt": updatedAt}, nil
}

type userRow struct {
	ID            int64
	OpenID        string
	Nickname      string
	AvatarURL     string
	Bio           string
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
	h.reconcileExpiredEvents()
	userID, ok := currentUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "missing user")
		return false
	}
	var count int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM event_members m JOIN ski_events e ON e.id=m.event_id
		WHERE m.event_id=? AND m.user_id=? AND m.status='active' AND e.deleted_at IS NULL AND e.status IN ('recruiting','full','finished','cancelled')`, eventID, userID).Scan(&count); err != nil {
		h.sqlError(c, err)
		return false
	}
	if count == 0 {
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "only members can access event chat")
		return false
	}
	return true
}

func (h *AppHandler) isActiveEventMember(eventID, userID int64) bool {
	var count int
	if err := h.db.QueryRow(`SELECT COUNT(*) FROM event_members WHERE event_id=? AND user_id=? AND status='active'`, eventID, userID).Scan(&count); err != nil {
		return false
	}
	return count > 0
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

func (h *AppHandler) refreshUserReviewStats(userID int64) error {
	var avg sql.NullFloat64
	var total, good int64
	if err := h.db.QueryRow(`SELECT AVG(score), COUNT(*), SUM(CASE WHEN score >= 4 THEN 1 ELSE 0 END) FROM reviews WHERE reviewee_id=? AND status='normal'`, userID).Scan(&avg, &total, &good); err != nil {
		return err
	}
	credit := 5.0
	if avg.Valid {
		credit = avg.Float64
	}
	goodRate := 100.0
	if total > 0 {
		goodRate = float64(good) * 100 / float64(total)
	}
	_, err := h.db.Exec(`UPDATE users SET credit_score=?, good_rate=? WHERE id=?`, credit, goodRate, userID)
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

func (h *AppHandler) rejectLocalRisk(c *gin.Context, values ...string) bool {
	if h == nil || h.textFilter == nil {
		return false
	}
	for _, value := range values {
		if value != "" && h.textFilter.Clean(value) != value {
			response.ContentRisk(c)
			return true
		}
	}
	return false
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
func pointerSlice(value *[]string) []string {
	if value == nil {
		return nil
	}
	return *value
}

func (h *AppHandler) isApprovedAvatar(userID int64, imageURL string) bool {
	var count int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM media_uploads WHERE user_id=? AND kind='avatars' AND public_url=? AND status='approved'`, userID, imageURL).Scan(&count)
	if count > 0 {
		return true
	}
	// Preserve an already stored legacy avatar without allowing a new URL.
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM users WHERE id=? AND avatar_url=?`, userID, imageURL).Scan(&count)
	return count > 0
}

func (h *AppHandler) isAllowedEventImage(userID int64, imageURL string) bool {
	var count int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM media_uploads WHERE user_id=? AND kind='events' AND public_url=? AND status='approved'`, userID, imageURL).Scan(&count)
	if count > 0 {
		return true
	}
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM ski_resorts WHERE image_url=? AND status='normal'`, imageURL).Scan(&count)
	if count > 0 {
		return true
	}
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM ski_events WHERE creator_id=? AND image_url=?`, userID, imageURL).Scan(&count)
	return count > 0
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

func (h *AppHandler) addNotification(userID int64, title, content, typ, targetType string, targetID int64) error {
	if userID == 0 {
		return nil
	}
	_, err := h.db.Exec(`INSERT INTO notifications (user_id, title, content, type, target_type, target_id) VALUES (?, ?, ?, ?, ?, ?)`, userID, title, content, typ, targetType, targetID)
	return err
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

func mustInt64(value string) int64 {
	id, _ := strconv.ParseInt(value, 10, 64)
	return id
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

func timeAgo(value time.Time) string {
	diff := time.Since(value)
	if diff < time.Minute {
		return "刚刚"
	}
	if diff < time.Hour {
		return strconv.Itoa(int(diff.Minutes())) + "分钟前"
	}
	if diff < 24*time.Hour {
		return strconv.Itoa(int(diff.Hours())) + "小时前"
	}
	return value.Format("01-02")
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

func normalizeUserMediaForRequest(c *gin.Context, user gin.H) {
	avatarURL, _ := user["avatarUrl"].(string)
	user["avatarUrl"] = mediaURLForRequest(c, avatarURL)
}

func mediaURLForRequest(c *gin.Context, value string) string {
	if value == "" {
		return value
	}
	parsed, err := url.Parse(value)
	if err != nil || !strings.HasPrefix(parsed.Path, "/uploads/") {
		return value
	}
	if parsed.IsAbs() {
		hostname := strings.ToLower(parsed.Hostname())
		ip := net.ParseIP(hostname)
		if hostname != "localhost" && (ip == nil || (!ip.IsLoopback() && !ip.IsPrivate() && !ip.IsUnspecified())) {
			return value
		}
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	} else if forwarded := strings.TrimSpace(strings.Split(c.GetHeader("X-Forwarded-Proto"), ",")[0]); forwarded == "http" || forwarded == "https" {
		scheme = forwarded
	}
	return fmt.Sprintf("%s://%s%s", scheme, c.Request.Host, parsed.RequestURI())
}
