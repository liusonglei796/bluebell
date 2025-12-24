package controller

import (
	"bluebell/logic"
	"bluebell/models"
	"bluebell/pkg/errorx"

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
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 2. 获取当前请求的用户ID
	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		HandleError(c, errorx.ErrNeedLogin)
		return
	}

	// 3. 具体投票业务逻辑
	if err := logic.VoteForPost(userID.(int64), p); err != nil {
		HandleError(c, err)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, nil)
}