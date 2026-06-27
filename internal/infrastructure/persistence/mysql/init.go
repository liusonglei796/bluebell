package database

import (
	"bluebell/internal/config"
	"bluebell/internal/infrastructure/logger"
	"bluebell/internal/infrastructure/persistence/mysql/model"
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
)

// Init 初始化 MySQL 连接，返回数据库连接实例。
// reg 为 Prometheus 注册表（可选），用于注册 database/sql 的连接池指标。
func Init(cfg *config.Config, reg prometheus.Registerer) (*gorm.DB, error) {
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

	// 使用 GORM → Zap 桥接日志适配器
	// - dev:  Info 级别，记录所有 SQL，慢查询阈值 10ms
	// - prod: Warn 级别，仅记录慢查询（阈值 200ms）和错误
	gormLogger := logger.NewGormLogger(cfg.App.Mode)

	gormConfig := &gorm.Config{
		Logger:                                   gormLogger,
		DisableForeignKeyConstraintWhenMigrating: true,
		PrepareStmt:                              true,
	}

	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("connect to mysql failed: %w", err)
	}

	// 注册 OpenTelemetry GORM 插件，自动为 SQL 操作创建子 Span
	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		zap.L().Error("register otelgorm plugin failed", zap.Error(err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB failed: %w", err)
	}

	// 注册 database/sql 的 DBStats collector，暴露 go_sql_connections_* 指标
	// 数据流: DBStats → Prometheus Registry → /metrics scrape → Mimir
	if reg != nil {
		if err := reg.Register(collectors.NewDBStatsCollector(sqlDB, "bluebell_mysql")); err != nil {
			zap.L().Warn("register DBStatsCollector failed", zap.Error(err))
		}
	}

	if err = sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql failed: %w", err)
	}

	sqlDB.SetMaxOpenConns(mysqlCfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(mysqlCfg.MaxIdleConns)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)
	sqlDB.SetConnMaxLifetime(2 * time.Hour)

	// 5. 自动迁移 (AutoMigrate)
	// 为什么：开发阶段自动创建/更新表结构，减少手动操作
	err = db.AutoMigrate(
		&model.User{},
		&model.Community{},
		&model.Post{},
		&model.Vote{},
		&model.Remark{},
		&model.UserProfile{},
		&model.Follow{},
		&model.Activity{},
	)
	if err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

	// 6. 数据初始化 (Seed Data)
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
		// 创建管理员账号
		adminUser := &model.User{
			UserName: "admin",
			Passwd:   "admin123", // 密码会被 BeforeCreate 钩子自动加密
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
