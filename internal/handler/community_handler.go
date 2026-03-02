package handler

import (
	"bluebell/pkg/errorx"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CommunityHandler 获取社区列表
func (h *Handlers) CommunityHandler(c *gin.Context) {
	data, err := h.Services.Community.GetCommunityList(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, data)
}

// CommunityHandlerByID 获取社区详情
func (h *Handlers) CommunityHandlerByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	data, err := h.Services.Community.GetCommunityDetail(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, data)
}
