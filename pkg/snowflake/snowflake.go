package snowflake

import (
	"time"

	// 建议给引入的包起个别名 sf，防止和你自己的包名 snowflake 冲突
	sf "github.com/bwmarrin/snowflake"
)

// 定义一个包级别的全局变量，这样 GetID 才能访问到它
// 为什么：snowflake 算法需要维护状态（如序列号），全局单例节点能保证 ID 的唯一性
var node *sf.Node

// Init 初始化雪花算法节点
// 为什么：需要设置起始时间（Epoch）和机器 ID，确保生成的 ID 在分布式环境中唯一且有序
func Init(startTime string, machineID int64) (err error) {
	var st time.Time
	// ⚠️ 关键修改：Go 语言的时间模版必须是 "2006-01-02" (代表 YYYY-MM-DD)
	// 不能乱写成 2005-10-28
	// 为什么：Go 语言使用特定的参考时间来定义格式，必须严格遵守
	st, err = time.Parse("2006-01-02", startTime)
	if err != nil {
		return
	}
	// 设置 Epoch (起始时间)，单位是毫秒
	// 为什么：雪花算法生成的 ID 包含时间戳部分，是相对于 Epoch 的偏移量
	sf.Epoch = st.UnixNano() / 1000000

	// 创建节点
	// 为什么：每个服务实例需要唯一的 machineID，防止多实例生成重复 ID
	node, err = sf.NewNode(machineID)
	if err != nil {
		return
	}
	return
}

// GetID 生成 ID
// 为什么：对外提供简单的接口获取唯一 ID
func GetID() int64 {
	// Generate() 返回的是 ID 类型，转换为 int64 需要用 Int64() 方法（首字母大写）
	return node.Generate().Int64()
}

// GenID 生成 ID 的别名函数，与教学文档保持一致
func GenID() int64 {
	return GetID()
}
