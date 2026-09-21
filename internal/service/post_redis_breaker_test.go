package service

import (
	"errors"
	"testing"
	"time"

	"bluebell/internal/model"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
)

// TestPostVote_RedisBreakerStateTransition 验证投票链路 Redis 故障时的断路器跳闸、快速失败与冷却自愈
func TestPostVote_RedisBreakerStateTransition(t *testing.T) {
	svc := &PostService{}

	// 自定义测试断路器：连续 3 次失败即跳闸，冷却时间 200ms
	svc.redisBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "TestRedisBreaker",
		MaxRequests: 1,
		Interval:    2 * time.Second,
		Timeout:     200 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	breaker := svc.getRedisBreaker()
	assert.Equal(t, gobreaker.StateClosed, breaker.State(), "初始状态必须为 StateClosed (正常)")

	redisDownErr := errors.New("dial tcp 127.0.0.1:6379: connectex: connection refused")

	// 模拟连续 3 次底层 Redis 连接故障
	for i := 0; i < 3; i++ {
		_, err := breaker.Execute(func() (interface{}, error) {
			return nil, redisDownErr
		})
		assert.ErrorIs(t, err, redisDownErr)
	}

	// 达到连续 3 次失败阈值，断路器必须跳闸为 StateOpen
	assert.Equal(t, gobreaker.StateOpen, breaker.State(), "Redis连续故障后，断路器必须跳闸为 StateOpen")

	// 再次尝试执行时，断路器必须立即快速拦截 (Fast-Fail)，不执行底层操作
	attempted := false
	_, err := breaker.Execute(func() (interface{}, error) {
		attempted = true
		return nil, nil
	})
	assert.ErrorIs(t, err, gobreaker.ErrOpenState, "处于 Open 状态时必须快速失败返回 ErrOpenState")
	assert.False(t, attempted, "熔断状态下绝不应再向已故障的 Redis 发送请求")

	// 等待 250ms 冷却时间后，状态应自动切换为 StateHalfOpen (半开试探)
	time.Sleep(250 * time.Millisecond)
	assert.Equal(t, gobreaker.StateHalfOpen, breaker.State(), "冷却期过后应当进入 StateHalfOpen 半开试探")

	// 试探请求成功，断路器自动闭合恢复正常
	_, err = breaker.Execute(func() (interface{}, error) {
		return "ok", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, gobreaker.StateClosed, breaker.State(), "试探成功后断路器自动恢复 Closed")
}

// TestPostVote_RedisBreakerIgnoresBusinessErrors 验证业务预期错误（如重复投票/过期）不会导致断路器误跳闸
func TestPostVote_RedisBreakerIgnoresBusinessErrors(t *testing.T) {
	svc := &PostService{}

	svc.redisBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "TestRedisBreakerBusiness",
		MaxRequests: 1,
		Interval:    2 * time.Second,
		Timeout:     200 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	breaker := svc.getRedisBreaker()

	// 模拟连续 10 次业务预期异常（如重复投票或时效过期）
	for i := 0; i < 10; i++ {
		var bizErr error
		_, err := breaker.Execute(func() (interface{}, error) {
			// 模拟 postCache.VoteForPost 返回业务错误
			mockErr := model.ErrVoteRepeated
			if errors.Is(mockErr, model.ErrVoteTimeExpire) || errors.Is(mockErr, model.ErrVoteRepeated) {
				bizErr = mockErr
				return nil, nil // 业务错误向断路器返回 nil，不计入系统故障
			}
			return nil, mockErr
		})

		assert.NoError(t, err, "断路器不应记录错误")
		assert.Equal(t, model.ErrVoteRepeated, bizErr)
	}

	// 验证断路器依然稳固保持在 StateClosed 状态，未被误触发跳闸
	assert.Equal(t, gobreaker.StateClosed, breaker.State(), "业务预期错误绝不应导致 Redis 断路器误跳闸")
}
