package user_handler

import (
	"errors"

	// 通用响应
	"bluebell/internal/backfront"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// DTO 请求
	"bluebell/internal/dto/request/user"

	// 基础设施 - 参数校验
	"bluebell/internal/infrastructure/translate"

	// 错误处理
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Handler 用户相关处理器
type Handler struct {
	userService svcdomain.UserService
}

// New 创建 Handler 实例
// 通过构造函数进行依赖注入
func New(userService svcdomain.UserService) *Handler {
	if userService == nil {
		panic("userService cannot be nil")
	}
	return &Handler{
		userService: userService,
	}
}

// SignUpHandler 处理用户注册请求
func (h *Handler) SignUpHandler(c *gin.Context) {
	p := &userreq.SignUpRequest{}
	if err := c.ShouldBindJSON(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				translate.RemoveTopStruct(translatedErrs))
			return
		}
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	if err := h.userService.SignUp(c.Request.Context(), p); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}

// LoginHandler 处理用户登录请求
func (h *Handler) LoginHandler(c *gin.Context) {
	p := &userreq.LoginRequest{}
	if err := c.ShouldBindJSON(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				translate.RemoveTopStruct(translatedErrs))
			return
		}
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	aToken, rToken, err := h.userService.Login(c.Request.Context(), p)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, map[string]string{
		"access_token":  aToken,
		"refresh_token": rToken,
	})
}

// RefreshTokenHandler 处理刷新令牌请求
func (h *Handler) RefreshTokenHandler(c *gin.Context) {
	p := &userreq.RefreshTokenRequest{}
	if err := c.ShouldBind(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				translate.RemoveTopStruct(translatedErrs))
			return
		}
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	newAToken, newRToken, err := h.userService.RefreshToken(c.Request.Context(), p)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
