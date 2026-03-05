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

/*这三个方法（

PostRepo()
、

CommunityRepo()
、

UserRepo()
）的存在，核心作用是为了实现领域驱动设计（DDD）架构中的

UnitOfWork
（工作单元）模式以及方便进行依赖注入*/
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
// uow 也就是此时的 r，它内部的 r.db 存的是【全局普通数据库连接对象 db】
func (r *Repositories) Transaction(fn func(uow repository.UnitOfWork) error) error {

	// r.db.Transaction 会要求 GORM 从连接池拿一根专属的长连接并开启事务，
	// 然后把这个绑定了长连接的特殊对象赋值给入参变量 `tx`（它就是【事务绑定的专属数据库连接对象】）。
	return r.db.Transaction(func(tx *gorm.DB) error {

		// 精髓来了！
		// 普通情况下，我们要被迫把 tx 一路作为函数参数传给所有的 Repo 方法。
		// 但我们没有这么做，而是直接利用这个 `tx` 对象，当场原地重新 New 了一整套 Repositories！
		txRepo := NewRepositories(tx)

		// 所有的 Dao 都还是用它们最喜欢的 r.db 去增删改查。
		// 但此时他们眼里的 r.db 已经不再是那个全局池子了，而是被我们狸猫换太子，偷偷塞进去的专属事务连接 `tx`！
		// 最后把这套崭新的 “卧底” 仓储群交给业务层的方法去折腾。
		return fn(txRepo)
	})
}

// Init 初始化 MySQL 连接，返回数据库连接实例
func Init(cfg *config.Config) (*gorm.DB, error) {
	if cfg == nil {
		return nil, fmt.Errorf("mysql.Init received nil config")
	}

	mysqlCfg := cfg.Mysql
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		mysqlCfg.User,
		mysqlCfg.Password,
		mysqlCfg.Host,
		mysqlCfg.Port,
		mysqlCfg.DbName,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gormConfig := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to mysql failed: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB failed: %w", err)
	}

	if err = sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql failed: %w", err)
	}

	sqlDB.SetMaxOpenConns(mysqlCfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(mysqlCfg.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	sqlDB.SetConnMaxLifetime(2 * time.Hour)

	zap.L().Info("init mysql success", zap.String("dsn_host", mysqlCfg.Host))

	return db, nil
}

// Close 关闭 MySQL 连接
func Close(db *gorm.DB) {
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil && sqlDB != nil {
			_ = sqlDB.Close()
		}
	}
}
