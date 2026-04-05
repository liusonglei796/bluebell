package config

import (
	"bluebell/pkg/errorx"
	"fmt"
	"strings"
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
	Password     string `mapstructure:"passwd"`
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

type timeoutConfig struct {
	Timeout string `mapstructure:"timeout"`
}

type rabbitmqConfig struct {
	URL string `mapstructure:"url"`
}

type SnowflakeConfig struct {
	StartTime int64 `mapstructure:"start_time"`
	MachineID int64 `mapstructure:"machine_id"`
}

type jwtConfig struct {
	Secret        string `mapstructure:"secret"`
	AccessExpiry  string `mapstructure:"access_expiry"`
	RefreshExpiry string `mapstructure:"refresh_expiry"`
}

type aiAuditConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
}

type esConfig struct {
	Addresses []string `mapstructure:"addresses"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

// otelConfig OpenTelemetry 配置结构体
// 定义在 config 包中以避免与 otel 包的循环依赖
type otelConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Endpoint    string `mapstructure:"endpoint"`
	ServiceName string `mapstructure:"service_name"`
}

// Config 全局配置结构体
// 使用指针类型以区分配置缺失和零值
type Config struct {
	App       *appConfig       `mapstructure:"app"`
	Mysql     *mysqlConfig     `mapstructure:"mysql"`
	Redis     *redisConfig     `mapstructure:"redis"`
	Log       *logConfig       `mapstructure:"log"`
	Snowflake *SnowflakeConfig `mapstructure:"snowflake"`
	RateLimit *rateLimitConfig `mapstructure:"ratelimit"`
	JWT       *jwtConfig       `mapstructure:"jwt"`
	Timeout   *timeoutConfig   `mapstructure:"timeout"`
	RabbitMQ  *rabbitmqConfig  `mapstructure:"rabbitmq"`
	AIAudit   *aiAuditConfig   `mapstructure:"ai_audit"`
	ES        *esConfig        `mapstructure:"es"`
	Otel      *otelConfig      `mapstructure:"otel"`
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
	// 允许使用环境变量覆盖配置
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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
