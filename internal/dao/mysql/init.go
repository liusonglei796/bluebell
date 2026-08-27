// Package mysql 提供 MySQL 数据访问层（DAO）
package mysql

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"bluebell/internal/config"
	"bluebell/internal/model"
	"bluebell/internal/snowflake"

	"go.uber.org/zap"
	gmysql "gorm.io/driver/mysql"
	gorm "gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

	// 根据环境配置 GORM Logger
	// - debug: Info 级别，打印所有 SQL（开发环境）
	// - test/release: Silent 级别，不打印任何 SQL（生产环境）
	var gormLogger logger.Interface

	if cfg.App.Mode == "dev" {
		// 独立创建一个慢查询日志文件，避免和 zap 冲突
		slowLogFile, _ := os.OpenFile("gorm_slow.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		gormLogger = logger.New(
			log.New(slowLogFile, "\r\n", log.LstdFlags), // 专门输出到新文件
			logger.Config{
				SlowThreshold:             10 * time.Millisecond,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		)
	} else {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	gormConfig := &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
		// 开启错误翻译：唯一索引冲突时 Create 返回 gorm.ErrDuplicatedKey，
		// 供上层将"并发注册同名用户"等竞态兜底映射为业务错误
		TranslateError: true,
	}

	db, err := gorm.Open(gmysql.Open(dsn), gormConfig)
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

	// 自动同步表结构
	if err := db.AutoMigrate(
		&model.User{},
		&model.Community{},
		&model.Post{},
		&model.PostAuthor{},
		&model.Vote{},
		&model.Comment{},
		&model.UserRelation{},
		&model.UserNotification{},
		&model.BookmarkFolder{},
		&model.PostBookmark{},
		&model.Tag{},
		&model.PostTag{},
		&model.ProcessedEvent{},
	); err != nil {
		zap.L().Warn("mysql auto migrate warning", zap.Error(err))
	}

	// 5. 数据初始化 (Seed Data)
	if err := seedData(db); err != nil {
		zap.L().Error("seed data failed", zap.Error(err))
	}
	zap.L().Info("init mysql success", zap.String("dsn_host", mysqlCfg.Host))

	return db, nil
}

// seedData 初始化基础数据
func seedData(db *gorm.DB) error {
	// 1. 初始化社区数据
	var communityCount int64
	db.Model(&model.Community{}).Count(&communityCount)
	if communityCount == 0 {
		communities := []model.Community{
			{CommunityName: "Go", Introduction: "Golang is the best language!"},
			{CommunityName: "Vue", Introduction: "Vue.js is a progressive JavaScript framework."},
			{CommunityName: "LeetCode", Introduction: "Practice coding and prepare for interviews."},
			{CommunityName: "Life", Introduction: "Everything about life outside of coding."},
			{CommunityName: "Python", Introduction: "A versatile programming language for everyone."},
			{CommunityName: "React", Introduction: "Build user interfaces with React."},
		}
		if err := db.Create(&communities).Error; err != nil {
			return fmt.Errorf("seed communities failed: %w", err)
		}
		zap.L().Info("seed communities success")
	}

	// 2. 初始化管理员账号
	var userCount int64
	db.Model(&model.User{}).Count(&userCount)
	if userCount == 0 {
		hashedPassword, err := model.HashPassword("admin123")
		if err != nil {
			return fmt.Errorf("hash admin password failed: %w", err)
		}
		// 创建管理员账号
		adminUser := &model.User{
			UserID:   snowflake.GenID(),
			UserName: "admin",
			Passwd:   hashedPassword,
			Role:     model.RoleAdmin,
		}
		if err := db.Create(adminUser).Error; err != nil {
			return fmt.Errorf("seed admin user failed: %w", err)
		}
		zap.L().Info("seed admin user success", zap.String("username", "admin"))
	}
	return nil
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
