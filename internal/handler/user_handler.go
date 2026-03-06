package handler

import (
	"bluebell/internal/backfront"
	"bluebell/internal/dto/request"
	myvalidator "bluebell/internal/infrastructure/validator"
	"bluebell/pkg/errorx"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// SignUpHandler 处理用户注册请求
func (h *UserHandler) SignUpHandler(c *gin.Context) {
	p := new(request.SignUpRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			backfront.HandleError(c, errorx.ErrInvalidParam)
			return
		}
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam, myvalidator.RemoveTopStruct(errs.Translate(myvalidator.Trans)))
		return
	}

	if err := h.userService.SignUp(c.Request.Context(), p); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}

// LoginHandler 处理用户登录请求
func (h *UserHandler) LoginHandler(c *gin.Context) {
	var p request.LoginRequest
	if err := c.ShouldBindJSON(&p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			backfront.HandleError(c, errorx.ErrInvalidParam)
			return
		}
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam, myvalidator.RemoveTopStruct(errs.Translate(myvalidator.Trans)))
		return
	}

	aToken, rToken, err := h.userService.Login(c.Request.Context(), &p)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, map[string]string{
		"access_token":  aToken,
		"refresh_token": rToken,
	})
}

// RefreshTokenHandler 刷新AccessToken
func (h *UserHandler) RefreshTokenHandler(c *gin.Context) {
	var p request.RefreshTokenRequest
	if err := c.ShouldBind(&p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			backfront.HandleError(c, errorx.ErrInvalidParam)
			return
		}
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam, myvalidator.RemoveTopStruct(errs.Translate(myvalidator.Trans)))
		return
	}

	parts := strings.SplitN(p.AccessToken, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidToken, "Token格式错误")
		return
	}
	aToken := parts[1]

	newAToken, newRToken, err := h.userService.RefreshToken(c.Request.Context(), aToken, p.RefreshToken)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
