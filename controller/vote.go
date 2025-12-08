package controller

import (
	"bluebell/logic"
	"bluebell/models"

	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
)

// PostVoteHandler 帖子投票
// @Summary 帖子投票
// @Description 为帖子投票，不允许重复投票
// @Tags 投票相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.ParamVoteData true "投票参数"
// @Success 200 {object} ResponseData
// @Router /vote [post]
func PostVoteHandler(c *gin.Context) {
	// 1. 参数校验
	p := new(models.ParamVoteData)
	if err := c.ShouldBindJSON(p); err != nil {
		zap.L().Error("PostVoteHandler with invalid param", zap.Error(err))
		ResponseError(c, CodeInvalidParam)
		return
	}

	// 2. 获取当前请求的用户ID
	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		zap.L().Error("user not login")
		CodeNotLogin := 0
		ResponseError(c, CodeNotLogin)
		return
	}

	// 3. 具体投票业务逻辑
	if err := logic.VoteForPost(userID.(int64), p); err != nil {
		zap.L().Error("logic.VoteForPost failed", zap.Error(err))
		ResponseError(c, CodeServerBusy)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, nil)
}