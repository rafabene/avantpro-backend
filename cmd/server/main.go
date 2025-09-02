package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/rafabene/avantpro-backend/docs"
	"github.com/rafabene/avantpro-backend/internal/config"
	"github.com/rafabene/avantpro-backend/internal/controllers"
	"github.com/rafabene/avantpro-backend/internal/database"
	"github.com/rafabene/avantpro-backend/internal/middleware"
	"github.com/rafabene/avantpro-backend/internal/repositories"
	"github.com/rafabene/avantpro-backend/internal/services"
	"github.com/rafabene/avantpro-backend/internal/worker"
)

// @title AvantPro Backend API
// @version 1.0
// @description User Management API with Profile support
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load environment variables in development
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Load configuration
	cfg := config.LoadConfig()

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Connect to database
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize repositories
	userRepo := repositories.NewUserRepository(db)
	orgRepo := repositories.NewOrganizationRepository(db)

	// Initialize services
	emailService := services.NewEmailService()
	orgService := services.NewOrganizationService(orgRepo, userRepo, emailService)

	// Initialize controllers
	orgController := controllers.NewOrganizationController(orgService)
	authService := services.NewAuthService(userRepo, cfg.JWT.Secret)
	authController := controllers.NewAuthController(authService)

	// Initialize and start worker for periodic maintenance tasks
	maintenanceWorker := worker.NewWorker(orgRepo)
	maintenanceWorker.Start()

	// Setup graceful shutdown for worker
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		maintenanceWorker.Stop()
		os.Exit(0)
	}()

	// Setup router
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:4200"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	router.Use(cors.New(config))

	// Configure trusted proxies
	if err := router.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		log.Printf("Warning: Failed to set trusted proxies: %v", err)
	}

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "avantpro-backend",
			"version": "1.0.0",
		})
	})

	// Swagger documentation (only in development)
	if cfg.IsDevelopment() {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// API routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/register", authController.Register)
			auth.POST("/password-reset", authController.RequestPasswordReset)
			auth.POST("/password-reset/confirm", authController.ResetPassword)
		}


		// Organization routes (protected)
		organizations := v1.Group("/organizations")
		organizations.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// Organization CRUD
			organizations.POST("", orgController.CreateOrganization)
			organizations.GET("/my", orgController.GetUserOrganizations)
			organizations.GET("/memberships", orgController.GetUserMemberships)
			organizations.GET("/:id", orgController.GetOrganization)
			organizations.PUT("/:id", orgController.UpdateOrganization)
			organizations.DELETE("/:id", orgController.DeleteOrganization)

			// Organization Members
			organizations.GET("/:id/members", orgController.GetOrganizationMembers)
			organizations.PUT("/:id/members/:userId", orgController.UpdateMemberRole)
			organizations.DELETE("/:id/members/:userId", orgController.RemoveMember)

			// Organization Invites
			organizations.POST("/:id/invites", orgController.InviteUser)
			organizations.GET("/:id/invites", orgController.GetOrganizationInvites)

			// Invite Management (by ID)
			organizations.POST("/invites/id/:inviteId/resend", orgController.ResendInvite)
			organizations.DELETE("/invites/id/:inviteId", orgController.RevokeInvite)

			// Invite Acceptance (by token)
			organizations.POST("/invites/token/:token/accept", orgController.AcceptInvite)
		}
	}

	// Start server
	log.Printf("Starting server on port %s in %s mode", cfg.Server.Port, cfg.Environment)
	if cfg.IsDevelopment() {
		log.Printf("Swagger UI available at: http://localhost:%s/swagger/index.html", cfg.Server.Port)
	}

	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
