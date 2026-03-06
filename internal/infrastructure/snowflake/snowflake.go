package snowflake

import (
	"bluebell/internal/config"
	"bluebell/pkg/errorx"

	// 建议给引入的包起个别名 sf，防止和你自己的包名 snowflake 冲突
	sf "github.com/bwmarrin/snowflake"
)

// 定义一个包级别的全局变量，这样 GetID 才能访问到它
// 为什么：snowflake 算法需要维护状态（如序列号），全局单例节点能保证 ID 的唯一性
var node *sf.Node

// Init 初始化雪花算法节点
// 为什么：需要设置起始时间（Epoch）和机器 ID，确保生成的 ID 在分布式环境中唯一且有序
func Init(cfg *config.Config) (err error) {
	// 设置 Epoch (起始时间)，单位是毫秒
	// 为什么：雪花算法生成的 ID 包含时间戳部分，是相对于 Epoch 的偏移量
	sf.Epoch = cfg.Snowflake.StartTime.UnixNano() / 1000000

	// 创建节点
	// 为什么：每个服务实例需要唯一的 machineID，防止多实例生成重复 ID
	node, err = sf.NewNode(cfg.Snowflake.MachineID)
	if err != nil {
		return errorx.Wrapf(err, errorx.CodeInfraError, "初始化雪花算法节点失败, machineID=%d", cfg.Snowflake.MachineID)
	}
	return
}

// GenID 生成 ID
func GenID() int64 {
	return node.Generate().Int64()
}
