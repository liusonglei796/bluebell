package handler

import (
	"bluebell/internal/backfront"
	"bluebell/internal/dto/request"
	myvalidator "bluebell/internal/infrastructure/validator"
	"bluebell/pkg/errorx"

	"errors"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// PostVoteHandler 帖子投票
func (h *voteHandlerStruct) PostVoteHandler(c *gin.Context) {
	p := &request.VoteRequest{}
	// 1. 尝试绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(p); err != nil {
		// 2. 判断是否为参数验证错误 (ValidationErrors)
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			// 如果是验证错误，进行翻译并去除结构体名前缀
			translatedErrs := errs.Translate(myvalidator.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				myvalidator.RemoveTopStruct(translatedErrs))
			return
		}

		// 3. 如果是其他类型的错误，返回通用参数错误
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 4. 调用 Service 层处理业务逻辑
	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	if err := h.voteService.VoteForPost(c.Request.Context(), userID.(int64), p); err != nil {
		backfront.HandleError(c, err)
		return
	}

	// 5. 业务处理成功
	backfront.ResponseSuccess(c, nil)
}
