package controller

import (
	"bluebell/dto/request"
	"bluebell/dto/response"
	"bluebell/logic"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreatePostHandler 创建帖子
// @Summary 创建帖子
// @Description 创建帖子接口
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body request.CreatePostRequest true "创建帖子参数"
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
	p := new(request.CreatePostRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}
	// 3. 将AuthorID填充到参数中
	p.AuthorID = userID.(int64)

	// 3. 调用业务逻辑创建帖子
	if _, err := logic.CreatePost(c.Request.Context(), p); err != nil {
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
// @Success 200 {object} ResponseData{data=response.PostDetailResponse}
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

// GetPostListHandler 获取帖子列表
// @Summary 获取帖子列表
// @Description 分页获取帖子列表接口，支持按社区和排序规则查询
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param page query int false "页码，默认1"
// @Param size query int false "每页数量，默认10"
// @Param order query string false "排序方式：time(时间)或score(分数)，默认time"
// @Param community_id query int false "社区ID，0表示所有社区"
// @Success 200 {object} ResponseData{data=[]response.PostDetailResponse}
// @Router /posts [get]
func GetPostListHandler(c *gin.Context) {
	// 1. 获取并校验参数
	p := &request.PostListRequest{
		Page:  1,
		Size:  10,
		Order: request.OrderTime, // 默认按时间排序
	}

	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler ShouldBindQuery failed", zap.Error(err))
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 2. 调度逻辑：根据 CommunityID 是否为 0，决定业务逻辑
	var data []*response.PostDetailResponse
	var err error

	if p.CommunityID == 0 {
		// 查询所有帖子
		data, err = logic.GetPostList(c.Request.Context(), p)
	} else {
		// 按社区查询帖子
		data, err = logic.GetCommunityPostList(c.Request.Context(), p)
	}

	if err != nil {
		HandleError(c, err)
		return
	}

	// 3. 返回响应
	ResponseSuccess(c, data)
}

// DeletePostHandler 删除帖子
// @Summary 删除帖子
// @Description 删除帖子接口（只有作者本人可以删除）
// @Tags 帖子相关
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path string true "帖子ID"
// @Success 200 {object} ResponseData
// @Router /post/{id} [delete]
func DeletePostHandler(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		HandleError(c, errorx.ErrNeedLogin)
		return
	}

	// 2. 获取帖子ID
	postIDStr := c.Param("id")
	postID, err := stringToInt64(postIDStr)
	if err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 3. 调用逻辑层删除帖子
	if err := logic.DeletePost(postID, userID.(int64)); err != nil {
		HandleError(c, err)
		return
	}

	// 4. 返回响应
	ResponseSuccess(c, nil)
}
