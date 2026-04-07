package post_handler

import (
	"errors"
	"strconv"

	// 通用响应
	"bluebell/internal/backfront"

	// 领域层 - Service 接口
	"bluebell/internal/domain/svcdomain"

	// DTO
	postreq "bluebell/internal/dto/request/post"
	postResp "bluebell/internal/dto/response/post"

	// 基础设施 - 参数校验
	"bluebell/internal/infrastructure/translate"

	// MQ 消息发布
	"bluebell/internal/service/mq"

	// 领域模型
	"bluebell/internal/model"

	// 错误处理
	"bluebell/pkg/errorx"

	// 中间件
	"bluebell/internal/middleware"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// Handler 帖子相关处理器
type Handler struct {
	postService svcdomain.PostService
	publisher   *mq.MQPublisher
}

// New 创建 Handler 实例
// 通过构造函数进行依赖注入
func New(postService svcdomain.PostService, publisher *mq.MQPublisher) *Handler {
	if postService == nil {
		panic("postService cannot be nil")
	}
	return &Handler{
		postService: postService,
		publisher:   publisher,
	}
}

// CreatePostHandler 创建帖子
// @Summary 创建帖子
// @Description 创建一个新的帖子
// @Tags 帖子相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body postreq.CreatePostRequest true "帖子参数"
// @Success 200 {object} backfront.ResponseData
// @Router /post [post]
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

	ctx, span := middleware.StartSpan(c, "bluebell/handler", "CreatePostHandler")
	defer span.End()

	postID, err := h.postService.CreatePost(ctx, p, userID.(int64))
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	// 异步发送审核消息
	if h.publisher != nil {
		auditMsg := &mq.AuditMessage{
			Title:    p.Title,
			Content:  p.Content,
			Type:     "post",
			AuthorID: userID.(int64),
		}
		if err := h.publisher.PublishAudit(ctx, auditMsg); err != nil {
			zap.L().Warn("publish audit message failed", zap.Error(err))
		}
	}

	// 异步发送搜索同步消息（索引）
	if h.publisher != nil {
		syncMsg := &mq.SyncMessage{
			PostID:      postID,
			AuthorID:    userID.(int64),
			CommunityID: p.CommunityID,
			PostTitle:   p.Title,
			Content:     p.Content,
			Status:      model.PostStatusPublished,
			CreatedAt:   time.Now().Format(time.RFC3339),
			Action:      "index",
		}
		if err := h.publisher.PublishSearch(ctx, syncMsg); err != nil {
			zap.L().Warn("publish search index message failed", zap.Error(err))
		}
	}

	backfront.ResponseSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情
// @Summary 获取帖子详情
// @Description 根据ID获取帖子详细信息
// @Tags 帖子相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "帖子ID"
// @Success 200 {object} backfront.ResponseData{data=postResp.DetailResponse}
// @Router /post/{id} [get]
func (h *Handler) GetPostDetailHandler(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	ctx, span := middleware.StartSpan(c, "bluebell/handler", "GetPostDetailHandler")
	defer span.End()

	data, err := h.postService.GetPostByID(ctx, postID)
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// GetPostListHandler 获取帖子列表
// @Summary 获取帖子列表
// @Description 分页获取帖子列表，可按时间或分数排序
// @Tags 帖子相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param page query int false "页码"
// @Param size query int false "每页数量"
// @Param order query string false "排序方式 (time/score)"
// @Param community_id query int false "社区ID"
// @Success 200 {object} backfront.ResponseData{data=[]postResp.DetailResponse}
// @Router /posts [get]
func (h *Handler) GetPostListHandler(c *gin.Context) {
	// 给予系统合理的默认设定值
	p := &postreq.PostListRequest{
		Page:  1,                 // 默认第一页
		Size:  10,                // 默认一页10条
		Order: postreq.OrderTime, // 默认按时间倒序
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

	ctx, span := middleware.StartSpan(c, "bluebell/handler", "GetPostListHandler")
	defer span.End()

	var data []*postResp.DetailResponse
	var err error

	if p.CommunityID == 0 {
		data, err = h.postService.GetPostList(ctx, p)
	} else {
		data, err = h.postService.GetCommunityPostList(ctx, p)
	}

	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, data)
}

// DeletePostHandler 删除帖子
// @Summary 删除帖子
// @Description 根据ID删除帖子（软删除）
// @Tags 帖子相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param id path int true "帖子ID"
// @Success 200 {object} backfront.ResponseData
// @Router /post/{id} [delete]
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

	ctx, span := middleware.StartSpan(c, "bluebell/handler", "DeletePostHandler")
	defer span.End()

	if err := h.postService.DeletePost(ctx, postID, userID.(int64)); err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	// 异步发送搜索同步消息（删除）
	if h.publisher != nil {
		syncMsg := map[string]interface{}{
			"post_id": strconv.FormatInt(postID, 10),
			"action":  "delete",
		}
		if err := h.publisher.PublishSearch(ctx, syncMsg); err != nil {
			zap.L().Warn("publish search sync message failed", zap.Error(err))
		}
	}

	backfront.ResponseSuccess(c, nil)
}

// PostVoteHandler 处理帖子投票请求
// @Summary 帖子投票
// @Description 为帖子点赞或踩
// @Tags 帖子相关
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body postreq.VoteRequest true "投票参数"
// @Success 200 {object} backfront.ResponseData
// @Router /vote [post]
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

	ctx, span := middleware.StartSpan(c, "bluebell/handler", "PostVoteHandler")
	defer span.End()

	if err := h.postService.VoteForPost(ctx, userID.(int64), p); err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	backfront.ResponseSuccess(c, nil)
}

// PostRemarkHandler 处理发表评论请求
func (h *Handler) PostRemarkHandler(c *gin.Context) {
	// 1. 获取当前用户 ID
	userID, exist := c.Get("UserIDKey")
	if !exist {
		backfront.HandleError(c, errorx.ErrNeedLogin)
		return
	}

	// 2. 绑定参数
	req := &postreq.RemarkRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
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

	// 3. 执行业务
	ctx, span := middleware.StartSpan(c, "bluebell/handler", "PostRemarkHandler")
	defer span.End()

	remarkID, err := h.postService.RemarkPost(ctx, req, userID.(int64))
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	// 异步发送评论审核消息
	if h.publisher != nil {
		auditMsg := &mq.AuditMessage{
			Content:  req.Content,
			Type:     "remark",
			RemarkID: remarkID,
			AuthorID: userID.(int64),
		}
		if err := h.publisher.PublishAudit(ctx, auditMsg); err != nil {
			zap.L().Warn("publish remark audit message failed", zap.Error(err))
		}
	}

	// 4. 响应
	backfront.ResponseSuccess(c, nil)
}

// GetPostRemarksHandler 获取帖子评论列表
func (h *Handler) GetPostRemarksHandler(c *gin.Context) {
	// 1. 获取参数
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		backfront.HandleError(c, errorx.ErrInvalidParam)
		return
	}

	// 2. 业务处理
	ctx, span := middleware.StartSpan(c, "bluebell/handler", "GetPostRemarksHandler")
	defer span.End()

	remarks, err := h.postService.GetPostRemarks(ctx, postID)
	if err != nil {
		middleware.RecordError(span, err)
		backfront.HandleError(c, err)
		return
	}

	// 3. 响应
	backfront.ResponseSuccess(c, remarks)
}
