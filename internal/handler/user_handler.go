package handler

import (
	"bluebell/internal/dto/request"
	"bluebell/pkg/errorx"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// SignUpHandler 处理用户注册请求
func (h *Handlers) SignUpHandler(c *gin.Context) {
	p := new(request.SignUpRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			HandleError(c, errorx.ErrInvalidParam)
			return
		}
		ResponseErrorWithMsg(c, errorx.CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}

	if err := h.Services.User.SignUp(c.Request.Context(), p); err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, nil)
}

// LoginHandler 处理用户登录请求
func (h *Handlers) LoginHandler(c *gin.Context) {
	var p request.LoginRequest
	if err := c.ShouldBindJSON(&p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			HandleError(c, errorx.ErrInvalidParam)
			return
		}
		ResponseErrorWithMsg(c, errorx.CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}

	aToken, rToken, err := h.Services.User.Login(c.Request.Context(), &p)
	if err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, map[string]string{
		"access_token":  aToken,
		"refresh_token": rToken,
	})
}

// RefreshTokenHandler 刷新AccessToken
func (h *Handlers) RefreshTokenHandler(c *gin.Context) {
	rt := c.Query("refresh_token")
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "请求头缺少Auth Token")
		c.Abort()
		return
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "Token格式错误")
		c.Abort()
		return
	}
	aToken := parts[1]

	newAToken, newRToken, err := h.Services.User.RefreshToken(c.Request.Context(), aToken, rt)
	if err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
