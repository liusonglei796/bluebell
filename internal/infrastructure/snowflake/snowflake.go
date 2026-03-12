package snowflake

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"
	"sync"
	"time"

	// 建议给引入的包起个别名 sf，防止和你自己的包名 snowflake 冲突
	sf "github.com/bwmarrin/snowflake"
	"go.uber.org/zap"
)

var (
	node          *sf.Node
	mu            sync.Mutex
	lastTimestamp int64 // 上次生成 ID 的毫秒时间戳
)

// Init 初始化雪花算法节点
func Init(cfg *config.Config) (err error) {
	// 设置 Epoch（直接使用毫秒时间戳）
	sf.Epoch = cfg.Snowflake.StartTime

	// 创建节点
	node, err = sf.NewNode(cfg.Snowflake.MachineID)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "初始化雪花算法节点失败, machineID=%d", cfg.Snowflake.MachineID)
	}
	return
}

// GenID 生成 ID（带时钟回拨保护）
func GenID() int64 {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now().UnixMilli()

	// 核心防御逻辑：检查当前时间是否小于上次生成 ID 的时间
	if now < lastTimestamp {
		offset := lastTimestamp - now
		zap.L().Warn("clock backwards detected", zap.Int64("offset_ms", offset))

		// 策略 1：如果是小范围回拨（如 10ms 内），尝试休眠等待时钟追赶
		if offset <= 10 {
			time.Sleep(time.Duration(offset+1) * time.Millisecond)
			now = time.Now().UnixMilli()
		}

		// 策略 2：如果休眠后依然回拨，或者回拨幅度过大，直接报错/抛出异常
		// 为什么：防止产生重复 ID 或 ID 乱序，保证分布式环境下的一致性
		if now < lastTimestamp {
			zap.L().Fatal("clock moved backwards too far, cannot generate ID",
				zap.Int64("last_timestamp", lastTimestamp),
				zap.Int64("current_now", now))
		}
	}

	lastTimestamp = now
	return node.Generate().Int64()
}
