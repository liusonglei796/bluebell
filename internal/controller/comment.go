package controller

import (
	"strconv"

	commentreq "bluebell/internal/dto/request/comment"
	"bluebell/internal/model"
	"bluebell/internal/service"

	"github.com/gin-gonic/gin"
)

// CommentController 二级评论控制器
type CommentController struct {
	commentSvc *service.CommentService
}

// NewCommentController 创建评论控制器实例
func NewCommentController(commentSvc *service.CommentService) *CommentController {
	return &CommentController{commentSvc: commentSvc}
}

// CreateCommentHandler 创建评论/回复
func (h *CommentController) CreateCommentHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &commentreq.CreateCommentRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()
	data, err := h.commentSvc.CreateComment(ctx, p, userID.(int64))
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// GetCommentListHandler 获取帖子根评论列表
func (h *CommentController) GetCommentListHandler(c *gin.Context) {
	p := &commentreq.CommentListRequest{
		Page:  1,
		Size:  20,
		Order: "hot",
	}

	if err := c.ShouldBindQuery(p); err != nil {
		HandleError(c, model.ErrInvalidParam)
		return
	}

	var currentUID int64
	if uid, exist := c.Get("UserIDKey"); exist {
		currentUID = uid.(int64)
	}

	ctx := c.Request.Context()
	data, err := h.commentSvc.GetCommentList(ctx, p, currentUID)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// GetSubRepliesHandler 展开根评论下的子回复
func (h *CommentController) GetSubRepliesHandler(c *gin.Context) {
	p := &commentreq.SubReplyListRequest{
		Page: 1,
		Size: 20,
	}

	if err := c.ShouldBindQuery(p); err != nil {
		HandleError(c, model.ErrInvalidParam)
		return
	}

	var currentUID int64
	if uid, exist := c.Get("UserIDKey"); exist {
		currentUID = uid.(int64)
	}

	ctx := c.Request.Context()
	data, err := h.commentSvc.GetSubReplies(ctx, p, currentUID)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// DeleteCommentHandler 删除评论
func (h *CommentController) DeleteCommentHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	commentIDStr := c.Param("id")
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		HandleError(c, model.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()
	if err := h.commentSvc.DeleteComment(ctx, commentID, userID.(int64)); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}
