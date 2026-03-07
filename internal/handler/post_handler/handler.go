package post_handler

import (
	"errors"
	"strconv"

	// 通用响应
	"bluebell/internal/backfront"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// DTO
	"bluebell/internal/dto/request/post"
	"bluebell/internal/dto/response/post"

	// 基础设施 - 参数校验
	"bluebell/internal/infrastructure/translate"

	// 错误处理
	"bluebell/pkg/errorx"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// Handler 帖子相关处理器
type Handler struct {
	postService svcdomain.PostService
}

// New 创建 Handler 实例
// 通过构造函数进行依赖注入
func New(postService svcdomain.PostService) *Handler {
	if postService == nil {
		panic("postService cannot be nil")
	}
	return &Handler{
		postService: postService,
	}
}

// CreatePostHandler 创建帖子
func (h *Handler) CreatePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	p := &postreq.CreatePostRequest{}
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

	if err := h.postService.CreatePost(c.Request.Context(), p, userID.(int64)); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情
func (h *Handler) GetPostDetailHandler(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}
	data, err := h.postService.GetPostByID(c.Request.Context(), postID)
	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// GetPostListHandler 获取帖子列表
func (h *Handler) GetPostListHandler(c *gin.Context) {
	// 给予系统合理的默认设定值
	p := &postreq.PostListRequest{
		Page:  1,                  // 默认第一页
		Size:  10,                 // 默认一页10条
		Order: postreq.OrderTime,  // 默认按时间倒序
	}

	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("GetPostListHandler ShouldBindQuery failed", zap.Error(err))
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

	// 用户传的值也可能并不合理，需要系统进行兜底强制干预（限制最大查阅量，防止恶意请求拖垮系统）
	if p.Size <= 0 || p.Size > 50 {
		p.Size = 10
	}
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.Order != postreq.OrderTime && p.Order != postreq.OrderScore {
		p.Order = postreq.OrderTime // 如果前端乱传了非法的一个order字符，强制使用默认的时间排序
	}

	var data []*postResp.DetailResponse
	var err error

	if p.CommunityID == 0 {
		data, err = h.postService.GetPostList(c.Request.Context(), p)
	} else {
		data, err = h.postService.GetCommunityPostList(c.Request.Context(), p)
	}

	if err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// DeletePostHandler 删除帖子
func (h *Handler) DeletePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	if err := h.postService.DeletePost(c.Request.Context(), postID, userID.(int64)); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}

// PostVoteHandler 处理帖子投票请求
func (h *Handler) PostVoteHandler(c *gin.Context) {
	p := &postreq.VoteRequest{}
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

	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	if err := h.postService.VoteForPost(c.Request.Context(), userID.(int64), p); err != nil {
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}
