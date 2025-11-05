package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	httphandlers "github.com/rafabene/avantpro-backend/internal/handlers/http"
	"github.com/rafabene/avantpro-backend/internal/handlers/middleware"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/config"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/logging"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/persistence/postgres"
	"github.com/rafabene/avantpro-backend/internal/services"
)

func main() {
	// Carregar configurações
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	// Inicializar logger
	logger := logging.NewSlogLogger(cfg.Logging.Level)
	logger.Info("starting avantpro backend",
		"env", cfg.Env,
		"version", "dev",
	)

	// Conectar ao banco de dados
	db, err := postgres.NewDatabaseConnection(&cfg.Database, logger)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		log.Fatal(err)
	}

	// Inicializar i18n
	i18nService, err := i18n.NewService("./internal/infrastructure/i18n/locales", "en")
	if err != nil {
		logger.Error("failed to initialize i18n", "error", err)
		log.Fatal(err)
	}
	logger.Info("i18n initialized",
		"default_language", i18nService.GetDefaultLanguage(),
		"supported_languages", i18nService.GetSupportedLanguages(),
	)

	// Inicializar repositories
	userRepo := postgres.NewUserRepository(db)
	uow := postgres.NewUnitOfWork(db)

	// Inicializar services
	userService := services.NewUserService(userRepo, uow, logger)

	// Inicializar handlers
	userHandler := httphandlers.NewUserHandler(userService)

	// Setup Gin
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	// Middleware global para adicionar base URL ao contexto
	router.Use(func(c *gin.Context) {
		c.Set("base_url", cfg.Server.BaseURL)
		c.Next()
	})

	// Middleware i18n
	i18nMiddleware := middleware.NewI18nMiddleware(i18nService)
	router.Use(i18nMiddleware.DetectLanguage())

	// Middleware CORS
	router.Use(middleware.CORS(cfg.CORS.AllowedOrigins))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"env":    cfg.Env,
		})
	})

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Users
		users := v1.Group("/users")
		{
			users.POST("", userHandler.CreateUser)
			users.GET("/:id", userHandler.GetUser)
			users.GET("", userHandler.ListUsers)
		}
	}

	// HTTP Server
	srv := &http.Server{
		Addr:              cfg.Server.Host + ":" + cfg.Server.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Info("server starting",
			"host", cfg.Server.Host,
			"port", cfg.Server.Port,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			log.Fatal(err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}

	logger.Info("server exited")
}
