package handler

import (
	"bluebell/internal/service"
)

// CtxUserIDKey 定义 Context 中 UserID 的 Key
const CtxUserIDKey = "userID"

// Handlers 聚合所有 Handler 方法
// 持有 Services 引用，作为 Handler 层的入口
type Handlers struct {
	Services *service.Services
}

// NewHandlers 创建 Handlers 实例
func NewHandlers(services *service.Services) *Handlers {
	return &Handlers{Services: services}
}
