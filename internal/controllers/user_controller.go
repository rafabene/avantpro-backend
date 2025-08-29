package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/rafabene/avantpro-backend/internal/errors"
	"github.com/rafabene/avantpro-backend/internal/models"
	"github.com/rafabene/avantpro-backend/internal/services"
)

// UserController handles HTTP requests for user operations
type UserController struct {
	service services.UserService
}

// NewUserController creates a new UserController instance
func NewUserController(service services.UserService) *UserController {
	return &UserController{service: service}
}

// CreateUser creates a new user
// @Summary Create a new user
// @Description Create a new user with optional profile information
// @Tags users
// @Accept json
// @Produce json
// @Param user body models.UserCreateRequest true "User creation data"
// @Success 201 {object} models.UserResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 409 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Conflict"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/users [post]
func (c *UserController) CreateUser(ctx *gin.Context) {
	var req models.UserCreateRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := errors.BadRequestError("Invalid JSON format: "+err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	user, err := c.service.CreateUser(&req)
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

	ctx.JSON(http.StatusCreated, user)
}

// GetUser retrieves a user by ID
// @Summary Get user by ID
// @Description Get a single user by their unique identifier
// @Tags users
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 404 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Not Found"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/users/{id} [get]
func (c *UserController) GetUser(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		prob := errors.BadRequestError("Invalid UUID format", errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	user, err := c.service.GetUserByID(id)
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

	ctx.JSON(http.StatusOK, user)
}

// GetUserByUsername retrieves a user by username
// @Summary Get user by username
// @Description Get a single user by their username (email)
// @Tags users
// @Produce json
// @Param username path string true "User username (email)"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 404 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Not Found"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/users/username/{username} [get]
func (c *UserController) GetUserByUsername(ctx *gin.Context) {
	username := ctx.Param("username")
	if username == "" {
		prob := errors.BadRequestError("Username is required", errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	user, err := c.service.GetUserByUsername(username)
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

	ctx.JSON(http.StatusOK, user)
}

// ListUsers retrieves a paginated list of users
// @Summary List users with pagination and sorting
// @Description Get a paginated list of users with optional sorting
// @Tags users
// @Produce json
// @Param page query int false "Page number" minimum(1) default(1)
// @Param limit query int false "Number of items per page" minimum(1) maximum(100) default(50)
// @Param sortBy query string false "Field to sort by" Enums(name, username, createdAt, updatedAt) default(createdAt)
// @Param sortOrder query string false "Sort order" Enums(asc, desc) default(desc)
// @Success 200 {object} models.UserListResponse
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/users [get]
func (c *UserController) ListUsers(ctx *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(ctx.DefaultQuery("limit", "50"))
	sortBy := ctx.DefaultQuery("sortBy", "createdAt")
	sortOrder := ctx.DefaultQuery("sortOrder", "desc")

	response, err := c.service.ListUsers(page, limit, sortBy, sortOrder)
	if err != nil {
		prob := errors.InternalError(errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateUser updates an existing user
// @Summary Update user
// @Description Update an existing user's information
// @Tags users
// @Accept json
// @Produce json
// @Param id path string true "User ID" format(uuid)
// @Param user body models.UserUpdateRequest true "User update data"
// @Success 200 {object} models.UserResponse
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 404 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Not Found"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/users/{id} [put]
func (c *UserController) UpdateUser(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		prob := errors.BadRequestError("Invalid UUID format", errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	var req models.UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		prob := errors.BadRequestError("Invalid JSON format: "+err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	user, err := c.service.UpdateUser(id, &req)
	if err != nil {
		if err.Error() == errors.ErrUserNotFound {
			prob := errors.NotFoundError("User", errors.GetInstance(ctx))
			errors.RespondWithProblem(ctx, prob)
			return
		}
		prob := errors.ValidationError(err.Error(), errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// DeleteUser deletes a user
// @Summary Delete user
// @Description Delete a user by their unique identifier
// @Tags users
// @Param id path string true "User ID" format(uuid)
// @Success 204
// @Failure 400 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Bad Request"
// @Failure 404 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Not Found"
// @Failure 500 {object} object{type=string,title=string,status=int,detail=string,instance=string} "Internal Server Error"
// @Router /api/v1/users/{id} [delete]
func (c *UserController) DeleteUser(ctx *gin.Context) {
	idParam := ctx.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		prob := errors.BadRequestError("Invalid UUID format", errors.GetInstance(ctx))
		errors.RespondWithProblem(ctx, prob)
		return
	}

	err = c.service.DeleteUser(id)
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

	ctx.Status(http.StatusNoContent)
}