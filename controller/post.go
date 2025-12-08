package controller

import (
	"bluebell/logic"
	"bluebell/models"

	"go.uber.org/zap"

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
		zap.L().Error("user not login")
		CodeNotLogin := 0
		ResponseError(c, CodeNotLogin)
		return
	}

	// 2. bind数据
	p := new(models.ParamPost)
	if err := c.ShouldBindJSON(p); err != nil {
		zap.L().Error("create post with invalid param", zap.Error(err))
		ResponseError(c, CodeInvalidParam)
		return
	}
	p.AuthorID = userID.(int64)

	// 3. 保存到数据库
	if _, err := logic.CreatePost(p); err != nil {
		zap.L().Error("logic.CreatePost failed", zap.Error(err))
		ResponseError(c, CodeServerBusy)
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
		zap.L().Error("invalid post id", zap.String("post_id", postIDStr), zap.Error(err))
		ResponseError(c, CodeInvalidParam)
		return
	}

	// 3. 根据id取出帖子详情
	data, err := logic.GetPostByID(postID)
	if err != nil {
		zap.L().Error("logic.GetPostByID failed", zap.Error(err))
		ResponseError(c, CodeServerBusy)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, data)
}

// GetPostListHandler 获取帖子列表
// @Summary 获取帖子列表(旧版)
// @Description 分页获取帖子列表接口(旧版)
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param page query string false "页码"
// @Param size query string false "每页数量"
// @Success 200 {object} ResponseData{data=[]models.ApiPostDetail}
// @Router /posts [get]
func GetPostListHandler(c *gin.Context) {
	// 获取分页参数
	page, size := getPageInfo(c)
	// 构造参数对象
	p := &models.ParamPostList{
		Page: page,
		Size: size,
	}
	// 获取数据
	data, err := logic.GetPostList(p)
	if err != nil {
		zap.L().Error("logic.GetPostList failed", zap.Error(err))
		ResponseError(c, CodeServerBusy)
		return
	}
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
func GetPostListHandler2(c *gin.Context) {
	// 根据前端传来的参数动态的获取帖子列表
	// 按创建时间或按照分数排序
	p := &models.ParamPostList{
		Page:  1,
		Size:  10,
		Order: models.OrderTime, // 默认按时间排序
	}

	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler2 with invalid param", zap.Error(err))
		ResponseError(c, CodeInvalidParam)
		return
	}

	data, err := logic.GetPostListNew(p) // 更新这里的逻辑
	if err != nil {
		zap.L().Error("logic.GetPostListNew failed", zap.Error(err))
		ResponseError(c, CodeServerBusy)
		return
	}
	ResponseSuccess(c, data)
}