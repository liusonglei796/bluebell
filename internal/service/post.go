// Package service 提供业务逻辑层（MVC 的 Service 层）
package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"bluebell/internal/dao/mysql"
	"bluebell/internal/dao/redis"
	postreq "bluebell/internal/dto/request/post"
	communityResp "bluebell/internal/dto/response/community"
	postResp "bluebell/internal/dto/response/post"
	"bluebell/internal/model"
	"bluebell/internal/mq"
	"bluebell/internal/snowflake"
	"bluebell/pkg/event"

	"go.uber.org/zap"
)

// PostService 帖子业务逻辑服务
type PostService struct {
	postDao       *mysql.PostDao
	postCache     *redis.PostCache
	localCache    *redis.LocalPostCache
	voteDao       *mysql.VoteDao
	commentDao    *mysql.CommentDao
	tagDao        *mysql.TagDao
	pinCache      *redis.PinCache
	bookmarkCache *redis.BookmarkCache
	relationDao   *mysql.RelationDao
	feedCache     *redis.FeedCache
	eventBus      *mq.EventBus
	communityDao  *mysql.CommunityDao
	userDao       *mysql.UserDao
}

// NewPostService 创建帖子服务实例
func NewPostService(
	postDao *mysql.PostDao,
	postCache *redis.PostCache,
	localCache *redis.LocalPostCache,
	voteDao *mysql.VoteDao,
	commentDao *mysql.CommentDao,
	tagDao *mysql.TagDao,
	pinCache *redis.PinCache,
	bookmarkCache *redis.BookmarkCache,
	relationDao *mysql.RelationDao,
	feedCache *redis.FeedCache,
	eventBus *mq.EventBus,
	communityDao *mysql.CommunityDao,
	userDao *mysql.UserDao,
) *PostService {
	if localCache == nil {
		localCache = redis.NewLocalPostCache()
	}
	return &PostService{
		postDao:       postDao,
		postCache:     postCache,
		localCache:    localCache,
		voteDao:       voteDao,
		commentDao:    commentDao,
		tagDao:        tagDao,
		pinCache:      pinCache,
		bookmarkCache: bookmarkCache,
		relationDao:   relationDao,
		feedCache:     feedCache,
		eventBus:      eventBus,
		communityDao:  communityDao,
		userDao:       userDao,
	}
}
func (s *PostService) CreatePost(ctx context.Context, p *postreq.CreatePostRequest, authorID int64) (postID string, err error) {
	postIDInt := snowflake.GenID()
	postID = strconv.FormatInt(postIDInt, 10)

	// 自动提取反范式冗余字段 (Plan 1)
	var authorName string
	if s.userDao != nil {
		if authorUser, _ := s.userDao.CheckUserExistsByID(ctx, authorID); authorUser != nil {
			authorName = authorUser.UserName
		}
	}
	var communityName string
	if s.communityDao != nil {
		if comm, _ := s.communityDao.GetCommunityDetailByID(ctx, p.CommunityID); comm != nil {
			communityName = comm.CommunityName
		}
	}
	var tagNames []string
	if len(p.TagIDs) > 0 && s.tagDao != nil {
		tagNames, _ = s.tagDao.GetTagNamesByIDs(ctx, p.TagIDs)
	}

	post := &model.Post{
		PostID:        postID,
		CommunityID:   p.CommunityID,
		PostTitle:     p.Title,
		Content:       p.Content,
		AuthorName:    authorName,
		CommunityName: communityName,
		TagNames:      strings.Join(tagNames, ","),
		Status:        model.PostStatusPublished,
	}
	post.ContentHash = post.ComputeContentHash()

	if !post.IsValid() {
		return "", model.ErrInvalidParam
	}

	// 防重复提交：基于请求指纹（用户ID + 接口 + 参数hash），5秒内相同请求视为手抖连点
	fingerprint := fmt.Sprintf("%d:/api/v1/post/create:%s:%s:%d", authorID, p.Title, p.Content, p.CommunityID)
	if err := s.postCache.CheckDuplicateSubmit(ctx, fingerprint, 5*time.Second); err != nil {
		if errors.Is(err, model.ErrDuplicateSubmit) {
			return "", err
		}
		zap.L().Error("postCache.CheckDuplicateSubmit failed",
			zap.Int64("author_id", authorID),
			zap.Error(err))
	}

	// 创建帖子并关联作者（多对多）
	author := &model.User{UserID: authorID}
	if err := s.postDao.CreatePostWithAuthor(ctx, post, author); err != nil {
		zap.L().Error("postDao.CreatePostWithAuthor failed",
			zap.Int64("post_id", postIDInt),
			zap.Error(err))
		return "", model.Wrap(model.ErrServerBusy, err)
	}

	// 绑定标签
	if len(p.TagIDs) > 0 && s.tagDao != nil {
		_ = s.tagDao.BindPostTags(ctx, postIDInt, p.CommunityID, p.TagIDs)
	}

	// 同步到 Redis 排序 ZSet
	err = s.postCache.CreatePost(ctx, postIDInt, p.CommunityID)
	if err != nil {
		zap.L().Error("postCache.CreatePost failed",
			zap.Int64("post_id", postIDInt),
			zap.Error(err))
	}

	// 预热/写入 Redis 实体快照缓存 (Plan 2)
	formatted := s.FormatPostListDTOs(ctx, []*model.Post{post}, []string{postID}, 0)
	if len(formatted) > 0 {
		_ = s.postCache.SetPostDetails(ctx, formatted, 24*time.Hour)
	}

	// 异步发布发帖领域事件
	if s.eventBus != nil {
		_ = s.eventBus.Publish(ctx, event.EventTypePostPublished, postID, authorID, event.PostPublishedEvent{
			PostID:      postIDInt,
			AuthorID:    authorID,
			CommunityID: p.CommunityID,
			Title:       p.Title,
		})
	}

	return postID, nil
}

