package controller

import (
	"bluebell/logic"
	"bluebell/models"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
)

// CreatePostHandler 创建帖子
// @Summary 创建帖子
// @Description 创建帖子接口
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.Post true "创建帖子参数"
// @Success 200 {object} ResponseData
// @Router /post [post]
func CreatePostHandler(c *gin.Context) {
	// 1. 获取参数 (从gin.Context中取到当前发请求的userID)
	// c.Get(key) 从上下文中取值，其中用到的key应该是常量，不应该是字符串，避免出错
	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		HandleError(c, errorx.ErrNeedLogin)
		return
	}

	// 2. bind数据
	p := new(models.ParamPost)
	if err := c.ShouldBindJSON(p); err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}
	// 3. 将UserID填充到参数中
	p.UserID = userID.(int64)

	// 4. 调用逻辑层创建帖子

	// 3. 调用业务逻辑
	if _, err := logic.CreatePost(p); err != nil {
		HandleError(c, err)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情
// @Summary 获取帖子详情
// @Description 获取帖子详情接口
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path string true "帖子ID"
// @Success 200 {object} ResponseData{data=models.ApiPostDetail}
// @Router /post/{id} [get]
func GetPostDetailHandler(c *gin.Context) {
	// 1. 获取参数 (从URL中获取帖子id)
	postIDStr := c.Param("id")

	// 2. 字符串转int64
	postID, err := stringToInt64(postIDStr)
	if err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 3. 根据id取出帖子详情
	data, err := logic.GetPostByID(postID)
	if err != nil {
		HandleError(c, err)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, data)
}



// GetPostListHandler2 升级版帖子列表接口
// @Summary 获取帖子列表(新版)
// @Description 升级版分页获取帖子列表接口，支持按社区和排序规则查询
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.ParamPostList true "获取帖子列表参数"
// @Success 200 {object} ResponseData{data=[]models.ApiPostDetail}
// @Router /posts2 [get]
func GetPostListHandler(c *gin.Context) {
	// 根据前端传来的参数动态的获取帖子列表
	// 按创建时间或按照分数排序
	p := &models.ParamPostList{
		Page:  1,
		Size:  10,
		Order: models.OrderTime, // 默认按时间排序
	}

	if err := c.ShouldBindQuery(p); err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	data, err := logic.GetPostListNew(p) // 更新这里的逻辑
	if err != nil {
		HandleError(c, err)
		return
	}
	ResponseSuccess(c, data)
}