package handler

import (
	"net/http"

	"github.com/brmcode/user-auth-service/internal/adapter/google"
	dto "github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/common"
	"github.com/brmcode/user-auth-service/internal/adapter/http/handler/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

type OAuthHandler struct {
	authService port.AuthenticationService
	verifier    *google.IDTokenVerifier
}

func NewOAuthHandler(authService port.AuthenticationService, verifier *google.IDTokenVerifier) *OAuthHandler {
	return &OAuthHandler{authService: authService, verifier: verifier}
}

func (o *OAuthHandler) Begin(ctx *gin.Context) {
	provider := ctx.Param("provider")
	if provider == "" {
		ctx.JSON(http.StatusBadRequest, response.NewError(http.StatusBadRequest, "provider parameter is required"))
		return
	}
	// Set provider in context for CompleteUserAuth
	ctx.Request = gothic.GetContextWithProvider(ctx.Request, provider)
	gothic.BeginAuthHandler(ctx.Writer, ctx.Request)
}

func (o *OAuthHandler) Callback(ctx *gin.Context) {
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

	res := o.authService.OAuthLogin(ctx, provider, user)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (o *OAuthHandler) GoogleAuthMobile(ctx *gin.Context) {
	var req dto.GoogleAuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	payload, err := o.verifier.Verify(ctx.Request.Context(), req.IDToken)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, response.NewError(http.StatusUnauthorized, err.Error()))
		return
	}

	res := o.authService.GoogleAuthMobile(ctx, payload)
	if !res.Success {
		ctx.JSON(res.StatusCode, res)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
