package database

import (
	// DAO 层 - MySQL 数据库访问
	"bluebell/internal/infrastructure/persistence/mysql/communitydb"
	"bluebell/internal/infrastructure/persistence/mysql/postdb"
	"bluebell/internal/infrastructure/persistence/mysql/socialdb"
	"bluebell/internal/infrastructure/persistence/mysql/userdb"
	"bluebell/internal/infrastructure/persistence/mysql/votedb"

	// 领域层 - Repository 接口
	"bluebell/internal/domain"

	"gorm.io/gorm"
)

// Repositories 聚合所有 MySQL 仓储实例
type Repositories struct {
	Post      domain.PostRepository
	Community domain.CommunityRepository
	User      domain.UserRepository
	Vote      domain.VoteRepository
	Remark    domain.RemarkRepository
	Social    domain.SocialRepository
}

// NewRepositories 创建 Repositories 实例
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Post:      postdb.NewPostRepo(db),
		Remark:    postdb.NewRemarkRepo(db),
		Community: communitydb.NewCommunityRepo(db),
		User:      userdb.NewUserRepo(db),
		Vote:      votedb.NewVoteRepo(db),
		Social:    socialdb.NewSocialRepo(db),
	}
}
