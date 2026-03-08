package database

import (
	"bluebell/internal/config"
	"bluebell/internal/model"
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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

	// 5. 自动迁移 (AutoMigrate)
	// 为什么：开发阶段自动创建/更新表结构，减少手动操作
	err = db.AutoMigrate(
		&model.User{},
		&model.Community{},
		&model.Post{},
		&model.Vote{},
		&model.Remark{},
	)
	if err != nil {
		return nil, fmt.Errorf("auto migrate failed: %w", err)
	}

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
