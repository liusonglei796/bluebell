package sqltrace

import (
	"context"
	"sync"
	"time"
)

// Item 单条 SQL 的执行追踪记录
type Item struct {
	SQL      string        `json:"sql"`
	Duration time.Duration `json:"duration"`
	Rows     int64         `json:"rows"`
	Error    string        `json:"error,omitempty"`
}

// Tracker 单个请求内的 SQL 统计追踪器
type Tracker struct {
	mu     sync.Mutex
	Traces []Item
}

// Add 记录一条 SQL 执行
func (t *Tracker) Add(sql string, d time.Duration, rows int64, err error) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	t.Traces = append(t.Traces, Item{
		SQL:      sql,
		Duration: d,
		Rows:     rows,
		Error:    errStr,
	})
}

// TotalCost 计算所有 SQL 累计执行耗时
func (t *Tracker) TotalCost() time.Duration {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	var total time.Duration
	for _, item := range t.Traces {
		total += item.Duration
	}
	return total
}

// Count 获取执行的 SQL 条数
func (t *Tracker) Count() int {
	if t == nil {
		return 0
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.Traces)
}

// GetTraces 获取所有记录快照
func (t *Tracker) GetTraces() []Item {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	res := make([]Item, len(t.Traces))
	copy(res, t.Traces)
	return res
}

type trackerKey struct{}

// WithTracker 将 Tracker 存入 Context
func WithTracker(ctx context.Context, t *Tracker) context.Context {
	return context.WithValue(ctx, trackerKey{}, t)
}

// FromContext 从 Context 提取 Tracker
func FromContext(ctx context.Context) *Tracker {
	if ctx == nil {
		return nil
	}
	if t, ok := ctx.Value(trackerKey{}).(*Tracker); ok {
		return t
	}
	return nil
}
