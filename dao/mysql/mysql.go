package mysql

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"bluebell/settings"
)

// 定义全局的数据库连接对象
// 为什么：gorm.DB 是线程安全的，整个应用共享一个连接池即可
var db *gorm.DB

// Init 初始化 MySQL 连接
// 为什么：建立数据库连接池，配置连接参数，确保应用启动时数据库可用
func Init(cfg *settings.MysqlConfig) (err error) {
	if cfg == nil {
		return fmt.Errorf("mysql.Init received nil config")
	}

	// 构建 DSN (Data Source Name)
	// parseTime=True: 自动解析时间字段为 time.Time
	// loc=Local: 使用本地时区
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DbName,
	)

	// 创建带超时的 context
	// 为什么：防止数据库连接卡死导致程序启动失败
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// GORM 配置
	gormConfig := &gorm.Config{
		// 日志配置：生产环境建议使用 logger.Silent
		Logger: logger.Default.LogMode(logger.Info),
		// 禁用外键约束（与现有数据库设计保持一致）
		DisableForeignKeyConstraintWhenMigrating: true,
		// 预编译语句缓存
		PrepareStmt: true,
	}

	// 连接数据库
	// gorm.Open 会自动进行连接池配置
	db, err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("connect to mysql failed: %w", err)
	}

	// 获取底层的 sql.DB 对象用于配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB failed: %w", err)
	}

	// 验证连接（带超时控制）
	if err = sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping mysql failed: %w", err)
	}

	// 设置最大打开连接数
	// 为什么：防止连接数过多压垮数据库
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)

	// 设置最大空闲连接数
	// 为什么：保持一定数量的空闲连接，避免频繁创建和销毁连接的开销
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)

	// 设置连接最大空闲时间
	// 为什么：防止空闲连接被数据库服务端或防火墙断开导致 "bad connection" 错误
	// 必须小于 MySQL 的 wait_timeout（默认8小时），这里设置为10分钟
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	// 设置连接最大存活时间
	// 为什么：防止连接长时间未使用被数据库服务端断开
	sqlDB.SetConnMaxLifetime(2 * time.Hour)

	zap.L().Info("init mysql success", zap.String("dsn_host", cfg.Host))
	return nil
}

// Close 关闭 MySQL 连接
// 为什么：程序退出时释放资源
func Close() {
	if db != nil {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

// GetDB 获取数据库连接实例
// 为什么：提供一个公共方法供其他包使用（如需要事务操作时）
func GetDB() *gorm.DB {
	return db
}