// FetchPostsWithCache 通过 Redis 实体缓存 (Plan 2) + MySQL 单表冗余兜底 (Plan 1) 高性能获取帖子列表 DTO
// FetchPostsMultiTier 三级缓存（L1 本地内存 -> L2 Redis -> L3 MySQL）获取帖子基础数据
func (s *PostService) FetchPostsMultiTier(ctx context.Context, orderedIDs []string) map[string]*postResp.DetailResponse {
	hitMap := make(map[string]*postResp.DetailResponse, len(orderedIDs))
	if len(orderedIDs) == 0 {
		return hitMap
	}

	var l1MissedIDs []string
	// 1. 【L1 本地内存缓存 (0 网络 RTT)】
	for _, id := range orderedIDs {
		if s.localCache != nil {
			if item, ok := s.localCache.Get(id); ok && item != nil {
				hitMap[id] = item
				continue
			}
		}
		l1MissedIDs = append(l1MissedIDs, id)
	}

	// 2. 【L2 Redis 分布式实体缓存】
	var l2MissedIDs []string
	if len(l1MissedIDs) > 0 {
		l2Hits, missed, err := s.postCache.MGetPostDetails(ctx, l1MissedIDs)
		if err != nil {
			zap.L().Warn("postCache.MGetPostDetails warning, fallback to DB", zap.Error(err))
			l2MissedIDs = l1MissedIDs
		} else {
			l2MissedIDs = missed
			for id, item := range l2Hits {
				hitMap[id] = item
				if s.localCache != nil {
					s.localCache.Set(id, item, 10*time.Second) // 回填 L1 本地缓存
				}
			}
		}
	}

	// 3. 【L3 MySQL 反范式单表极速兜底 (0 JOIN)】
	if len(l2MissedIDs) > 0 {
		dbPosts, err := s.postDao.GetPostListByIDsSingleTable(ctx, l2MissedIDs)
		if err != nil || len(dbPosts) == 0 {
			dbPosts, err = s.postDao.GetPostListByIDsWithPreload(ctx, l2MissedIDs)
		}
		if err != nil {
			zap.L().Error("fetch missed posts from DB failed", zap.Error(err))
		} else if len(dbPosts) > 0 {
			formattedDB := s.FormatPostListDTOs(ctx, dbPosts, l2MissedIDs, 0)
			// 同步回填 L2 与 L1
			_ = s.postCache.SetPostDetails(ctx, formattedDB, 24*time.Hour)
			for _, item := range formattedDB {
				hitMap[item.ID] = item
				if s.localCache != nil {
					s.localCache.Set(item.ID, item, 10*time.Second)
				}
			}
		}
	}

	return hitMap
}

