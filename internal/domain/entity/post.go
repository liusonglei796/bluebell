// Package entity 定义领域实体
//
// 领域实体是 DDD 中的核心概念，它们：
// 1. 拥有唯一标识（ID）
// 2. 包含业务属性和行为
// 3. 不依赖任何基础设施（GORM、Redis 等）
// 4. 是业务逻辑的核心载体
package entity

import (
	"strings"
	"time"
)

// 帖子状态常量
const (
	PostStatusPublished = 1  // 已发布
	PostStatusDeleted   = 0  // 已删除（软删除）
)

// Post 帖子领域实体
type Post struct {
	PostID      string
	AuthorID    int64
	CommunityID int64
	PostTitle   string
	Content     string
	Status      int8
	CreatedAt   time.Time
	Author      *User
	Community   *Community
}

// Validate 校验帖子内容是否合法
func (p *Post) Validate() error {
	if p == nil || p.PostID == "" {
		return ErrInvalidParam
	}
	if strings.TrimSpace(p.PostTitle) == "" || strings.TrimSpace(p.Content) == "" {
		return ErrInvalidParam
	}
	return nil
}

// IsValid 检查帖子数据完整性（保留以兼容旧代码）
func (p *Post) IsValid() bool {
	return p.Validate() == nil
}

// HasAuthor 检查帖子是否有关联的作者信息
func (p *Post) HasAuthor() bool {
	return p.Author != nil && p.Author.UserID != 0
}

// HasCommunity 检查帖子是否有关联的社区信息
func (p *Post) HasCommunity() bool {
	return p.Community != nil && p.Community.ID != 0
}

// CanBeDeletedBy 校验指定用户是否有权删除此帖子
// 核心业务规则：只有帖子的作者才能删除自己的帖子
func (p *Post) CanBeDeletedBy(userID int64) error {
	if p.AuthorID != userID {
		return ErrForbidden
	}
	return nil
}

// IsPublished 判断帖子是否处于已发布状态
func (p *Post) IsPublished() bool {
	return p.Status == PostStatusPublished
}
