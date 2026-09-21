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

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

// PostService 帖子业务逻辑服务
type PostService struct {
	postDao       *mysql.PostDao
	postCache     *redis.PostCache
	commentDao    *mysql.CommentDao
	tagDao        *mysql.TagDao
	pinCache      *redis.PinCache
	bookmarkCache *redis.BookmarkCache
	relationDao   *mysql.RelationDao
	feedCache     *redis.FeedCache
	eventBus      *mq.EventBus
	communityDao  *mysql.CommunityDao
	userDao       *mysql.UserDao
	sfGroup       singleflight.Group        // 防击穿：单节点并发请求单飞合并
	dbBreaker     *gobreaker.CircuitBreaker // 熔断器：详情回源 MySQL 时的保护断路器
	redisBreaker  *gobreaker.CircuitBreaker // 熔断器：Redis 投票与热榜更新时的保护断路器
}

// NewPostService 创建帖子服务实例
func NewPostService(
	postDao *mysql.PostDao,
	postCache *redis.PostCache,
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
	breakerSettings := gobreaker.Settings{
		Name:        "PostDBBreaker",
		MaxRequests: 3,                // 半开状态下允许的最大试探请求数
		Interval:    10 * time.Second, // 统计滑动窗口
		Timeout:     5 * time.Second,  // 熔断后冷却 5 秒进入半开状态
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// 触发熔断条件：连续失败达到 5 次，或请求数 >= 10 且失败率 >= 50%
			return counts.ConsecutiveFailures >= 5 || (counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			zap.L().Warn("post db circuit breaker state changed",
				zap.String("breaker", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()))
		},
	}
	dbBreaker := gobreaker.NewCircuitBreaker(breakerSettings)

	redisBreakerSettings := gobreaker.Settings{
		Name:        "PostRedisBreaker",
		MaxRequests: 3,                // 半开状态下允许试探的请求数
		Interval:    10 * time.Second, // 统计滑动窗口
		Timeout:     5 * time.Second,  // 熔断后冷却 5 秒进入半开状态
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// 连续失败 5 次，或请求数 >= 10 且失败率 >= 50% 时触发熔断
			return counts.ConsecutiveFailures >= 5 || (counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			zap.L().Warn("post redis circuit breaker state changed",
				zap.String("breaker", name),
				zap.String("from", from.String()),
				zap.String("to", to.String()))
		},
	}
	redisBreaker := gobreaker.NewCircuitBreaker(redisBreakerSettings)

	return &PostService{
		postDao:       postDao,
		postCache:     postCache,
		commentDao:    commentDao,
		tagDao:        tagDao,
		pinCache:      pinCache,
		bookmarkCache: bookmarkCache,
		relationDao:   relationDao,
		feedCache:     feedCache,
		eventBus:      eventBus,
		communityDao:  communityDao,
		userDao:       userDao,
		dbBreaker:     dbBreaker,
		redisBreaker:  redisBreaker,
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
		PostID:        postIDInt,
		AuthorID:      authorID,
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
	if s.postCache != nil {
		fingerprint := fmt.Sprintf("%d:/api/v1/post/create:%s:%s:%d", authorID, p.Title, p.Content, p.CommunityID)
		if err := s.postCache.CheckDuplicateSubmit(ctx, fingerprint, 5*time.Second); err != nil {
			if errors.Is(err, model.ErrDuplicateSubmit) {
				return "", err
			}
			zap.L().Error("postCache.CheckDuplicateSubmit failed",
				zap.Int64("author_id", authorID),
				zap.Error(err))
		}
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

	// 同步到 Redis 排序 ZSet 与缓存预热
	if s.postCache != nil {
		err = s.postCache.CreatePost(ctx, postIDInt, p.CommunityID)
		if err != nil {
			zap.L().Error("postCache.CreatePost failed",
				zap.Int64("post_id", postIDInt),
				zap.Error(err))
		}

		formatted := s.FormatPostListDTOs(ctx, []*model.Post{post}, []string{postID}, 0)
		if len(formatted) > 0 {
			_ = s.postCache.SetPostDetails(ctx, formatted, 24*time.Hour)
		}

		// 防穿透：将新发布的帖子ID写入分布式布隆过滤器
		_ = s.postCache.AddPostBloom(ctx, postID)
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

// getDBBreaker 延迟获取或初始化断路器（保障单测等零值初始化的安全性）
func (s *PostService) getDBBreaker() *gobreaker.CircuitBreaker {
	if s.dbBreaker == nil {
		s.dbBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "PostDBBreaker",
			MaxRequests: 3,
			Interval:    10 * time.Second,
			Timeout:     5 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 5 || (counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5)
			},
		})
	}
	return s.dbBreaker
}

// getRedisBreaker 延迟获取或初始化 Redis 断路器（保障单测等零值初始化的安全性）
func (s *PostService) getRedisBreaker() *gobreaker.CircuitBreaker {
	if s.redisBreaker == nil {
		s.redisBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:        "PostRedisBreaker",
			MaxRequests: 3,
			Interval:    10 * time.Second,
			Timeout:     5 * time.Second,
			ReadyToTrip: func(counts gobreaker.Counts) bool {
				return counts.ConsecutiveFailures >= 5 || (counts.Requests >= 10 && float64(counts.TotalFailures)/float64(counts.Requests) >= 0.5)
			},
		})
	}
	return s.redisBreaker
}



