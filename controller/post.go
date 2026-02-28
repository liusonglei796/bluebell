package controller

import (
	"bluebell/dto/request"
	"bluebell/dto/response"
	"bluebell/logic"
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PostController 帖子控制器
type PostController struct {
	postService *logic.PostService
}

// NewPostController 创建帖子控制器实例
func NewPostController(postService *logic.PostService) *PostController {
	return &PostController{postService: postService}
}

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
func (pc *PostController) CreatePostHandler(c *gin.Context) {
	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		HandleError(c, errorx.ErrNeedLogin)
		return
	}

	p := new(request.CreatePostRequest)
	if err := c.ShouldBindJSON(p); err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}
	p.AuthorID = userID.(int64)

	if _, err := pc.postService.CreatePost(c.Request.Context(), p); err != nil {
		HandleError(c, err)
		return
	}

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
func (pc *PostController) GetPostDetailHandler(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := stringToInt64(postIDStr)
	if err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	data, err := pc.postService.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		HandleError(c, err)
		return
	}

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
func (pc *PostController) GetPostListHandler(c *gin.Context) {
	p := &request.PostListRequest{
		Page:  1,
		Size:  10,
		Order: request.OrderTime,
	}

	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler ShouldBindQuery failed", zap.Error(err))
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	var data []*response.PostDetailResponse
	var err error

	if p.CommunityID == 0 {
		data, err = pc.postService.GetPostList(c.Request.Context(), p)
	} else {
		data, err = pc.postService.GetCommunityPostList(c.Request.Context(), p)
	}

	if err != nil {
		HandleError(c, err)
		return
	}

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
func (pc *PostController) DeletePostHandler(c *gin.Context) {
	userID, exist := c.Get(CtxUserIDKey)
	if !exist {
		HandleError(c, errorx.ErrNeedLogin)
		return
	}

	postIDStr := c.Param("id")
	postID, err := stringToInt64(postIDStr)
	if err != nil {
		HandleError(c, errorx.ErrInvalidParam)
		return
	}

	if err := pc.postService.DeletePost(c.Request.Context(), postID, userID.(int64)); err != nil {
		HandleError(c, err)
		return
	}

	ResponseSuccess(c, nil)
}
