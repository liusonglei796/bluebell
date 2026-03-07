package community_handler

import (
	"errors"

	// 通用响应
	"bluebell/internal/backfront"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// DTO 请求
	communityreq "bluebell/internal/dto/request/community"

	// 基础设施 - 参数校验
	"bluebell/internal/infrastructure/translate"

	// 错误处理
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Handler 社区相关处理器
type Handler struct {
	communityService svcdomain.CommunityService
}

// New 创建 Handler 实例
// 通过构造函数进行依赖注入
func New(communityService svcdomain.CommunityService) *Handler {
	if communityService == nil {
		panic("communityService cannot be nil")
	}
	return &Handler{
		communityService: communityService,
	}
}

// GetCommunityListHandler 获取社区列表
func (h *Handler) GetCommunityListHandler(c *gin.Context) {
	data, err := h.communityService.GetCommunityList(c.Request.Context())
	if err != nil {
		backfront.HandleError(c, err)
		return
	}
	backfront.ResponseSuccess(c, data)
}

// GetCommunityDetailHandler 获取社区详情
func (h *Handler) GetCommunityDetailHandler(c *gin.Context) {
	p := &communityreq.CommunityDetailRequest{}
	if err := c.ShouldBindUri(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				translate.RemoveTopStruct(translatedErrs))
			return
		}
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	data, err := h.communityService.GetCommunityDetail(c.Request.Context(), p.ID)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}
	backfront.ResponseSuccess(c, data)
}

// CreateCommunityHandler 创建社区（仅管理员）
func (h *Handler) CreateCommunityHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	p := &communityreq.CreateCommunityRequest{}
	if err := c.ShouldBindJSON(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			backfront.HandleErrorWithMsg(c, errorx.CodeInvalidParam,
				translate.RemoveTopStruct(translatedErrs))
			return
		}
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	if err := h.communityService.CreateCommunity(c.Request.Context(), p.Name, p.Introduction, userID.(int64)); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}
