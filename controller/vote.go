package controller

import (
	"bluebell/dto/request"
	"bluebell/logic"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
)

// VoteController 投票控制器
type VoteController struct {
	voteService *logic.VoteService
}

// NewVoteController 创建投票控制器实例
func NewVoteController(voteService *logic.VoteService) *VoteController {
	return &VoteController{voteService: voteService}
}

// PostVoteHandler 帖子投票
// @Summary 帖子投票
// @Description 为帖子投票，不允许重复投票
// @Tags 投票相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body request.VoteRequest true "投票参数"
// @Success 200 {object} ResponseData
// @Router /vote [post]
func (vc *VoteController) PostVoteHandler(c *gin.Context) {
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

	if err := vc.voteService.VoteForPost(c.Request.Context(), userID.(int64), p); err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, nil)
}
