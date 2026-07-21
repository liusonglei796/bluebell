package user_handler

import (
	"errors"
	"net/http"

	// 领域层 - Service 接口
	"bluebell/internal/application"

	// DTO 请求
	userreq "bluebell/internal/interfaces/http/dto/request/user"

	// 基础设施 - 参数校验
	"bluebell/internal/infrastructure/translate"

	// 错误处理
	"bluebell/internal/domain/entity"
	"bluebell/internal/interfaces/http/render"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Handler 用户相关处理器
type Handler struct {
	userService application.UserService
}

// New 创建 Handler 实例
// 通过构造函数进行依赖注入
func New(userService application.UserService) *Handler {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": translate.RemoveTopStruct(translatedErrs)})
			return
		}
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()

	if err := h.userService.SignUp(ctx, p); err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, nil)
}

// LoginHandler 处理用户登录请求
func (h *Handler) LoginHandler(c *gin.Context) {
	p := &userreq.LoginRequest{}
	if err := c.ShouldBindJSON(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			c.JSON(http.StatusBadRequest, gin.H{"error": translate.RemoveTopStruct(translatedErrs)})
			return
		}
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()

	aToken, rToken, err := h.userService.Login(ctx, p)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	// 获取用户信息（用于返回给前端）
	userInfo, err := h.userService.GetUserByUsername(ctx, p.Username)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, map[string]interface{}{
		"access_token":  aToken,
		"refresh_token": rToken,
		"user_id":       userInfo.UserID,
		"username":      userInfo.UserName,
		"role":          userInfo.Role,
	})
}

// LogoutHandler 处理用户登出请求
func (h *Handler) LogoutHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	ctx := c.Request.Context()

	if err := h.userService.Logout(ctx, userID.(int64)); err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, nil)
}

// RefreshTokenHandler 处理刷新令牌请求
func (h *Handler) RefreshTokenHandler(c *gin.Context) {
	p := &userreq.RefreshTokenRequest{}
	if err := c.ShouldBind(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			c.JSON(http.StatusBadRequest, gin.H{"error": translate.RemoveTopStruct(translatedErrs)})
			return
		}
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()

	newAToken, newRToken, err := h.userService.RefreshToken(ctx, p)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
