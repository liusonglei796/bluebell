package post_handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"bluebell/internal/application"
	"bluebell/internal/infrastructure/mq"
	"bluebell/internal/infrastructure/persistence/mysql/model"
	"bluebell/internal/infrastructure/translate"
	"bluebell/internal/interfaces/http/dto/request/post"
	"bluebell/internal/interfaces/http/dto/response/post"
	"bluebell/internal/domain/entity"
	"bluebell/internal/interfaces/http/render"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type Handler struct {
	postService application.PostService
	publisher   *mq.Publisher
}

func New(postService application.PostService, publisher *mq.Publisher) *Handler {
	return &Handler{postService: postService, publisher: publisher}
}


func (h *Handler) CreatePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	p := &postreq.CreatePostRequest{}
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
	postID, err := h.postService.CreatePost(ctx, p, userID.(int64))
	if err != nil {
		render.HandleError(c, err)
		return
	}

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

	render.HandleSuccess(c, nil)
}

// GetPostDetailHandler 获取帖子详情
func (h *Handler) GetPostDetailHandler(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()
	data, err := h.postService.GetPostByID(ctx, postID)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, data)
}

// GetPostListHandler 获取帖子列表
func (h *Handler) GetPostListHandler(c *gin.Context) {
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
			c.JSON(http.StatusBadRequest, gin.H{"error": translate.RemoveTopStruct(translatedErrs)})
			return
		}
		render.HandleError(c, entity.ErrInvalidParam)
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

	ctx := c.Request.Context()
	var data []*postResp.DetailResponse
	var err error

	if p.CommunityID == 0 {
		data, err = h.postService.GetPostList(ctx, p)
	} else {
		data, err = h.postService.GetCommunityPostList(ctx, p)
	}

	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, data)
}

// DeletePostHandler 删除帖子
func (h *Handler) DeletePostHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()
	if err := h.postService.DeletePost(ctx, postID, userID.(int64)); err != nil {
		render.HandleError(c, err)
		return
	}

	if h.publisher != nil {
		syncMsg := map[string]interface{}{
			"post_id": strconv.FormatInt(postID, 10),
			"action":  "delete",
		}
		if err := h.publisher.PublishSearch(ctx, syncMsg); err != nil {
			zap.L().Warn("publish search sync message failed", zap.Error(err))
		}
	}

	render.HandleSuccess(c, nil)
}

// PostVoteHandler 处理帖子投票请求
func (h *Handler) PostVoteHandler(c *gin.Context) {
	p := &postreq.VoteRequest{}
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

	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	ctx := c.Request.Context()
	if err := h.postService.VoteForPost(ctx, userID.(int64), p); err != nil {
		if errors.Is(err, entity.ErrVoteRepeated) {
			// 重复投票不记录成功指标，避免虚增
			render.HandleSuccess(c, nil)
			return
		}
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, nil)
}

// PostRemarkHandler 处理发表评论请求
func (h *Handler) PostRemarkHandler(c *gin.Context) {
	userID, exist := c.Get("UserIDKey")
	if !exist {
		render.HandleError(c, entity.ErrNeedLogin)
		return
	}

	req := &postreq.RemarkRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
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
	_, err := h.postService.RemarkPost(ctx, req, userID.(int64))
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, nil)
}

// GetPostRemarksHandler 获取帖子评论列表
func (h *Handler) GetPostRemarksHandler(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		render.HandleError(c, entity.ErrInvalidParam)
		return
	}

	ctx := c.Request.Context()
	remarks, err := h.postService.GetPostRemarks(ctx, postID)
	if err != nil {
		render.HandleError(c, err)
		return
	}

	render.HandleSuccess(c, remarks)
}
