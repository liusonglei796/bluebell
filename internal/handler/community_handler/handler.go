package community_handler

import (
	"errors"

	// 通用响应
	"bluebell/internal/backfront"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	communityreq "bluebell/internal/dto/request/community"
	communityResp "bluebell/internal/dto/response/community"

	// 基础设施 - 参数校验
	"bluebell/internal/infrastructure/translate"

	// 错误处理
	"bluebell/pkg/errorx"

	// 中间件
	"bluebell/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var _ = communityResp.Response{} // 取消编译器对未使用导包的检查，保留给 Swagger 使用

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
// @Summary 获取社区列表
// @Description 获取所有社区的列表
// @Tags 社区相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} backfront.ResponseData{data=[]communityResp.Response}
// @Router /community [get]
func (h *Handler) GetCommunityListHandler(c *gin.Context) {
	ctx, span := middleware.StartSpan(c, "bluebell/handler", "GetCommunityListHandler")
	defer span.End()

	data, err := h.communityService.GetCommunityList(ctx)
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}
	backfront.ResponseSuccess(c, data)
}

// GetCommunityDetailHandler 获取社区详情
// @Summary 获取社区详情
// @Description 根据社区ID获取社区详细信息
// @Tags 社区相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "社区ID"
// @Success 200 {object} backfront.ResponseData{data=communityResp.Response}
// @Router /community/{id} [get]
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

	ctx, span := middleware.StartSpan(c, "bluebell/handler", "GetCommunityDetailHandler")
	defer span.End()

	data, err := h.communityService.GetCommunityDetail(ctx, p.ID)
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}
	backfront.ResponseSuccess(c, data)
}

// CreateCommunityHandler 创建社区（仅管理员）
// @Summary 创建社区
// @Description 创建一个新的社区（仅管理员）
// @Tags 社区相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body communityreq.CreateCommunityRequest true "社区参数"
// @Success 200 {object} backfront.ResponseData
// @Router /community [post]
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

	ctx, span := middleware.StartSpan(c, "bluebell/handler", "CreateCommunityHandler")
	defer span.End()

	if err := h.communityService.CreateCommunity(ctx, p.Name, p.Introduction, userID.(int64)); err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}
