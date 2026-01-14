// internal/http/router.go
package http

import (
	"investment_monitoring/config"

	_ "investment_monitoring/internal/http/internal_middleware"
	subexample_controller "investment_monitoring/internal/module/example/subexample/controller"
	subexample_service "investment_monitoring/internal/module/example/subexample/service"
	token_service "investment_monitoring/internal/module/support/token/service"
	repository "investment_monitoring/internal/repository/entity"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func NewRouter(db *repository.SQLStore, rdb *redis.Client, cfg *config.Config) chi.Router {
	// 1. =========== INITIAL SERVICE / MIDDLEWARE ===========
	// TODO: create new service here
	_ = token_service.NewService(cfg)
	// _ = internal_middleware.AuthMiddleware(*tokenSvc)
	subexampleSvc := subexample_service.NewService(db)

	// 2. =========== INITIAL HANDLER ===========
	// TODO: create new http handler here
	subexampleHandler := subexample_controller.NewHandler(subexampleSvc)

	// 3. =========== MANAGE ROUTING ===========
	r := chi.NewRouter()

	r.Route("/example", func(r chi.Router) {
		// r.Use(auth)
		r.Mount("/subexample", subexampleHandler.Routes())
	})

	return r
}
