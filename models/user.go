package models

// User 用户模型
// 为什么：对应数据库中的 user 表结构，用于 ORM 映射
type User struct {
	// db:"user_id" 标签用于 sqlx 库将数据库列名映射到结构体字段
	UserID   int64  `json:"user_id,string" db:"user_id"`
	Username string `db:"username"`
	Password string `db:"password"`
}
