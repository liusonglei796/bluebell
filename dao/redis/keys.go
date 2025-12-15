package redis

const (
	KeyPrefix              = "bluebell:"
	KeyUserAccessToken     = "active_access_token:"  // bluebell:active_access_token:1001
	KeyUserRefreshToken    = "active_refresh_token:" // bluebell:active_refresh_token:1001
	KeyPostTimeZSet        = "post:time"             // bluebell:post:time - 所有帖子按时间排序
	KeyPostScoreZSet       = "post:score"            // bluebell:post:score - 所有帖子按分数排序
	KeyPostVotedZSetPrefix = "post:voted:"           // bluebell:post:voted:{postID} - 帖子的投票记录

	// 社区维度的帖子排序 Key
	// bluebell:community:post:time:{communityID} - 特定社区的帖子按时间排序
	KeyCommunityPostTimePrefix  = "community:post:time:"
	// bluebell:community:post:score:{communityID} - 特定社区的帖子按分数排序
	KeyCommunityPostScorePrefix = "community:post:score:"
)

func getRedisKey(key string) string {
	return KeyPrefix + key
}
