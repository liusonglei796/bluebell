package controller

import (
	"strconv"

	tagreq "bluebell/internal/dto/request/tag"
	"bluebell/internal/model"
	"bluebell/internal/service"

	"github.com/gin-gonic/gin"
)

// TagController 标签控制器
type TagController struct {
	tagSvc *service.TagService
}

// NewTagController 创建标签控制器实例
func NewTagController(tagSvc *service.TagService) *TagController {
	return &TagController{tagSvc: tagSvc}
}

// CreateTagHandler 社区创建标签
func (h *TagController) CreateTagHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &tagreq.CreateTagRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()
	data, err := h.tagSvc.CreateTag(ctx, userID.(int64), p)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// GetCommunityTagsHandler 获取社区下的标签列表
func (h *TagController) GetCommunityTagsHandler(c *gin.Context) {
	cidStr := c.Query("community_id")
	communityID, err := strconv.ParseInt(cidStr, 10, 64)
	if err != nil || communityID == 0 {
		HandleError(c, model.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()
	data, err := h.tagSvc.GetCommunityTags(ctx, communityID)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}
