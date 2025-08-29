package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/rafabene/avantpro-backend/docs"
	"github.com/rafabene/avantpro-backend/internal/config"
	"github.com/rafabene/avantpro-backend/internal/controllers"
	"github.com/rafabene/avantpro-backend/internal/database"
	"github.com/rafabene/avantpro-backend/internal/repositories"
	"github.com/rafabene/avantpro-backend/internal/services"
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

// @securityDefinitions.basic BasicAuth

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

	// Initialize services
	userService := services.NewUserService(userRepo)

	// Initialize controllers
	userController := controllers.NewUserController(userService)
	authService := services.NewAuthService(userRepo, cfg.JWT.Secret)
	authController := controllers.NewAuthController(authService)

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

		// User routes
		users := v1.Group("/users")
		{
			users.POST("", userController.CreateUser)
			users.GET("", userController.ListUsers)
			users.GET("/:id", userController.GetUser)
			users.GET("/username/:username", userController.GetUserByUsername)
			users.PUT("/:id", userController.UpdateUser)
			users.DELETE("/:id", userController.DeleteUser)
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