// fetchPostWithSingleflight 使用 singleflight 并发单飞合并打库，同时实施防击穿、防雪崩与布隆自愈
func (s *PostService) fetchPostWithSingleflight(ctx context.Context, id string) (*postResp.DetailResponse, error) {
	val, err, _ := s.sfGroup.Do(id, func() (interface{}, error) {
		// 1. 二次确认缓存（Double Check）：可能前面并发拿到锁的协程已经查完 DB 并回填 Redis 了
		if s.postCache != nil {
			if item, _ := s.postCache.GetPostDetail(ctx, id); item != nil {
				return item, nil
			}
		}

		// 2. 回源 MySQL 单表极速查询 (防击穿：同节点内单飞合并；断路器熔断保护：DB故障时跳闸快速降级)
		breakerResult, breakerErr := s.getDBBreaker().Execute(func() (interface{}, error) {
			dbPosts, err := s.postDao.GetPostListByIDsSingleTable(ctx, []string{id})
			if err != nil || len(dbPosts) == 0 {
				dbPosts, err = s.postDao.GetPostListByIDsWithPreload(ctx, []string{id})
			}
			return dbPosts, err
		})

		if breakerErr != nil {
			if errors.Is(breakerErr, gobreaker.ErrOpenState) || errors.Is(breakerErr, gobreaker.ErrTooManyRequests) {
				zap.L().Warn("post db circuit breaker open, fast degrading",
					zap.String("post_id", id),
					zap.Error(breakerErr))
				return nil, model.ErrServerBusy // 熔断跳闸快速降级，保护 MySQL 不被打死
			}
			return nil, breakerErr
		}

		dbPosts, _ := breakerResult.([]*model.Post)
		if len(dbPosts) == 0 {
			// 用户明确要求：防穿透不使用空值缓存，查无此帖直接返回 nil，不写任何空对象到 Redis
			return nil, nil
		}

		// 3. 格式化装配 DTO
		formattedDB := s.FormatPostListDTOs(ctx, dbPosts, []string{id}, 0)
		if len(formattedDB) == 0 {
			return nil, nil
		}
		item := formattedDB[0]

		// 4. 防雪崩：回填 Redis 实体快照时注入随机抖动 (Jitter 0~30分钟)
		if s.postCache != nil {
			_ = s.postCache.SetPostDetailWithJitter(ctx, item, 24*time.Hour, 30*time.Minute)
			// 5. 布隆过滤器自愈：预热该 ID 记录到布隆过滤器
			_ = s.postCache.AddPostBloom(ctx, id)
		}

		return item, nil
	})

	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	return val.(*postResp.DetailResponse), nil
}

