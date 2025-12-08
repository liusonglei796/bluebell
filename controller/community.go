package controller

import (
	"errors"
	"strconv"

	"bluebell/logic"
	"bluebell/pkg/errno"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CommunityHandler 获取社区列表
// @Summary 获取社区列表
// @Description 获取所有社区的列表信息
// @Tags 社区相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} ResponseData{data=[]models.CommunityDetail}
// @Router /community [get]
// @Security ApiKeyAuth
func CommunityHandler(c *gin.Context) {
	// 1. 查询所有的社区 (community_id, community_name) 以列表的形式返回
	data, err := logic.GetCommunityList()
	if err != nil {
		zap.L().Error("logic.GetCommunityList() failed", zap.Error(err))
		ResponseError(c, CodeServerBusy) // 统一的错误响应封装
		return
	}
	
	// 2. 返回成功响应
	ResponseSuccess(c, data)
}

// CommunityDetailHandler 获取社区详情
// @Summary 获取社区详情
// @Description 根据社区ID获取社区的详细信息
// @Tags 社区相关
// @Accept application/json
// @Produce application/json
// @Param id path int true "社区ID"
// @Param Authorization header string true "Bearer 用户令牌"
// @Success 200 {object} ResponseData{data=models.CommunityDetail}
// @Failure 400 {object} ResponseData
// @Failure 500 {object} ResponseData
// @Router /community/{id} [get]
// @Security ApiKeyAuth
func CommunityDetailHandler(c *gin.Context) {
	// 1. 获取路径参数 (community_id)
	idStr := c.Param("id")
	
	// 2. 参数类型转换 (String -> Int64)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		// 如果转换失败，说明前端传来的不是数字，返回参数错误
		ResponseError(c, CodeInvalidParam)
		return
	}

	// 3. 调用 Logic 层获取详情
	data, err := logic.GetCommunityDetail(id)
	if err != nil {
		zap.L().Error("logic.GetCommunityDetail() failed", zap.Error(err))
		// 根据 error 类型返回不同的错误码
		if errors.Is(err, errno.ErrorInvalidID) {
			ResponseError(c, CodeInvalidParam) // 无效ID返回参数错误
		} else {
			ResponseError(c, CodeServerBusy) // 其他错误返回服务繁忙
		}
		return
	}

	// 4. 返回成功响应
	ResponseSuccess(c, data)
}