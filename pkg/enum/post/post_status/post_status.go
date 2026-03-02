package post_status

// 帖子状态
// NORMAL: 正常（可见）
// DELETED: 已删除（软删除）
// HIDDEN: 已隐藏（管理员操作）
const (
	NORMAL  = 1
	DELETED = 0
	HIDDEN  = 2
)
