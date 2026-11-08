package main

import (
	"context"
	"flag"
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

	"github.com/lib/pq"
)

const (
	outboxChannel = "outbox_events"
	batchSize     = 25
)

func main() {
	pollInterval := flag.Duration("poll-interval", 5*time.Second, "fallback polling interval for outbox events")
	listen := flag.Bool("listen", true, "wake on PostgreSQL NOTIFY in addition to polling")
	flag.Parse()

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

	ticker := time.NewTicker(*pollInterval)
	defer ticker.Stop()

	// LISTEN wakes the worker right after an event commits; the ticker stays as
	// a fallback for missed notifications or when LISTEN is unavailable.
	var notify <-chan *pq.Notification
	if *listen {
		listener := pq.NewListener(cfg.DATABASE_URL, time.Second, time.Minute, func(ev pq.ListenerEventType, err error) {
			if err != nil {
				log.Println("Outbox listener:", err)
			}
		})
		if err := listener.Listen(outboxChannel); err != nil {
			log.Println("Outbox LISTEN unavailable, falling back to polling only:", err)
		} else {
			defer listener.Close()
			notify = listener.Notify
		}
	}

	log.Printf("Outbox worker started poll_interval=%s listen=%v", *pollInterval, notify != nil)

	process := func() {
		if err := processor.ProcessBatch(ctx, batchSize); err != nil {
			log.Println("Outbox worker error:", err)
		}
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Outbox worker stopped")
			return
		case <-ticker.C:
			process()
		case <-notify:
			process()
		}
	}
}
