package handler

import (
	"net/http"

	dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/adapter/validator"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	validator   *validator.Validator
	userService port.UserService
	authService port.AuthenticationService
}

// RegisterAndLogin handles user registration and immediate login
func (a *AuthHandler) RegisterAndLogin(ctx *gin.Context) {
	var req dto.RegisterUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if err := a.validator.Validate(req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	userResult := a.authService.Register(ctx, req)
	if !userResult.Success {
		ctx.JSON(userResult.StatusCode, userResult)
		return
	}

	// Now login the user
	loginModel := dto.LoginModel{
		Email:    req.Email,
		Password: req.Password,
	}
	loginResult := a.authService.Login(ctx, loginModel)
	if !loginResult.Success {
		ctx.JSON(loginResult.StatusCode, loginResult)
		return
	}

	ctx.JSON(http.StatusCreated, loginResult)
}

func NewAuthHandler(validator *validator.Validator, userService port.UserService, authService port.AuthenticationService) *AuthHandler {
	return &AuthHandler{validator: validator, userService: userService, authService: authService}
}

func (a *AuthHandler) Login(ctx *gin.Context) {
	var input dto.LoginModel

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if err := a.validator.Validate(input); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	res := a.authService.Login(ctx, input)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (a *AuthHandler) RefreshToken(ctx *gin.Context) {
	var input dto.ReNewAccessTokenRequest

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	res := a.authService.ReNewAccessToken(ctx, input)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (a *AuthHandler) Logout(ctx *gin.Context) {
	var input dto.ReNewAccessTokenRequest

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	res := a.authService.Logout(ctx, input)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (a *AuthHandler) Register(ctx *gin.Context) {
	var req dto.RegisterUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if err := a.validator.Validate(req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	user := a.authService.Register(ctx, req)
	if !user.Success {
		ctx.JSON(user.StatusCode, user)
		return
	}

	ctx.JSON(http.StatusCreated, user)
}
