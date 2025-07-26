package controller

import (
	"net/http"

	"github.com/brmcode/user-auth-service/domain"
	"github.com/brmcode/user-auth-service/dto"
	"github.com/brmcode/user-auth-service/dto/response"
	"github.com/brmcode/user-auth-service/service"
	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService service.UserService
}

func NewUserController(userService service.UserService) *UserController {
	return &UserController{userService: userService}
}

func (u *UserController) GetUser(ctx *gin.Context) {
	username := ctx.Param("username")
	if domain.GetUsername(ctx) != username {
		ctx.JSON(http.StatusForbidden, response.NewError(403, "you are not authorized to access this resource"))
		return
	}

	res, errRes := u.userService.GetUser(username)
	if errRes != nil {
		ctx.JSON(errRes.Code, errRes)
		return
	}

	ctx.JSON(http.StatusOK, res)
}

func (u *UserController) CreateUser(ctx *gin.Context) {
	var req dto.CreateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	user, resErr := u.userService.CreateUser(req)
	if resErr != nil {
		ctx.JSON(resErr.Code, resErr)
		return
	}

	ctx.JSON(http.StatusCreated, user)
}

func (u *UserController) UpdateUser(ctx *gin.Context) {
	username := ctx.Param("username")

	var req dto.UpdateUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, err.Error()))
		return
	}

	if username != req.Username {
		ctx.JSON(http.StatusBadRequest, response.NewError(400, "username not match"))
		return
	}

	if domain.GetUsername(ctx) != username {
		ctx.JSON(http.StatusForbidden, response.NewError(403, "you are not allowed to update this user"))
		return
	}

	user, resErr := u.userService.UpdateUser(req)
	if resErr != nil {
		ctx.JSON(resErr.Code, resErr)
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

	if resErr := u.userService.DeleteUser(username); resErr != nil {
		ctx.JSON(resErr.Code, resErr)
		return
	}

	ctx.Status(http.StatusNoContent)
}
