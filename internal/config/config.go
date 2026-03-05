package config

import (
	"bluebell/pkg/errorx"
	"fmt"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type appConfig struct {
	Name    string `mapstructure:"name"`
	Mode    string `mapstructure:"mode"`
	Version string `mapstructure:"version"`
	Port    int    `mapstructure:"port"`
}

type logConfig struct {
	Level      string `mapstructure:"level"`
	FileName   string `mapstructure:"file_name"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

type mysqlConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DbName       string `mapstructure:"db_name"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

type redisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db_name"`
	PoolSize int    `mapstructure:"pool_size"`
}

type rateLimitConfig struct {
	FillInterval string `mapstructure:"fill_interval"`
	Capacity     int64  `mapstructure:"capacity"`
}

type snowflakeConfig struct {
	StartTime string `mapstructure:"start_time"`
	MachineID int64  `mapstructure:"machine_id"`
}

type jwtConfig struct {
	Secret        string `mapstructure:"secret"`
	AccessExpiry  string `mapstructure:"access_expiry"`
	RefreshExpiry string `mapstructure:"refresh_expiry"`
}

// Config 全局配置结构体
// 使用指针类型以区分配置缺失和零值
type Config struct {
	App       *appConfig       `mapstructure:"app"`
	Mysql     *mysqlConfig     `mapstructure:"mysql"`
	Redis     *redisConfig     `mapstructure:"redis"`
	Log       *logConfig       `mapstructure:"log"`
	Snowflake *snowflakeConfig `mapstructure:"snowflake"`
	RateLimit *rateLimitConfig `mapstructure:"ratelimit"`
	JWT       *jwtConfig       `mapstructure:"jwt"`
}

var atva atomic.Value

// Get returns the current configuration
func Get() *Config {
	if c, ok := atva.Load().(*Config); ok {
		return c
	}
	return nil
}

// Init Initialize configuration from file using Viper
func Init(filePath string) (*Config, error) {
	viper.SetConfigFile(filePath)
	err := viper.ReadInConfig()
	if err != nil {
		return nil, errorx.Wrap(err, errorx.CodeConfigError, "Read config failed")
	}

	conf := &Config{}
	if err := viper.Unmarshal(conf); err != nil {
		return nil, errorx.Wrap(err, errorx.CodeConfigError, "Unmarshal config failed")
	}
	//把这个对象安全地发布给其他并发读取的 goroutine。
	atva.Store(conf)
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Printf("Config file changed: %s\n", in.Name)
		newConf := new(Config)
		// On reload, unmarshal to a completely new object
		if err := viper.Unmarshal(newConf); err != nil {
			fmt.Printf("Config hot reload failed: %v\n", err)
		} else {
			atva.Store(newConf)
		}
	})

	return conf, nil
}
