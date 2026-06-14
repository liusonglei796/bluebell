package bookmark_handler

import (
	"bluebell/internal/application/port"
	bookmarkreq "bluebell/internal/application/dto/request/bookmark"
	"bluebell/internal/domain/entity"
	"bluebell/internal/interfaces/http/render"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	bookmarkService port.BookmarkService
}

func New(bookmarkService port.BookmarkService) *Handler {
	return &Handler{bookmarkService: bookmarkService}
}

func (h *Handler) CreateBookmarkHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	var req bookmarkreq.CreateBookmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		render.HandleError(c, err)
		return
	}

	ctx := c.Request.Context()
	if err := h.bookmarkService.CreateBookmark(ctx, userID.(int64), req.PostID); err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, gin.H{"message": "bookmarked"})
}

func (h *Handler) DeleteBookmarkHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()
	if err := h.bookmarkService.DeleteBookmark(ctx, userID.(int64), postID); err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, gin.H{"message": "bookmark deleted"})
}

func (h *Handler) GetUserBookmarksHandler(c *gin.Context) {
	userIDStr := c.Param("id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	var req bookmarkreq.BookmarkListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		render.HandleError(c, err)
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 20
	}

	ctx := c.Request.Context()
	data, err := h.bookmarkService.GetUserBookmarks(ctx, userID, req.Page, req.Size)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, data)
}

func (h *Handler) IsBookmarkedHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		c.JSON(http.StatusOK, gin.H{"code": 1000, "data": gin.H{"bookmarked": false, "count": 0}})
		return
	}

	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()
	data, err := h.bookmarkService.IsBookmarked(ctx, userID.(int64), postID)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, data)
}
