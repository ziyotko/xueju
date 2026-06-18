package main

import (
	"log"

	"xueju/backend/internal/config"
	"xueju/backend/internal/database"
	"xueju/backend/internal/router"
)

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.MySQLDSN)
	if err != nil {
		log.Printf("mysql connection skipped: %v", err)
	}
	if db != nil {
		defer db.Close()
	}

	engine := router.New(cfg, db)
	addr := ":" + cfg.Port

	log.Printf("%s listening on %s", cfg.AppName, addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}

