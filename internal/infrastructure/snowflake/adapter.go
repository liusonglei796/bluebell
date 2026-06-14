package snowflake

import (
	"bluebell/internal/application/port"
)

// idGenerator 是 port.IDGenerator 的 Snowflake 适配器
type idGenerator struct{}

// NewIDGenerator 创建 port.IDGenerator 的适配器实例
func NewIDGenerator() port.IDGenerator {
	return &idGenerator{}
}

func (g *idGenerator) GenID() int64 {
	return GenID()
}
