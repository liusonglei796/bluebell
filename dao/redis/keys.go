package redis

const (
	KeyPrefix           = "bluebell:"
	KeyUserAccessToken  = "active_access_token:"  // bluebell:active_access_token:1001
	KeyUserRefreshToken = "active_refresh_token:" // bluebell:active_refresh_token:1001
    KeyPostTimeZSet="post:time"
	KeyPostScoreZSet="post:score"
	KeyPostVotedZSetPrefix="post:voted:"
)

func getRedisKey(key string) string {
	return KeyPrefix + key
}
