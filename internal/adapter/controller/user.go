package controller

import (
	"net/http"

	"github.com/brmcode/user-auth-service/internal/adapter/validator"
	"github.com/brmcode/user-auth-service/internal/core/domain"
	"github.com/brmcode/user-auth-service/internal/core/dto/request"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	validator   *validator.Validator
	userService port.UserService
}

func NewUserController(validator *validator.Validator, userService port.UserService) *UserController {
	return &UserController{validator: validator, userService: userService}
}

func (u *UserController) GetUser(ctx *gin.Context) {
	username := ctx.Param("username")
	if domain.GetUsername(ctx) != username {
		ctx.JSON(http.StatusForbidden, response.NewError(403, "you are not authorized to access this resource"))
		return
	}

	res, errRes := u.userService.GetUser(ctx, username)
	if errRes != nil {
		ctx.JSON(errRes.StatusCode, errRes)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (u *UserController) CreateUser(ctx *gin.Context) {
	var req request.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if err := u.validator.Validate(req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	user, resErr := u.userService.CreateUser(ctx, req)
	if resErr != nil {
		ctx.JSON(resErr.StatusCode, resErr)
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

func (u *UserController) UpdateUser(ctx *gin.Context) {
	username := ctx.Param("username")
	payload := domain.GetPayload(ctx)
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
		ctx.JSON(http.StatusBadRequest, response.NewError(400, "username not match"))
		return
	}

	if payload.Username != username {
		ctx.JSON(http.StatusForbidden, response.NewError(403, "you are not allowed to update this user"))
		return
	}

	if payload.Role != domain.ADMIN_ROLE && req.Role != payload.Role {
		ctx.JSON(http.StatusForbidden, response.NewError(403, "insufficient permissions to change role"))
		return
	}

	user, resErr := u.userService.UpdateUser(ctx, req)
	if resErr != nil {
		ctx.JSON(resErr.StatusCode, resErr)
		return
	}

	ctx.JSON(http.StatusOK, user)
}

func (u *UserController) DeleteUser(ctx *gin.Context) {
	username := ctx.Param("username")

	if domain.GetUsername(ctx) != username {
		ctx.JSON(http.StatusForbidden, response.NewError(403, "you are not allowed to delete this user"))
		return
	}

	if resErr := u.userService.DeleteUser(ctx, username); resErr != nil {
		ctx.JSON(resErr.StatusCode, resErr)
		return
	}

	ctx.Status(http.StatusNoContent)
}
