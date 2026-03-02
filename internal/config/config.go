package config

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type AppConfig struct {
	Name    string `mapstructure:"name"`
	Mode    string `mapstructure:"mode"`
	Version string `mapstructure:"version"`
	Port    int    `mapstructure:"port"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	FileName   string `mapstructure:"file_name"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

type MysqlConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DbName       string `mapstructure:"db_name"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db_name"`
	PoolSize int    `mapstructure:"pool_size"`
}

type RateLimitConfig struct {
	FillInterval string `mapstructure:"fill_interval"`
	Capacity     int64  `mapstructure:"capacity"`
}

type SnowflakeConfig struct {
	StartTime string `mapstructure:"start_time"`
	MachineID int64  `mapstructure:"machine_id"`
}

type JWTConfig struct {
	Secret              string `mapstructure:"secret"`
	AccessExpiryMinutes int    `mapstructure:"access_expiry_minutes"`
	RefreshExpiryHours  int    `mapstructure:"refresh_expiry_hours"`
}

// Config 全局配置结构体
// 使用指针类型以区分配置缺失和零值
type Config struct {
	App       *AppConfig       `mapstructure:"app"`
	Mysql     *MysqlConfig     `mapstructure:"mysql"`
	Redis     *RedisConfig     `mapstructure:"redis"`
	Log       *LogConfig       `mapstructure:"log"`
	Snowflake *SnowflakeConfig `mapstructure:"snowflake"`
	RateLimit *RateLimitConfig `mapstructure:"ratelimit"`
	JWT       *JWTConfig       `mapstructure:"jwt"`
}

// Conf 全局配置变量
var Conf = new(Config)

// Init 初始化配置
func Init(filePath string) (err error) {
	viper.SetConfigFile(filePath)

	if err = viper.ReadInConfig(); err != nil {
		return fmt.Errorf("viper.ReadInConfig() failed: %w", err)
	}

	if err = viper.Unmarshal(Conf); err != nil {
		return fmt.Errorf("viper.Unmarshal() failed: %w", err)
	}

	// 监控配置文件变化，支持热加载
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("配置文件被修改了...")
		if err := viper.Unmarshal(Conf); err != nil {
			fmt.Printf("配置文件热加载失败: %v\n", err)
		}
	})

	return
}
