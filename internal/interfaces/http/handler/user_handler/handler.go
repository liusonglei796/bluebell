package user_handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	// 领域层 - Service 接口
	"bluebell/internal/application"

	// DTO 请求
	userreq "bluebell/internal/application/dto/request/user"

	// 基础设施 - 参数校验
	"bluebell/internal/infrastructure/snowflake"
	"bluebell/internal/infrastructure/translate"

	// 错误处理
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/metrics"
	"bluebell/internal/interfaces/http/render"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ... (Handler struct and New function)

// GitHubLoginHandler 重定向到 GitHub 登录页面
func (h *Handler) GitHubLoginHandler(c *gin.Context) {
	// 实际项目中应使用随机 state 并存入 session/cookie
	// 此处简化处理
	url := "https://github.com/login/oauth/authorize?client_id=MOCK_CLIENT_ID&scope=user"
	c.Redirect(http.StatusTemporaryRedirect, url)
}

// GitHubCallbackHandler 处理 GitHub 回调
func (h *Handler) GitHubCallbackHandler(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		// Mock: 如果没有 code 或未配置 GitHub，返回模拟成功
		at, rt, _ := h.userService.SocialLogin(c.Request.Context(), "mock_github_id", "mock_github_user", "mock@github.com", "https://github.com/identicons/mock.png")
		render.HandleSuccess(c, map[string]interface{}{
			"access_token":  at,
			"refresh_token": rt,
			"username":      "mock_github_user",
		})
		return
	}

	// 1. 换取 Token (此处应调用基础设施层的 oauth2 config)
	// 2. 获取用户信息
	// 3. 调用 s.SocialLogin
	render.HandleSuccess(c, gin.H{"msg": "GitHub Login Success (Simulation)"})
}

// Handler 用户相关处理器
type Handler struct {
	userService *application.UserService
	uploadDir   string // 上传文件存储根目录
}

// New 创建 Handler 实例
// 通过构造函数进行依赖注入
// uploadDir 是上传文件存储根目录（来自 config.Upload.Dir）
func New(userService *application.UserService, uploadDir string) *Handler {
	return &Handler{
		userService: userService,
		uploadDir:   uploadDir,
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

	metrics.RecordSuccess(ctx, metrics.UsersRegistered)
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

	// 获取头像 URL
	avatarURL, _ := h.userService.GetAvatarURL(ctx, userInfo.UserID)

	metrics.RecordSuccess(ctx, metrics.UsersLoggedIn)
	render.HandleSuccess(c, map[string]interface{}{
		"access_token":  aToken,
		"refresh_token": rToken,
		"user_id":       userInfo.UserID,
		"username":      userInfo.UserName,
		"role":          userInfo.Role,
		"avatar_url":    avatarURL,
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

// allowedAvatarExts 允许的头像文件扩展名
var allowedAvatarExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
}

const avatarMaxSize = 10 << 20 // 10MB

// UploadAvatarHandler 处理头像上传请求（multipart/form-data）
// 被 authGroup 路由 POST /upload/avatar 调用（需 JWT 认证）
// 设计说明：
//   - Handler 层负责解析 HTTP 请求、校验文件类型和大小、保存文件到磁盘
//   - Application Service 层只负责更新 user_profile 表的 avatar_url
//   - 这样文件 I/O 集中在接口层，领域层和应用层保持纯逻辑
func (h *Handler) UploadAvatarHandler(c *gin.Context) {
	// 1. 从 JWT 上下文中获取当前用户 ID
	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	// 2. 限制请求体大小（在解析 multipart 之前设置）
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, avatarMaxSize)

	// 3. 解析 multipart form，获取上传文件
	// 使用 ParseMultipartForm 而非 FormFile，以保证设置了 MaxBytesReader 后能正确拦截超大文件
	if err := c.Request.ParseMultipartForm(avatarMaxSize); err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}
	defer file.Close()

	// 4. 校验文件扩展名
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedAvatarExts[ext] {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	// 5. 生成唯一文件名：snowflake ID + 扩展名
	// 使用 snowflake 生成唯一 ID 避免文件名冲突
	filename := fmt.Sprintf("%d%s", snowflake.GenID(), ext)

	// 6. 确保上传目录存在（avatars 子目录）
	// uploadDir 来自配置文件 config.yaml 的 upload.dir 字段
	avatarDir := filepath.Join(h.uploadDir, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		render.HandleError(c, entity.ErrServerBusy)
		return
	}

	// 7. 写入文件到磁盘
	dst := filepath.Join(avatarDir, filename)
	out, err := os.Create(dst)
	if err != nil {
		render.HandleError(c, entity.ErrServerBusy)
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		render.HandleError(c, entity.ErrServerBusy)
		return
	}

	// 8. 构造可访问的 URL 路径（相对路径，通过 Gin Static 提供服务）
	// 例：/uploads/avatars/123456789.jpg
	avatarURL := fmt.Sprintf("/uploads/avatars/%s", filename)

	// 9. 调用应用服务更新数据库中的 avatar_url
	ctx := c.Request.Context()
	if err := h.userService.UploadAvatar(ctx, userID.(int64), avatarURL); err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, map[string]string{
		"avatar_url": avatarURL,
	})
}

// RefreshTokenHandler 处理刷新令牌请求
// 注意：前端将 refresh_token 放在 Authorization header 中（而不是请求体），
// 不可使用 c.ShouldBind（它只处理 JSON/Form 请求体，不处理 header binding 标签）。
// 见 frontend/src/api/request.ts:34
func (h *Handler) RefreshTokenHandler(c *gin.Context) {
	p := &userreq.RefreshTokenRequest{
		RefreshToken: c.GetHeader("Authorization"),
	}

	if p.RefreshToken == "" {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()

	newAToken, newRToken, err := h.userService.RefreshToken(ctx, p)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	metrics.RecordSuccess(ctx, metrics.TokenRefreshes)
	render.HandleSuccess(c, map[string]string{
		"access_token":  newAToken,
		"refresh_token": newRToken,
	})
}
