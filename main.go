package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
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

const jwtSecretConfigKey = "jwt_secret"

// initJWTSecret initializes the JWT secret from config or storage
// If config has a secret, use it and save to storage
// If config doesn't have a secret, try to load from storage, or generate a new one
func initJWTSecret(st store.Store, configSecret string) (string, error) {
	// If config has a secret set, use it and save to storage
	if configSecret != "" {
		if err := st.SetConfig(jwtSecretConfigKey, configSecret); err != nil {
			return "", fmt.Errorf("failed to save JWT secret to storage: %w", err)
		}
		log.Println("Using JWT secret from configuration")
		return configSecret, nil
	}

	// Try to load from storage
	secret, err := st.GetConfig(jwtSecretConfigKey)
	if err == nil && secret != "" {
		log.Println("Using JWT secret from storage")
		return secret, nil
	}

	// Generate a new secret
	secret, err = generateRandomSecret(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate JWT secret: %w", err)
	}

	// Save to storage
	if err := st.SetConfig(jwtSecretConfigKey, secret); err != nil {
		return "", fmt.Errorf("failed to save generated JWT secret: %w", err)
	}

	log.Println("Generated and saved new JWT secret")
	return secret, nil
}

// generateRandomSecret generates a cryptographically secure random string
func generateRandomSecret(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

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

	// Initialize JWT secret from config or storage
	jwtSecret, err := initJWTSecret(st, cfg.JWT.Secret)
	if err != nil {
		log.Fatalf("Failed to initialize JWT secret: %v", err)
	}

	// Initialize JWT service
	jwtService := auth.NewJWTService(jwtSecret, cfg.JWT.AccessTokenExpiry, cfg.JWT.RefreshTokenExpiry)

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

	// Initialize auth middleware (with store for API key support)
	authMiddleware := authMiddleware.NewAuthMiddlewareWithStore(jwtService, st)

	// Initialize handlers
	healthHandler := handlers.HealthCheck
	webhookHandler := handlers.NewWebhookHandlerWithNotifier(st, notificationManager)
	agentHandler := handlers.NewAgentHandler(st)
	authHandler := handlers.NewAuthHandler(st, jwtService, emailService)
	apiKeyHandler := handlers.NewAPIKeyHandler(st)

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

	// Protected API routes (JWT only)
	r.Route("/api", func(r chi.Router) {
		r.Use(authMiddleware.RequireAuth)
		r.Post("/auth/logout", authHandler.Logout)
		r.Get("/auth/me", authHandler.Me)

		// API Key management
		r.Route("/apikeys", func(r chi.Router) {
			r.Get("/", apiKeyHandler.List)
			r.Post("/", apiKeyHandler.Create)
			r.Delete("/{id}", apiKeyHandler.Revoke)
		})

		r.Route("/agents", func(r chi.Router) {
			r.Get("/", agentHandler.ListAgents)
			r.Get("/{agent_id}", agentHandler.GetAgent)
			r.Get("/{agent_id}/sessions", agentHandler.ListSessions)
			r.Get("/{agent_id}/sessions/{session_topic}", agentHandler.GetSession)
			r.Get("/{agent_id}/status", agentHandler.GetAgentStatus)
		})
	})

	// Webhook requires authentication (supports both JWT and API Key)
	r.Route("/webhook", func(r chi.Router) {
		r.Use(authMiddleware.RequireAuthOrAPIKey)
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

	// Shutdown HTTP server first (stop accepting new connections)
	log.Println("Shutting down HTTP server...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)

	// Start shutdown in goroutine
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
		} else {
			log.Println("HTTP server shutdown complete")
		}
		shutdownCancel() // Cancel context after shutdown completes
	}()

	// Wait for shutdown or force exit after timeout
	select {
	case <-shutdownDone:
		// Normal shutdown completed
	case <-time.After(5 * time.Second):
		log.Println("Shutdown timeout, forcing exit...")
		os.Exit(0)
	}

	// Shutdown notification manager (wait for pending notifications)
	log.Println("Shutting down notification manager...")
	notifyShutdownCtx, notifyCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer notifyCancel()

	if err := notificationManager.Shutdown(notifyShutdownCtx); err != nil {
		log.Printf("Warning: Notification manager shutdown error: %v", err)
	}

	log.Println("Notification manager shutdown complete")

	// Close database connection with timeout
	if closeDB != nil {
		log.Println("Closing database connection...")
		dbCloseDone := make(chan struct{})
		go func() {
			closeDB()
			close(dbCloseDone)
		}()

		select {
		case <-dbCloseDone:
			log.Println("Database connection closed")
		case <-time.After(3 * time.Second):
			log.Println("Database connection close timeout, continuing...")
		}
	}

	log.Println("Server exited")
}