// HydrateAndRerankPosts 内存聚合与多维动态重排序（置顶优先 -> 动态 Gravity 实时热度算分 / 时间倒序）
func (s *PostService) HydrateAndRerankPosts(ctx context.Context, orderedIDs []string, currentUID int64, order string) []*postResp.DetailResponse {
	if len(orderedIDs) == 0 {
		return make([]*postResp.DetailResponse, 0)
	}

	// 1. 三级缓存读取基础实体快照
	hitMap := s.FetchPostsMultiTier(ctx, orderedIDs)

	// 2. 批量拉取实时指标与个性化状态
	var voteData []int64
	if s.postCache != nil {
		voteData, _ = s.postCache.GetPostsVoteData(ctx, orderedIDs)
	}
	var bookmarkMap map[string]bool
	if s.bookmarkCache != nil {
		bookmarkMap, _ = s.bookmarkCache.BatchIsBookmarked(ctx, currentUID, orderedIDs)
	}
	// 3. 内存聚合装配
	res := make([]*postResp.DetailResponse, 0, len(orderedIDs))
	for idx, id := range orderedIDs {
		item, ok := hitMap[id]
		if !ok || item == nil {
			continue
		}
		clone := *item
		if idx < len(voteData) {
			clone.VoteNum = voteData[idx]
			clone.Score = voteData[idx]
		}
		clone.IsBookmarked = bookmarkMap[id]
		res = append(res, &clone)
	}

	// 4. 内存中执行多维复合重排序 (In-Memory Rerank)
	if order == postreq.OrderScore {
		// 动态 Gravity 热度重排：置顶优先 -> 实时衰减分数降序 -> 创建时间降序
		sort.SliceStable(res, func(i, j int) bool {
			if res[i].IsPinned != res[j].IsPinned {
				return res[i].IsPinned
			}
			scoreI := redis.CalculateGravityScore(res[i].VoteNum, 0, res[i].CreateTime)
			scoreJ := redis.CalculateGravityScore(res[j].VoteNum, 0, res[j].CreateTime)
			if scoreI != scoreJ {
				return scoreI > scoreJ
			}
			return res[i].CreateTime.After(res[j].CreateTime)
		})
	} else {
		// 时间序：置顶优先 -> 创建时间降序
		sort.SliceStable(res, func(i, j int) bool {
			if res[i].IsPinned != res[j].IsPinned {
				return res[i].IsPinned
			}
			return res[i].CreateTime.After(res[j].CreateTime)
		})
	}

	return res
}

// GetPostByID 查询单个帖子详情（三级缓存 + 动态水合）
func (s *PostService) GetPostByID(ctx context.Context, pid int64, currentUID int64) (*postResp.DetailResponse, error) {
	pidStr := strconv.FormatInt(pid, 10)
	res := s.HydrateAndRerankPosts(ctx, []string{pidStr}, currentUID, postreq.OrderTime)
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return res[0], nil
}

