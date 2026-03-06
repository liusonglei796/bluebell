package handler

import (
	"bluebell/internal/backfront"
	"bluebell/internal/dto/request"
	myvalidator "bluebell/internal/infrastructure/validator"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// PostVoteHandler 帖子投票
func (h *VoteHandler) PostVoteHandler(c *gin.Context) {
	p := new(request.VoteRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			backfront.HandleError(c, errorx.ErrInvalidParam)
			return
		}
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam, myvalidator.RemoveTopStruct(errs.Translate(myvalidator.Trans)))
		return
	}

	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	if err := h.voteService.VoteForPost(c.Request.Context(), userID.(int64), p); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}
