package controller

import (
	"strconv"

	relationreq "bluebell/internal/dto/request/relation"
	"bluebell/internal/model"
	"bluebell/internal/service"

	"github.com/gin-gonic/gin"
)

// RelationController 社交关系控制器
type RelationController struct {
	relationSvc *service.RelationService
}

// NewRelationController 创建关系控制器实例
func NewRelationController(relationSvc *service.RelationService) *RelationController {
	return &RelationController{relationSvc: relationSvc}
}

// FollowHandler 关注用户
func (h *RelationController) FollowHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &relationreq.FollowRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()
	var err error
	if p.Action == 1 {
		err = h.relationSvc.FollowUser(ctx, userID.(int64), p.TargetUserID)
	} else {
		err = h.relationSvc.UnfollowUser(ctx, userID.(int64), p.TargetUserID)
	}

	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}

// GetFollowingListHandler 获取关注列表
func (h *RelationController) GetFollowingListHandler(c *gin.Context) {
	targetUIDStr := c.Param("id")
	targetUID, err := strconv.ParseInt(targetUIDStr, 10, 64)
	if err != nil {
		HandleError(c, model.ErrInvalidParam)
		return
	}

	p := &relationreq.RelationListRequest{Page: 1, Size: 20}
	_ = c.ShouldBindQuery(p)

	var viewerUID int64
	if uid, exist := c.Get("UserIDKey"); exist {
		viewerUID = uid.(int64)
	}

	ctx := c.Request.Context()
	data, err := h.relationSvc.GetFollowingList(ctx, targetUID, viewerUID, p.Page, p.Size)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// GetFollowerListHandler 获取粉丝列表
func (h *RelationController) GetFollowerListHandler(c *gin.Context) {
	targetUIDStr := c.Param("id")
	targetUID, err := strconv.ParseInt(targetUIDStr, 10, 64)
	if err != nil {
		HandleError(c, model.ErrInvalidParam)
		return
	}

	p := &relationreq.RelationListRequest{Page: 1, Size: 20}
	_ = c.ShouldBindQuery(p)

	var viewerUID int64
	if uid, exist := c.Get("UserIDKey"); exist {
		viewerUID = uid.(int64)
	}

	ctx := c.Request.Context()
	data, err := h.relationSvc.GetFollowerList(ctx, targetUID, viewerUID, p.Page, p.Size)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}
