// Package model 提供数据库模型与业务规则（MVC 的 M 层）
//
// 模型同时承担数据映射（GORM）与核心业务校验规则，业务逻辑集中在模型方法中，
// 服务层负责流程编排，控制器负责 HTTP 交互。
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"gorm.io/gorm"
)

// 帖子状态常量
const (
	PostStatusPublished = 1 // 已发布
	PostStatusDeleted   = 0 // 已删除（软删除）
)

// Post 帖子模型（内存对齐优化建议：把相同类型的字段放在一起，宽字段如 int64, string 放在前面）
// 对应数据库 post 表
type Post struct {
	gorm.Model
	PostID        string     `gorm:"column:post_id;not null;uniqueIndex;size:255"`
	CommunityID   int64      `gorm:"column:community_id"`
	PostTitle     string     `gorm:"column:post_title;not null;type:text"`
	AuthorName    string     `gorm:"column:author_name;type:varchar(64);not null;default:''"`
	CommunityName string     `gorm:"column:community_name;type:varchar(128);not null;default:''"`
	TagNames      string     `gorm:"column:tag_names;type:varchar(255);not null;default:''"`
	Authors       []User     `gorm:"many2many:post_author;"`
	Community     *Community
	Tags          []Tag      `gorm:"many2many:post_tag;foreignKey:PostID;joinForeignKey:PostID;References:ID;joinReferences:TagID"`
	Content       string     `gorm:"column:content;type:text;not null"`
	ContentHash   string     `gorm:"column:content_hash;size:64;not null;uniqueIndex:idx_community_hash"`
	Status        int8       `gorm:"column:status;default:1"`
	IsPinned      int8       `gorm:"column:is_pinned;not null;default:0"`
	IsHighlighted int8       `gorm:"column:is_highlighted;not null;default:0"`
	BookmarkCount int        `gorm:"column:bookmark_count;not null;default:0"`
	CommentCount  int        `gorm:"column:comment_count;not null;default:0"`
}

// TableName 自定义表名
func (Post) TableName() string {
	return "post"
}

// ComputeContentHash 计算内容指纹：SHA256(title + content)，取前 32 字符
func (p *Post) ComputeContentHash() string {
	h := sha256.Sum256([]byte(p.PostTitle + p.Content))
	return hex.EncodeToString(h[:16])
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

// IsValid 检查帖子数据完整性
func (p *Post) IsValid() bool {
	return p.Validate() == nil
}

// HasAuthors 检查帖子是否有关联的作者信息
func (p *Post) HasAuthors() bool {
	return len(p.Authors) > 0
}

// HasCommunity 检查帖子是否有关联的社区信息
func (p *Post) HasCommunity() bool {
	return p.Community != nil && p.Community.ID != 0
}

// CanBeDeletedBy 校验指定用户是否有权删除此帖子
// 核心业务规则：只有帖子的作者之一才能删除帖子
func (p *Post) CanBeDeletedBy(userID int64) error {
	for _, a := range p.Authors {
		if a.UserID == userID {
			return nil
		}
	}
	return ErrForbidden
}

// IsPublished 判断帖子是否处于已发布状态
func (p *Post) IsPublished() bool {
	return p.Status == PostStatusPublished
}
