package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/middleware"
	"gitlab.chabokan.net/niflheim/wtf-backend/internal/service"
)

type RouterConfig struct {
	AllowedOrigin     string
	RequestsPerMinute int
}

type Dependencies struct {
	Health  HealthChecker
	Auth    *service.AuthService
	Regions *service.RegionService
	Request *service.RequestService
	Metrics *service.MetricsService
}

func NewRouter(deps Dependencies, cfg RouterConfig) http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS(cfg.AllowedOrigin))

	healthHandler := NewHealthHandler(deps.Health)
	authHandler := NewAuthHandler(deps.Auth)
	regionHandler := NewRegionHandler(deps.Regions)
	requestHandler := NewRequestHandler(deps.Request)
	metricsHandler := NewMetricsHandler(deps.Metrics)
	rateLimiter := middleware.NewRateLimiter(cfg.RequestsPerMinute, cfg.RequestsPerMinute)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthHandler.Live)
		r.Get("/health/ready", healthHandler.Ready)
		r.Get("/regions", regionHandler.List)
		r.With(rateLimiter.Middleware).Post("/requests", requestHandler.Create)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(deps.Auth))
			r.Get("/dispatcher/requests", requestHandler.List)
			r.Get("/dispatcher/requests/{id}", requestHandler.Detail)
			r.Patch("/dispatcher/requests/{id}/status", requestHandler.UpdateStatus)
			r.Get("/dispatcher/metrics/summary", metricsHandler.Summary)
			r.Get("/dispatcher/metrics/by-region", metricsHandler.ByRegion)
			r.Get("/dispatcher/metrics/by-need-type", metricsHandler.ByNeedType)
		})
	})

	return r
}
