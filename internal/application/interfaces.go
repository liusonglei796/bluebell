package application

import (
	postreq "bluebell/internal/application/dto/request/post"
	votereq "bluebell/internal/application/dto/request/vote"
	voteresp "bluebell/internal/application/dto/response/vote"

	"context"
)




// ========== Vote Service 接口 ==========

// VoteService 投票与排行榜业务逻辑服务接口
type VoteService interface {
	// VoteForPost 为帖子投票
	VoteForPost(ctx context.Context, userID int64, p *postreq.VoteRequest) error
	// GetLeaderboard 获取排行榜
	GetLeaderboard(ctx context.Context, p *votereq.LeaderboardRequest) (*voteresp.LeaderboardResponse, error)
}

