package handler

import (
	"strconv"

	"bluebell/internal/dto/request"
	dtoResp "bluebell/internal/dto/response"
	"bluebell/internal/backfront"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreatePostHandler 创建帖子
func (h *Handlers) CreatePostHandler(c *gin.Context) {
	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	p := new(request.CreatePostRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}
	p.AuthorID = strconv.FormatInt(userID.(int64), 10)

	if _, err := h.Services.Post.CreatePost(c.Request.Context(), p); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情
func (h *Handlers) GetPostDetailHandler(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	data, err := h.Services.Post.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// GetPostListHandler 获取帖子列表
func (h *Handlers) GetPostListHandler(c *gin.Context) {
	p := &request.PostListRequest{
		Page:  1,
		Size:  10,
		Order: request.OrderTime,
	}

	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler ShouldBindQuery failed", zap.Error(err))
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	var data []*dtoResp.PostDetailResponse
	var err error

	if p.CommunityID == 0 {
		data, err = h.Services.Post.GetPostList(c.Request.Context(), p)
	} else {
		data, err = h.Services.Post.GetCommunityPostList(c.Request.Context(), p)
	}

	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// DeletePostHandler 删除帖子
func (h *Handlers) DeletePostHandler(c *gin.Context) {
	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	if err := h.Services.Post.DeletePost(c.Request.Context(), postID, userID.(int64)); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}



