package handler

import (
	"bluebell/internal/backfront"
	"bluebell/internal/dto/request"
	myvalidator "bluebell/internal/infrastructure/validator"
	"bluebell/pkg/errorx"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func (h *userHandlerStruct) SignUpHandler(c *gin.Context) {
	p := &request.SignUpRequest{}
	// 1. 尝试绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(p); err != nil {
		// 2. 判断是否为参数验证错误 (ValidationErrors)
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			// 如果是验证错误，进行翻译并去除结构体名前缀
			translatedErrs := errs.Translate(myvalidator.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				myvalidator.RemoveTopStruct(translatedErrs))
			return
		}

		// 3. 如果是其他类型的错误（如 JSON 格式不正确），返回通用参数错误
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 4. 调用 Service 层处理业务逻辑
	if err := h.userService.SignUp(c.Request.Context(), p); err != nil {
		backfront.HandleError(c, err)
		return
	}

	// 5. 业务处理成功
	backfront.ResponseSuccess(c, nil)
}

// LoginHandler 处理用户登录请求
func (h *userHandlerStruct) LoginHandler(c *gin.Context) {
	p := &request.LoginRequest{}
	// 1. 尝试绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(p); err != nil {
		// 2. 判断是否为参数验证错误 (ValidationErrors)
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			// 如果是验证错误，进行翻译并去除结构体名前缀
			translatedErrs := errs.Translate(myvalidator.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				myvalidator.RemoveTopStruct(translatedErrs))
			return
		}

		// 3. 如果是其他类型的错误（如 JSON 格式不正确），返回通用参数错误
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 4. 调用 Service 层处理业务逻辑
	aToken, rToken, err := h.userService.Login(c.Request.Context(), p)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	// 5. 业务处理成功
	backfront.ResponseSuccess(c, map[string]string{
		"access_token":  aToken,
		"refresh_token": rToken,
	})
}

// RefreshTokenHandler 刷新AccessToken
func (h *userHandlerStruct) RefreshTokenHandler(c *gin.Context) {
	p := &request.RefreshTokenRequest{}
	// 1. 尝试绑定数据
	if err := c.ShouldBind(p); err != nil {
		// 2. 判断是否为参数验证错误 (ValidationErrors)
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			// 如果是验证错误，进行翻译并去除结构体名前缀
			translatedErrs := errs.Translate(myvalidator.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				myvalidator.RemoveTopStruct(translatedErrs))
			return
		}

		// 3. 如果是其他类型的错误，返回通用参数错误
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	parts := strings.SplitN(p.AccessToken, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidToken, "Token格式错误")
		return
	}
	aToken := parts[1]

	// 4. 调用 Service 层处理业务逻辑
	newAToken, newRToken, err := h.userService.RefreshToken(c.Request.Context(), aToken, p.RefreshToken)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	// 5. 业务处理成功
	backfront.ResponseSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
