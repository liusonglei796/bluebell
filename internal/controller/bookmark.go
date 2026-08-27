package controller

import (
	bookmarkreq "bluebell/internal/dto/request/bookmark"
	"bluebell/internal/model"
	"bluebell/internal/service"

	"github.com/gin-gonic/gin"
)

// BookmarkController 收藏夹控制器
type BookmarkController struct {
	bookmarkSvc *service.BookmarkService
}

// NewBookmarkController 创建收藏夹控制器实例
func NewBookmarkController(bookmarkSvc *service.BookmarkService) *BookmarkController {
	return &BookmarkController{bookmarkSvc: bookmarkSvc}
}

// CreateFolderHandler 创建自定义收藏夹
func (h *BookmarkController) CreateFolderHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &bookmarkreq.CreateFolderRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()
	data, err := h.bookmarkSvc.CreateFolder(ctx, userID.(int64), p)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// GetUserFoldersHandler 获取用户所有收藏夹
func (h *BookmarkController) GetUserFoldersHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	ctx := c.Request.Context()
	data, err := h.bookmarkSvc.GetUserFolders(ctx, userID.(int64))
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// AddBookmarkHandler 添加收藏
func (h *BookmarkController) AddBookmarkHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &bookmarkreq.AddBookmarkRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()
	if err := h.bookmarkSvc.AddBookmark(ctx, userID.(int64), p); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}

// RemoveBookmarkHandler 取消收藏
func (h *BookmarkController) RemoveBookmarkHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &bookmarkreq.RemoveBookmarkRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()
	if err := h.bookmarkSvc.RemoveBookmark(ctx, userID.(int64), p); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}

// GetBookmarkPostListHandler 获取收藏夹内帖子列表
func (h *BookmarkController) GetBookmarkPostListHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &bookmarkreq.BookmarkListRequest{Page: 1, Size: 20}
	_ = c.ShouldBindQuery(p)

	ctx := c.Request.Context()
	data, err := h.bookmarkSvc.GetBookmarkPostList(ctx, userID.(int64), p)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}
