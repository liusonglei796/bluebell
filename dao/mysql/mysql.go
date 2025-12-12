package mysql

import (
	"context"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"bluebell/settings"
)

// 定义全局的数据库连接对象
// 为什么：sqlx.DB 是线程安全的，整个应用共享一个连接池即可
var db *sqlx.DB

// Init 初始化 MySQL 连接
// 为什么:建立数据库连接池,配置连接参数,确保应用启动时数据库可用
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
	// 为什么:防止数据库连接卡死导致程序启动失败
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 连接数据库
	// sqlx.Connect 相当于 sql.Open + db.Ping,确保连接可用
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		return fmt.Errorf("connect to mysql failed: %w", err)
	}

	// 验证连接(带超时控制)
	if err = db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping mysql failed: %w", err)
	}

	// 设置最大打开连接数
	// 为什么:防止连接数过多压垮数据库
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	// 设置最大空闲连接数
	// 为什么:保持一定数量的空闲连接,避免频繁创建和销毁连接的开销
	db.SetMaxIdleConns(cfg.MaxIdleConns)

	// 设置连接最大空闲时间
	// 为什么:防止空闲连接被数据库服务端或防火墙断开导致 "bad connection" 错误
	// 必须小于 MySQL 的 wait_timeout(默认8小时),这里设置为10分钟
	db.SetConnMaxIdleTime(10 * time.Minute)

	// 设置连接最大存活时间
	// 为什么:防止连接长时间未使用被数据库服务端断开
	db.SetConnMaxLifetime(2 * time.Hour)

	zap.L().Info("init mysql success", zap.String("dsn_host", cfg.Host))
	return nil
}

// Close 关闭 MySQL 连接
// 为什么：程序退出时释放资源
func Close() {
	_ = db.Close()
}
