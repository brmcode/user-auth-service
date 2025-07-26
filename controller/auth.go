package controller

import (
	"net/http"

	"github.com/brmcode/user-auth-service/dto"
	"github.com/brmcode/user-auth-service/dto/response"
	"github.com/brmcode/user-auth-service/service"
	"github.com/gin-gonic/gin"
)

type AuthController struct {
	userService service.UserService
	authService service.AuthenticationService
}

func NewAuthController(userService service.UserService, authService service.AuthenticationService) *AuthController {
	return &AuthController{userService: userService, authService: authService}
}

func (a *AuthController) Login(ctx *gin.Context) {
	var input dto.LoginModel

	if err := ctx.ShouldBindJSON(&input); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	res, resErr := a.authService.Login(input)
	if resErr != nil {
		ctx.JSON(resErr.Code, resErr)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (a *AuthController) Register(ctx *gin.Context) {
	var req dto.CreateUserRequest

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
