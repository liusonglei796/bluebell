package constants

// ==================== Redis 缓存 Key 前缀 ====================
// 统一使用 `:` 作为分隔符，格式: {模块}:{子模块}:{标识}
const (
	// Key 前缀
	CacheKeyPrefix = "bluebell:"

	// 用户相关
	CacheKeyUserAccessToken  = "user:access_token:"  // 用户 Access Token   bluebell:user:access_token:{userID}
	CacheKeyUserRefreshToken = "user:refresh_token:" // 用户 Refresh Token  bluebell:user:refresh_token:{userID}

	// 帖子相关
	CacheKeyPostTime  = "post:time"   // 全局帖子按时间排序 (ZSET)
	CacheKeyPostScore = "post:score"  // 全局帖子按分数排序 (ZSET)
	CacheKeyPostVoted = "post:voted:" // 帖子投票记录       bluebell:post:voted:{postID}

	// 社区帖子相关
	CacheKeyCommunityPostTime  = "community:post:time:"  // 社区帖子按时间排序 bluebell:community:post:time:{communityID}
	CacheKeyCommunityPostScore = "community:post:score:" // 社区帖子按分数排序 bluebell:community:post:score:{communityID}
)
const ScorePerVote = 432 // 每票对应的分数权重: 86400/200
