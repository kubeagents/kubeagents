package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/kubeagents/kubeagents/config"
	"github.com/kubeagents/kubeagents/handlers"
	"github.com/kubeagents/kubeagents/notifier"
	"github.com/kubeagents/kubeagents/store"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize store
	st := store.NewStore()

	// Initialize notification manager
	notificationManager := notifier.NewNotificationManager(
		cfg.NotificationWebhookURL,
		cfg.NotificationTimeout,
	)

	// Initialize handlers
	healthHandler := handlers.HealthCheck
	webhookHandler := handlers.NewWebhookHandlerWithNotifier(st, notificationManager)
	agentHandler := handlers.NewAgentHandler(st)
	
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
	
	// Routes
	r.Get("/health", healthHandler)
	r.Post("/webhook/status", webhookHandler.ServeHTTP)
	
	// API routes
	r.Route("/api/agents", func(r chi.Router) {
		r.Get("/", agentHandler.ListAgents)
		r.Get("/{agent_id}", agentHandler.GetAgent)
		r.Get("/{agent_id}/sessions", agentHandler.ListSessions)
		r.Get("/{agent_id}/sessions/{session_topic}", agentHandler.GetSession)
		r.Get("/{agent_id}/status", agentHandler.GetAgentStatus)
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

	log.Println("Server exited")
}
