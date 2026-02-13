package controller

import (
	"net/http"

	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

type OAuthController struct {
	authService port.AuthenticationService
}

func NewOAuthController(authService port.AuthenticationService) *OAuthController {
	return &OAuthController{authService: authService}
}

func (o *OAuthController) Begin(ctx *gin.Context) {
	provider := ctx.Param("provider")
	if provider == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError(http.StatusBadRequest, "provider parameter is required"))
		return
	}
	// Set provider in context for CompleteUserAuth
	ctx.Request = gothic.GetContextWithProvider(ctx.Request, provider)
	gothic.BeginAuthHandler(ctx.Writer, ctx.Request)
}

func (o *OAuthController) Callback(ctx *gin.Context) {
	provider := ctx.Param("provider")
	if provider == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError(http.StatusBadRequest, "provider parameter is required"))
		return
	}

	// Set provider in context for CompleteUserAuth
	ctx.Request = gothic.GetContextWithProvider(ctx.Request, provider)

	user, err := gothic.CompleteUserAuth(ctx.Writer, ctx.Request)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewError(500, err.Error()))
		return
	}

	res, resErr := o.authService.OAuthLogin(ctx, provider, user)
	if resErr != nil {
		ctx.JSON(resErr.StatusCode, resErr)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
