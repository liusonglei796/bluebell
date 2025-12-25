package settings

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// 1. 保持命名一致性，统一使用 Config 后缀
// 为什么：结构体命名规范，清晰表明该结构体用于配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Mode    string `mapstructure:"mode"`
	Version string `mapstructure:"version"`
	Port    int    `mapstructure:"port"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	FileName   string `mapstructure:"file_name"`   // 修正: 对应 YAML 中的 file_name
	MaxSize    int    `mapstructure:"max_size"`    // 新增: 补全 YAML 中的配置
	MaxBackups int    `mapstructure:"max_backups"` // 新增
	MaxAge     int    `mapstructure:"max_age"`     // 新增
}

type MysqlConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DbName       string `mapstructure:"db_name"`
	MaxOpenConns int    `mapstructure:"max_open_conns"` // 修正: 从 LogConfig 移回这里
	MaxIdleConns int    `mapstructure:"max_idle_conns"` // 修正: 从 LogConfig 移回这里
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db_name"`   // 修正: YAML key 是 db_name，类型是 int
	PoolSize int    `mapstructure:"pool_size"` // 新增: YAML 中有 pool_size
}

type RateLimitConfig struct {
	FillInterval string `mapstructure:"fill_interval"` // 令牌填充间隔（如 "10ms"）
	Capacity     int64  `mapstructure:"capacity"`      // 令牌桶容量
}

// 为什么用指针
// 1.为了区分配置缺失和零值
// 如果是值类型 (MysqlConfig)： 如果配置文件中完全没有写 mysql 相关的配置，Go 会给它赋零值（Zero Value）。
// 即 Host 为空字符串，Port 为 0。你无法区分“用户没配 MySQL”还是“用户故意配了个空值的 MySQL”。
// 如果是指针类型 (*MysqlConfig)： 如果配置文件中没有 mysql 区块，解析库（如 mapstructure）会将该字段保持为 nil。
// 2.避免大结构体的拷贝
// 为什么：配置对象通常较大，传递指针比传递值更高效
type Config struct {
	App       *AppConfig       `mapstructure:"app"`
	Mysql     *MysqlConfig     `mapstructure:"mysql"`
	Redis     *RedisConfig     `mapstructure:"redis"`
	Log       *LogConfig       `mapstructure:"log"`
	Snowflake *SnowflakeConfig `mapstructure:"snowflake"`
	RateLimit *RateLimitConfig `mapstructure:"ratelimit"` // 新增: 限流配置
}

type SnowflakeConfig struct {
	StartTime string `mapstructure:"start_time"`
	MachineID int64  `mapstructure:"machine_id"`
}

// 定义全局变量
// 为什么：配置在整个应用生命周期内都需要访问，全局变量方便各处调用
var Conf = new(Config)

func Init(filePath string) (err error) {
	// 1. 指定配置文件路径
	// 为什么：viper 支持多种配置源，这里指定文件路径
	viper.SetConfigFile(filePath)

	// 2. 读取配置信息
	// 为什么：加载文件内容到 viper 的内存结构中
	if err = viper.ReadInConfig(); err != nil {
		// 这里只需返回错误，由调用者决定是否打印，或者使用 fmt.Errorf 包装
		return fmt.Errorf("viper.ReadInConfig() failed: %w", err)
	}

	// 3. 将读取的配置信息反序列化到 Conf 变量中
	// 为什么：viper 内部是 map 结构，反序列化到结构体方便代码中使用强类型访问，且利用 mapstructure 标签映射字段
	if err = viper.Unmarshal(Conf); err != nil {
		return fmt.Errorf("viper.Unmarshal() failed: %w", err)
	}

	// 4. 监控配置文件变化
	// 为什么：支持热加载，无需重启服务即可更新配置
	viper.WatchConfig()
	viper.OnConfigChange(func(in fsnotify.Event) {
		fmt.Println("配置文件被修改了...")
		// 配置文件发生变化后，需要重新反序列化到 Conf 变量中，否则代码中使用的还是旧值
		if err := viper.Unmarshal(Conf); err != nil {
			fmt.Printf("配置文件热加载失败: %v\n", err)
		}
	})

	return
}
