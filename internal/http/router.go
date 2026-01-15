// internal/http/router.go
package http

import (
	"investment_monitoring/config"

	"investment_monitoring/internal/http/internal_middleware"

	// Example module
	subexample_controller "investment_monitoring/internal/module/example/subexample/controller"
	subexample_service "investment_monitoring/internal/module/example/subexample/service"

	// IAM module
	auth_controller "investment_monitoring/internal/module/iam/auth/controller"
	auth_service "investment_monitoring/internal/module/iam/auth/service"
	organization_controller "investment_monitoring/internal/module/iam/organization/controller"
	organization_service "investment_monitoring/internal/module/iam/organization/service"

	// Project module
	project_controller "investment_monitoring/internal/module/project/controller"
	project_service "investment_monitoring/internal/module/project/service"

	// Monitoring module
	monitoring_controller "investment_monitoring/internal/module/monitoring/controller"
	monitoring_service "investment_monitoring/internal/module/monitoring/service"

	// Tracker module
	tracker_controller "investment_monitoring/internal/module/tracker/controller"
	tracker_service "investment_monitoring/internal/module/tracker/service"

	// Finance module
	finance_controller "investment_monitoring/internal/module/finance/controller"
	finance_service "investment_monitoring/internal/module/finance/service"

	// Support services
	mail_service "investment_monitoring/internal/module/support/mail/service"
	token_service "investment_monitoring/internal/module/support/token/service"

	repository "investment_monitoring/internal/repository/entity"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func NewRouter(db *repository.SQLStore, rdb *redis.Client, cfg *config.Config) chi.Router {
	// 1. =========== INITIAL SERVICE / MIDDLEWARE ===========
	// Support services
	tokenSvc := token_service.NewService(cfg)
	mailSvc := mail_service.NewService(cfg)

	// Auth middleware
	authMiddleware := internal_middleware.AuthMiddleware(*tokenSvc)

	// Example service
	subexampleSvc := subexample_service.NewService(db)

	// IAM services
	authSvc := auth_service.NewService(db, tokenSvc, cfg)
	orgSvc := organization_service.NewService(db, mailSvc, cfg)

	// Project services
	projectSvc := project_service.NewService(db, cfg)

	// Monitoring services
	monitoringSvc := monitoring_service.NewService(db, cfg)

	// Tracker services
	trackerSvc := tracker_service.NewService(db, cfg)

	// Finance services
	financeSvc := finance_service.NewService(db, cfg)

	// 2. =========== INITIAL HANDLER ===========
	subexampleHandler := subexample_controller.NewHandler(subexampleSvc)
	authHandler := auth_controller.NewHandler(authSvc)
	orgHandler := organization_controller.NewHandler(orgSvc, db)
	projectHandler := project_controller.NewHandler(projectSvc, db)
	monitoringHandler := monitoring_controller.NewHandler(monitoringSvc, db)
	trackerHandler := tracker_controller.NewHandler(trackerSvc, db)
	financeHandler := finance_controller.NewHandler(financeSvc, db)

	// 3. =========== MANAGE ROUTING ===========
	r := chi.NewRouter()

	// Example routes
	r.Route("/example", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Mount("/subexample", subexampleHandler.Routes())
	})

	// IAM routes
	r.Route("/iam", func(r chi.Router) {
		// Auth routes (public, no auth required)
		r.Mount("/auth", authHandler.Routes())

		// Organization routes
		r.Route("/organization", func(r chi.Router) {
			// Public route - no auth required
			r.Get("/registered", orgHandler.GetRegisteredOrganizations)

			// Protected routes - require auth
			r.Group(func(r chi.Router) {
				r.Use(authMiddleware)
				r.Mount("/", orgHandler.Routes())
			})
		})
	})

	// Project routes (protected)
	r.Route("/project", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Mount("/", projectHandler.Routes())
	})

	// Monitoring routes (protected)
	r.Route("/monitoring", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Mount("/", monitoringHandler.Routes())
	})

	// Tracker routes (protected)
	r.Route("/tracker", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Mount("/", trackerHandler.Routes())
	})

	// Finance routes (protected)
	r.Route("/finance", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Mount("/", financeHandler.Routes())
	})

	return r
}
