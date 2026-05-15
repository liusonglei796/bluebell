package search_handler

import (
	"bluebell/internal/domain/entity"
	"bluebell/internal/infrastructure/es"
	"bluebell/internal/interfaces/http/render"
	searchreq "bluebell/internal/interfaces/http/dto/request/search"

	"github.com/gin-gonic/gin"
)

// Handler 搜索相关处理器
type Handler struct {
	esClient *es.Client
}

// New 创建搜索处理器实例
func New(esClient *es.Client) *Handler {
	return &Handler{
		esClient: esClient,
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

	esReq := &es.SearchRequest{
		Keyword:  req.Keyword,
		Page:     req.Page,
		PageSize: req.PageSize,
	}

	resp, err := h.esClient.Search(ctx, esReq)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, resp)
}