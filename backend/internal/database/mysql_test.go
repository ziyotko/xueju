package database

import (
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestNormalizeDSNAppliesApplicationTimezone(t *testing.T) {
	dsn, err := normalizeDSN("user:pass@tcp(127.0.0.1:3306)/xueju?parseTime=true", "Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Loc == nil || cfg.Loc.String() != "Asia/Shanghai" {
		t.Fatalf("unexpected mysql location: %v", cfg.Loc)
	}
	if cfg.Params["time_zone"] != "'+08:00'" {
		t.Fatalf("unexpected mysql session timezone: %q", cfg.Params["time_zone"])
	}
}

func TestNormalizeDSNPreservesExplicitTimezone(t *testing.T) {
	dsn, err := normalizeDSN("user:pass@tcp(127.0.0.1:3306)/xueju?time_zone=%27%2B00%3A00%27", "Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Params["time_zone"] != "'+00:00'" {
		t.Fatalf("explicit mysql timezone was overwritten: %q", cfg.Params["time_zone"])
	}
}
