package handler

import (
	"net/http"
	"slices"

	"github.com/brmcode/user-auth-service/internal/adapter/auth"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/request"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/adapter/validator"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	validator   *validator.Validator
	userService port.UserService
}

func NewUserHandler(v *validator.Validator, userService port.UserService) *UserHandler {
	return &UserHandler{validator: v, userService: userService}
}

func (u *UserHandler) GetUser(ctx *gin.Context) {
	username := ctx.Param("username")
	if auth.GetUsername(ctx) != username {
		ctx.JSON(http.StatusForbidden, response.NewError(403, "you are not authorized to access this resource"))
		return
	}

	res := u.userService.GetUser(ctx, username)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (u *UserHandler) CreateUser(ctx *gin.Context) {
	var req request.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if err := u.validator.Validate(req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	res := u.userService.CreateUser(ctx, req)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.JSON(http.StatusCreated, res)
}

func (u *UserHandler) UpdateUser(ctx *gin.Context) {
	username := ctx.Param("username")
	payload := auth.GetPayload(ctx)
	if payload == nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError(500, "payload not found"))
		return
	}

	var req request.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if err := u.validator.Validate(req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if username != req.Username {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, "username in path does not match body"))
		return
	}

	isAdmin := payload.HasRole(domain.ADMIN_ROLE)
	isOwner := payload.Username == username

	// Only the owner or an admin can update a user.
	if !isOwner && !isAdmin {
		ctx.JSON(
			http.StatusForbidden,
			response.NewError(403, "you are not allowed to update this user"),
		)
		return
	}

	// Only admins can change roles.
	if !isAdmin && !roleCodesEqual(payload.Roles, req.Roles) {
		ctx.JSON(
			http.StatusForbidden,
			response.NewError(403, "insufficient permissions to change roles"),
		)
		return
	}

	res := u.userService.UpdateUser(ctx, req)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (u *UserHandler) DeleteUser(ctx *gin.Context) {
	username := ctx.Param("username")

	if auth.GetUsername(ctx) != username {
		ctx.JSON(http.StatusForbidden, response.NewError(403, "you are not allowed to delete this user"))
		return
	}

	res := u.userService.DeleteUser(ctx, username)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.Status(http.StatusNoContent)
}

func roleCodesEqual(currentRoles, requestedRoles []string) bool {
	if len(currentRoles) != len(requestedRoles) {
		return false
	}

	sortedCurrentRoles := slices.Clone(currentRoles)
	sortedRequestedRoles := slices.Clone(requestedRoles)

	slices.Sort(sortedCurrentRoles)
	slices.Sort(sortedRequestedRoles)

	return slices.Equal(sortedCurrentRoles, sortedRequestedRoles)
}