// FetchPostsMultiTier 通过 Redis 原生客户端缓存(CSC) + MySQL 单表冗余兜底(Plan 1)
// 实施布隆过滤器防穿透、singleflight 防击穿与 TTL Jitter 防雪崩
func (s *PostService) FetchPostsMultiTier(ctx context.Context, orderedIDs []string) map[string]*postResp.DetailResponse {
	hitMap := make(map[string]*postResp.DetailResponse, len(orderedIDs))
	if len(orderedIDs) == 0 {
		return hitMap
	}

	var dbMissedIDs []string
	// L1(进程内 CSC) + L2(Redis) 合并层：rdb.Get 自动走客户端缓存
	for _, id := range orderedIDs {
		// 1. 防穿透：布隆过滤器前置快速校验，若绝对不存在则直接略过，绝不打库
		if s.postCache != nil {
			exists, err := s.postCache.CheckPostInBloom(ctx, id)
			if err == nil && !exists {
				continue
			}
		}

		item, err := s.postCache.GetPostDetail(ctx, id)
		if err != nil {
			zap.L().Warn("postCache.GetPostDetail warning, fallback to DB", zap.Error(err))
			dbMissedIDs = append(dbMissedIDs, id)
			continue
		}
		if item == nil {
			dbMissedIDs = append(dbMissedIDs, id) // redis.Nil → 未命中
			continue
		}
		hitMap[id] = item
	}

	// L3 MySQL 反范式单表极速兜底 + singleflight 防击穿 + Jitter 防雪崩
	if len(dbMissedIDs) > 0 {
		for _, id := range dbMissedIDs {
			item, err := s.fetchPostWithSingleflight(ctx, id)
			if err != nil {
				zap.L().Error("fetch missed post with singleflight failed",
					zap.String("post_id", id),
					zap.Error(err))
				continue
			}
			if item != nil {
				hitMap[item.ID] = item
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
	var authorIDsToFetch []int64
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

		// 动态刷新社区信息（优先从本地进程内存缓存 mapCache 读取，0 SQL，0 网络开销）
		if s.communityDao != nil && clone.CommunityID > 0 {
			if comm, err := s.communityDao.GetCommunityDetailByID(ctx, clone.CommunityID); err == nil && comm != nil {
				clone.CommunityName = comm.CommunityName
				if clone.Community != nil {
					clone.Community.Name = comm.CommunityName
					clone.Community.Introduction = comm.Introduction
				}
			}
		}

		for _, aidStr := range clone.AuthorIDs {
			if aid, err := strconv.ParseInt(aidStr, 10, 64); err == nil && aid > 0 {
				authorIDsToFetch = append(authorIDsToFetch, aid)
			}
		}

		res = append(res, &clone)
	}

	// 动态批量刷新最新的作者名称（单次主键批量查询，彻底消除改名后缓存数据漂移）
	if len(authorIDsToFetch) > 0 && s.userDao != nil {
		if users, err := s.userDao.GetUsersByIDs(ctx, authorIDsToFetch); err == nil {
			userMap := make(map[int64]string, len(users))
			for _, u := range users {
				userMap[u.UserID] = u.UserName
			}
			for _, item := range res {
				for _, aidStr := range item.AuthorIDs {
					if aid, err := strconv.ParseInt(aidStr, 10, 64); err == nil {
						if freshName, ok := userMap[aid]; ok && freshName != "" {
							item.AuthorName = freshName
							item.AuthorNames = []string{freshName}
						}
					}
				}
			}
		}
	}

	// 4. 内存中执行多维复合重排序 (In-Memory Rerank)
	return rerankPosts(res, order)
}

// rerankPosts 内存中执行多维复合重排序：置顶优先 -> 热度/时间降序。
// 纯函数（输入需已按 ZSet 顺序装配好动态字段），便于单测。
func rerankPosts(items []*postResp.DetailResponse, order string) []*postResp.DetailResponse {
	if order == postreq.OrderScore {
		// 动态 Gravity 热度重排：置顶优先 -> 实时衰减分数降序 -> 创建时间降序
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].IsPinned != items[j].IsPinned {
				return items[i].IsPinned
			}
			scoreI := redis.CalculateGravityScore(items[i].VoteNum, 0, items[i].CreateTime)
			scoreJ := redis.CalculateGravityScore(items[j].VoteNum, 0, items[j].CreateTime)
			if scoreI != scoreJ {
				return scoreI > scoreJ
			}
			return items[i].CreateTime.After(items[j].CreateTime)
		})
	} else {
		// 时间序：置顶优先 -> 创建时间降序
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].IsPinned != items[j].IsPinned {
				return items[i].IsPinned
			}
			return items[i].CreateTime.After(items[j].CreateTime)
		})
	}
	return items
}

