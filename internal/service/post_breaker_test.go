package service

import (
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker"
	"github.com/stretchr/testify/assert"
)

// TestPostService_CircuitBreakerStateTransition 验证断路器在下游错误时的跳闸与快速失败
func TestPostService_CircuitBreakerStateTransition(t *testing.T) {
	svc := &PostService{}

	// 配置断路器：连续 3 次失败即熔断跳闸，冷却时间 200ms
	svc.dbBreaker = gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:        "TestBreaker",
		MaxRequests: 1,
		Interval:    2 * time.Second,
		Timeout:     200 * time.Millisecond,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 3
		},
	})

	breaker := svc.getDBBreaker()
	assert.Equal(t, gobreaker.StateClosed, breaker.State(), "初始状态应为 StateClosed (正常)")

	dbErr := errors.New("mysql connection refused")

	// 模拟连续 3 次 DB 故障
	for i := 0; i < 3; i++ {
		_, err := breaker.Execute(func() (interface{}, error) {
			return nil, dbErr
		})
		assert.ErrorIs(t, err, dbErr)
	}

	// 此时达到 3 次连续失败，断路器必须跳闸为 StateOpen
	assert.Equal(t, gobreaker.StateOpen, breaker.State(), "达到失败阈值后断路器必须跳闸为 StateOpen")

	// 再次调用时，必须直接由断路器拦截快速失败，绝不执行底层闭包！
	executed := false
	_, err := breaker.Execute(func() (interface{}, error) {
		executed = true
		return "data", nil
	})
	assert.ErrorIs(t, err, gobreaker.ErrOpenState, "处于 Open 状态时必须返回 ErrOpenState")
	assert.False(t, executed, "熔断状态下绝不应该执行底层 DB 操作")

	// 等待 250ms 冷却时间过后，断路器应进入 StateHalfOpen (半开状态)
	time.Sleep(250 * time.Millisecond)
	assert.Equal(t, gobreaker.StateHalfOpen, breaker.State(), "冷却期过后应当进入 StateHalfOpen 半开试探")

	// 半开状态下试探成功 -> 应当自动自愈恢复为 StateClosed
	_, err = breaker.Execute(func() (interface{}, error) {
		return "recovered_data", nil
	})
	assert.NoError(t, err)
	assert.Equal(t, gobreaker.StateClosed, breaker.State(), "试探请求成功后必须自动恢复为 StateClosed")
}
