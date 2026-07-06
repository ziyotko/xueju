package database

import "database/sql"

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
			UNIQUE KEY uniq_join_pending (event_id, applicant_id, status),
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
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	if err := ensureColumn(db, "ski_events", "image_url", "VARCHAR(500) NOT NULL DEFAULT '' AFTER remark"); err != nil {
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
