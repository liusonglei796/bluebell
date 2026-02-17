package logic

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/dto/request"
	"bluebell/dto/response"
	"bluebell/models"
	"bluebell/pkg/errorx"
	"bluebell/pkg/snowflake"

	"go.uber.org/zap"
)

// CreatePost 创建帖子,返回新创建的帖子ID
func CreatePost(p *request.CreatePostRequest) (postID int64, err error) {
	// 1. 生成帖子ID
	postID = snowflake.GenID()

	// 2. 构造Post结构体
	post := &models.Post{
		ID:          postID,
		AuthorID:    p.AuthorID,
		CommunityID: p.CommunityID,
		Title:       p.Title,
		Content:     p.Content,
		Status:      1, // 默认状态为1,表示正常
	}

	// 3. 保存到数据库
	err = mysql.CreatePost(post)
	if err != nil {
		// 系统错误：记录日志并返回通用错误
		zap.L().Error("mysql.CreatePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return 0, errorx.ErrServerBusy
	}

	// 4. 同步到 Redis (用于排序和投票功能)
	// 初始化帖子的时间排序和分数排序
	err = redis.CreatePost(postID, p.CommunityID)
	if err != nil {
		// Redis 写入失败不影响主流程,记录日志即可
		zap.L().Error("redis.CreatePost failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		// 不返回错误,因为 MySQL 已经成功了
	}

	return postID, nil
}

// GetPostByID 查询单个帖子详情
// 优化版：使用 GORM Preload 预加载，从 3 次查询优化为 3 次查询（保持不变但代码更简洁）
func GetPostByID(pid int64) (data *response.PostDetailResponse, err error) {
	// 1. 使用 Preload 查询帖子及其关联的作者和社区信息
	// Preload 会自动执行:
	//   - SELECT * FROM post WHERE post_id = ?
	//   - SELECT * FROM user WHERE user_id = ? (post.author_id)
	//   - SELECT * FROM community WHERE community_id = ? (post.community_id)
	post, err := mysql.GetPostByID(pid)
	if err != nil {
		// 系统错误
		zap.L().Error("mysql.GetPostByID failed",
			zap.Int64("post_id", pid),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 2. 检查是否找到了帖子
	if post == nil || post.ID == 0 {
		// 业务错误：帖子不存在
		return nil, errorx.ErrNotFound
	}

	// 3. 检查关联数据是否加载成功
	if post.Author == nil || post.Author.UserID == 0 {
		zap.L().Warn("author not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("author_id", post.AuthorID))
		return nil, errorx.ErrNotFound
	}

	if post.Community == nil || post.Community.CommunityID == 0 {
		zap.L().Warn("community not found for post",
			zap.Int64("post_id", pid),
			zap.Int64("community_id", post.CommunityID))
		return nil, errorx.ErrNotFound
	}

	// 4. 组装返回数据（数据已经通过 Preload 加载）
	data = &response.PostDetailResponse{
		Post:       post,
		AuthorName: post.Author.Username, // 直接从预加载的 Author 获取
	}

	return data, nil
}

// GetPostList 从 Redis 获取排序后的 ID，再从 MySQL 查询详情，最后组装投票数据
// 优化版：使用 GORM Preload 预加载，从 1+N+N 次查询优化为 3 次查询
func GetPostList(p *request.PostListRequest) (data []*response.PostDetailResponse, err error) {
	// 1. 从 Redis 查询帖子 ID 列表（已按时间或分数排序）
	ids, err := redis.GetPostIDsInOrder(p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("redis.GetPostIDsInOrder failed",
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 2. 处理空数据
	if len(ids) == 0 {
		zap.L().Warn("redis.GetPostIDsInOrder() return 0 data")
		// 返回空切片而不是 nil
		data = make([]*response.PostDetailResponse, 0)
		return
	}

	// 记录调试日志
	zap.L().Debug("GetPostList", zap.Any("ids", ids))

	// 3. 使用 Preload 批量查询帖子及关联数据（作者、社区）
	// 从原来的 1 + N + N 次查询优化为 1 + 1 + 1 = 3 次查询
	posts, err := mysql.GetPostListByIDsWithPreload(ids)
	if err != nil {
		zap.L().Error("mysql.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	zap.L().Debug("GetPostListByIDsWithPreload", zap.Any("posts", posts))

	// 4. 使用 Pipeline 批量查询每个帖子的投票数据
	voteData, err := redis.GetPostsVoteData(ids)
	if err != nil {
		zap.L().Error("redis.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	for idx, post := range posts {
		// 安全检查：确保 Preload 成功加载了关联数据
		var authorName string
		if post.Author != nil {
			authorName = post.Author.Username
		} else {
			zap.L().Error("author not preloaded for post",
				zap.Int64("post_id", post.ID),
				zap.Int64("author_id", post.AuthorID))
			authorName = ""
		}

		// 组装最终数据
		postDetail := &response.PostDetailResponse{
			AuthorName: authorName,
			Post:       post,
			VoteNum:    voteData[idx], // 填充投票数
		}
		data = append(data, postDetail)
	}

	return
}

// GetCommunityPostList 根据社区ID获取帖子列表
// 优化版：使用 GORM Preload 预加载，解决 N+1 问题
func GetCommunityPostList(p *request.PostListRequest) (data []*response.PostDetailResponse, err error) {
	// 1. 从 Redis 查询指定社区的帖子 ID 列表 (已按时间或分数排序)
	ids, err := redis.GetCommunityPostIDsInOrder(p.CommunityID, p.Order, p.Page, p.Size)
	if err != nil {
		zap.L().Error("redis.GetCommunityPostIDsInOrder failed",
			zap.Int64("community_id", p.CommunityID),
			zap.String("order", p.Order),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 2. 处理空数据情况
	if len(ids) == 0 {
		zap.L().Info("GetCommunityPostList: no posts found",
			zap.Int64("community_id", p.CommunityID))
		// 返回空切片而不是 nil
		data = make([]*response.PostDetailResponse, 0)
		return data, nil
	}

	zap.L().Debug("GetCommunityPostList", zap.Any("ids", ids))

	// 3. 使用 Preload 批量查询帖子及关联数据（作者、社区）
	posts, err := mysql.GetPostListByIDsWithPreload(ids)
	if err != nil {
		zap.L().Error("mysql.GetPostListByIDsWithPreload failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 4. 使用 Pipeline 批量查询每个帖子的投票数据
	voteData, err := redis.GetPostsVoteData(ids)
	if err != nil {
		zap.L().Error("redis.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 5. 组装数据: 填充作者、社区、投票数据
	data = make([]*response.PostDetailResponse, 0, len(posts))
	for idx, post := range posts {
		// 安全检查：确保 Preload 成功加载了关联数据
		var authorName string
		if post.Author != nil {
			authorName = post.Author.Username
		} else {
			zap.L().Error("author not preloaded for post",
				zap.Int64("post_id", post.ID),
				zap.Int64("author_id", post.AuthorID))
			authorName = ""
		}

		// 组装最终数据
		postDetail := &response.PostDetailResponse{
			AuthorName: authorName,
			Post:       post,
			VoteNum:    voteData[idx], // 填充投票数
		}
		data = append(data, postDetail)
	}

	return data, nil
}

// DeletePost 删除帖子（软删除）
// 只有作者本人可以删除自己的帖子
func DeletePost(postID, userID int64) error {
	// 1. 查询帖子（已排除已删除的）
	post, err := mysql.GetPostByID(postID)
	if err != nil {
		zap.L().Error("mysql.GetPostByID failed",
			zap.Int64("post_id", postID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}
	if post == nil {
		return errorx.ErrNotFound
	}

	// 2. 验证权限（只有作者能删除）
	if post.AuthorID != userID {
		return errorx.ErrForbidden
	}

	// 3. 软删除（更新 status 为 0）
	err = mysql.DeletePostByAuthor(postID, userID)
	if err != nil {
		zap.L().Error("mysql.DeletePostByAuthor failed",
			zap.Int64("post_id", postID),
			zap.Int64("user_id", userID),
			zap.Error(err))
		return errorx.ErrServerBusy
	}

	// 4. 删除 Redis 数据（可选，保留投票数据供分析）
	// err = redis.DeletePost(postID, post.CommunityID)
	// if err != nil {
	//     zap.L().Warn("redis.DeletePost failed", ...)
	// }

	return nil
}
