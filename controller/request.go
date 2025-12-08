package controller

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
)

var ErrorUserNotLogin = errors.New("用户未登录")

// GetCurrentUser 从 Gin 上下文中获取当前登录的用户ID
func GetCurrentUser(c *gin.Context) (userID int64, err error) {
	// 1. 从上下文中获取值
	uid, ok := c.Get(CtxUserIDKey)
	if !ok {
		// 如果获取不到，说明JWT中间件没有成功设置，即用户未登录
		err = ErrorUserNotLogin
		return
	}

	// 2. 进行类型断言
	userID, ok = uid.(int64)
	if !ok {
		// 如果类型断言失败（理论上不应该发生，除非你在其他地方用同一个Key存了不同类型）
		err = ErrorUserNotLogin // 同样视为未登录或无效状态
		return
	}

	// 3. 成功返回
	return userID, nil
}

// stringToInt64 将字符串转换为int64
func stringToInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

// getPageInfo 从Gin上下文中获取分页参数
func getPageInfo(c *gin.Context) (page, size int64) {
	// 获取page参数,默认为1
	pageStr := c.Query("page")
	if pageStr == "" {
		page = 1
	} else {
		page, _ = strconv.ParseInt(pageStr, 10, 64)
		if page <= 0 {
			page = 1
		}
	}

	// 获取size参数,默认为10
	sizeStr := c.Query("size")
	if sizeStr == "" {
		size = 10
	} else {
		size, _ = strconv.ParseInt(sizeStr, 10, 64)
		if size <= 0 || size > 100 { // 限制最大为100
			size = 10
		}
	}

	return page, size
}
