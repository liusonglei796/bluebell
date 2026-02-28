package controller

import (
	"bluebell/logic"
	"bluebell/pkg/errorx"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CommunityController 社区控制器
type CommunityController struct {
	communityService *logic.CommunityService
}

// NewCommunityController 创建社区控制器实例
func NewCommunityController(communityService *logic.CommunityService) *CommunityController {
	return &CommunityController{communityService: communityService}
}

// CommunityHandler 获取社区列表
// @Summary 获取社区列表
// @Description 获取所有社区的列表信息
// @Tags 社区相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} ResponseData{data=[]models.Community}
// @Router /community [get]
// @Security ApiKeyAuth
func (cc *CommunityController) CommunityHandler(c *gin.Context) {
	data, err := cc.communityService.GetCommunityList(c.Request.Context())
	if err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, data)
}

// CommunityHandlerByID 获取社区详情
// @Summary 获取社区详情
// @Description 根据社区ID获取社区的详细信息
// @Tags 社区相关
// @Accept application/json
// @Produce application/json
// @Param id path int true "社区ID"
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} ResponseData{data=models.Community}
// @Failure 400 {object} ResponseData
// @Failure 500 {object} ResponseData
// @Router /community/{id} [get]
// @Security ApiKeyAuth
func (cc *CommunityController) CommunityHandlerByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	data, err := cc.communityService.GetCommunityDetail(c.Request.Context(), id)
	if err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, data)
}
