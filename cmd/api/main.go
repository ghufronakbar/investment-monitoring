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
	"investment_monitoring/internal/audit"
	handler "investment_monitoring/internal/http"
	"investment_monitoring/internal/http/internal_middleware"
	"investment_monitoring/pkg/logger"

	repository "investment_monitoring/internal/repository/entity"

	"github.com/go-chi/chi/v5"
	chiMw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	cfg := config.Load()

	db, err := config.ConnectDB(cfg.DATABASE_URL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	store := repository.NewStore(db)

	// Redis is currently not used by any service/handler.
	// Keep this disabled so the API can run without a local Redis instance.
	// rdb, err := config.ConnectRedis(cfg)
	// if err != nil {
	// 	panic("Cannot connect to Redis" + err.Error())
	// }

	// HTTP router root
	r := chi.NewRouter()

	// --- SETUP CORS: ALLOW ALL ---
	r.Use(cors.Handler(cors.Options{
		// AllowedOrigins:   []string{"https://foo.com"}, // Gunakan ini untuk restrict domain tertentu
		AllowedOrigins:   []string{"https://*", "http://*"}, // Allow semua https dan http
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	}))
	// -----------------------------

	r.Use(chiMw.RequestID)
	r.Use(audit.Middleware)
	r.Use(internal_middleware.RequestLogger)
	r.Use(chiMw.StripSlashes)
	r.Use(chiMw.Recoverer)

	r.Mount("/api", handler.NewRouter(store, nil, cfg))

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
