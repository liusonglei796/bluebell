package handler

import (
	"bluebell/internal/backfront"
	"bluebell/internal/dto/request"
	myvalidator "bluebell/internal/infrastructure/validator"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// GetCommunityListHandler 获取社区列表
func (h *CommunityHandler) GetCommunityListHandler(c *gin.Context) {
	data, err := h.communityService.GetCommunityList(c.Request.Context())
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// GetCommunityDetailHandler 获取社区详情
func (h *CommunityHandler) GetCommunityDetailHandler(c *gin.Context) {
	p := new(request.CommunityDetailRequest)

	if err := c.ShouldBindUri(p); err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			backfront.HandleError(c, errorx.ErrInvalidParam)
			return
		}
		backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam, myvalidator.RemoveTopStruct(errs.Translate(myvalidator.Trans)))
		return
	}

	data, err := h.communityService.GetCommunityDetail(c.Request.Context(), p.ID)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}
