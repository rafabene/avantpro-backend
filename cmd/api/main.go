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
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/rafabene/avantpro-backend/internal/handlers/middleware"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/config"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/i18n"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/logging"
	"github.com/rafabene/avantpro-backend/internal/infrastructure/persistence/postgres"

	_ "github.com/rafabene/avantpro-backend/docs" // Import generated docs
)

// @title AvantPro Backend API
// @version 1.0
// @description REST API para gerenciamento de assinaturas seguindo princípios de Clean Architecture
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@avantpro.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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
	_ = postgres.NewUnitOfWork(db) // TODO: Usar quando implementar specs

	// Inicializar services
	// TODO: Adicionar services conforme specs

	// Inicializar handlers
	// TODO: Adicionar handlers conforme specs

	// Setup Gin
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.Env == "development" {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(gin.Logger())   // Middleware de logging
	router.Use(gin.Recovery()) // Middleware de recovery para panics

	// Swagger documentation - apenas em development
	if cfg.Env == "development" {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		logger.Info("swagger UI enabled", "url", "http://"+cfg.Server.Host+":"+cfg.Server.Port+"/swagger/index.html")
	}

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
	_ = router.Group("/api/v1") // TODO: Adicionar rotas conforme specs

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
