package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// AuthController handles HTTP requests for authentication operations
type AuthController struct {
	authService services.AuthService
}

// NewAuthController creates a new AuthController instance
func NewAuthController(authService services.AuthService) *AuthController {
	return &AuthController{authService: authService}
}

// Login authenticates a user and returns a token
// @Summary Login user
// @Description Authenticate user with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.LoginRequest true "Login credentials"
// @Success 200 {object} models.LoginResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 401 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Unauthorized"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/auth/login [post]
func (c *AuthController) Login(ctx *gin.Context) {
	var req models.LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := errors.BadRequestError("Invalid JSON format: "+err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	response, err := c.authService.Login(&req)
	if err != nil {
		if err.Error() == "invalid credentials" {
			prob := errors.UnauthorizedError("Invalid email or password", errors.GetInstance(ctx))
			errors.RespondWithProblem(ctx, prob)
			return
		}
		prob := errors.InternalError(errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// Register creates a new user account
// @Summary Register new user
// @Description Create a new user account
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.RegisterRequest true "Registration data"
// @Success 201 {object} models.LoginResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 409 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Conflict"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/auth/register [post]
func (c *AuthController) Register(ctx *gin.Context) {
	var req models.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := errors.BadRequestError("Invalid JSON format: "+err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	response, err := c.authService.Register(&req)
	if err != nil {
		if err.Error() == errors.ErrUsernameAlreadyExists {
			prob := errors.ConflictError(err.Error(), errors.GetInstance(ctx))
			errors.RespondWithProblem(ctx, prob)
			return
		}
		prob := errors.ValidationError(err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusCreated, response)
}

// RequestPasswordReset sends password reset email
// @Summary Request password reset
// @Description Send password reset email to user
// @Tags auth
// @Accept json
// @Produce json
// @Param email body models.PasswordResetRequest true "Email for password reset"
// @Success 200 {object} models.MessageResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 404 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Not Found"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/auth/password-reset [post]
func (c *AuthController) RequestPasswordReset(ctx *gin.Context) {
	var req models.PasswordResetRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := errors.BadRequestError("Invalid JSON format: "+err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	err := c.authService.RequestPasswordReset(req.Email)
	if err != nil {
		if err.Error() == errors.ErrUserNotFound {
			prob := errors.NotFoundError("User", errors.GetInstance(ctx))
			errors.RespondWithProblem(ctx, prob)
			return
		}
		prob := errors.InternalError(errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, models.MessageResponse{
		Message: "Password reset email sent successfully",
	})
}

// ResetPassword resets user password with token
// @Summary Reset password
// @Description Reset user password using reset token
// @Tags auth
// @Accept json
// @Produce json
// @Param reset body models.PasswordResetConfirmRequest true "Password reset data"
// @Success 200 {object} models.MessageResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 404 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Not Found"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/auth/password-reset/confirm [post]
func (c *AuthController) ResetPassword(ctx *gin.Context) {
	var req models.PasswordResetConfirmRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := errors.BadRequestError("Invalid JSON format: "+err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	err := c.authService.ResetPassword(req.Token, req.NewPassword)
	if err != nil {
		if err.Error() == "invalid or expired token" {
			prob := errors.BadRequestError("Invalid or expired reset token", errors.GetInstance(ctx))
			errors.RespondWithProblem(ctx, prob)
			return
		}
		prob := errors.InternalError(errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, models.MessageResponse{
		Message: "Password reset successfully",
	})
}

// UpdateLastSelectedOrganization updates user's last selected organization preference
// @Summary Update last selected organization
// @Description Update user's last selected organization preference
// @Tags auth
// @Accept json
// @Produce json
// @Param organization body models.UpdateLastSelectedOrganizationRequest true "Organization preference data"
// @Success 200 {object} models.MessageResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 401 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Unauthorized"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/auth/last-selected-organization [put]
// @Security Bearer
func (c *AuthController) UpdateLastSelectedOrganization(ctx *gin.Context) {
	var req models.UpdateLastSelectedOrganizationRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := errors.BadRequestError("Invalid JSON format: "+err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	// Get user ID from context (set by JWT middleware)
	userIDStr, exists := ctx.Get("user_id")
	if !exists {
		prob := errors.UnauthorizedError("User not authenticated", errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		prob := errors.BadRequestError("Invalid user ID", errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	err = c.authService.UpdateLastSelectedOrganization(userID, req.OrganizationID)
	if err != nil {
		prob := errors.InternalError(errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, models.MessageResponse{
		Message: "Last selected organization updated successfully",
	})
}
