package controller

import (
	"bluebell/dao/mysql"
	"bluebell/logic"
	"bluebell/models"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// SignUpHandler 处理用户注册请求
// @Summary 用户注册
// @Description 用户注册接口
// @Tags 用户相关
// @Accept application/json
// @Produce application/json
// @Param object body models.ParamSignUp true "注册参数"
// @Success 200 {object} ResponseData
// @Router /signup [post]
func SignUpHandler(c *gin.Context) {
	// 1. 参数校验
	p := new(models.ParamSignUp)
	if err := c.ShouldBindJSON(p); err != nil {
		// 获取validator.ValidationErrors类型的errors
		// 为什么：validator 库返回的错误类型是 ValidationErrors，包含了具体的字段错误信息
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			// 非validator.ValidationErrors类型错误直接返回
			// 可能是 JSON 格式错误等
			ResponseError(c, CodeInvalidParam)
			return
		}
		// validator.ValidationErrors类型错误则进行翻译
		// 为什么：默认的错误信息是英文且包含结构体名，需要翻译成中文并去除结构体名前缀，提升用户体验
		ResponseErrorWithMsg(c, CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}

	// 2. 业务处理
	// 调用 logic 层进行具体的注册逻辑
	if err := logic.SignUp(p); err != nil {
		// 3. 错误处理
		// 根据 logic 层返回的错误类型，返回对应的业务状态码
		if errors.Is(err, mysql.ErrorUserExist) {
			ResponseError(c, CodeUserExist)
			return
		}
		// 其他未知错误，返回服务繁忙
		ResponseError(c, CodeServerBusy)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, nil)
}

// LoginHandler 处理用户登录请求
func LoginHandler(c *gin.Context) {
	var p models.ParamLogin
	// 1. 参数校验
	if err := c.ShouldBindJSON(&p); err != nil {
		// 获取validator.ValidationErrors类型的errors
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			// 非validator.ValidationErrors类型错误直接返回
			ResponseError(c, CodeInvalidParam)
			return
		}
		// validator.ValidationErrors类型错误则进行翻译
		ResponseErrorWithMsg(c, CodeInvalidParam, removeTopStruct(errs.Translate(trans)))
		return
	}

	// 2. 业务处理
	aToken, rToken, err := logic.Login(&p)
	if err != nil {
		// 记录错误日志，方便排查
		zap.L().Error("logic.Login failed", zap.Error(err))
		// 3. 错误处理
		if errors.Is(err, mysql.ErrorUserNotExist) {
			ResponseError(c, CodeUserNotExist)
			return
		}
		if errors.Is(err, mysql.ErrorInvalidPassword) {
			ResponseError(c, CodeInvalidPassword)
			return
		}
		ResponseError(c, CodeServerBusy)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, map[string]string{
		"access_token":  aToken,
		"refresh_token": rToken,
	})
}

// RefreshTokenHandler 刷新AccessToken
func RefreshTokenHandler(c *gin.Context) {
	rt := c.Query("refresh_token")
	// 客户端需要在 Header 中携带 Authorization: Bearer <access_token>
	authHeader := c.Request.Header.Get("Authorization")
	if authHeader == "" {
		ResponseErrorWithMsg(c, CodeInvalidToken, "请求头缺少Auth Token")
		c.Abort()
		return
	}
	// 按空格分割
	parts := strings.SplitN(authHeader, " ", 2)
	if !(len(parts) == 2 && parts[0] == "Bearer") {
		ResponseErrorWithMsg(c, CodeInvalidToken, "Token格式错误")
		c.Abort()
		return
	}
	aToken := parts[1]

	newAToken, newRToken, err := logic.RefreshToken(aToken, rt)
	if err != nil {
		zap.L().Error("logic.RefreshToken failed", zap.Error(err))
		ResponseError(c, CodeInvalidToken)
		return
	}

	ResponseSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