// GetPostByID 查询单个帖子详情（三级缓存 + 防穿透 + 防击穿 + 防雪崩）
func (s *PostService) GetPostByID(ctx context.Context, pid int64, currentUID int64) (*postResp.DetailResponse, error) {
	// 1. 防穿透：参数有效性前置拦截
	if pid <= 0 {
		return nil, model.ErrInvalidParam
	}

	// 无 Redis 模式：直接查 MySQL
	if s.postCache == nil {
		post, err := s.postDao.GetPostByID(ctx, pid)
		if err != nil {
			return nil, model.Wrap(model.ErrServerBusy, err)
		}
		if post == nil {
			return nil, model.ErrNotFound
		}
		dtos := s.FormatPostListDTOs(ctx, []*model.Post{post}, []string{strconv.FormatInt(pid, 10)}, currentUID)
		if len(dtos) == 0 {
			return nil, model.ErrNotFound
		}
		return dtos[0], nil
	}

	pidStr := strconv.FormatInt(pid, 10)

	// 2. 防穿透：布隆过滤器拦截绝对不存在的 ID（0 次查询 MySQL 与实体缓存）
	if s.postCache != nil {
		exists, err := s.postCache.CheckPostInBloom(ctx, pidStr)
		if err == nil && !exists {
			return nil, model.ErrNotFound
		}
	}

	res := s.HydrateAndRerankPosts(ctx, []string{pidStr}, currentUID, postreq.OrderTime)
	if len(res) == 0 {
		return nil, model.ErrNotFound
	}
	return res[0], nil
}

