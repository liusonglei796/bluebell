package redis

import (
	"context"
	"fmt"

	"bluebell/settings" // 替换你的包名

	"github.com/redis/go-redis/v9" // 假设使用 v9，如果是 v8 用法也一样
	"go.uber.org/zap"
)

// 定义全局的 Redis 客户端
// 为什么：redis.Client 是线程安全的，整个应用共享一个连接池即可
var (
	rdb *redis.Client
	ctx = context.Background() // 全局上下文,用于 Redis 操作
)

// Init 初始化 Redis 连接
// 为什么：建立 Redis 连接池，确保应用启动时缓存服务可用
func Init(cfg *settings.RedisConfig) error {
	if cfg == nil {
		return fmt.Errorf("redis config is nil")
	}

	// 创建 Redis 客户端
	// 为什么：配置连接地址、密码、数据库编号和连接池大小
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})
	// Ping 需要 context (使用全局 ctx)
	// 【优化点 1】：去掉 zap.L().Error
	// 如果连接失败，直接返回包装后的 error，交给 main 函数处理
	// 为什么：Ping 操作可以验证连接是否真正建立成功
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("connect to redis failed: %w", err)
	}

	// 【优化点 2】：连接成功，打印一条 Info 日志
	// 记录地址和 DB 号，方便确认连接的是哪个库
	zap.L().Info("init redis success",
		zap.String("addr", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		zap.Int("db", cfg.DB),
	)

	return nil
}

// Close 关闭 Redis 连接
// 为什么：程序退出时释放资源
func Close() {
	_ = rdb.Close()
}
