package controller

import (
	"strings"
	"time"

	userreq "bluebell/internal/dto/request/user"
	"bluebell/internal/model"
	"bluebell/internal/service"

	"github.com/gin-gonic/gin"
)

// UserController 用户相关控制器
type UserController struct {
	userSvc *service.UserService
}

// NewUserController 创建用户控制器实例
func NewUserController(userSvc *service.UserService) *UserController {
	return &UserController{
		userSvc: userSvc,
	}
}

// SignUpHandler 处理用户注册请求
func (h *UserController) SignUpHandler(c *gin.Context) {
	p := &userreq.SignUpRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()

	if err := h.userSvc.SignUp(ctx, p); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}

// LoginHandler 处理用户登录请求
func (h *UserController) LoginHandler(c *gin.Context) {
	p := &userreq.LoginRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()

	aToken, rToken, err := h.userSvc.Login(ctx, p)
	if err != nil {
		HandleError(c, err)
		return
	}

	// 获取用户信息（用于返回给前端）
	userInfo, err := h.userSvc.GetUserByUsername(ctx, p.Username)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, map[string]interface{}{
		"access_token":  aToken,
		"refresh_token": rToken,
		"user_id":       userInfo.UserID,
		"username":      userInfo.UserName,
		"role":          userInfo.Role,
	})
}

// LogoutHandler 处理用户登出请求
func (h *UserController) LogoutHandler(c *gin.Context) {
	_, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	jtiVal, _ := c.Get("JTIKey")
	expVal, _ := c.Get("TokenExpKey")

	jti, _ := jtiVal.(string)
	exp, _ := expVal.(time.Time)

	ctx := c.Request.Context()

	if err := h.userSvc.Logout(ctx, jti, exp); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}

// RefreshTokenHandler 处理刷新令牌请求
func (h *UserController) RefreshTokenHandler(c *gin.Context) {
	p := &userreq.RefreshTokenRequest{}
	_ = c.ShouldBind(p)

	authHeader := c.Request.Header.Get("Authorization")
	if p.Authorization == "" && authHeader != "" {
		p.Authorization = authHeader
	}

	// 如果 Body 中未传递 refresh_token，但 Header 传递了 Bearer Token
	if p.RefreshToken == "" && authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			p.RefreshToken = parts[1]
		} else {
			p.RefreshToken = authHeader
		}
	}

	if p.RefreshToken == "" {
		HandleError(c, model.ErrInvalidParam)
		return
	}
	ctx := c.Request.Context()

	newAToken, newRToken, err := h.userSvc.RefreshToken(ctx, p)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
