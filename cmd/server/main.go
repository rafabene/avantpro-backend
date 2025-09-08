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
// @description APIs do Avant Pro
// @termsOfService http://swagger.io/terms/

// @contact.name Suporte da API
// @contact.url http://www.swagger.io/support
// @contact.email rafabene@gmail.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Digite "Bearer" seguido de um espaço e o token JWT.

func main() {
	// Carregar variáveis de ambiente em desenvolvimento
	if err := godotenv.Load(); err != nil {
		log.Println("Nenhum arquivo .env encontrado, usando variáveis de ambiente")
	}

	// Carregar configuração
	cfg := config.LoadConfig()

	// Definir modo do Gin
	gin.SetMode(cfg.Server.GinMode)

	// Conectar ao banco de dados
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("Falha ao conectar ao banco de dados: %v", err)
	}

	// Inicializar repositórios
	userRepo := repositories.NewUserRepository(db)
	orgRepo := repositories.NewOrganizationRepository(db)
	notificationRepo := repositories.NewNotificationRepository(db)
	notificationPrefRepo := repositories.NewNotificationPreferenceRepository(db)
	passwordResetRepo := repositories.NewPasswordResetRepository(db)

	// Inicializar serviços
	emailService := services.NewEmailService()
	notificationService := services.NewNotificationService(notificationRepo, orgRepo, userRepo, nil)
	notificationPrefService := services.NewNotificationPreferenceService(notificationPrefRepo, notificationRepo, orgRepo, notificationService)
	orgService := services.NewOrganizationService(orgRepo, userRepo, emailService, notificationService, notificationPrefRepo)

	// Inicializar controladores
	orgController := controllers.NewOrganizationController(orgService)
	authService := services.NewAuthService(userRepo, passwordResetRepo, cfg.JWT.Secret, &cfg.Auth, &cfg.JWT)
	authController := controllers.NewAuthController(authService)
	notificationController := controllers.NewNotificationController(notificationService)
	notificationPrefController := controllers.NewNotificationPreferenceController(notificationPrefService)

	// Inicializar e iniciar worker para tarefas de manutenção periódica
	maintenanceWorker := worker.NewWorker(orgRepo)
	maintenanceWorker.Start()

	// Configurar desligamento gracioso para o worker
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Sinal de desligamento recebido")
		maintenanceWorker.Stop()
		os.Exit(0)
	}()

	// Configurar roteador
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Middleware CORS
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = cfg.CORS.AllowOrigins
	corsConfig.AllowMethods = cfg.CORS.AllowMethods
	corsConfig.AllowHeaders = cfg.CORS.AllowHeaders
	router.Use(cors.New(corsConfig))

	// Configurar proxies confiáveis
	if err := router.SetTrustedProxies(cfg.Server.TrustedProxies); err != nil {
		log.Printf("Aviso: Falha ao definir proxies confiáveis: %v", err)
	}

	// Endpoint de verificação de saúde
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "avantpro-backend",
			"version": "1.0.0",
		})
	})

	// Documentação Swagger (apenas em desenvolvimento)
	if cfg.IsDevelopment() {
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	// Rotas da API
	v1 := router.Group("/api/v1")
	{
		// Rotas de autenticação
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authController.Login)
			auth.POST("/register", authController.Register)
			auth.POST("/password-reset", authController.RequestPasswordReset)
			auth.POST("/password-reset/confirm", authController.ResetPassword)

			// Rotas de autenticação protegidas
			authProtected := auth.Group("")
			authProtected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
			{
				authProtected.PUT("/last-selected-organization", authController.UpdateLastSelectedOrganization)
			}
		}

		// Rotas públicas (sem autenticação necessária)
		public := v1.Group("")
		{
			// Validação de convite (público)
			public.GET("/invites/token/:token/validate", orgController.ValidateInvite)
		}

		// Rotas protegidas
		protected := v1.Group("")
		protected.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
		{
			// Rotas de organização que não requerem validação de associação
			organizations := protected.Group("/organizations")
			{
				// CRUD de organização
				organizations.POST("", orgController.CreateOrganization)
				organizations.GET("/my", orgController.GetUserOrganizations)
			}

			// Rotas de organização que requerem validação de associação
			organizationsWithMembership := protected.Group("/organizations")
			organizationsWithMembership.Use(middleware.OrganizationMembershipMiddleware(orgService))
			{
				organizationsWithMembership.GET("", orgController.GetOrganization)       // Usa header Organization-ID
				organizationsWithMembership.PUT("", orgController.UpdateOrganization)    // Usa header Organization-ID
				organizationsWithMembership.DELETE("", orgController.DeleteOrganization) // Usa header Organization-ID
			}

			// Rotas centralizadas de Membros (requerem validação de associação)
			members := protected.Group("/members")
			members.Use(middleware.OrganizationMembershipMiddleware(orgService))
			{
				members.GET("", orgController.GetOrganizationMembers)   // Usa header Organization-ID
				members.PUT("/:userId", orgController.UpdateMemberRole) // Usa header Organization-ID
				members.DELETE("/:userId", orgController.RemoveMember)  // Usa header Organization-ID
			}

			// Rotas centralizadas de Associações (não requerem Organization-ID)
			memberships := protected.Group("/memberships")
			{
				memberships.GET("", orgController.GetUserMemberships)
			}

			// Rotas centralizadas de Convites
			invites := protected.Group("/invites")
			{
				// Rotas que requerem validação de associação
				invitesWithMembership := invites.Group("")
				invitesWithMembership.Use(middleware.OrganizationMembershipMiddleware(orgService))
				{
					invitesWithMembership.POST("", orgController.InviteUser)            // Usa header Organization-ID
					invitesWithMembership.GET("", orgController.GetOrganizationInvites) // Usa header Organization-ID
				}

				// Rotas que não requerem header Organization-ID
				invites.POST("/:inviteId/resend", orgController.ResendInvite)
				invites.DELETE("/:inviteId", orgController.RevokeInvite)

				// Operações baseadas em token
				invites.POST("/token/:token/accept", orgController.AcceptInvite)
			}

			// Rotas centralizadas de Notificações (requerem validação de associação)
			notifications := protected.Group("/notifications")
			notifications.Use(middleware.OrganizationMembershipMiddleware(orgService))
			{
				notifications.GET("", notificationController.GetUserNotifications)           // Usa header Organization-ID
				notifications.GET("/unread", notificationController.GetUnreadNotifications)  // Usa header Organization-ID
				notifications.GET("/unread-count", notificationController.GetUnreadCount)    // Usa header Organization-ID
				notifications.PUT("/mark-all-read", notificationController.MarkAllAsRead)    // Usa header Organization-ID
				notifications.PUT("/:notifId/read", notificationController.MarkAsRead)       // Usa header Organization-ID
				notifications.DELETE("", notificationController.DeleteAllNotifications)      // Usa header Organization-ID
				notifications.DELETE("/:notifId", notificationController.DeleteNotification) // Usa header Organization-ID
			}

			// Rotas centralizadas de Preferências de Notificação (requerem validação de associação)
			notificationPrefs := protected.Group("/notification-preferences")
			notificationPrefs.Use(middleware.OrganizationMembershipMiddleware(orgService))
			{
				notificationPrefs.GET("", notificationPrefController.GetOrganizationPreferences)     // Usa header Organization-ID
				notificationPrefs.PUT("", notificationPrefController.UpdateOrganizationPreferences)  // Usa header Organization-ID
				notificationPrefs.PUT("/:event", notificationPrefController.UpdateSinglePreference)  // Usa header Organization-ID
				notificationPrefs.POST("/reset", notificationPrefController.ResetToDefaults)         // Usa header Organization-ID
				notificationPrefs.GET("/events", notificationPrefController.GetAvailableEvents)      // Header Organization-ID não requerido
				notificationPrefs.POST("/test", notificationPrefController.GenerateTestNotification) // Usa header Organization-ID
			}
		}
	}

	// Iniciar servidor
	log.Printf("Iniciando servidor na porta %s em modo %s", cfg.Server.Port, cfg.Environment)
	if cfg.IsDevelopment() {
		log.Printf("Interface Swagger disponível em: http://localhost:%s/swagger/index.html", cfg.Server.Port)
	}

	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("Falha ao iniciar servidor: %v", err)
	}
}
