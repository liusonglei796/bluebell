package controller

import (
	"errors"
	"net/http"
	"strconv"

	"bluebell/internal/model"
	"bluebell/internal/service"
	"bluebell/internal/translate"

	postreq "bluebell/internal/dto/request/post"
	postResp "bluebell/internal/dto/response/post"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// PostController 帖子相关控制器
type PostController struct {
	postSvc *service.PostService
}

// NewPostController 创建帖子控制器实例
func NewPostController(postSvc *service.PostService) *PostController {
	return &PostController{postSvc: postSvc}
}

// bindJSON 解析请求体，参数错误时直接返回 400
func bindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		var errs validator.ValidationErrors
		if errors.As(err, &errs) {
			translatedErrs := errs.Translate(translate.Trans)
			c.JSON(http.StatusBadRequest, gin.H{
				"code":  CodeInvalidParam,
				"msg":   model.ErrInvalidParam.Error(),
				"error": translate.RemoveTopStruct(translatedErrs),
			})
			return false
		}
		HandleError(c, model.ErrInvalidParam)
		return false
	}
	return true
}

// CreatePostHandler 创建帖子
func (h *PostController) CreatePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &postreq.CreatePostRequest{}
	if !bindJSON(c, p) {
		return
	}

	ctx := c.Request.Context()
	_, err := h.postSvc.CreatePost(ctx, p, userID.(int64))
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情
func (h *PostController) GetPostDetailHandler(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		HandleError(c, model.ErrInvalidParam)
		return
	}

	var currentUID int64
	if uid, exist := c.Get("UserIDKey"); exist {
		currentUID = uid.(int64)
	}

	ctx := c.Request.Context()
	data, err := h.postSvc.GetPostByID(ctx, postID, currentUID)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// GetPostListHandler 获取帖子列表
func (h *PostController) GetPostListHandler(c *gin.Context) {
	p := &postreq.PostListRequest{
		Page:  1,
		Size:  10,
		Order: postreq.OrderTime,
	}

	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler ShouldBindQuery failed", zap.Error(err))
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

	if p.Size <= 0 || p.Size > 50 {
		p.Size = 10
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Order != postreq.OrderTime && p.Order != postreq.OrderScore {
		p.Order = postreq.OrderTime
	}

	var currentUID int64
	if uid, exist := c.Get("UserIDKey"); exist {
		currentUID = uid.(int64)
	}

	ctx := c.Request.Context()
	var data []*postResp.DetailResponse
	var err error

	if p.CommunityID == 0 {
		data, err = h.postSvc.GetPostList(ctx, p, currentUID)
	} else {
		data, err = h.postSvc.GetCommunityPostList(ctx, p, currentUID)
	}

	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}

// DeletePostHandler 删除帖子
func (h *PostController) DeletePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		HandleError(c, model.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()
	if err := h.postSvc.DeletePost(ctx, postID, userID.(int64)); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}

// PostVoteHandler 处理帖子投票请求
func (h *PostController) PostVoteHandler(c *gin.Context) {
	p := &postreq.VoteRequest{}
	if !bindJSON(c, p) {
		return
	}

	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	ctx := c.Request.Context()
	if err := h.postSvc.VoteForPost(ctx, userID.(int64), p); err != nil {
		if errors.Is(err, model.ErrVoteRepeated) {
			// 重复投票不记录成功指标，避免虚增
			HandleSuccess(c, nil)
			return
		}
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}


// PinPostHandler 置顶/取消置顶帖子 (仅管理员/版主)
func (h *PostController) PinPostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &postreq.PinPostRequest{}
	if !bindJSON(c, p) {
		return
	}

	communityIDStr := c.Query("community_id")
	communityID, _ := strconv.ParseInt(communityIDStr, 10, 64)

	ctx := c.Request.Context()
	if err := h.postSvc.PinPost(ctx, p.PostID, communityID, p.IsPinned == 1, userID.(int64)); err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, nil)
}

// GetFeedListHandler 获取用户关注者的发帖动态 Feed 流
func (h *PostController) GetFeedListHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		HandleError(c, model.ErrNeedLogin)
		return
	}

	p := &postreq.FeedListRequest{
		Size: 20,
	}
	_ = c.ShouldBindQuery(p)

	ctx := c.Request.Context()
	data, err := h.postSvc.GetTimelineFeed(ctx, userID.(int64), p.Cursor, p.Size)
	if err != nil {
		HandleError(c, err)
		return
	}

	HandleSuccess(c, data)
}
