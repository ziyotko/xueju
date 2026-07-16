package database

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

func Open(dsn string, timezone ...string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("MYSQL_DSN is empty")
	}
	timezoneName := "Asia/Shanghai"
	if len(timezone) > 0 && strings.TrimSpace(timezone[0]) != "" {
		timezoneName = strings.TrimSpace(timezone[0])
	}
	normalizedDSN, err := normalizeDSN(dsn, timezoneName)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", normalizedDSN)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func normalizeDSN(dsn, timezoneName string) (string, error) {
	location, err := time.LoadLocation(timezoneName)
	if err != nil {
		return "", fmt.Errorf("load mysql timezone %q: %w", timezoneName, err)
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("parse MYSQL_DSN: %w", err)
	}
	cfg.Loc = location
	if cfg.Params == nil {
		cfg.Params = map[string]string{}
	}
	if _, configured := cfg.Params["time_zone"]; !configured {
		_, offsetSeconds := time.Now().In(location).Zone()
		cfg.Params["time_zone"] = mysqlTimezoneOffset(offsetSeconds)
	}
	return cfg.FormatDSN(), nil
}

func mysqlTimezoneOffset(offsetSeconds int) string {
	sign := "+"
	if offsetSeconds < 0 {
		sign = "-"
		offsetSeconds = -offsetSeconds
	}
	hours := offsetSeconds / 3600
	minutes := (offsetSeconds % 3600) / 60
	return fmt.Sprintf("'%s%02d:%02d'", sign, hours, minutes)
}
