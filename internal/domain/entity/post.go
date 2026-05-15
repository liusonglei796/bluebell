// Package entity 定义领域实体
//
// 领域实体是 DDD 中的核心概念，它们：
// 1. 拥有唯一标识（ID）
// 2. 包含业务属性和行为
// 3. 不依赖任何基础设施（GORM、Redis 等）
// 4. 是业务逻辑的核心载体
package entity

// Post 帖子领域实体
type Post struct {
	PostID      string
	AuthorID    int64
	CommunityID int64
	PostTitle   string
	Content     string
	Status      int8
	Author      *User
	Community   *Community
}
