package search_handler

import (
	"bluebell/internal/application"
	"bluebell/internal/domain/entity"
	"bluebell/internal/interfaces/http/render"
	searchreq "bluebell/internal/application/dto/request/search"

	"github.com/gin-gonic/gin"
)

// Handler 搜索相关处理器
type Handler struct {
	postSvc *application.PostService
}

// New 创建搜索处理器实例
func New(postSvc *application.PostService) *Handler {
	return &Handler{
		postSvc: postSvc,
	}
}

// SearchHandler 处理全文搜索请求
func (h *Handler) SearchHandler(c *gin.Context) {
	req := &searchreq.SearchRequest{}
	if err := c.ShouldBindQuery(req); err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()

	resp, err := h.postSvc.SearchPosts(ctx, req.Keyword, req.Page, req.PageSize)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, resp)
}