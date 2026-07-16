package database

import (
	"database/sql"
	"fmt"
	"strings"
)

func Migrate(db *sql.DB) error {
	if db == nil {
		return nil
	}

	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			openid VARCHAR(128) NOT NULL UNIQUE,
			unionid VARCHAR(128) NULL,
			nickname VARCHAR(64) NOT NULL DEFAULT '雪友',
			avatar_url VARCHAR(255) NOT NULL DEFAULT '',
			gender TINYINT NOT NULL DEFAULT 0,
			gender_visible TINYINT NOT NULL DEFAULT 1,
			phone VARCHAR(32) NULL,
			bio VARCHAR(500) NOT NULL DEFAULT '',
			city VARCHAR(64) NOT NULL DEFAULT '',
			ski_type VARCHAR(32) NOT NULL DEFAULT '',
			ski_level VARCHAR(32) NOT NULL DEFAULT '',
			style_tags JSON NULL,
			favorite_resorts JSON NULL,
			has_car TINYINT NOT NULL DEFAULT 0,
			credit_score DECIMAL(3,1) NOT NULL DEFAULT 5.0,
			event_count INT NOT NULL DEFAULT 0,
			join_count INT NOT NULL DEFAULT 0,
			good_rate DECIMAL(5,2) NOT NULL DEFAULT 100.00,
			real_name_status VARCHAR(32) NOT NULL DEFAULT 'unverified',
			real_name_verified_at DATETIME NULL,
			verification_status VARCHAR(32) NOT NULL DEFAULT 'unverified',
			verification_method VARCHAR(32) NOT NULL DEFAULT '',
			verified_at DATETIME NULL,
			phone_masked VARCHAR(32) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'normal',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			deleted_at DATETIME NULL
		)`,
		`CREATE TABLE IF NOT EXISTS ski_resorts (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(128) NOT NULL,
			city VARCHAR(64) NOT NULL,
			province VARCHAR(64) NOT NULL DEFAULT '',
			image_url VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'normal',
			sort INT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ski_events (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			title VARCHAR(128) NOT NULL,
			creator_id BIGINT NOT NULL,
			resort_id BIGINT NOT NULL DEFAULT 0,
			resort_name VARCHAR(128) NOT NULL,
			event_date DATE NULL,
			start_time DATETIME NULL,
			depart_city VARCHAR(64) NOT NULL DEFAULT '',
			depart_area VARCHAR(128) NOT NULL DEFAULT '',
			meet_place VARCHAR(255) NOT NULL DEFAULT '',
			traffic_type VARCHAR(32) NOT NULL DEFAULT '',
			max_members INT NOT NULL DEFAULT 1,
			current_members INT NOT NULL DEFAULT 1,
			ski_type_req VARCHAR(32) NOT NULL DEFAULT '',
			level_req VARCHAR(32) NOT NULL DEFAULT '',
			purpose_tags JSON NULL,
			allow_beginner TINYINT NOT NULL DEFAULT 0,
			same_gender_only TINYINT NOT NULL DEFAULT 0,
			allow_car_pool TINYINT NOT NULL DEFAULT 0,
			allow_room_share TINYINT NOT NULL DEFAULT 0,
			allow_photo TINYINT NOT NULL DEFAULT 0,
			cost_desc VARCHAR(255) NOT NULL DEFAULT '',
			remark TEXT NULL,
			image_url VARCHAR(500) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'recruiting',
			view_count INT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			deleted_at DATETIME NULL,
			INDEX idx_ski_events_creator (creator_id),
			INDEX idx_ski_events_status_date (status, event_date)
		)`,
		`CREATE TABLE IF NOT EXISTS event_members (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			event_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			role VARCHAR(32) NOT NULL DEFAULT 'member',
			status VARCHAR(32) NOT NULL DEFAULT 'active',
			joined_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_event_user (event_id, user_id),
			INDEX idx_event_members_user (user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS join_requests (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			event_id BIGINT NOT NULL,
			applicant_id BIGINT NOT NULL,
			creator_id BIGINT NOT NULL,
			ski_level VARCHAR(32) NOT NULL DEFAULT '',
			ski_type VARCHAR(32) NOT NULL DEFAULT '',
			has_car TINYINT NOT NULL DEFAULT 0,
			can_carry_people TINYINT NOT NULL DEFAULT 0,
			depart_area VARCHAR(128) NOT NULL DEFAULT '',
			message VARCHAR(500) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			reject_reason VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_join_event_applicant (event_id, applicant_id),
			INDEX idx_join_applicant (applicant_id),
			INDEX idx_join_creator (creator_id)
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			event_id BIGINT NOT NULL,
			sender_id BIGINT NOT NULL,
			message_type VARCHAR(32) NOT NULL DEFAULT 'text',
			content TEXT NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'normal',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_chat_event (event_id, id)
		)`,
		`CREATE TABLE IF NOT EXISTS chat_reads (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			event_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			last_read_message_id BIGINT NOT NULL DEFAULT 0,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_chat_read_user_event (event_id, user_id),
			INDEX idx_chat_reads_user (user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS reviews (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			event_id BIGINT NOT NULL,
			reviewer_id BIGINT NOT NULL,
			reviewee_id BIGINT NOT NULL,
			score INT NOT NULL DEFAULT 5,
			positive_tags JSON NULL,
			negative_tags JSON NULL,
			content VARCHAR(500) NOT NULL DEFAULT '',
			is_anonymous TINYINT NOT NULL DEFAULT 0,
			status VARCHAR(32) NOT NULL DEFAULT 'normal',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_review_once (event_id, reviewer_id, reviewee_id),
			INDEX idx_reviews_reviewee (reviewee_id)
		)`,
		`CREATE TABLE IF NOT EXISTS reports (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			reporter_id BIGINT NOT NULL,
			target_type VARCHAR(32) NOT NULL,
			target_id BIGINT NOT NULL,
			reason VARCHAR(255) NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			result VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_reports_status (status)
		)`,
		`CREATE TABLE IF NOT EXISTS event_favorites (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			event_id BIGINT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_favorite_user_event (user_id, event_id),
			INDEX idx_favorites_user (user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_follows (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			follower_id BIGINT NOT NULL,
			followee_id BIGINT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_follow_user (follower_id, followee_id),
			INDEX idx_follows_follower (follower_id),
			INDEX idx_follows_followee (followee_id)
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			title VARCHAR(128) NOT NULL,
			content VARCHAR(500) NOT NULL DEFAULT '',
			type VARCHAR(32) NOT NULL DEFAULT 'system',
			target_type VARCHAR(32) NOT NULL DEFAULT '',
			target_id BIGINT NOT NULL DEFAULT 0,
			is_read TINYINT NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_notifications_user_read (user_id, is_read, created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS join_request_history (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			join_request_id BIGINT NOT NULL,
			event_id BIGINT NOT NULL,
			applicant_id BIGINT NOT NULL,
			from_status VARCHAR(32) NOT NULL DEFAULT '',
			to_status VARCHAR(32) NOT NULL,
			operator_id BIGINT NOT NULL DEFAULT 0,
			reason VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_join_history_request (join_request_id, created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS media_uploads (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			kind VARCHAR(32) NOT NULL,
			path VARCHAR(500) NOT NULL,
			public_url VARCHAR(500) NOT NULL DEFAULT '',
			review_token VARCHAR(64) NOT NULL DEFAULT '',
			trace_id VARCHAR(128) NOT NULL DEFAULT '',
			mime_type VARCHAR(64) NOT NULL,
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			review_result VARCHAR(255) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_media_status (status, created_at),
			INDEX idx_media_trace (trace_id),
			INDEX idx_media_user (user_id, created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS admin_action_logs (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			admin_username VARCHAR(128) NOT NULL,
			resource VARCHAR(64) NOT NULL,
			target_id VARCHAR(64) NOT NULL,
			action VARCHAR(64) NOT NULL,
			before_status VARCHAR(32) NOT NULL DEFAULT '',
			after_status VARCHAR(32) NOT NULL DEFAULT '',
			detail VARCHAR(1000) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_admin_logs_created (created_at),
			INDEX idx_admin_logs_target (resource, target_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_verifications (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			status VARCHAR(32) NOT NULL,
			method VARCHAR(32) NOT NULL DEFAULT 'sms_phone',
			provider VARCHAR(64) NOT NULL DEFAULT 'aliyun_sms_auth',
			provider_reference VARCHAR(255) NOT NULL DEFAULT '',
			phone_encrypted TEXT NOT NULL,
			phone_hash CHAR(64) NOT NULL,
			active_phone_hash CHAR(64) NULL,
			phone_masked VARCHAR(32) NOT NULL,
			verified_at DATETIME NULL,
			revoked_at DATETIME NULL,
			revoked_by VARCHAR(128) NOT NULL DEFAULT '',
			revoke_reason VARCHAR(500) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uniq_active_phone_hash (active_phone_hash),
			INDEX idx_verifications_user (user_id, created_at),
			INDEX idx_verifications_phone (phone_hash)
		)`,
		`CREATE TABLE IF NOT EXISTS verification_sms_requests (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			request_id VARCHAR(64) NOT NULL UNIQUE,
			user_id BIGINT NOT NULL,
			phone_hash CHAR(64) NOT NULL,
			phone_masked VARCHAR(32) NOT NULL,
			provider_reference VARCHAR(255) NOT NULL DEFAULT '',
			status VARCHAR(32) NOT NULL DEFAULT 'pending',
			attempts INT NOT NULL DEFAULT 0,
			expires_at DATETIME NOT NULL,
			ip_address VARCHAR(64) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_sms_user_created (user_id, created_at),
			INDEX idx_sms_phone_created (phone_hash, created_at),
			INDEX idx_sms_ip_created (ip_address, created_at)
		)`,
		`CREATE TABLE IF NOT EXISTS verification_audit_logs (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			verification_id BIGINT NOT NULL DEFAULT 0,
			action VARCHAR(64) NOT NULL,
			operator_type VARCHAR(32) NOT NULL,
			operator_id VARCHAR(128) NOT NULL DEFAULT '',
			old_status VARCHAR(32) NOT NULL DEFAULT '',
			new_status VARCHAR(32) NOT NULL DEFAULT '',
			request_id VARCHAR(64) NOT NULL DEFAULT '',
			provider_reference VARCHAR(255) NOT NULL DEFAULT '',
			ip_address VARCHAR(64) NOT NULL DEFAULT '',
			user_agent VARCHAR(500) NOT NULL DEFAULT '',
			remark VARCHAR(500) NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_verification_audit_user (user_id, created_at),
			INDEX idx_verification_audit_request (request_id)
		)`,
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version BIGINT PRIMARY KEY,
			name VARCHAR(128) NOT NULL,
			applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	if err := ensureColumn(db, "ski_events", "image_url", "VARCHAR(500) NOT NULL DEFAULT '' AFTER remark"); err != nil {
		return err
	}
	if err := applyMigration(db, 2026071301, "launch hardening", func(tx *sql.Tx) error {
		if err := ensureColumnTx(tx, "users", "bio", "VARCHAR(500) NOT NULL DEFAULT '' AFTER phone"); err != nil {
			return err
		}
		// Phone numbers are no longer part of the product. Keep the nullable
		// column for rollback compatibility, but erase legacy PII once.
		if _, err := tx.Exec(`UPDATE users SET phone=NULL WHERE phone IS NOT NULL`); err != nil {
			return err
		}
		// Older schemas allowed one row per status, which made a second reject
		// collide with the first. Keep the newest row and enforce one current
		// application per event/user pair.
		if _, err := tx.Exec(`DELETE older FROM join_requests older JOIN join_requests newer
			ON older.event_id=newer.event_id AND older.applicant_id=newer.applicant_id AND older.id<newer.id`); err != nil {
			return err
		}
		if err := dropIndexIfExists(tx, "join_requests", "uniq_join_pending"); err != nil {
			return err
		}
		return ensureUniqueIndex(tx, "join_requests", "uniq_join_event_applicant", "event_id, applicant_id")
	}); err != nil {
		return err
	}
	if err := ensureColumn(db, "media_uploads", "review_token", "VARCHAR(64) NOT NULL DEFAULT '' AFTER public_url"); err != nil {
		return err
	}
	if err := ensureColumn(db, "media_uploads", "trace_id", "VARCHAR(128) NOT NULL DEFAULT '' AFTER review_token"); err != nil {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX idx_media_trace ON media_uploads (trace_id)`); err != nil && !isDuplicateIndexError(err) {
		return err
	}
	if err := applyMigration(db, 2026071302, "register legacy media", func(tx *sql.Tx) error {
		// Media that was already referenced by a user or event predates the
		// asynchronous review table. Grandfather only those referenced files;
		// all newly uploaded media continues through the pending review flow.
		if _, err := tx.Exec(`INSERT INTO media_uploads (user_id, kind, path, public_url, mime_type, status, review_result)
			SELECT u.id, 'avatars', SUBSTRING_INDEX(u.avatar_url, '/uploads/', -1), u.avatar_url,
				CASE WHEN LOWER(u.avatar_url) LIKE '%.png' THEN 'image/png' ELSE 'image/jpeg' END,
				'approved', 'legacy_migration'
			FROM users u
			WHERE u.avatar_url LIKE '%/uploads/%'
			AND NOT EXISTS (
				SELECT 1 FROM media_uploads m
				WHERE m.path=SUBSTRING_INDEX(u.avatar_url, '/uploads/', -1)
			)`); err != nil {
			return err
		}
		_, err := tx.Exec(`INSERT INTO media_uploads (user_id, kind, path, public_url, mime_type, status, review_result)
			SELECT e.creator_id, 'events', SUBSTRING_INDEX(e.image_url, '/uploads/', -1), e.image_url,
				CASE WHEN LOWER(e.image_url) LIKE '%.png' THEN 'image/png' ELSE 'image/jpeg' END,
				'approved', 'legacy_migration'
			FROM ski_events e
			WHERE e.image_url LIKE '%/uploads/%'
			AND NOT EXISTS (
				SELECT 1 FROM media_uploads m
				WHERE m.path=SUBSTRING_INDEX(e.image_url, '/uploads/', -1)
			)`)
		return err
	}); err != nil {
		return err
	}
	if err := applyMigration(db, 2026071501, "real name and group compliance", func(tx *sql.Tx) error {
		if err := ensureColumnTx(tx, "users", "real_name_status", "VARCHAR(32) NOT NULL DEFAULT 'unverified' AFTER good_rate"); err != nil {
			return err
		}
		if err := ensureColumnTx(tx, "users", "real_name_verified_at", "DATETIME NULL AFTER real_name_status"); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE ski_events SET max_members=20 WHERE max_members>20 AND current_members<=20`); err != nil {
			return err
		}
		var oversized int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM ski_events WHERE current_members>20 AND deleted_at IS NULL`).Scan(&oversized); err != nil {
			return err
		}
		if oversized > 0 {
			return fmt.Errorf("%d existing groups exceed the 20-member compliance limit and require manual remediation", oversized)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := applyMigration(db, 2026071502, "self service sms phone verification", func(tx *sql.Tx) error {
		columns := []struct{ name, definition string }{
			{"verification_status", "VARCHAR(32) NOT NULL DEFAULT 'unverified' AFTER real_name_verified_at"},
			{"verification_method", "VARCHAR(32) NOT NULL DEFAULT '' AFTER verification_status"},
			{"verified_at", "DATETIME NULL AFTER verification_method"},
			{"phone_masked", "VARCHAR(32) NOT NULL DEFAULT '' AFTER verified_at"},
		}
		for _, column := range columns {
			if err := ensureColumnTx(tx, "users", column.name, column.definition); err != nil {
				return err
			}
		}
		// Administrator-only legacy approvals have no SMS proof and must not
		// remain valid after enabling self-service phone verification.
		_, err := tx.Exec(`UPDATE users SET verification_status=CASE WHEN real_name_status='verified' THEN 'reverify_required' ELSE 'unverified' END, verification_method='', verified_at=NULL, phone_masked=''`)
		return err
	}); err != nil {
		return err
	}

	seed := []string{
		`INSERT INTO ski_resorts (name, city, province, image_url, sort)
		 SELECT '崇礼 · 万龙滑雪场', '张家口', '河北', 'https://images.unsplash.com/photo-1740137660688-3d3f2b5422b6?auto=format&fit=crop&w=900&q=80', 10
		 WHERE NOT EXISTS (SELECT 1 FROM ski_resorts WHERE name = '崇礼 · 万龙滑雪场')`,
		`INSERT INTO ski_resorts (name, city, province, image_url, sort)
		 SELECT '南山滑雪场', '北京', '北京', 'https://images.unsplash.com/photo-1678973751386-66d7bd0f5abf?auto=format&fit=crop&w=900&q=80', 20
		 WHERE NOT EXISTS (SELECT 1 FROM ski_resorts WHERE name = '南山滑雪场')`,
		`INSERT INTO ski_resorts (name, city, province, image_url, sort)
		 SELECT '云顶滑雪公园', '张家口', '河北', 'https://images.unsplash.com/photo-1707290796500-95016769aa11?auto=format&fit=crop&w=900&q=80', 30
		 WHERE NOT EXISTS (SELECT 1 FROM ski_resorts WHERE name = '云顶滑雪公园')`,
		`INSERT INTO ski_resorts (name, city, province, image_url, sort)
		 SELECT '太舞滑雪小镇', '张家口', '河北', 'https://images.unsplash.com/photo-1558733467-11cef06eb6d8?auto=format&fit=crop&w=900&q=80', 40
		 WHERE NOT EXISTS (SELECT 1 FROM ski_resorts WHERE name = '太舞滑雪小镇')`,
	}
	for _, statement := range seed {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}

func applyMigration(db *sql.DB, version int64, name string, fn func(*sql.Tx) error) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, version).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return fmt.Errorf("migration %d %s: %w", version, name, err)
	}
	if _, err := tx.Exec(`INSERT INTO schema_migrations (version, name) VALUES (?, ?)`, version, name); err != nil {
		return err
	}
	return tx.Commit()
}

func ensureColumnTx(tx *sql.Tx, table, column, definition string) error {
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME=?`, table, column).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := tx.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}

func dropIndexIfExists(tx *sql.Tx, table, index string) error {
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND INDEX_NAME=?`, table, index).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	_, err := tx.Exec(`ALTER TABLE ` + table + ` DROP INDEX ` + index)
	return err
}

func ensureUniqueIndex(tx *sql.Tx, table, index, columns string) error {
	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND INDEX_NAME=?`, table, index).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := tx.Exec(`ALTER TABLE ` + table + ` ADD UNIQUE INDEX ` + index + ` (` + columns + `)`)
	return err
}

func isDuplicateIndexError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key name") || strings.Contains(message, "already exists")
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? AND COLUMN_NAME=?`, table, column).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err := db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + column + ` ` + definition)
	return err
}
