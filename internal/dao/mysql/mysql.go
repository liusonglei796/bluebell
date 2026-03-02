package mysql

import (
	"bluebell/internal/config"
	"bluebell/internal/domain/repository"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// db 全局数据库连接对象
// gorm.DB 是线程安全的，整个应用共享一个连接池即可
var db *gorm.DB

// Repositories 聚合所有 Repository 实例
// 作为依赖注入的入口，Service 层通过此结构访问数据层
// 实现了 repository.UnitOfWork 接口
type Repositories struct {
	db        *gorm.DB                       // GORM 数据库实例
	Post      repository.PostRepository      // 帖子 Repository
	Community repository.CommunityRepository // 社区 Repository
	User      repository.UserRepository      // 用户 Repository
}

// NewRepositories 创建所有 Repository 实例
func NewRepositories(gormDB *gorm.DB) *Repositories {
	return &Repositories{
		db:        gormDB,
		Post:      NewPostRepo(gormDB),
		Community: NewCommunityRepo(gormDB),
		User:      NewUserRepo(gormDB),
	}
}

// PostRepo 返回 PostRepository 实例
func (r *Repositories) PostRepo() repository.PostRepository {
	return r.Post
}

// CommunityRepo 返回 CommunityRepository 实例
func (r *Repositories) CommunityRepo() repository.CommunityRepository {
	return r.Community
}

// UserRepo 返回 UserRepository 实例
func (r *Repositories) UserRepo() repository.UserRepository {
	return r.User
}

// Transaction 在数据库事务中执行函数
// 回调参数 uow 是绑定了事务连接的新 UnitOfWork 实例
func (r *Repositories) Transaction(fn func(uow repository.UnitOfWork) error) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		txRepo := NewRepositories(tx)
		return fn(txRepo)
	})
}

// Init 初始化 MySQL 连接
func Init(cfg *config.MysqlConfig) (err error) {
	if cfg == nil {
		return fmt.Errorf("mysql.Init received nil config")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DbName,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gormConfig := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	}

	db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("connect to mysql failed: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB failed: %w", err)
	}

	if err = sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping mysql failed: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	sqlDB.SetConnMaxLifetime(2 * time.Hour)

	zap.L().Info("init mysql success", zap.String("dsn_host", cfg.Host))
	return nil
}

// Close 关闭 MySQL 连接
func Close() {
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

// GetDB 获取数据库连接实例
func GetDB() *gorm.DB {
	return db
}
