package handler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/christopherdang/vibecloud/backend/auth"
	"github.com/christopherdang/vibecloud/backend/config"
	authhandler "github.com/christopherdang/vibecloud/backend/handler"
	"github.com/christopherdang/vibecloud/backend/repository"
	"github.com/christopherdang/vibecloud/backend/response"
	"github.com/christopherdang/vibecloud/backend/service"
)

var (
	router http.Handler
	once   sync.Once
	pool   *pgxpool.Pool
)

func setup() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	pool, err = pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.Timeout(30 * time.Second))

	// CORS
	allowedOrigins := []string{"http://localhost:3000"}
	if prodURL := os.Getenv("VERCEL_PROJECT_PRODUCTION_URL"); prodURL != "" {
		allowedOrigins = append(allowedOrigins, "https://"+prodURL)
	}
	allowedOrigins = append(allowedOrigins, "https://www.vibecloudai.com", "https://vibecloudai.com")
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Rate limit
	r.Use(httprate.LimitByIP(100, time.Minute))

	// Health check
	r.Get("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			response.Error(w, 503, "DB_UNHEALTHY", fmt.Sprintf("database ping failed: %v", err))
			return
		}
		response.JSON(w, 200, map[string]string{"status": "ok"})
	})

	// Dependency wiring
	userRepo := repository.NewUserRepository(pool)
	authService := service.NewAuthService(userRepo, cfg.JWTSigningSecret)
	authH := authhandler.NewAuthHandler(authService)
	authMiddleware := auth.NewMiddleware(cfg.JWTSigningSecret)

	deviceCodeRepo := repository.NewDeviceCodeRepository(pool)
	deviceCodeService := service.NewDeviceCodeService(deviceCodeRepo, authService)
	deviceCodeHandler := authhandler.NewDeviceCodeHandler(deviceCodeService)

	deployLogRepo := repository.NewDeployLogRepository(pool)
	deployService := service.NewDeployService(deployLogRepo)
	deployHandler := authhandler.NewDeployHandler(deployService)

	// Public auth routes (no JWT required)
	r.Route("/api/v1/auth", func(r chi.Router) {
		r.Use(httprate.LimitByIP(10, time.Minute))
		r.Post("/register", authH.Register)
		r.Post("/login", authH.Login)
		r.Post("/refresh", authH.Refresh)
		r.Post("/device-code/exchange", deviceCodeHandler.Exchange)

		// Protected device code generation (nested inside auth group)
		r.Group(func(r chi.Router) {
			r.Use(authMiddleware.Authenticate)
			r.Post("/device-code", deviceCodeHandler.Generate)
		})
	})

	// Protected routes (JWT required)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware.Authenticate)
		r.Get("/me", authH.Me)
		r.Patch("/tier", authH.UpdateTier)
		r.Post("/deploys/check", deployHandler.CheckLimit)
		r.Post("/deploys/log", deployHandler.LogDeploy)
	})

	router = r
}

func Handler(w http.ResponseWriter, r *http.Request) {
	once.Do(setup)
	router.ServeHTTP(w, r)
}
