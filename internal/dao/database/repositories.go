package database

import (
	// DAO 层 - MySQL 数据库访问
	"bluebell/internal/dao/database/communitydb"
	"bluebell/internal/dao/database/postdb"
	"bluebell/internal/dao/database/userdb"
	"bluebell/internal/dao/database/votedb"

	// 领域层 - Repository 接口
	"bluebell/internal/domain/dbdomain"

	"gorm.io/gorm"
)

// Repositories 聚合所有 MySQL 仓储实例
type Repositories struct {
	Post      dbdomain.PostRepository
	Community dbdomain.CommunityRepository
	User      dbdomain.UserRepository
	Vote      dbdomain.VoteRepository
}

// NewRepositories 创建 Repositories 实例
func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		Post:      postdb.NewPostRepo(db),
		Community: communitydb.NewCommunityRepo(db),
		User:      userdb.NewUserRepo(db),
		Vote:      votedb.NewVoteRepo(db),
	}
}
