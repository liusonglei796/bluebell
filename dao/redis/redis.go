package redis

import (
	"context"
	"fmt"
	"time"

	"bluebell/settings"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// 定义全局的 Redis 客户端
// 为什么：redis.Client 是线程安全的，整个应用共享一个连接池即可
var (
	rdb *redis.Client
	ctx = context.Background() // 全局上下文,用于 Redis 操作
)

// Init 初始化 Redis 连接
// 为什么:建立 Redis 连接池,确保应用启动时缓存服务可用
func Init(cfg *settings.RedisConfig) error {
	if cfg == nil {
		return fmt.Errorf("redis config is nil")
	}

	// 创建 Redis 客户端
	// 为什么:配置连接地址、密码、数据库编号和连接池大小
	rdb = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,
	})

	// 创建带超时的 context
	// 为什么:防止 Redis 连接卡死导致程序启动失败
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ping 验证连接(带超时控制)
	if err := rdb.Ping(pingCtx).Err(); err != nil {
		return fmt.Errorf("connect to redis failed: %w", err)
	}

	// 连接成功,打印日志
	// 记录地址和 DB 号,方便确认连接的是哪个库
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
