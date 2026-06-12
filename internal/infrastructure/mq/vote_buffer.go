package mq

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// VoteBuffer 投票本地缓冲区
type VoteBuffer struct {
	votes         map[string]int8 // key: postID:userID, value: direction
	flushFunc     func(ctx context.Context, votes map[string]int8) error
	flushInterval time.Duration
	mu            sync.Mutex
}

// NewVoteBuffer 创建一个新的投票缓冲区
func NewVoteBuffer(interval time.Duration, flushFunc func(context.Context, map[string]int8) error) *VoteBuffer {
	return &VoteBuffer{
		votes:         make(map[string]int8),
		flushInterval: interval,
		flushFunc:     flushFunc,
	}
}

// AddVote 添加投票到缓冲区
func (vb *VoteBuffer) AddVote(postID, userID string, direction int8) {
	vb.mu.Lock()
	defer vb.mu.Unlock()
	key := fmt.Sprintf("%s:%s", postID, userID)
	vb.votes[key] = direction
}

// Start 启动定时刷新
func (vb *VoteBuffer) Start(ctx context.Context) {
	ticker := time.NewTicker(vb.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			vb.flush(ctx)
		case <-ctx.Done():
			// 退出前执行最后一次刷新
			vb.flush(context.Background())
			return
		}
	}
}

// flush 执行刷新逻辑
func (vb *VoteBuffer) flush(ctx context.Context) {
	vb.mu.Lock()
	if len(vb.votes) == 0 {
		vb.mu.Unlock()
		return
	}
	// 复制当前的投票数据并清空原 map
	votesToFlush := vb.votes
	vb.votes = make(map[string]int8)
	vb.mu.Unlock()

	// 调用外部刷新函数
	if err := vb.flushFunc(ctx, votesToFlush); err != nil {
		// 如果失败，目前简单的记录日志（或者可以在这里考虑重试逻辑，但根据任务要求暂不复杂化）
		// 在实际生产中，可能需要将失败的数据重新放回缓冲区或记录到专门的错误日志中
		fmt.Printf("failed to flush votes: %v\n", err)
	}
}
