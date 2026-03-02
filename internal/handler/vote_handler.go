package handler

import (
	"bluebell/internal/dto/request"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
)

// PostVoteHandler 帖子投票
func (h *Handlers) PostVoteHandler(c *gin.Context) {
	p := new(request.VoteRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		HandleError(c, errorx.ErrNeedLogin)
		return
	}

	if err := h.Services.Vote.VoteForPost(c.Request.Context(), userID.(int64), p); err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, nil)
}
