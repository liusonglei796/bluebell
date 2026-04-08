package user_handler

import (
	"errors"

	// 通用响应
	"bluebell/internal/backfront"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// DTO 请求
	userreq "bluebell/internal/dto/request/user"

	// 基础设施 - 参数校验
	"bluebell/internal/infrastructure/translate"

	// 错误处理
	"bluebell/pkg/errorx"

	// 中间件
	"bluebell/internal/middleware"

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
// @Summary 用户注册
// @Description 注册新用户
// @Tags 用户相关
// @Accept json
// @Produce json
// @Param object body userreq.SignUpRequest true "注册参数"
// @Success 200 {object} backfront.ResponseData
// @Router /signup [post]
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

	ctx, span := middleware.StartSpan(c, "SignUpHandler")
	defer span.End()

	if err := h.userService.SignUp(ctx, p); err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}

// LoginHandler 处理用户登录请求
// @Summary 用户登录
// @Description 用户登录并获取令牌
// @Tags 用户相关
// @Accept json
// @Produce json
// @Param object body userreq.LoginRequest true "登录参数"
// @Success 200 {object} backfront.ResponseData
// @Router /login [post]
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

	ctx, span := middleware.StartSpan(c, "LoginHandler")
	defer span.End()

	aToken, rToken, err := h.userService.Login(ctx, p)
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	// 获取用户信息（用于返回给前端）
	userInfo, err := h.userService.GetUserByUsername(ctx, p.Username)
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, map[string]interface{}{
		"access_token":  aToken,
		"refresh_token": rToken,
		"user_id":       userInfo.UserID,
		"username":      userInfo.UserName,
		"role":          userInfo.Role,
	})
}

// RefreshTokenHandler 处理刷新令牌请求
// @Summary 刷新令牌
// @Description 使用刷新令牌获取新的访问令牌
// @Tags 用户相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param refresh_token query string true "刷新令牌"
// @Success 200 {object} backfront.ResponseData
// @Router /refresh_token [post]
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

	ctx, span := middleware.StartSpan(c, "RefreshTokenHandler")
	defer span.End()

	newAToken, newRToken, err := h.userService.RefreshToken(ctx, p)
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