// GetPostList 获取帖子列表（L1/L2/L3 多级缓存 + 内存聚合与动态重排序）
func (s *PostService) GetPostList(ctx context.Context, p *postreq.PostListRequest, currentUID int64) ([]*postResp.DetailResponse, error) {
	// 无 Redis 模式：直接查 MySQL 分页列表
	if s.postCache == nil {
		posts, err := s.postDao.GetPostList(ctx, p.Page, p.Size, p.Order)
		if err != nil {
			zap.L().Error("postDao.GetPostList failed", zap.Error(err))
			return nil, model.Wrap(model.ErrServerBusy, err)
		}
		ids := make([]string, len(posts))
		for i, post := range posts {
			ids[i] = strconv.FormatInt(post.PostID, 10)
		}
		return s.FormatPostListDTOs(ctx, posts, ids, currentUID), nil
	}

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
	err = s.postDao.DeletePostByAuthor(ctx, postID, userID)
	if err != nil {
		zap.L().Error("postDao.DeletePostByAuthor failed",
			zap.Int64("post_id", postID),
			zap.Int64("user_id", userID),
			zap.Error(err))
		return model.Wrap(model.ErrServerBusy, err)
	}

	// 清理缓存：DeletePost 触发的 Redis DEL 会经 CLIENT TRACKING 自动失效进程内 L1(CSC)，
	// 无需手动维护本地 map。
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

	// 无 Redis 模式：直接执行 MySQL 事务更新并加显式行锁
	if s.postCache == nil {
		return s.postDao.VotePost(ctx, p.PostID, p.Direction)
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

	// 2. Redis Lua 原子更新 (ZSet + Hash + Gravity score)；挂载断路器熔断保护
	var bizErr error
	_, breakerErr := s.getRedisBreaker().Execute(func() (interface{}, error) {
		err := s.postCache.VoteForPost(ctx, userIDStr, postIDStr, communityIDStr, float64(p.Direction))
		if err != nil {
			// 关键：过滤业务预期错误，不计入断路器失败统计
			if errors.Is(err, model.ErrVoteTimeExpire) || errors.Is(err, model.ErrVoteRepeated) || errors.Is(err, model.ErrNotFound) {
				bizErr = err
				return nil, nil // 返回 nil 避免触发断路器误跳闸
			}
			// 真正的 Redis 基础设施故障（如网络超时、连接拒绝、OOM等），计入断路器失败指标
			return nil, err
		}
		return nil, nil
	})

	if breakerErr != nil {
		if errors.Is(breakerErr, gobreaker.ErrOpenState) || errors.Is(breakerErr, gobreaker.ErrTooManyRequests) {
			zap.L().Warn("vote redis circuit breaker open, fast failing to protect service",
				zap.Int64("user_id", userID),
				zap.Int64("post_id", p.PostID),
				zap.Error(breakerErr))
			return model.ErrServerBusy // 熔断跳闸快速降级，防止请求堆积引发服务雪崩
		}
		zap.L().Error("postCache.VoteForPost failed",
			zap.String("post_id", postIDStr),
			zap.String("user_id", userIDStr),
			zap.Error(breakerErr))
		return model.Wrap(model.ErrServerBusy, breakerErr)
	}

	if bizErr != nil {
		if errors.Is(bizErr, model.ErrVoteRepeated) {
			// 重复投票是幂等操作，不报错
			return nil
		}
		return bizErr
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
			pIDs[i] = p.PostID
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

	var voteData []int64
	if s.postCache != nil {
		voteData, _ = s.postCache.GetPostsVoteData(ctx, orderedIDs)
	}
	var bookmarkMap map[string]bool
	if s.bookmarkCache != nil {
		bookmarkMap, _ = s.bookmarkCache.BatchIsBookmarked(ctx, currentUID, orderedIDs)
	}

	// 1. 批量提取涉及的作者 ID（去重）
	authorIDMap := make(map[int64]struct{})
	for _, post := range posts {
		if post.AuthorID != 0 {
			authorIDMap[post.AuthorID] = struct{}{}
		}
		for _, a := range post.Authors {
			if a.UserID != 0 {
				authorIDMap[a.UserID] = struct{}{}
			}
		}
	}
	authorIDs := make([]int64, 0, len(authorIDMap))
	for id := range authorIDMap {
		authorIDs = append(authorIDs, id)
	}

	// 2. 批量拉取最新的作者信息 (0 笛卡尔积，单次主键/唯一索引批量点查)
	userMap := make(map[int64]*model.User, len(authorIDs))
	if len(authorIDs) > 0 && s.userDao != nil {
		if users, err := s.userDao.GetUsersByIDs(ctx, authorIDs); err == nil {
			for _, u := range users {
				userMap[u.UserID] = u
			}
		}
	}

	data := make([]*postResp.DetailResponse, 0, len(posts))
	for idx, post := range posts {
		// 动态组装作者信息（优先使用最新从 User 表/缓存查出的数据，彻底规避改名后数据不一致）
		var authorNames []string
		var primaryAuthorName string

		if u, ok := userMap[post.AuthorID]; ok && u.UserName != "" {
			primaryAuthorName = u.UserName
			authorNames = []string{u.UserName}
		} else {
			if post.HasAuthors() {
				for _, a := range post.Authors {
					if liveU, exists := userMap[a.UserID]; exists && liveU.UserName != "" {
						authorNames = append(authorNames, liveU.UserName)
					} else {
						authorNames = append(authorNames, a.UserName)
					}
				}
			}
			if primaryAuthorName == "" {
				primaryAuthorName = post.AuthorName
				if primaryAuthorName == "" && len(authorNames) > 0 {
					primaryAuthorName = authorNames[0]
				}
			}
		}
		if len(authorNames) == 0 && primaryAuthorName != "" {
			authorNames = []string{primaryAuthorName}
		}

		// 动态组装社区信息（优先从本地进程内存缓存 mapCache 读取，0 SQL，0 网络开销）
		communityName := post.CommunityName
		var communityObj *communityResp.Response
		if s.communityDao != nil && post.CommunityID > 0 {
			if comm, err := s.communityDao.GetCommunityDetailByID(ctx, post.CommunityID); err == nil && comm != nil {
				communityName = comm.CommunityName
				communityObj = &communityResp.Response{
					ID:           strconv.FormatInt(int64(comm.ID), 10),
					Name:         comm.CommunityName,
					Introduction: comm.Introduction,
					CreateTime:   comm.CreatedAt,
				}
			}
		}
		if communityObj == nil && communityName != "" {
			communityObj = &communityResp.Response{
				ID:         strconv.FormatInt(post.CommunityID, 10),
				Name:       communityName,
				CreateTime: post.CreatedAt,
			}
		}

		var voteNum int64 = post.VoteNum
		if idx < len(voteData) {
			voteNum = voteData[idx]
		}

		// 提取标签名称（优先使用反范式冗余字段 tag_names，若为空再回退查 tagDao）
		var tagNames []string
		if post.TagNames != "" {
			tagNames = strings.Split(post.TagNames, ",")
		} else if s.tagDao != nil {
			tagNames, _ = s.tagDao.GetPostTags(ctx, post.PostID)
		}
		postIDStr := strconv.FormatInt(post.PostID, 10)
		postDetail := &postResp.DetailResponse{
			ID:            postIDStr,
			AuthorIDs:     formatAuthorIDs(post.Authors, post.AuthorID),
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
			IsBookmarked:  bookmarkMap[postIDStr],
			Tags:          tagNames,
		}
		data = append(data, postDetail)
	}

	return data
}

// formatAuthorIDs 格式化作者ID列表
func formatAuthorIDs(users []model.User, primaryAuthorID int64) []string {
	if len(users) > 0 {
		ids := make([]string, len(users))
		for i, u := range users {
			ids[i] = strconv.FormatInt(u.UserID, 10)
		}
		return ids
	}
	if primaryAuthorID != 0 {
		return []string{strconv.FormatInt(primaryAuthorID, 10)}
	}
	return []string{}
}