// GetPostList 获取帖子列表（L1/L2/L3 多级缓存 + 内存聚合与动态重排序）
func (s *PostService) GetPostList(ctx context.Context, p *postreq.PostListRequest, currentUID int64) ([]*postResp.DetailResponse, error) {
	ids, err := s.postCache.GetPostIDsInOrder(ctx, p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("postCache.GetPostIDsInOrder failed",
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, model.Wrap(model.ErrServerBusy, err)
	}
	return s.HydrateAndRerankPosts(ctx, ids, currentUID, p.Order), nil
}

// GetCommunityPostList 根据社区ID获取帖子列表（多路召回置顶帖 + 内存归并与重排）
func (s *PostService) GetCommunityPostList(ctx context.Context, p *postreq.PostListRequest, currentUID int64) ([]*postResp.DetailResponse, error) {
	ids, err := s.postCache.GetCommunityPostIDsInOrder(ctx, p.CommunityID, p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("postCache.GetCommunityPostIDsInOrder failed",
			zap.Int64("community_id", p.CommunityID),
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	// 多路召回：第一页合并社区置顶帖
	if p.Page <= 1 && s.pinCache != nil {
		pinnedIDs, _ := s.pinCache.GetCommunityPinned(ctx, p.CommunityID)
		if len(pinnedIDs) > 0 {
			// 内存去重并置顶前置
			seen := make(map[string]bool, len(pinnedIDs)+len(ids))
			combined := make([]string, 0, len(pinnedIDs)+len(ids))
			for _, pid := range pinnedIDs {
				if !seen[pid] {
					seen[pid] = true
					combined = append(combined, pid)
				}
			}
			for _, pid := range ids {
				if !seen[pid] {
					seen[pid] = true
					combined = append(combined, pid)
				}
			}
			ids = combined
		}
	}

	return s.HydrateAndRerankPosts(ctx, ids, currentUID, p.Order), nil
}
// DeletePost 删除帖子及其评论（级联软删除）
func (s *PostService) DeletePost(ctx context.Context, postID int64, userID int64) error {
	postIDStr := strconv.FormatInt(postID, 10)
	post, err := s.postDao.GetPostByID(ctx, postID)
	if err != nil {
		zap.L().Error("postDao.GetPostByID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}
	if !post.IsValid() {
		return model.ErrNotFound
	}

	// 权限校验
	if err := post.CanBeDeletedBy(userID); err != nil {
		return err
	}

	// 1. 删除该帖子的所有评论
	if err := s.commentDao.DeleteCommentsByPostID(ctx, postID); err != nil {
		zap.L().Error("commentDao.DeleteCommentsByPostID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	// 2. 软删除帖子 (status = 0)
	err = s.postDao.DeletePostByAuthor(ctx, postIDStr, userID)
	if err != nil {
		zap.L().Error("postDao.DeletePostByAuthor failed",
			zap.Int64("post_id", postID),
			zap.Int64("user_id", userID),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	// 清理多级缓存 (L1 + L2)
	if s.localCache != nil {
		s.localCache.Delete(postIDStr)
	}
	if err := s.postCache.DeletePost(ctx, postID, post.CommunityID); err != nil {
		zap.L().Error("postCache.DeletePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		// 缓存清理失败不影响主流程，仅记录日志
	}
	return nil
}

// VoteForPost 投票业务逻辑
// 请求 → Redis Lua 原子校验与更新(ZSet+Hash+Gravity score) → MySQL UPSERT 持久化
func (s *PostService) VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error {
	// 业务校验
	vote := &model.Vote{
		PostID:    p.PostID,
		UserID:    userID,
		Direction: p.Direction,
	}
	if err := vote.Validate(); err != nil {
		return err
	}

	postIDStr := strconv.FormatInt(p.PostID, 10)
	userIDStr := strconv.FormatInt(userID, 10)

	// 1. 获取 community_id (优先 Redis → 回退 MySQL)
	communityID, err := s.postCache.GetPostCommunityID(ctx, p.PostID)
	if err != nil {
		// Redis 缓存缺失，回退到 MySQL 查找帖子
		post, err := s.postDao.GetPostByID(ctx, p.PostID)
		if err != nil {
			return model.Wrap(model.ErrServerBusy, err)
		}
		if post == nil {
			return model.ErrNotFound
		}
		communityID = post.CommunityID
		// 引导 Redis 缓存，让后续投票走快路径
		if err := s.postCache.CreatePost(ctx, p.PostID, communityID); err != nil {
			zap.L().Error("postCache.CreatePost bootstrap failed", zap.Error(err))
		}
	}
	communityIDStr := strconv.FormatInt(communityID, 10)

	// 2. Redis Lua 原子更新 (ZSet + Hash + Gravity score)
	err = s.postCache.VoteForPost(ctx, userIDStr, postIDStr, communityIDStr, float64(p.Direction))
	if err != nil {
		if errors.Is(err, model.ErrVoteTimeExpire) {
			return err
		}
		if errors.Is(err, model.ErrVoteRepeated) {
			// 重复投票是幂等操作，不报错
			return nil
		}
		zap.L().Error("postCache.VoteForPost failed",
			zap.String("post_id", postIDStr),
			zap.String("user_id", userIDStr),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	// 3. MySQL 持久化落库 (UPSERT 幂等保障)
	if err := s.voteDao.SaveVote(ctx, userID, p.PostID, p.Direction); err != nil {
		zap.L().Error("voteDao.SaveVote failed",
			zap.Int64("user_id", userID),
			zap.Int64("post_id", p.PostID),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	return nil
}


// PinPost 置顶/取消置顶帖子 (仅管理员或版主)
func (s *PostService) PinPost(ctx context.Context, postID, communityID int64, isPinned bool, userID int64) error {
	if err := s.pinCache.SetPinned(ctx, communityID, postID, isPinned); err != nil {
		return model.Wrap(model.ErrServerBusy, err)
	}
	return nil
}

// GetTimelineFeed 获取用户关注的人的发帖动态 Timeline Feed (基于读扩散 + 游标分页)
func (s *PostService) GetTimelineFeed(ctx context.Context, userID int64, cursor, size int64) ([]*postResp.DetailResponse, error) {
	if size <= 0 || size > 50 {
		size = 20
	}

	// 1. 尝试从用户 Feed 缓存中拉取游标分页 ID
	postIDStrs, err := s.feedCache.GetUserFeedPage(ctx, userID, cursor, size)
	if err != nil || len(postIDStrs) == 0 {
		// 2. 缓存未命中，执行读扩散多路归并
		followingIDs, err := s.relationDao.GetFollowingIDs(ctx, userID)
		if err != nil || len(followingIDs) == 0 {
			return make([]*postResp.DetailResponse, 0), nil
		}

		// 从 MySQL 多路归并出最新 100 条帖子
		posts, err := s.postDao.GetPostListByAuthorIDs(ctx, followingIDs, 100)
		if err != nil || len(posts) == 0 {
			return make([]*postResp.DetailResponse, 0), nil
		}

		pIDs := make([]int64, len(posts))
		timestamps := make([]int64, len(posts))
		for i, p := range posts {
			pidInt, _ := strconv.ParseInt(p.PostID, 10, 64)
			pIDs[i] = pidInt
			timestamps[i] = p.CreatedAt.UnixMilli()
		}

		// 预热写入 Feed ZSet (带 10min TTL)
		_ = s.feedCache.SetUserFeed(ctx, userID, pIDs, timestamps, 10*time.Minute)
		postIDStrs, _ = s.feedCache.GetUserFeedPage(ctx, userID, cursor, size)
	}

	if len(postIDStrs) == 0 {
		return make([]*postResp.DetailResponse, 0), nil
	}

	posts, err := s.postDao.GetPostListByIDsWithPreload(ctx, postIDStrs)
	if err != nil {
		return nil, model.Wrap(model.ErrServerBusy, err)
	}

	return s.FormatPostListDTOs(ctx, posts, postIDStrs, userID), nil
}

// FormatPostListDTOs 通用批量组装帖子列表 DTO (包含作者、社区、标签、点赞分数、是否收藏)
func (s *PostService) FormatPostListDTOs(ctx context.Context, posts []*model.Post, orderedIDs []string, currentUID int64) []*postResp.DetailResponse {
	if len(posts) == 0 {
		return make([]*postResp.DetailResponse, 0)
	}

	voteData, _ := s.postCache.GetPostsVoteData(ctx, orderedIDs)
	bookmarkMap, _ := s.bookmarkCache.BatchIsBookmarked(ctx, currentUID, orderedIDs)

	data := make([]*postResp.DetailResponse, 0, len(posts))
	for idx, post := range posts {
		var authorNames []string
		if post.HasAuthors() {
			for _, a := range post.Authors {
				authorNames = append(authorNames, a.UserName)
			}
		}

		primaryAuthorName := post.AuthorName
		if primaryAuthorName == "" {
			if len(authorNames) > 0 {
				primaryAuthorName = authorNames[0]
			}
		} else if len(authorNames) == 0 {
			authorNames = []string{primaryAuthorName}
		}

		communityName := post.CommunityName
		var communityObj *communityResp.Response
		if post.Community != nil {
			communityName = post.Community.CommunityName
			communityObj = &communityResp.Response{
				ID:           strconv.FormatInt(int64(post.Community.ID), 10),
				Name:         post.Community.CommunityName,
				Introduction: post.Community.Introduction,
				CreateTime:   post.Community.CreatedAt,
			}
		} else if communityName != "" {
			communityObj = &communityResp.Response{
				ID:         strconv.FormatInt(post.CommunityID, 10),
				Name:       communityName,
				CreateTime: post.CreatedAt,
			}
		}

		var voteNum int64
		if idx < len(voteData) {
			voteNum = voteData[idx]
		}

		// 提取标签名称（优先使用反范式冗余字段 tag_names，若为空再回退查 tagDao）
		var tagNames []string
		if post.TagNames != "" {
			tagNames = strings.Split(post.TagNames, ",")
		} else if s.tagDao != nil {
			pidInt, _ := strconv.ParseInt(post.PostID, 10, 64)
			tagNames, _ = s.tagDao.GetPostTags(ctx, pidInt)
		}
		postDetail := &postResp.DetailResponse{
			ID:            post.PostID,
			AuthorIDs:     formatAuthorIDs(post.Authors),
			AuthorNames:   authorNames,
			AuthorName:    primaryAuthorName,
			CommunityID:   post.CommunityID,
			CommunityName: communityName,
			Community:     communityObj,
			Status:        post.Status,
			Title:         post.PostTitle,
			Content:       post.Content,
			CreateTime:    post.CreatedAt,
			VoteNum:       voteNum,
			Score:         voteNum,
			IsPinned:      post.IsPinned == 1,
			IsHighlighted: post.IsHighlighted == 1,
			BookmarkCount: post.BookmarkCount,
			CommentCount:  post.CommentCount,
			IsBookmarked:  bookmarkMap[post.PostID],
			Tags:          tagNames,
		}
		data = append(data, postDetail)
	}

	return data
}

// formatAuthorIDs 格式化作者ID列表
func formatAuthorIDs(users []model.User) []string {
	ids := make([]string, len(users))
	for i, u := range users {
		ids[i] = strconv.FormatInt(u.UserID, 10)
	}
	return ids
}
