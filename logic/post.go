package logic

import (
	"bluebell/dao/mysql"
	"bluebell/dao/redis"
	"bluebell/models"
	"bluebell/pkg/errorx"
	"bluebell/pkg/snowflake"

	"go.uber.org/zap"
)

// CreatePost 创建帖子,返回新创建的帖子ID
func CreatePost(p *models.ParamPost) (postID int64, err error) {
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
func GetPostByID(pid int64) (data *models.ApiPostDetail, err error) {
	// 1. 查询帖子信息
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

	// 3. 查询作者信息
	user, err := mysql.GetUserByID(post.AuthorID)
	if err != nil {
		// 系统错误
		zap.L().Error("mysql.GetUserByID failed",
			zap.Int64("author_id", post.AuthorID),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 检查是否找到了用户
	if user == nil || user.UserID == 0 {
		zap.L().Warn("user not found", zap.Int64("author_id", post.AuthorID))
		// 业务错误：作者不存在
		return nil, errorx.ErrNotFound
	}

	// 4. 查询社区信息
	community, err := mysql.GetCommunityDetailByID(post.CommunityID)
	if err != nil {
		// 系统错误
		zap.L().Error("mysql.GetCommunityDetailByID failed",
			zap.Int64("community_id", post.CommunityID),
			zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 检查是否找到了社区
	if community == nil || community.ID == 0 {
		zap.L().Warn("community not found", zap.Int64("community_id", post.CommunityID))
		// 业务错误：社区不存在
		return nil, errorx.ErrNotFound
	}

	// 5. 组装数据
	data = &models.ApiPostDetail{
		Post:            post,
		AuthorName:      user.Username,
		CommunityDetail: community,
	}

	return data, nil
}

// 从 Redis 获取排序后的 ID，再从 MySQL 查询详情，最后组装投票数据
func GetPostList(p *models.ParamPostList) (data []*models.ApiPostDetail, err error) {
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
		data = make([]*models.ApiPostDetail, 0)
		return
	}

	// 记录调试日志
	zap.L().Debug("GetPostList2", zap.Any("ids", ids))

	// 3. 根据 ID 列表从 MySQL 查询帖子详细信息（保持顺序）
	posts, err := mysql.GetPostListByIDs(ids)
	if err != nil {
		zap.L().Error("mysql.GetPostListByIDs failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	zap.L().Debug("GetPostListByIDs", zap.Any("posts", posts))

	// 4. 使用 Pipeline 批量查询每个帖子的投票数据
	voteData, err := redis.GetPostsVoteData(ids)
	if err != nil {
		zap.L().Error("redis.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 5. 收集所有用户ID和社区ID
	userIDs := make([]int64, 0, len(posts))
	communityIDs := make([]int64, 0, len(posts))
	
	for _, post := range posts {
		userIDs = append(userIDs, post.AuthorID)
		communityIDs = append(communityIDs, post.CommunityID)
	}

	// 6. 批量查询用户信息
	users, err := mysql.GetUsersByIDs(userIDs)
	if err != nil {
		zap.L().Error("mysql.GetUsersByIDs failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}
	
	// 构建用户ID到用户名的映射
	userMap := make(map[int64]string, len(users))
	for _, user := range users {
		userMap[user.UserID] = user.Username
	}

	// 7. 批量查询社区信息
	communities, err := mysql.GetCommunitiesByIDs(communityIDs)
	if err != nil {
		zap.L().Error("mysql.GetCommunitiesByIDs failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}
	
	// 构建社区ID到社区详情的映射
	communityMap := make(map[int64]*models.CommunityDetail, len(communities))
	for _, community := range communities {
		communityMap[community.ID] = community
	}

	// 8. 组装数据：填充作者、社区、投票数据
	data = make([]*models.ApiPostDetail, 0, len(posts))
	for idx, post := range posts {
		// 从映射中获取作者名和社区详情
		authorName, ok := userMap[post.AuthorID]
		if !ok {
			zap.L().Error("user not found for post", zap.Int64("author_id", post.AuthorID))
			authorName = "" // 设置默认值
		}
		
		community, ok := communityMap[post.CommunityID]
		if !ok {
			zap.L().Error("community not found for post", zap.Int64("community_id", post.CommunityID))
			community = &models.CommunityDetail{} // 设置默认值
		}

		// 组装最终数据
		postDetail := &models.ApiPostDetail{
			AuthorName:      authorName,
			CommunityDetail: community,
			Post:            post,
			VoteNum:         voteData[idx], // 填充投票数
		}
		data = append(data, postDetail)
	}

	return
}

// GetCommunityPostList 根据社区ID获取帖子列表 (优化版: 解决N+1问题)
func GetCommunityPostList(p *models.ParamPostList) (data []*models.ApiPostDetail, err error) {
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
		data = make([]*models.ApiPostDetail, 0)
		return data, nil
	}

	zap.L().Debug("GetCommunityPostList", zap.Any("ids", ids))

	// 3. 根据 ID 列表从 MySQL 批量查询帖子详细信息 (保持顺序)
	posts, err := mysql.GetPostListByIDs(ids)
	if err != nil {
		zap.L().Error("mysql.GetPostListByIDs failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 4. 使用 Pipeline 批量查询每个帖子的投票数据
	voteData, err := redis.GetPostsVoteData(ids)
	if err != nil {
		zap.L().Error("redis.GetPostsVoteData failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 5. 收集所有唯一的用户ID和社区ID
	// 注意: 同一个社区的所有帖子社区ID相同,但作者可能不同
	userIDs := make([]int64, 0, len(posts))
	communityIDs := make([]int64, 0, 1) // 社区ID只有一个

	// 用于去重用户ID
	userIDSet := make(map[int64]struct{})

	for _, post := range posts {
		// 用户ID去重
		if _, exists := userIDSet[post.AuthorID]; !exists {
			userIDSet[post.AuthorID] = struct{}{}
			userIDs = append(userIDs, post.AuthorID)
		}

		// 社区ID (理论上所有帖子的社区ID应该相同)
		if len(communityIDs) == 0 || communityIDs[0] != post.CommunityID {
			communityIDs = append(communityIDs, post.CommunityID)
		}
	}

	// 6. 批量查询用户信息
	users, err := mysql.GetUsersByIDs(userIDs)
	if err != nil {
		zap.L().Error("mysql.GetUsersByIDs failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 构建用户ID到用户名的映射
	userMap := make(map[int64]string, len(users))
	for _, user := range users {
		userMap[user.UserID] = user.Username
	}

	// 7. 批量查询社区信息
	communities, err := mysql.GetCommunitiesByIDs(communityIDs)
	if err != nil {
		zap.L().Error("mysql.GetCommunitiesByIDs failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}

	// 构建社区ID到社区详情的映射
	communityMap := make(map[int64]*models.CommunityDetail, len(communities))
	for _, community := range communities {
		communityMap[community.ID] = community
	}

	// 8. 组装数据: 填充作者、社区、投票数据
	data = make([]*models.ApiPostDetail, 0, len(posts))
	for idx, post := range posts {
		// 从映射中获取作者名和社区详情
		authorName, ok := userMap[post.AuthorID]
		if !ok {
			zap.L().Error("user not found for post", zap.Int64("author_id", post.AuthorID))
			authorName = "" // 设置默认值
		}

		community, ok := communityMap[post.CommunityID]
		if !ok {
			zap.L().Error("community not found for post", zap.Int64("community_id", post.CommunityID))
			community = &models.CommunityDetail{} // 设置默认值
		}

		// 组装最终数据
		postDetail := &models.ApiPostDetail{
			AuthorName:      authorName,
			CommunityDetail: community,
			Post:            post,
			VoteNum:         voteData[idx], // 填充投票数
		}
		data = append(data, postDetail)
	}

	return data, nil
}

// GetPostListNew 是一个新的、统一的帖子列表获取函数
// 它充当一个"调度器"或"分发器"
func GetPostListNew(p *models.ParamPostList) (data []*models.ApiPostDetail, err error) {
	
	// 关键判断：根据 CommunityID 是否为 0，来决定执行哪种查询逻辑
	if p.CommunityID == 0 {
		// 1. CommunityID 为 0 (或未提供)
		// 执行"查询所有帖子"的逻辑
		// GetPostList2 是原有的、用于获取所有帖子的逻辑函数 (视频中已存在)
		data, err = GetPostList(p)
	} else {
		// 2. CommunityID 不为 0 (已提供)
		// 执行"根据社区ID查询帖子"的逻辑
		// GetCommunityPostList 是原有的、用于按社区ID获取帖子的逻辑函数 (视频中已存在)
		data, err = GetCommunityPostList(p)
	}
	
	// 统一的错误处理
	if err != nil {
		// 记录日志，方便排查问题
		zap.L().Error("logic.GetPostListNew failed", zap.Error(err))
		return nil, errorx.ErrServerBusy
	}
	
	// 成功则返回数据和 nil 错误
	return data, nil
}
