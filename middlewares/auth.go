package middlewares

import (
	"bluebell/controller"
	"bluebell/dao/redis"
	"bluebell/pkg/errorx"
	"bluebell/pkg/jwt"
	"strings"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware 基于JWT的认证中间件
func JWTAuthMiddleware() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 1. 获取 Authorization header
		// 客户端携带 Token 的三种方式 1.放在请求头 2.放在请求体 3.放在URI
		// 这里假设 Token 放在 Header 的 Authorization 中，并使用 Bearer 开头
		// Authorization: Bearer xxxxxxx.xxx.xxx
		authHeader := c.Request.Header.Get("Authorization")
		if authHeader == "" {
			controller.ResponseError(c, errorx.ErrNeedLogin)
			c.Abort()
			return
		}

		// 2. 按空格分割
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			controller.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 3. 解析 Token
		mc, err := jwt.ParseToken(parts[1])
		if err != nil {
			controller.ResponseError(c, errorx.ErrInvalidToken)
			c.Abort()
			return
		}

		// 4. 单点登录校验：检查 Redis 中的 Token 是否与当前请求的 Token 一致
		redisToken, err := redis.GetUserAccessToken(mc.UserID)
		if err != nil {
			// Redis 查询失败（可能是 Redis 挂了，或者 Key 不存在/过期）
			// 如果 Key 不存在，说明用户未登录或 Token 已过期
			controller.ResponseError(c, errorx.ErrNeedLogin)
			c.Abort()
			return
		}

		// 如果 Redis 中的 Token 与请求的 Token 不一致，说明账号已在其他设备登录
		if parts[1] != redisToken {
			controller.ResponseErrorWithMsg(c, errorx.CodeInvalidToken, "账号已在其他设备登录")
			c.Abort()
			return
		}

		// 5. 将当前请求的 userID 信息保存到请求的上下文 c 上
		// 后续的处理函数可以用过 c.Get(controller.CtxUserIDKey) 来获取当前请求的用户信息
		c.Set(controller.CtxUserIDKey, mc.UserID)
		c.Next()
	}
}
