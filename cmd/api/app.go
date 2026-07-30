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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"task-api/internal/config"
	"task-api/internal/handler"
	"task-api/internal/repository"
	"task-api/internal/service"
)

func Run() error {
	_ = godotenv.Load()

	cfg := config.Load()

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	log.Println("Connected to database")

	userRepo := repository.NewUserRepository(pool)
	taskRepo := repository.NewTaskRepository(pool)

	userService := service.NewUserService(userRepo, cfg.JWT.SecretKey, cfg.JWT.AccessTokenTTL, cfg.JWT.RefreshTokenTTL)
	taskService := service.NewTaskService(taskRepo)
	authService := service.NewAuthService(userService)

	userHandler := handler.NewUserHandler(userService)
	taskHandler := handler.NewTaskHandler(taskService)
	authHandler := handler.NewAuthHandler(authService)

	r := chi.NewRouter()

	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer, middleware.Timeout(60*time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", authHandler.Register)
			r.Post("/login", authHandler.Login)
			r.Post("/refresh", authHandler.Refresh)
		})

		r.Group(func(r chi.Router) {
			r.Use(authHandler.AuthMiddleware)

			r.Route("/users", func(r chi.Router) {
				r.Get("/me", userHandler.GetMe)
				r.Put("/me", userHandler.UpdateMe)
				r.Delete("/me", userHandler.DeleteMe)
			})

			r.Route("/tasks", func(r chi.Router) {
				r.Post("/", taskHandler.Create)
				r.Get("/", taskHandler.List)
				r.Get("/{id}", taskHandler.GetByID)
				r.Put("/{id}", taskHandler.Update)
				r.Delete("/{id}", taskHandler.Delete)
				r.Patch("/{id}/status", taskHandler.UpdateStatus)
			})
		})
	})

	srv := &http.Server{
		Addr:         cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	go func() {
		log.Printf("Server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server exited properly")
	return nil
}