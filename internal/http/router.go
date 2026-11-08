// internal/http/router.go
package http

import (
	"investment_monitoring/config"

	"investment_monitoring/internal/http/internal_middleware"

	// Example module

	// IAM module
	auth_controller "investment_monitoring/internal/module/iam/auth/controller"
	auth_service "investment_monitoring/internal/module/iam/auth/service"
	organization_controller "investment_monitoring/internal/module/iam/organization/controller"
	organization_service "investment_monitoring/internal/module/iam/organization/service"
	profile_controller "investment_monitoring/internal/module/iam/profile/controller"
	profile_service "investment_monitoring/internal/module/iam/profile/service"

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

	// Realtime module
	websocket_controller "investment_monitoring/internal/module/realtime/websocket/controller"
	websocket_service "investment_monitoring/internal/module/realtime/websocket/service"

	// Support services
	dashboard_controller "investment_monitoring/internal/module/support/dashboard/controller"
	dashboard_service "investment_monitoring/internal/module/support/dashboard/service"
	mail_service "investment_monitoring/internal/module/support/mail/service"
	token_service "investment_monitoring/internal/module/support/token/service"
	uploader_controller "investment_monitoring/internal/module/support/uploader/controller"
	uploader_service "investment_monitoring/internal/module/support/uploader/service"

	repository "investment_monitoring/internal/repository/entity"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func NewRouter(db *repository.SQLStore, rdb *redis.Client, cfg *config.Config) (chi.Router, *websocket_service.Service) {
	// 1. =========== INITIAL SERVICE / MIDDLEWARE ===========
	// Support services
	tokenSvc := token_service.NewService(cfg)
	mailSvc := mail_service.NewService(cfg)

	// Auth middleware
	authMiddleware := internal_middleware.AuthMiddleware(*tokenSvc)
	authRateLimiter := internal_middleware.NewAuthRateLimiter(rdb, cfg.REDIS_PREFIX)

	// Example service

	// IAM services
	authSvc := auth_service.NewService(db, tokenSvc, rdb, cfg)
	orgSvc := organization_service.NewService(db, mailSvc, cfg)
	profileSvc := profile_service.NewService(db, cfg)

	// Project services
	projectSvc := project_service.NewService(db, rdb, cfg)

	// Monitoring services
	monitoringSvc := monitoring_service.NewService(db, rdb, cfg)

	// Tracker services
	trackerSvc := tracker_service.NewService(db, rdb, cfg)

	// Finance services
	financeSvc := finance_service.NewService(db, rdb, cfg)

	// Realtime services
	websocketSvc := websocket_service.NewService(cfg, tokenSvc, db, rdb)

	// Uploader service
	uploaderSvc, _ := uploader_service.NewService(db, cfg)

	// Dashboard service
	dashboardSvc := dashboard_service.NewService(db, rdb, cfg)

	// 2. =========== INITIAL HANDLER ===========
	authHandler := auth_controller.NewHandler(authSvc)
	orgHandler := organization_controller.NewHandler(orgSvc, db)
	profileHandler := profile_controller.NewHandler(profileSvc)
	projectHandler := project_controller.NewHandler(projectSvc, db)
	monitoringHandler := monitoring_controller.NewHandler(monitoringSvc, db)
	trackerHandler := tracker_controller.NewHandler(trackerSvc, db)
	financeHandler := finance_controller.NewHandler(financeSvc, db)
	websocketHandler := websocket_controller.NewHandler(websocketSvc)
	uploaderHandler := uploader_controller.NewHandler(uploaderSvc)
	dashboardHandler := dashboard_controller.NewHandler(dashboardSvc, db)

	// 3. =========== MANAGE ROUTING ===========
	r := chi.NewRouter()

	// IAM routes
	r.Route("/iam", func(r chi.Router) {
		// Auth routes (public, no auth required)
		r.Route("/auth", func(r chi.Router) {
			r.With(authRateLimiter.Limit("check-email")).Post("/check-email", authHandler.CheckEmail)
			r.With(authRateLimiter.Limit("login")).Post("/login", authHandler.Login)
			r.Post("/set-password", authHandler.SetPassword)
			r.With(authRateLimiter.Limit("refresh-token")).Post("/refresh-token", authHandler.RefreshToken)
			r.Post("/logout", authHandler.Logout)
			r.Post("/revoke-all", authHandler.RevokeAll)
		})

		// Profile routes (protected)
		r.Route("/profile", func(r chi.Router) {
			r.Use(authMiddleware)
			r.Mount("/", profileHandler.Routes())
		})

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

	// Support routes (protected)
	r.Route("/support", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Mount("/upload", uploaderHandler.Routes())
		r.Mount("/dashboard", dashboardHandler.Routes())
	})

	// Realtime routes (handshake auth is handled by websocket service)
	r.Route("/realtime", func(r chi.Router) {
		r.Mount("/ws", websocketHandler.Routes())
	})

	return r, websocketSvc
}
