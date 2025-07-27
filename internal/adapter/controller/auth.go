package controller

import (
	"net/http"

	dto "github.com/brmcode/user-auth-service/internal/core/dto/common"
	"github.com/brmcode/user-auth-service/internal/core/dto/request"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	userService port.UserService
	authService port.AuthenticationService
}

func NewAuthController(userService port.UserService, authService port.AuthenticationService) *AuthController {
	return &AuthController{userService: userService, authService: authService}
}

func (a *AuthController) Login(ctx *gin.Context) {
	var input dto.LoginModel

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	res, resErr := a.authService.Login(ctx, input)
	if resErr != nil {
		ctx.JSON(resErr.Code, resErr)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (a *AuthController) RefreshToken(ctx *gin.Context) {
	var input dto.ReNewAccessTokenRequest

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	res, resErr := a.authService.ReNewAccessToken(input)
	if resErr != nil {
		ctx.JSON(resErr.Code, resErr)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (a *AuthController) Register(ctx *gin.Context) {
	var req request.CreateUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	user, resErr := a.authService.Register(req)
	if resErr != nil {
		ctx.JSON(resErr.Code, resErr)
		return
	}

	ctx.JSON(http.StatusCreated, user)
}
