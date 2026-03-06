package serviceinterface

import (
	"bluebell/internal/dto/request"
	"context"
)

// VoteService 投票业务逻辑服务接口
type VoteService interface {
	// VoteForPost 为帖子投票
	VoteForPost(ctx context.Context, userID int64, p *request.VoteRequest) error
}
