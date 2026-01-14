// cmd/api/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"investment_monitoring/config"
	handler "investment_monitoring/internal/http"
	"investment_monitoring/internal/http/internal_middleware"
	"investment_monitoring/pkg/logger"

	repository "investment_monitoring/internal/repository/entity"

	"github.com/go-chi/chi/v5"
	chiMw "github.com/go-chi/chi/v5/middleware"
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
		panic("Cannot connect to Redis" + err.Error())
	}

	// HTTP router root
	r := chi.NewRouter()
	r.Use(chiMw.RequestID)
	r.Use(internal_middleware.RequestLogger)
	r.Use(chiMw.StripSlashes)
	r.Use(chiMw.Recoverer)

	r.Mount("/api", handler.NewRouter(store, rdb, cfg))

	srv := &http.Server{
		Addr:    ":" + cfg.PORT,
		Handler: r,
	}

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Server running on port", cfg.PORT)
		chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			logger.L().Info("route", "method", method, "route", route)
			return nil
		})

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)
}
