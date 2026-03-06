package handler

import (
	"strconv"

	"bluebell/internal/backfront"
	"bluebell/internal/dto/request"
	dtoResp "bluebell/internal/dto/response"
	myvalidator "bluebell/internal/infrastructure/validator"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)


// CreatePostHandler 创建帖子
func (h *PostHandler) CreatePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	p := new(request.CreatePostRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			backfront.HandleError(c, errorx.ErrInvalidParam)
			return
		}
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam, myvalidator.RemoveTopStruct(errs.Translate(myvalidator.Trans)))
		return
	}
	if _, err := h.postService.CreatePost(c.Request.Context(), p, userID.(int64)); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情
func (h *PostHandler) GetPostDetailHandler(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	data, err := h.postService.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// GetPostListHandler 获取帖子列表
func (h *PostHandler) GetPostListHandler(c *gin.Context) {
	p := &request.PostListRequest{
		Page:  1,
		Size:  10,
		Order: request.OrderTime,
	}

	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler ShouldBindQuery failed", zap.Error(err))
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			backfront.HandleError(c, errorx.ErrInvalidParam)
			return
		}
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam, myvalidator.RemoveTopStruct(errs.Translate(myvalidator.Trans)))
		return
	}

	var data []*dtoResp.PostDetailResponse
	var err error

	if p.CommunityID == 0 {
		data, err = h.postService.GetPostList(c.Request.Context(), p)
	} else {
		data, err = h.postService.GetCommunityPostList(c.Request.Context(), p)
	}

	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// DeletePostHandler 删除帖子
func (h *PostHandler) DeletePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
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

	if err := h.postService.DeletePost(c.Request.Context(), postID, userID.(int64)); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}
