package port

// IDGenerator ID 生成器端口
//
// 应用层通过此接口生成全局唯一 ID，不直接依赖 Snowflake / UUID 等具体实现。
// 基础设施层提供适配器（如 infrastructure/snowflake）。
type IDGenerator interface {
	// GenID 生成一个全局唯一的 int64 ID
	GenID() int64
}
