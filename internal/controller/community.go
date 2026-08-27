package controller

import (
	"errors"
	"net/http"

	"bluebell/internal/dto/request/community"
	communityResp "bluebell/internal/dto/response/community"
	"bluebell/internal/model"
	"bluebell/internal/service"
	"bluebell/internal/translate"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var _ = communityResp.Response{} // 保留给 Swagger 使用

// CommunityController 社区相关控制器
type CommunityController struct {
	communitySvc *service.CommunityService
}

// NewCommunityController 创建社区控制器实例
func NewCommunityController(communitySvc *service.CommunityService) *CommunityController {
	return &CommunityController{
		communitySvc: communitySvc,
	}
}

// GetCommunityListHandler 获取社区列表
func (h *CommunityController) GetCommunityListHandler(c *gin.Context) {
	ctx := c.Request.Context()

	data, err := h.communitySvc.GetCommunityList(ctx)
	if err != nil {
		HandleError(c, err)
		return
	}
	HandleSuccess(c, data)
}

// GetCommunityDetailHandler 获取社区详情
func (h *CommunityController) GetCommunityDetailHandler(c *gin.Context) {
	p := &communityreq.CommunityDetailRequest{}
	if err := c.ShouldBindUri(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":  CodeInvalidParam,
				"msg":   model.ErrInvalidParam.Error(),
				"error": translate.RemoveTopStruct(translatedErrs),
			})
			return
		}
		HandleError(c, model.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()

	data, err := h.communitySvc.GetCommunityDetail(ctx, p.ID)
	if err != nil {
		HandleError(c, err)
		return
	}
	HandleSuccess(c, data)
}

// CreateCommunityHandler 创建社区
func (h *CommunityController) CreateCommunityHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &communityreq.CreateCommunityRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()

	if err := h.communitySvc.CreateCommunity(ctx, p.Name, p.Introduction, userID.(int64)); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}
