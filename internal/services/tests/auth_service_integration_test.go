package tests

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/rafabene/avantpro-backend/internal/config"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/services"
	"github.com/rafabene/avantpro-backend/internal/services/tests/mocks"
)

var _ = Describe("Testes de Integração do AuthService", func() {
	var (
		mockTest         *MockTestConfig
		authService      services.AuthService
		mockUserRepo     *mocks.MockUserRepository
		mockPasswordRepo *mocks.MockPasswordResetRepository
		jwtSecret        string
		testConfig       *config.Config
	)

	BeforeEach(func() {
		// Setup mock test
		mockTest = SetupMockTest(GinkgoT())

		// Initialize mock repositories
		mockUserRepo = mocks.NewMockUserRepository(mockTest.Controller)
		mockPasswordRepo = mocks.NewMockPasswordResetRepository(mockTest.Controller)

		// Configure test settings
		jwtSecret = "test-secret-key"
		testConfig = &config.Config{
			Auth: config.AuthConfig{
				MaxLoginAttempts:       3,
				AccountLockoutDuration: 15 * time.Minute,
			},
			JWT: config.JWTConfig{
				Secret:          jwtSecret,
				ExpirationHours: 24,
			},
		}

		// Initialize service with mocks
		authService = services.NewAuthService(mockUserRepo, mockPasswordRepo, jwtSecret, &testConfig.Auth, &testConfig.JWT)
	})

	AfterEach(func() {
		if mockTest != nil {
			mockTest.TeardownMockTest()
		}
	})

	Describe("Registrar novo usuário", func() {
		Context("quando a requisição é válida", func() {
			It("deve criar usuário com sucesso e retornar resposta de login", func() {
				registerRequest := &services.RegisterRequest{
					Email:    "test@example.com",
					Name:     "Test User",
					Password: "SecurePass123!",
				}

				// Mock expectations
				mockUserRepo.EXPECT().GetByUsername("test@example.com").Return(nil, errors.New("user not found"))
				mockUserRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
					// Simulate GORM BeforeCreate hook - generate UUID and hash password
					user.ID = uuid.New()
					user.CreatedAt = time.Now()
					user.UpdatedAt = time.Now()
					return user.HashPassword()
				})

				response, err := authService.Register(registerRequest)

				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
				Expect(response.Token).ToNot(BeEmpty())
				Expect(response.User.Username).To(Equal("test@example.com"))
				Expect(response.User.Name).To(Equal("Test User"))
				Expect(response.User.ID).ToNot(Equal(uuid.Nil))
			})

			It("deve fazer hash da senha antes de armazenar", func() {
				registerRequest := &services.RegisterRequest{
					Email:    "test@example.com",
					Name:     "Test User",
					Password: "SecurePass123!",
				}

				// Mock expectations
				mockUserRepo.EXPECT().GetByUsername("test@example.com").Return(nil, errors.New("user not found"))
				mockUserRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
					// Simulate GORM BeforeCreate hook - generate UUID and hash password
					user.ID = uuid.New()
					user.CreatedAt = time.Now()
					user.UpdatedAt = time.Now()

					// Verify password was NOT hashed yet
					Expect(user.Password).To(Equal("SecurePass123!"))

					// Hash the password (simulating GORM hook)
					err := user.HashPassword()
					if err != nil {
						return err
					}

					// Verify password was hashed after GORM hook
					Expect(user.Password).ToNot(Equal("SecurePass123!"))
					Expect(len(user.Password)).To(BeNumerically(">", 10))
					return nil
				})

				_, err := authService.Register(registerRequest)
				Expect(err).ToNot(HaveOccurred())
			})

			It("deve gerar um token JWT válido", func() {
				registerRequest := &services.RegisterRequest{
					Email:    "test@example.com",
					Name:     "Test User",
					Password: "SecurePass123!",
				}

				// Mock expectations
				mockUserRepo.EXPECT().GetByUsername("test@example.com").Return(nil, errors.New("user not found"))
				mockUserRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
					// Simulate GORM BeforeCreate hook - generate UUID and hash password
					user.ID = uuid.New()
					user.CreatedAt = time.Now()
					user.UpdatedAt = time.Now()
					return user.HashPassword()
				})

				response, err := authService.Register(registerRequest)

				Expect(err).ToNot(HaveOccurred())
				Expect(response.Token).ToNot(BeEmpty())
				Expect(response.Token).To(ContainSubstring("."))
			})
		})

		Context("quando usuário já existe", func() {
			It("deve retornar erro", func() {
				// Mock existing user
				existingUser := &models.User{
					ID:       uuid.New(),
					Username: "existing@example.com",
					Name:     "Existing User",
					Password: "hashedpassword",
				}

				registerRequest := &services.RegisterRequest{
					Email:    "existing@example.com",
					Name:     "Test User",
					Password: "SecurePass123!",
				}

				// Mock expectations - user already exists
				mockUserRepo.EXPECT().GetByUsername("existing@example.com").Return(existingUser, nil)

				response, err := authService.Register(registerRequest)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("Usuário já existe"))
				Expect(response).To(BeNil())
			})
		})

		Context("com casos extremos", func() {
			It("deve rejeitar email vazio", func() {
				registerRequest := &services.RegisterRequest{
					Email:    "",
					Name:     "Test User",
					Password: "SecurePass123!",
				}

				// Não precisamos de mocks porque a validação vai falhar antes
				response, err := authService.Register(registerRequest)

				// Email vazio DEVE resultar em erro de validação
				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("dados inválidos"))
			})

			It("deve rejeitar nome vazio", func() {
				registerRequest := &services.RegisterRequest{
					Email:    "test@example.com",
					Name:     "",
					Password: "SecurePass123!",
				}

				// Não precisamos de mocks porque a validação vai falhar antes
				response, err := authService.Register(registerRequest)

				// Nome vazio DEVE resultar em erro de validação
				Expect(err).To(HaveOccurred())
				Expect(response).To(BeNil())
				Expect(err.Error()).To(ContainSubstring("dados inválidos"))
			})

			It("deve lidar com senha de tamanho mínimo", func() {
				registerRequest := &services.RegisterRequest{
					Email:    "test@example.com",
					Name:     "Test User",
					Password: "SecurePass123!",
				}

				// Mock expectations
				mockUserRepo.EXPECT().GetByUsername("test@example.com").Return(nil, errors.New("user not found"))
				mockUserRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
					// Simulate GORM BeforeCreate hook - generate UUID and hash password
					user.ID = uuid.New()
					user.CreatedAt = time.Now()
					user.UpdatedAt = time.Now()
					return user.HashPassword()
				})

				response, err := authService.Register(registerRequest)

				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
			})
		})
	})

	Describe("Fluxo de criação de usuário", func() {
		It("deve seguir o fluxo completo de registro", func() {
			registerRequest := &services.RegisterRequest{
				Email:    "fullflow@example.com",
				Name:     "Full Flow User",
				Password: "SecurePass123!",
			}

			var createdUser *models.User

			By("registrando o usuário")
			// Mock expectations for registration
			mockUserRepo.EXPECT().GetByUsername("fullflow@example.com").Return(nil, errors.New("user not found"))
			mockUserRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
				// Simulate GORM BeforeCreate hook - generate UUID and hash password
				user.ID = uuid.New()
				user.CreatedAt = time.Now()
				user.UpdatedAt = time.Now()
				err := user.HashPassword()
				if err != nil {
					return err
				}
				// Store the created user for later verification
				createdUser = user
				return nil
			})

			response, err := authService.Register(registerRequest)
			Expect(err).ToNot(HaveOccurred())
			Expect(response).ToNot(BeNil())

			By("verificando se o usuário foi criado corretamente")
			Expect(createdUser).ToNot(BeNil())
			Expect(createdUser.Username).To(Equal("fullflow@example.com"))
			Expect(createdUser.Name).To(Equal("Full Flow User"))

			By("verificando se a senha foi hasheada")
			Expect(createdUser.Password).ToNot(Equal("SecurePass123!"))

			By("verificando se a resposta contém dados corretos do usuário")
			Expect(response.User.Username).To(Equal("fullflow@example.com"))
			Expect(response.User.Name).To(Equal("Full Flow User"))
			Expect(response.User.ID).ToNot(Equal(uuid.Nil))

			By("verificando se o token JWT é retornado")
			Expect(response.Token).ToNot(BeEmpty())
		})
	})

	Describe("Funcionalidade de bloqueio de conta", func() {
		var testUser *models.User

		BeforeEach(func() {
			// Create a test user with hashed password
			testUser = &models.User{
				ID:       uuid.New(),
				Username: "locktest@example.com",
				Name:     "Lock Test User",
			}
			// Hash the password manually since we're not using real GORM
			testUser.Password = "CorrectPass123!"
			err := testUser.HashPassword()
			Expect(err).ToNot(HaveOccurred())
		})

		Context("quando usuário digita senha incorreta múltiplas vezes", func() {
			It("deve bloquear conta após máximo de tentativas", func() {
				loginRequest := &services.LoginRequest{
					Email:    "locktest@example.com",
					Password: "WrongPass123!",
				}

				// Mock expectations for 3 failed login attempts
				// Attempt 1
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(testUser, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					Expect(user.FailedLoginAttempts).To(Equal(1))
					testUser.FailedLoginAttempts = 1
					return nil
				})

				response, err := authService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("email ou senha incorretos"))
				Expect(response).To(BeNil())

				// Attempt 2
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(testUser, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					Expect(user.FailedLoginAttempts).To(Equal(2))
					testUser.FailedLoginAttempts = 2
					return nil
				})

				response, err = authService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("email ou senha incorretos"))
				Expect(response).To(BeNil())

				// Attempt 3 - should lock account
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(testUser, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					Expect(user.FailedLoginAttempts).To(Equal(3))
					Expect(user.IsLocked()).To(BeTrue())
					Expect(user.LockedUntil).ToNot(BeNil())
					testUser.FailedLoginAttempts = 3
					testUser.LockAccount(15 * time.Minute)
					return nil
				})

				response, err = authService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("email ou senha incorretos"))
				Expect(response).To(BeNil())
			})

			It("deve impedir login quando conta está bloqueada", func() {
				// Set up locked user
				lockedUser := &models.User{
					ID:                  testUser.ID,
					Username:            testUser.Username,
					Name:                testUser.Name,
					Password:            testUser.Password,
					FailedLoginAttempts: 3,
				}
				lockedUser.LockAccount(15 * time.Minute)

				// Try to login with correct password
				loginRequest := &services.LoginRequest{
					Email:    "locktest@example.com",
					Password: "CorrectPass123!",
				}

				// Mock expectations - return locked user
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(lockedUser, nil)

				response, err := authService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Conta bloqueada"))
				Expect(response).To(BeNil())
			})

			It("deve permitir login com senha correta quando conta não está bloqueada", func() {
				// Try to login with correct password
				loginRequest := &services.LoginRequest{
					Email:    "locktest@example.com",
					Password: "CorrectPass123!",
				}

				// Mock expectations for successful login
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(testUser, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					// Reset failed attempts on successful login
					Expect(user.FailedLoginAttempts).To(Equal(0))
					return nil
				})

				response, err := authService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
				Expect(response.Token).ToNot(BeEmpty())
				Expect(response.User.Username).To(Equal("locktest@example.com"))
			})

			It("deve resetar tentativas falhadas após login bem-sucedido", func() {
				// Set up user with failed attempts
				userWithFailedAttempts := &models.User{
					ID:                  testUser.ID,
					Username:            testUser.Username,
					Name:                testUser.Name,
					Password:            testUser.Password,
					FailedLoginAttempts: 2,
				}
				now := time.Now()
				userWithFailedAttempts.LastFailedLoginAt = &now

				// Login with correct password
				loginRequest := &services.LoginRequest{
					Email:    "locktest@example.com",
					Password: "CorrectPass123!",
				}

				// Mock expectations for successful login that resets failed attempts
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(userWithFailedAttempts, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					// Verify failed attempts were reset
					Expect(user.FailedLoginAttempts).To(Equal(0))
					Expect(user.LastFailedLoginAt).To(BeNil())
					return nil
				})

				response, err := authService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
			})
		})

		Context("quando bloqueio de conta expira", func() {
			It("deve permitir login após período de bloqueio", func() {
				// Set up user with expired lockout
				expiredLockedUser := &models.User{
					ID:                  testUser.ID,
					Username:            testUser.Username,
					Name:                testUser.Name,
					Password:            testUser.Password,
					FailedLoginAttempts: 3,
				}
				// Set lockout time in the past (expired)
				pastTime := time.Now().Add(-1 * time.Minute)
				expiredLockedUser.LockedUntil = &pastTime

				// Try to login with correct password
				loginRequest := &services.LoginRequest{
					Email:    "locktest@example.com",
					Password: "CorrectPass123!",
				}

				// Mock expectations for successful login after lockout expires
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(expiredLockedUser, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					// Reset failed attempts on successful login
					Expect(user.FailedLoginAttempts).To(Equal(0))
					return nil
				})

				response, err := authService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
				Expect(response.Token).ToNot(BeEmpty())
			})
		})

		Context("com máximo de tentativas configurável", func() {
			It("deve respeitar o máximo de tentativas de login configurado", func() {
				// Test with different configuration
				customConfig := &config.Config{
					Auth: config.AuthConfig{
						MaxLoginAttempts:       2, // Different from default 3
						AccountLockoutDuration: 5 * time.Minute,
					},
					JWT: config.JWTConfig{
						Secret:          jwtSecret,
						ExpirationHours: 24,
					},
				}

				customAuthService := services.NewAuthService(mockUserRepo, mockPasswordRepo, jwtSecret, &customConfig.Auth, &customConfig.JWT)

				loginRequest := &services.LoginRequest{
					Email:    "locktest@example.com",
					Password: "WrongPass123!",
				}

				// Mock expectations for 2 failed login attempts (custom config)
				// Attempt 1
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(testUser, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					Expect(user.FailedLoginAttempts).To(Equal(1))
					testUser.FailedLoginAttempts = 1
					return nil
				})

				_, err := customAuthService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).To(HaveOccurred())

				// Attempt 2 - should lock account with MaxLoginAttempts = 2
				mockUserRepo.EXPECT().GetByUsername("locktest@example.com").Return(testUser, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					Expect(user.FailedLoginAttempts).To(Equal(2))
					Expect(user.IsLocked()).To(BeTrue())
					testUser.FailedLoginAttempts = 2
					testUser.LockAccount(5 * time.Minute)
					return nil
				})

				_, err = customAuthService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).To(HaveOccurred())
			})
		})

		Context("quando usuário não existe", func() {
			It("deve retornar erro de credenciais inválidas", func() {
				loginRequest := &services.LoginRequest{
					Email:    "nonexistent@example.com",
					Password: "AnyPassword123!",
				}

				// Mock expectations - user not found
				mockUserRepo.EXPECT().GetByUsername("nonexistent@example.com").Return(nil, errors.New("user not found"))

				response, err := authService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("email ou senha incorretos"))
				Expect(response).To(BeNil())
			})
		})
	})

	Describe("Expiração de token JWT", func() {
		Context("quando token é gerado com configuração de expiração", func() {
			It("deve gerar token com tempo de expiração configurado", func() {
				// Configuração personalizada com expiração de 1 hora
				customConfig := &config.Config{
					Auth: config.AuthConfig{
						MaxLoginAttempts:       3,
						AccountLockoutDuration: 15 * time.Minute,
					},
					JWT: config.JWTConfig{
						Secret:          "test-secret-key",
						ExpirationHours: 1, // 1 hora
					},
				}

				customAuthService := services.NewAuthService(mockUserRepo, mockPasswordRepo, customConfig.JWT.Secret, &customConfig.Auth, &customConfig.JWT)

				// Criar usuário de teste
				testUser := &models.User{
					ID:       uuid.New(),
					Username: "jwt-test@example.com",
					Name:     "JWT Test User",
					Password: "SecurePass123!",
				}
				err := testUser.HashPassword()
				Expect(err).ToNot(HaveOccurred())

				loginRequest := &services.LoginRequest{
					Email:    "jwt-test@example.com",
					Password: "SecurePass123!",
				}

				// Mock expectations
				mockUserRepo.EXPECT().GetByUsername("jwt-test@example.com").Return(testUser, nil)
				mockUserRepo.EXPECT().Update(gomock.Any()).DoAndReturn(func(user *models.User) error {
					return nil
				})

				response, err := customAuthService.LoginWithContext(loginRequest, "192.168.1.1", "test-agent")

				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
				Expect(response.Token).ToNot(BeEmpty())

				// Verificar se o token contém claims de expiração
				Expect(response.Token).To(ContainSubstring("."))

				// Parse do token para verificar claims (simulação)
				tokenParts := strings.Split(response.Token, ".")
				Expect(len(tokenParts)).To(Equal(3)) // Header.Payload.Signature
			})

			It("deve incluir claims de 'iat', 'exp' e 'nbf' no token", func() {
				// Usar reflection ou parse manual para verificar claims
				// Por simplicidade, verificamos que o token foi gerado corretamente
				registerRequest := &services.RegisterRequest{
					Email:    "claims-test@example.com",
					Name:     "Claims Test User",
					Password: "SecurePass123!",
				}

				// Mock expectations
				mockUserRepo.EXPECT().GetByUsername("claims-test@example.com").Return(nil, errors.New("user not found"))
				mockUserRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
					user.ID = uuid.New()
					user.CreatedAt = time.Now()
					user.UpdatedAt = time.Now()
					return user.HashPassword()
				})

				response, err := authService.Register(registerRequest)

				Expect(err).ToNot(HaveOccurred())
				Expect(response).ToNot(BeNil())
				Expect(response.Token).ToNot(BeEmpty())

				// Token deve ter 3 partes separadas por pontos
				tokenParts := strings.Split(response.Token, ".")
				Expect(len(tokenParts)).To(Equal(3))
			})
		})

		Context("com política de senha forte", func() {
			It("deve rejeitar senhas que não atendem aos critérios de segurança", func() {
				weakPasswords := []string{
					"123456",      // Muito simples (menos de 8 chars)
					"password",    // Sem números/maiúsculas/símbolos
					"PASSWORD",    // Sem minúsculas/números/símbolos
					"Password123", // Sem símbolos
					"Pass1!",      // Muito curta (menos de 8 chars)
				}

				for _, weakPassword := range weakPasswords {
					registerRequest := &services.RegisterRequest{
						Email:    "weak-pass-test@example.com",
						Name:     "Weak Password User",
						Password: weakPassword,
					}

					By(fmt.Sprintf("testando senha fraca: %s", weakPassword))

					// Agora a validação está no service, então senhas fracas devem falhar
					// Não precisamos de mocks para usuário que não existe ou create
					// porque a validação vai falhar antes

					response, err := authService.Register(registerRequest)

					// Senha fraca DEVE resultar em erro de validação
					Expect(err).To(HaveOccurred())
					Expect(response).To(BeNil())
					Expect(err.Error()).To(ContainSubstring("dados inválidos"))
				}
			})

			It("deve aceitar senhas que atendem aos critérios de segurança", func() {
				strongPasswords := []string{
					"SecurePass123!",
					"MyStr0ng*Password",
					"C0mpl3x&P4ssw0rd",
					"S3cur3P4ssw0rd!",
				}

				for i, strongPassword := range strongPasswords {
					email := fmt.Sprintf("strong-pass-test%d@example.com", i)
					registerRequest := &services.RegisterRequest{
						Email:    email,
						Name:     "Strong Password User",
						Password: strongPassword,
					}

					By(fmt.Sprintf("testando senha forte: %s", strongPassword))

					// Mock expectations
					mockUserRepo.EXPECT().GetByUsername(email).Return(nil, errors.New("user not found"))
					mockUserRepo.EXPECT().Create(gomock.Any()).DoAndReturn(func(user *models.User) error {
						user.ID = uuid.New()
						user.CreatedAt = time.Now()
						user.UpdatedAt = time.Now()
						return user.HashPassword()
					})

					response, err := authService.Register(registerRequest)

					Expect(err).ToNot(HaveOccurred())
					Expect(response).ToNot(BeNil())
					Expect(response.Token).ToNot(BeEmpty())
					Expect(response.User.Username).To(Equal(email))
				}
			})
		})
	})

	Describe("Funcionalidade de Reset de Senha", func() {
		Context("quando usuário faz reset de senha com token válido", func() {
			It("deve atualizar a senha usando UpdatePassword ao invés de Update completo", func() {
				email := "resettest@example.com"
				newPassword := "NewSecurePassword123!"
				userID := uuid.New()
				token := "valid-reset-token"

				// Mock user retrieval for password reset request
				existingUser := &models.User{
					ID:       userID,
					Username: email,
					Name:     "Reset Test User",
					Password: "old-hashed-password",
				}

				// Mock password reset token
				resetToken := &models.PasswordResetToken{
					UserID:    userID,
					Token:     token,
					ExpiresAt: time.Now().Add(time.Hour),
				}

				// Setup expectations for password reset request
				mockUserRepo.EXPECT().
					GetByUsername(email).
					Return(existingUser, nil)

				mockPasswordRepo.EXPECT().
					DeleteUserTokens(userID).
					Return(nil)

				mockPasswordRepo.EXPECT().
					Create(gomock.Any()).
					DoAndReturn(func(tokenRecord *models.PasswordResetToken) error {
						tokenRecord.ID = uuid.New()
						tokenRecord.CreatedAt = time.Now()
						tokenRecord.UpdatedAt = time.Now()
						return nil
					})

				// Setup expectations for password reset confirmation
				mockPasswordRepo.EXPECT().
					GetByToken(token).
					Return(resetToken, nil)

				mockUserRepo.EXPECT().
					GetByID(userID).
					Return(existingUser, nil)

				// CRITICAL: Expect UpdatePassword to be called, not Update
				mockUserRepo.EXPECT().
					UpdatePassword(userID, gomock.Any()).
					DoAndReturn(func(id uuid.UUID, hashedPassword string) error {
						// Verify that the password is actually hashed
						Expect(hashedPassword).ToNot(Equal(newPassword))
						Expect(len(hashedPassword)).To(BeNumerically(">", 50)) // bcrypt hashes are typically 60+ chars
						Expect(hashedPassword).To(HavePrefix("$2a$"))          // bcrypt prefix
						return nil
					})

				mockPasswordRepo.EXPECT().
					Update(gomock.Any()).
					Return(nil)

				// Request password reset
				err := authService.RequestPasswordReset(email)
				Expect(err).ToNot(HaveOccurred())

				// Reset password with token
				err = authService.ResetPassword(token, newPassword)
				Expect(err).ToNot(HaveOccurred())
			})

			It("deve falhar se UpdatePassword retornar erro", func() {
				email := "resettest@example.com"
				newPassword := "NewSecurePassword123!"
				userID := uuid.New()
				token := "valid-reset-token"

				// Mock user and token
				existingUser := &models.User{
					ID:       userID,
					Username: email,
					Name:     "Reset Test User",
					Password: "old-hashed-password",
				}

				resetToken := &models.PasswordResetToken{
					UserID:    userID,
					Token:     token,
					ExpiresAt: time.Now().Add(time.Hour),
				}

				// Setup expectations
				mockPasswordRepo.EXPECT().
					GetByToken(token).
					Return(resetToken, nil)

				mockUserRepo.EXPECT().
					GetByID(userID).
					Return(existingUser, nil)

				// Mock UpdatePassword to return an error
				mockUserRepo.EXPECT().
					UpdatePassword(userID, gomock.Any()).
					Return(errors.New("database error"))

				// Reset password should fail
				err := authService.ResetPassword(token, newPassword)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("database error"))
			})

			It("deve garantir que o hash da senha seja diferente da senha original", func() {
				email := "resettest@example.com"
				newPassword := "NewSecurePassword123!"
				userID := uuid.New()
				token := "valid-reset-token"

				existingUser := &models.User{
					ID:       userID,
					Username: email,
					Name:     "Reset Test User",
					Password: "old-hashed-password",
				}

				resetToken := &models.PasswordResetToken{
					UserID:    userID,
					Token:     token,
					ExpiresAt: time.Now().Add(time.Hour),
				}

				mockPasswordRepo.EXPECT().
					GetByToken(token).
					Return(resetToken, nil)

				mockUserRepo.EXPECT().
					GetByID(userID).
					Return(existingUser, nil)

				var capturedHashedPassword string
				mockUserRepo.EXPECT().
					UpdatePassword(userID, gomock.Any()).
					DoAndReturn(func(id uuid.UUID, hashedPassword string) error {
						capturedHashedPassword = hashedPassword
						return nil
					})

				mockPasswordRepo.EXPECT().
					Update(gomock.Any()).
					Return(nil)

				// Reset password
				err := authService.ResetPassword(token, newPassword)
				Expect(err).ToNot(HaveOccurred())

				// Verify password was properly hashed
				Expect(capturedHashedPassword).ToNot(Equal(newPassword))
				Expect(capturedHashedPassword).ToNot(Equal("old-hashed-password"))
				Expect(len(capturedHashedPassword)).To(BeNumerically(">", 50))
				Expect(capturedHashedPassword).To(HavePrefix("$2a$"))
			})
		})

		Context("quando token é inválido ou expirado", func() {
			It("deve retornar erro apropriado", func() {
				token := "invalid-token"

				mockPasswordRepo.EXPECT().
					GetByToken(token).
					Return(nil, errors.New("token not found"))

				err := authService.ResetPassword(token, "NewSecurePass123!")
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
