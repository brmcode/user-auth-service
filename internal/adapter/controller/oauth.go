package controller

import (
	"net/http"

	"github.com/brmcode/user-auth-service/internal/adapter/validator"
	dto "github.com/brmcode/user-auth-service/internal/core/dto/common"
	"github.com/brmcode/user-auth-service/internal/core/dto/response"
	"github.com/brmcode/user-auth-service/internal/core/port"
	"github.com/gin-gonic/gin"
	"github.com/markbates/goth/gothic"
)

type OAuthController struct {
	validator   *validator.Validator
	authService port.AuthenticationService
}

func NewOAuthController(validator *validator.Validator, authService port.AuthenticationService) *OAuthController {
	return &OAuthController{validator: validator, authService: authService}
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

	// res, resErr := c.authService.OAuthLogin(ctx, provider, user)
	// if resErr != nil {
	// 	ctx.JSON(resErr.Code, resErr)
	// 	return
	// }

	ctx.JSON(http.StatusOK, gin.H{"user": &dto.User{
		Provider:       user.Provider,
		ProviderUserID: user.UserID,
		Email:          user.Email,
		Name:           user.Name,
		FirstName:      user.FirstName,
		LastName:       user.LastName,
		AvatarURL:      user.AvatarURL,
	}})
}

func (o *OAuthController) OAuthLogin(ctx *gin.Context) {
	var req dto.OAuthRegisterUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if err := o.validator.Validate(req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	res, resErr := o.authService.OAuthLogin(ctx, req)
	if resErr != nil {
		ctx.JSON(resErr.Code, resErr)
		return
	}

	ctx.JSON(http.StatusOK, res)
}
