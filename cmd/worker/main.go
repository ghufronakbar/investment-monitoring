package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"investment_monitoring/config"
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
	processor := outbox.NewProcessor(store)

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
