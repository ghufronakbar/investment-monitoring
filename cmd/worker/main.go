package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"investment_monitoring/config"
	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"
	token_service "investment_monitoring/internal/module/support/token/service"
	"investment_monitoring/internal/outbox"
	repository "investment_monitoring/internal/repository/entity"
)

func main() {
	cfg := config.Load()

	db, err := config.ConnectDB(cfg.DATABASE_URL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	store := repository.NewStore(db)

	rdb, err := config.ConnectRedis(cfg)
	if err != nil {
		log.Println("Redis unavailable, realtime outbox delivery to API instances is disabled:", err)
	} else {
		defer rdb.Close()
	}

	tokenSvc := token_service.NewService(cfg)
	realtimePublisher := websocket_service.NewService(cfg, tokenSvc, store, rdb)
	processor := outbox.NewProcessor(store, realtimePublisher)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("Outbox worker started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox worker stopped")
			return
		case <-ticker.C:
			if err := processor.ProcessBatch(ctx, 25); err != nil {
				log.Println("Outbox worker error:", err)
			}
		}
	}
}
