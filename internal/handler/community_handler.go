package handler

import (
	"bluebell/internal/response"
	"bluebell/pkg/errorx"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CommunityHandler 获取社区列表
func (h *Handlers) CommunityHandler(c *gin.Context) {
	data, err := h.Services.Community.GetCommunityList(c.Request.Context())
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.ResponseSuccess(c, data)
}

// CommunityHandlerByID 获取社区详情
func (h *Handlers) CommunityHandlerByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	data, err := h.Services.Community.GetCommunityDetail(c.Request.Context(), id)
	if err != nil {
		response.HandleError(c, err)
		return
	}

	response.ResponseSuccess(c, data)
}
