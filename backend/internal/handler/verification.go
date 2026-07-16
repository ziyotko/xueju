package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"xueju/backend/internal/phoneverification"
	"xueju/backend/internal/response"
)

const (
	verificationMethod   = "sms_phone"
	verificationProvider = "aliyun_sms_auth"
)

var smsCodePattern = regexp.MustCompile(`^\d{4,8}$`)

func (h *AppHandler) VerificationStatus(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var status, method, masked string
	var verifiedAt sql.NullTime
	if err := h.db.QueryRow(`SELECT verification_status, verification_method, phone_masked, verified_at FROM users WHERE id=?`, userID).Scan(&status, &method, &masked, &verifiedAt); err != nil {
		h.sqlError(c, err)
		return
	}
	response.Success(c, verificationPayload(status, method, masked, verifiedAt))
}

func (h *AppHandler) SendPhoneVerificationCode(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var req struct {
		Phone       string `json:"phone"`
		Agreed      bool   `json:"agreed"`
		ChangePhone bool   `json:"changePhone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !req.Agreed {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "请先阅读并同意用户协议和隐私政策")
		return
	}
	var currentStatus string
	if err := h.db.QueryRow(`SELECT verification_status FROM users WHERE id=?`, userID).Scan(&currentStatus); err != nil {
		h.sqlError(c, err)
		return
	}
	if currentStatus == "verified" && !req.ChangePhone {
		response.Error(c, http.StatusConflict, response.CodePhoneAlreadyVerified, "手机号已完成认证，如需更换请先进入更换手机号流程")
		return
	}
	if currentStatus != "verified" && req.ChangePhone {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "当前账号无需更换手机号，请直接完成认证")
		return
	}
	if currentStatus == "revoked" {
		response.Error(c, http.StatusForbidden, response.CodePhoneVerificationRevoked, "手机号认证已被撤销，请联系管理员处理")
		return
	}
	phone, err := phoneverification.Normalize(req.Phone)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "请输入正确的中国大陆手机号")
		return
	}
	if h.cfg.PhoneHashSecret == "" || h.cfg.PhoneEncryptionKey == "" {
		response.Error(c, http.StatusServiceUnavailable, response.CodeServerError, "phone verification is not configured")
		return
	}
	phoneHash := phoneverification.Hash(phone, h.cfg.PhoneHashSecret)
	if !h.allowSmsSend(c, userID, phoneHash) {
		return
	}

	requestID := secureRequestID()
	expires := normalizedPositive(h.cfg.SmsCodeExpireSeconds, 300)
	masked := phoneverification.Mask(phone)
	expiresAt := time.Now().Add(time.Duration(expires) * time.Second)
	if _, err := h.db.Exec(`INSERT INTO verification_sms_requests (request_id, user_id, phone_hash, phone_masked, expires_at, ip_address) VALUES (?, ?, ?, ?, ?, ?)`, requestID, userID, phoneHash, masked, expiresAt, c.ClientIP()); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}

	result, err := h.phoneVerifier.Send(c.Request.Context(), phone, requestID)
	if err != nil {
		_, _ = h.db.Exec(`UPDATE verification_sms_requests SET status='provider_failed' WHERE request_id=?`, requestID)
		h.writeVerificationAudit(userID, 0, "sms_send_failed", "user", userIDString(userID), "", "", requestID, "", c, "provider request failed")
		response.Error(c, http.StatusBadGateway, response.CodeServerError, err.Error())
		return
	}
	_, _ = h.db.Exec(`UPDATE verification_sms_requests SET provider_reference=? WHERE request_id=?`, result.ProviderReference, requestID)
	_, _ = h.db.Exec(`UPDATE users SET verification_status=IF(verification_status='verified','verified','pending') WHERE id=?`, userID)
	auditAction := "sms_code_sent"
	if req.ChangePhone {
		auditAction = "phone_change_code_sent"
	}
	h.writeVerificationAudit(userID, 0, auditAction, "user", userIDString(userID), currentStatus, "pending", requestID, result.ProviderReference, c, masked)
	response.Success(c, gin.H{"success": true, "requestId": requestID, "expiresIn": expires})
}

func (h *AppHandler) CheckPhoneVerificationCode(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok || !h.ensureActive(c, userID) {
		return
	}
	var req struct {
		Phone       string `json:"phone"`
		Code        string `json:"code"`
		ChangePhone bool   `json:"changePhone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "手机号或验证码格式错误")
		return
	}
	var currentStatus string
	if err := h.db.QueryRow(`SELECT verification_status FROM users WHERE id=?`, userID).Scan(&currentStatus); err != nil {
		h.sqlError(c, err)
		return
	}
	if currentStatus == "verified" && !req.ChangePhone {
		response.Error(c, http.StatusConflict, response.CodePhoneAlreadyVerified, "手机号已完成认证，如需更换请先进入更换手机号流程")
		return
	}
	if currentStatus != "verified" && req.ChangePhone {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "当前账号无需更换手机号，请直接完成认证")
		return
	}
	if currentStatus == "revoked" {
		response.Error(c, http.StatusForbidden, response.CodePhoneVerificationRevoked, "手机号认证已被撤销，请联系管理员处理")
		return
	}
	phone, err := phoneverification.Normalize(req.Phone)
	if err != nil || !smsCodePattern.MatchString(strings.TrimSpace(req.Code)) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "手机号或验证码格式错误")
		return
	}
	phoneHash := phoneverification.Hash(phone, h.cfg.PhoneHashSecret)
	var requestID, providerReference, status string
	var attempts int
	err = h.db.QueryRow(`SELECT request_id, provider_reference, status, attempts FROM verification_sms_requests WHERE user_id=? AND phone_hash=? AND expires_at>NOW() ORDER BY id DESC LIMIT 1`, userID, phoneHash).Scan(&requestID, &providerReference, &status, &attempts)
	if errors.Is(err, sql.ErrNoRows) || status != "pending" {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "验证码已过期，请重新获取")
		return
	}
	if err != nil {
		h.sqlError(c, err)
		return
	}
	if attempts >= 5 {
		response.Error(c, http.StatusTooManyRequests, response.CodeBadRequest, "验证码错误次数过多，请重新获取")
		return
	}

	checked, err := h.phoneVerifier.Check(c.Request.Context(), phone, strings.TrimSpace(req.Code), requestID)
	if err != nil {
		h.writeVerificationAudit(userID, 0, "sms_check_failed", "user", userIDString(userID), "", "", requestID, providerReference, c, "provider request failed")
		response.Error(c, http.StatusBadGateway, response.CodeServerError, err.Error())
		return
	}
	if !checked.Passed {
		_, _ = h.db.Exec(`UPDATE verification_sms_requests SET attempts=attempts+1, status=IF(attempts+1>=5,'locked','pending') WHERE request_id=? AND status='pending'`, requestID)
		h.writeVerificationAudit(userID, 0, "sms_check_failed", "user", userIDString(userID), "", "", requestID, checked.ProviderReference, c, "verification code rejected")
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "验证码错误")
		return
	}

	encrypted, err := phoneverification.Encrypt(phone, h.cfg.PhoneEncryptionKey)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	masked := phoneverification.Mask(phone)
	tx, err := h.db.BeginTx(c.Request.Context(), nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	defer tx.Rollback()
	var lockedStatus string
	if err := tx.QueryRow(`SELECT status FROM verification_sms_requests WHERE request_id=? AND user_id=? AND phone_hash=? AND expires_at>NOW() FOR UPDATE`, requestID, userID, phoneHash).Scan(&lockedStatus); err != nil || lockedStatus != "pending" {
		response.Error(c, http.StatusConflict, response.CodeBadRequest, "验证码已使用或已过期")
		return
	}
	var duplicate int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM user_verifications WHERE active_phone_hash=? AND user_id<>?`, phoneHash, userID).Scan(&duplicate); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if duplicate > 0 {
		response.Error(c, http.StatusConflict, response.CodeBadRequest, "该手机号已绑定其他账号")
		return
	}
	var oldStatus string
	if err := tx.QueryRow(`SELECT verification_status FROM users WHERE id=? FOR UPDATE`, userID).Scan(&oldStatus); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	_, _ = tx.Exec(`UPDATE user_verifications SET active_phone_hash=NULL, status='replaced', revoked_at=NOW(), revoke_reason='phone_changed' WHERE user_id=? AND active_phone_hash IS NOT NULL`, userID)
	providerRef := firstValue(checked.ProviderReference, providerReference)
	result, err := tx.Exec(`INSERT INTO user_verifications (user_id, status, method, provider, provider_reference, phone_encrypted, phone_hash, active_phone_hash, phone_masked, verified_at) VALUES (?, 'verified', ?, ?, ?, ?, ?, ?, ?, NOW())`, userID, verificationMethod, verificationProvider, providerRef, encrypted, phoneHash, phoneHash, masked)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			response.Error(c, http.StatusConflict, response.CodeBadRequest, "该手机号已绑定其他账号")
			return
		}
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	verificationID, _ := result.LastInsertId()
	if _, err := tx.Exec(`UPDATE users SET verification_status='verified', verification_method=?, verified_at=NOW(), phone_masked=? WHERE id=?`, verificationMethod, masked, userID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`UPDATE verification_sms_requests SET status='verified', provider_reference=? WHERE request_id=?`, providerRef, requestID); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if _, err := tx.Exec(`INSERT INTO verification_audit_logs (user_id, verification_id, action, operator_type, operator_id, old_status, new_status, request_id, provider_reference, ip_address, user_agent, remark) VALUES (?, ?, 'sms_check_succeeded', 'user', ?, ?, 'verified', ?, ?, ?, ?, ?)`, userID, verificationID, userIDString(userID), oldStatus, requestID, providerRef, c.ClientIP(), limitedUserAgent(c), masked); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	action := "phone_bound"
	if oldStatus == "verified" {
		action = "phone_changed"
	}
	if _, err := tx.Exec(`INSERT INTO verification_audit_logs (user_id, verification_id, action, operator_type, operator_id, old_status, new_status, request_id, provider_reference, ip_address, user_agent, remark) VALUES (?, ?, ?, 'user', ?, ?, 'verified', ?, ?, ?, ?, ?)`, userID, verificationID, action, userIDString(userID), oldStatus, requestID, providerRef, c.ClientIP(), limitedUserAgent(c), masked); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	if err := tx.Commit(); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeServerError, err.Error())
		return
	}
	response.Success(c, gin.H{"success": true, "verificationStatus": "verified", "verificationMethod": verificationMethod, "phoneMasked": masked, "verifiedAt": time.Now()})
}

func (h *AppHandler) allowSmsSend(c *gin.Context, userID int64, phoneHash string) bool {
	interval := normalizedPositive(h.cfg.SmsSendIntervalSeconds, 60)
	var last sql.NullTime
	_ = h.db.QueryRow(`SELECT MAX(created_at) FROM verification_sms_requests WHERE phone_hash=?`, phoneHash).Scan(&last)
	if last.Valid && time.Since(last.Time) < time.Duration(interval)*time.Second {
		response.Error(c, http.StatusTooManyRequests, response.CodeBadRequest, "发送过于频繁，请稍后再试")
		return false
	}
	checks := []struct {
		query string
		args  []interface{}
		limit int
	}{
		{`SELECT COUNT(*) FROM verification_sms_requests WHERE phone_hash=? AND created_at>=CURDATE()`, []interface{}{phoneHash}, normalizedPositive(h.cfg.SmsDailyLimitPerPhone, 10)},
		{`SELECT COUNT(*) FROM verification_sms_requests WHERE user_id=? AND created_at>=CURDATE()`, []interface{}{userID}, normalizedPositive(h.cfg.SmsDailyLimitPerUser, 10)},
		{`SELECT COUNT(*) FROM verification_sms_requests WHERE ip_address=? AND created_at>=DATE_SUB(NOW(), INTERVAL 1 HOUR)`, []interface{}{c.ClientIP()}, normalizedPositive(h.cfg.SmsHourlyLimitPerIP, 30)},
	}
	for _, check := range checks {
		var count int
		if err := h.db.QueryRow(check.query, check.args...).Scan(&count); err != nil {
			h.sqlError(c, err)
			return false
		}
		if count >= check.limit {
			response.Error(c, http.StatusTooManyRequests, response.CodeBadRequest, "验证码发送次数已达上限，请稍后再试")
			return false
		}
	}
	return true
}

func (h *AppHandler) requireVerifiedUser(c *gin.Context, userID int64) bool {
	var status string
	if err := h.db.QueryRow(`SELECT verification_status FROM users WHERE id=? AND deleted_at IS NULL`, userID).Scan(&status); err != nil {
		h.sqlError(c, err)
		return false
	}
	switch status {
	case "verified":
		return true
	case "reverify_required":
		response.Error(c, http.StatusForbidden, response.CodePhoneReverificationRequired, "手机号认证状态已失效，请重新完成认证")
	case "revoked":
		response.Error(c, http.StatusForbidden, response.CodePhoneVerificationRevoked, "手机号认证已被撤销，暂时无法使用该功能")
	default:
		response.Error(c, http.StatusForbidden, response.CodePhoneVerificationRequired, "完成手机号认证后方可使用该功能")
	}
	return false
}

func verificationPayload(status, method, masked string, verifiedAt sql.NullTime) gin.H {
	result := gin.H{"verificationStatus": status, "verificationMethod": method, "phoneMasked": masked, "verifiedAt": nil}
	if verifiedAt.Valid {
		result["verifiedAt"] = verifiedAt.Time
	}
	return result
}

func nullableTime(value sql.NullTime) interface{} {
	if value.Valid {
		return value.Time
	}
	return nil
}

func (h *AppHandler) writeVerificationAudit(userID, verificationID int64, action, operatorType, operatorID, oldStatus, newStatus, requestID, providerRef string, c *gin.Context, remark string) {
	_, _ = h.db.Exec(`INSERT INTO verification_audit_logs (user_id, verification_id, action, operator_type, operator_id, old_status, new_status, request_id, provider_reference, ip_address, user_agent, remark) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, userID, verificationID, action, operatorType, operatorID, oldStatus, newStatus, requestID, providerRef, c.ClientIP(), limitedUserAgent(c), remark)
}

func secureRequestID() string {
	value := make([]byte, 16)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}

func normalizedPositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func userIDString(id int64) string { return strconv.FormatInt(id, 10) }
func limitedUserAgent(c *gin.Context) string {
	value := c.Request.UserAgent()
	if len(value) > 500 {
		return value[:500]
	}
	return value
}
func firstValue(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
