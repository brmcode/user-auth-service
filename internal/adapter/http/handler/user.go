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
	"github.com/brmcode/user-auth-service/pkg/i18n"
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
		ctx.JSON(http.StatusForbidden, response.NewError(403, i18n.Translate("auth.unauthorized.resource")))
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
		ctx.JSON(http.StatusBadRequest, response.NewError(400, i18n.Translate("request.invalid_body")))
		return
	}

	if err := u.validator.Validate(req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, i18n.Translate("request.validation_failed")))
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
		ctx.JSON(http.StatusInternalServerError, response.NewError(500, i18n.Translate("auth.payload.not_found")))
		return
	}

	var req request.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, i18n.Translate("request.invalid_body")))
		return
	}

	if err := u.validator.Validate(req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, i18n.Translate("request.validation_failed")))
		return
	}

	if username != req.Username {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, i18n.Translate("auth.username.path_mismatch")))
		return
	}

	isAdmin := payload.HasRole(domain.ADMIN_ROLE)
	isOwner := payload.Username == username

	// Only the owner or an admin can update a user.
	if !isOwner && !isAdmin {
		ctx.JSON(
			http.StatusForbidden,
			response.NewError(403, i18n.Translate("auth.user.update_not_allowed")),
		)
		return
	}

	// Only admins can change roles.
	if !isAdmin && !roleCodesEqual(payload.Roles, req.Roles) {
		ctx.JSON(
			http.StatusForbidden,
			response.NewError(403, i18n.Translate("auth.roles.insufficient_permissions")),
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
		ctx.JSON(http.StatusForbidden, response.NewError(403, i18n.Translate("auth.user.delete_not_allowed")))
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
