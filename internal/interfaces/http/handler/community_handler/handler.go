package community_handler

import (
	"errors"
	"net/http"

	"bluebell/internal/application"
	communityreq "bluebell/internal/application/dto/request/community"
	communityResp "bluebell/internal/application/dto/response/community"
	"bluebell/internal/infrastructure/translate"
	"bluebell/internal/domain/entity"
	"bluebell/internal/interfaces/http/render"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var _ = communityResp.Response{} // 取消编译器对未使用导包的检查，保留给 Swagger 使用

// Handler 社区相关处理器
type Handler struct {
	communityService *application.CommunityService
}

// New 创建 Handler 实例
// 通过构造函数进行依赖注入
func New(communityService *application.CommunityService) *Handler {
	return &Handler{
		communityService: communityService,
	}
}


func (h *Handler) GetCommunityListHandler(c *gin.Context) {
	ctx := c.Request.Context()

	data, err := h.communityService.GetCommunityList(ctx)
	if err != nil {
		render.HandleError(c, err)
		return
	}
	render.HandleSuccess(c, data)
}

// GetCommunityDetailHandler 获取社区详情
func (h *Handler) GetCommunityDetailHandler(c *gin.Context) {
	p := &communityreq.CommunityDetailRequest{}
	if err := c.ShouldBindUri(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			c.JSON(http.StatusBadRequest, gin.H{"error": translate.RemoveTopStruct(translatedErrs)})
			return
		}
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()

	data, err := h.communityService.GetCommunityDetail(ctx, p.ID)
	if err != nil {
		render.HandleError(c, err)
		return
	}
	render.HandleSuccess(c, data)
}

func (h *Handler) CreateCommunityHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	p := &communityreq.CreateCommunityRequest{}
	if err := c.ShouldBindJSON(p); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			c.JSON(http.StatusBadRequest, gin.H{"error": translate.RemoveTopStruct(translatedErrs)})
			return
		}
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()

	if err := h.communityService.CreateCommunity(ctx, p.Name, p.Introduction, userID.(int64)); err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, nil)
}
