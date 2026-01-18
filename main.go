package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kubeagents/kubeagents/auth"
	"github.com/kubeagents/kubeagents/config"
	"github.com/kubeagents/kubeagents/email"
	"github.com/kubeagents/kubeagents/handlers"
	authMiddleware "github.com/kubeagents/kubeagents/middleware"
	"github.com/kubeagents/kubeagents/notifier"
	"github.com/kubeagents/kubeagents/store"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize store (PostgreSQL if configured, otherwise memory)
	var st store.Store
	var pgStore *store.PostgresStore
	var closeDB func()

	if cfg.Database.DBName != "" {
		// Use PostgreSQL
		connString := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			cfg.Database.Host,
			cfg.Database.Port,
			cfg.Database.User,
			cfg.Database.Password,
			cfg.Database.DBName,
			cfg.Database.SSLMode,
		)

		var err error
		pgStore, err = store.NewPostgresStore(context.Background(), connString)
		if err != nil {
			log.Fatalf("Failed to connect to database: %v", err)
		}

		// Run database migrations
		conn, err := pgStore.Pool().Acquire(context.Background())
		if err != nil {
			log.Fatalf("Failed to acquire database connection: %v", err)
		}
		defer conn.Release()

		if err := store.RunMigrations(context.Background(), conn.Conn()); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}

		st = pgStore
		closeDB = func() { pgStore.Close() }
		log.Println("Using PostgreSQL storage")
	} else {
		// Use memory storage
		st = store.NewMemoryStore()
		log.Println("Using in-memory storage")
	}

	// Initialize notification manager
	notificationManager := notifier.NewNotificationManager(
		cfg.NotificationWebhookURL,
		cfg.NotificationTimeout,
	)

	// Initialize JWT service
	jwtService := auth.NewJWTService(cfg.JWT.Secret, cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)

	// Initialize email service (optional - will be nil if SMTP not configured)
	var emailService *email.EmailService
	if cfg.SMTP.Host != "" && cfg.SMTP.User != "" {
		emailService = email.NewEmailService(email.EmailConfig{
			SMTPHost:   cfg.SMTP.Host,
			SMTPPort:   cfg.SMTP.Port,
			SMTPUser:   cfg.SMTP.User,
			SMTPPass:   cfg.SMTP.Password,
			FromEmail:  cfg.SMTP.FromEmail,
			AppBaseURL: cfg.AppBaseURL,
		})
		log.Println("Email service initialized")
	} else {
		log.Println("Warning: SMTP not configured, email verification disabled")
	}

	// Initialize auth middleware
	authMiddleware := authMiddleware.NewAuthMiddleware(jwtService)

	// Initialize handlers
	healthHandler := handlers.HealthCheck
	webhookHandler := handlers.NewWebhookHandlerWithNotifier(st, notificationManager)
	agentHandler := handlers.NewAgentHandler(st)
	authHandler := handlers.NewAuthHandler(st, jwtService, emailService)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Public routes
	r.Get("/health", healthHandler)

	// Auth routes (public)
	r.Route("/api/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Get("/verify", authHandler.VerifyEmail)
		r.Post("/login", authHandler.Login)
		r.Post("/refresh", authHandler.Refresh)
		r.Post("/resend-verify", authHandler.ResendVerify)
	})

	// Protected API routes
	r.Route("/api", func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Post("/auth/logout", authHandler.Logout)
		r.Get("/auth/me", authHandler.Me)

		r.Route("/agents", func(r chi.Router) {
			r.Get("/", agentHandler.ListAgents)
			r.Get("/{agent_id}", agentHandler.GetAgent)
			r.Get("/{agent_id}/sessions", agentHandler.ListSessions)
			r.Get("/{agent_id}/sessions/{session_topic}", agentHandler.GetSession)
			r.Get("/{agent_id}/status", agentHandler.GetAgentStatus)
		})
	})

	// Webhook requires authentication (Agent binds to user)
	r.Route("/webhook", func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Post("/status", webhookHandler.ServeHTTP)
	})

	// Start background goroutine for session expiration check
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				st.CheckExpiredSessions()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	cancel()

	// Shutdown notification manager first (wait for pending notifications)
	notifyShutdownCtx, notifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer notifyCancel()

	if err := notificationManager.Shutdown(notifyShutdownCtx); err != nil {
		log.Printf("Warning: Notification manager shutdown error: %v", err)
	}

	// Shutdown HTTP server
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	if closeDB != nil {
		closeDB()
	}

	log.Println("Server exited")
}
