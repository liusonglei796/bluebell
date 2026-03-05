package handler

import (
	"bluebell/internal/dto/request"
	"bluebell/internal/response"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
)

// PostVoteHandler 帖子投票
func (h *Handlers) PostVoteHandler(c *gin.Context) {
	p := new(request.VoteRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		response.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		response.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	if err := h.Services.Vote.VoteForPost(c.Request.Context(), userID.(int64), p); err != nil {
		response.HandleError(c, err)
		return
	}

	response.ResponseSuccess(c, nil)
}
