package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	commentreq "bluebell/internal/dto/request/comment"
	commentresp "bluebell/internal/dto/response/comment"
	"bluebell/internal/model"
	"bluebell/internal/mq"
	"bluebell/internal/snowflake"
	"bluebell/pkg/event"

	"go.uber.org/zap"
)

// CommentService 评论业务服务
type CommentService struct {
	commentDao *mysql.CommentDao
	postDao    *mysql.PostDao
	userDao    *mysql.UserDao
	cache      *redis.PostCache
	eventBus   *mq.EventBus
}

// NewCommentService 创建评论服务实例
func NewCommentService(
	commentDao *mysql.CommentDao,
	postDao *mysql.PostDao,
	userDao *mysql.UserDao,
	cache *redis.PostCache,
	eventBus *mq.EventBus,
) *CommentService {
	return &CommentService{
		commentDao: commentDao,
		postDao:    postDao,
		userDao:    userDao,
		cache:      cache,
		eventBus:   eventBus,
	}
}

// CreateComment 创建二级评论/子回复
func (s *CommentService) CreateComment(ctx context.Context, p *commentreq.CreateCommentRequest, authorID int64) (*commentresp.CommentResponse, error) {
	// 1. 验证帖子存在
	post, err := s.postDao.GetPostByID(ctx, p.PostID)
	if err != nil || post == nil {
		return nil, model.ErrNotFound
	}

	// 2. 短时防重复提交指纹拦截
	fingerprint := fmt.Sprintf("%d:/api/v1/comment/create:%d:%d:%s", authorID, p.PostID, p.ParentID, p.Content)
	if err := s.cache.CheckDuplicateSubmit(ctx, fingerprint, 3*time.Second); err != nil {
		return nil, err
	}

	commentID := snowflake.GenID()

	// 3. 确定被回复人
	replyToUID := p.ReplyToUID
	if replyToUID == 0 {
		if p.ParentID > 0 {
			parentComment, _ := s.commentDao.GetCommentByID(ctx, p.ParentID)
			if parentComment != nil {
				replyToUID = parentComment.AuthorID
			}
		} else {
			// 一级评论，回复的是帖子作者
			if len(post.Authors) > 0 {
				replyToUID = post.Authors[0].UserID
			}
		}
	}

	comment := &model.Comment{
		ID:         commentID,
		PostID:     p.PostID,
		RootID:     p.RootID,
		ParentID:   p.ParentID,
		AuthorID:   authorID,
		ReplyToUID: replyToUID,
		Content:    p.Content,
		Status:     model.CommentStatusNormal,
	}

	if err := comment.Validate(); err != nil {
		return nil, err
	}

	if err := s.commentDao.CreateComment(ctx, comment); err != nil {
		zap.L().Error("commentDao.CreateComment failed", zap.Error(err))
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	// 4. 异步发出评论创建领域事件
	contentRunes := []rune(p.Content)
	previewLen := 100
	if len(contentRunes) < previewLen {
		previewLen = len(contentRunes)
	}

	_ = s.eventBus.Publish(ctx, event.EventTypeCommentCreated, strconv.FormatInt(commentID, 10), authorID, event.CommentCreatedEvent{
		CommentID:      commentID,
		PostID:         p.PostID,
		RootID:         p.RootID,
		ParentID:       p.ParentID,
		AuthorID:       authorID,
		ReplyToUserID:  replyToUID,
		ContentPreview: string(contentRunes[:previewLen]),
	})

	author, _ := s.userDao.CheckUserExistsByID(ctx, authorID)
	authorName := "用户"
	if author != nil {
		authorName = author.UserName
	}

	return &commentresp.CommentResponse{
		ID:         strconv.FormatInt(commentID, 10),
		PostID:     strconv.FormatInt(p.PostID, 10),
		RootID:     strconv.FormatInt(p.RootID, 10),
		ParentID:   strconv.FormatInt(p.ParentID, 10),
		AuthorID:   strconv.FormatInt(authorID, 10),
		AuthorName: authorName,
		Content:    p.Content,
		CreatedAt:  time.Now(),
	}, nil
}

// GetCommentList 分页获取帖子根评论列表 (带前 3 条子回复批量预览)
func (s *CommentService) GetCommentList(ctx context.Context, p *commentreq.CommentListRequest, currentUID int64) (*commentresp.CommentListResponse, error) {
	if p.Size <= 0 || p.Size > 50 {
		p.Size = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}

	rootComments, total, err := s.commentDao.GetRootComments(ctx, p.PostID, p.Order, p.Page, p.Size)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	if len(rootComments) == 0 {
		return &commentresp.CommentListResponse{Total: total, Comments: make([]*commentresp.CommentResponse, 0)}, nil
	}

	// 批量提取 rootIDs 预览前 3 条子评论
	rootIDs := make([]int64, len(rootComments))
	authorIDSet := make(map[int64]struct{})
	for i, rc := range rootComments {
		rootIDs[i] = rc.ID
		authorIDSet[rc.AuthorID] = struct{}{}
		if rc.ReplyToUID > 0 {
			authorIDSet[rc.ReplyToUID] = struct{}{}
		}
	}

	previewMap, _ := s.commentDao.GetSubRepliesPreview(ctx, p.PostID, rootIDs, 3)
	for _, subList := range previewMap {
		for _, sc := range subList {
			authorIDSet[sc.AuthorID] = struct{}{}
			if sc.ReplyToUID > 0 {
				authorIDSet[sc.ReplyToUID] = struct{}{}
			}
		}
	}

	// 批量查询所有涉及的用户昵称
	authorIDs := make([]int64, 0, len(authorIDSet))
	for uid := range authorIDSet {
		authorIDs = append(authorIDs, uid)
	}
	users, _ := s.userDao.GetUsersByIDs(ctx, authorIDs)
	userMap := make(map[int64]string, len(users))
	for _, u := range users {
		userMap[u.UserID] = u.UserName
	}

	resList := make([]*commentresp.CommentResponse, len(rootComments))
	for i, rc := range rootComments {
		resList[i] = s.formatCommentDTO(rc, userMap, previewMap[rc.ID])
	}

	return &commentresp.CommentListResponse{
		Total:    total,
		Comments: resList,
	}, nil
}

// GetSubReplies 展开某根评论下的全部子回复
func (s *CommentService) GetSubReplies(ctx context.Context, p *commentreq.SubReplyListRequest, currentUID int64) (*commentresp.CommentListResponse, error) {
	if p.Size <= 0 || p.Size > 50 {
		p.Size = 20
	}
	if p.Page <= 0 {
		p.Page = 1
	}

	replies, total, err := s.commentDao.GetSubReplies(ctx, p.PostID, p.RootID, p.Page, p.Size)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	authorIDSet := make(map[int64]struct{})
	for _, sc := range replies {
		authorIDSet[sc.AuthorID] = struct{}{}
		if sc.ReplyToUID > 0 {
			authorIDSet[sc.ReplyToUID] = struct{}{}
		}
	}
	authorIDs := make([]int64, 0, len(authorIDSet))
	for uid := range authorIDSet {
		authorIDs = append(authorIDs, uid)
	}
	users, _ := s.userDao.GetUsersByIDs(ctx, authorIDs)
	userMap := make(map[int64]string, len(users))
	for _, u := range users {
		userMap[u.UserID] = u.UserName
	}

	resList := make([]*commentresp.CommentResponse, len(replies))
	for i, sc := range replies {
		resList[i] = s.formatCommentDTO(sc, userMap, nil)
	}

	return &commentresp.CommentListResponse{
		Total:    total,
		Comments: resList,
	}, nil
}

// DeleteComment 软删除评论
func (s *CommentService) DeleteComment(ctx context.Context, commentID, authorID int64) error {
	return s.commentDao.DeleteComment(ctx, commentID, authorID)
}

func (s *CommentService) formatCommentDTO(c *model.Comment, userMap map[int64]string, subReplies []*model.Comment) *commentresp.CommentResponse {
	authorName := userMap[c.AuthorID]
	if authorName == "" {
		authorName = "已注销用户"
	}
	replyToName := userMap[c.ReplyToUID]

	content := c.Content
	isDeleted := false
	if c.Status == model.CommentStatusDeleted {
		content = "[该评论已被作者删除]"
		isDeleted = true
	} else if c.Status == model.CommentStatusBlocked {
		content = "[该评论因违规已被屏蔽]"
		isDeleted = true
	}

	var subDTOs []*commentresp.CommentResponse
	if len(subReplies) > 0 {
		subDTOs = make([]*commentresp.CommentResponse, len(subReplies))
		for j, sc := range subReplies {
			subDTOs[j] = s.formatCommentDTO(sc, userMap, nil)
		}
	}

	return &commentresp.CommentResponse{
		ID:          strconv.FormatInt(c.ID, 10),
		PostID:      strconv.FormatInt(c.PostID, 10),
		RootID:      strconv.FormatInt(c.RootID, 10),
		ParentID:    strconv.FormatInt(c.ParentID, 10),
		AuthorID:    strconv.FormatInt(c.AuthorID, 10),
		AuthorName:  authorName,
		ReplyToUID:  strconv.FormatInt(c.ReplyToUID, 10),
		ReplyToName: replyToName,
		Content:     content,
		LikeCount:   c.LikeCount,
		ReplyCount:  c.ReplyCount,
		IsDeleted:   isDeleted,
		CreatedAt:   c.CreatedAt,
		SubReplies:  subDTOs,
	}
}
