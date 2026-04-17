package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

// Config 是整个应用启动时读取的顶层配置。
// 这里先覆盖服务启动最需要的配置项，后面接数据库、Redis、MQ 时再继续往里扩展。
type Config struct {
	App    AppConfig    `yaml:"app"`
	Server ServerConfig `yaml:"server"`
	Log    LogConfig    `yaml:"log"`
	MySQL  MySQLConfig  `yaml:"mysql"`
	Redis  RedisConfig  `yaml:"redis"`
}

type AppConfig struct {
	Name string `yaml:"name" env:"GO_SECKILL_APP_NAME" env-default:"go-seckill"`
	Env  string `yaml:"env" env:"GO_SECKILL_APP_ENV" env-default:"dev"`
}

type ServerConfig struct {
	Port            int           `yaml:"port" env:"GO_SECKILL_SERVER_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"GO_SECKILL_SERVER_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"GO_SECKILL_SERVER_WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"GO_SECKILL_SERVER_IDLE_TIMEOUT" env-default:"30s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"GO_SECKILL_SERVER_SHUTDOWN_TIMEOUT" env-default:"10s"`
}

type LogConfig struct {
	Level       string `yaml:"level" env:"GO_SECKILL_LOG_LEVEL" env-default:"info"`
	Development bool   `yaml:"development" env:"GO_SECKILL_LOG_DEVELOPMENT" env-default:"true"`
}

type MySQLConfig struct {
	Host            string        `yaml:"host" env:"GO_SECKILL_MYSQL_HOST" env-default:"127.0.0.1"`
	Port            int           `yaml:"port" env:"GO_SECKILL_MYSQL_PORT" env-default:"3306"`
	User            string        `yaml:"user" env:"GO_SECKILL_MYSQL_USER" env-default:"go_seckill"`
	Password        string        `yaml:"password" env:"GO_SECKILL_MYSQL_PASSWORD" env-default:"go_seckill123"`
	Database        string        `yaml:"database" env:"GO_SECKILL_MYSQL_DATABASE" env-default:"go_seckill"`
	Charset         string        `yaml:"charset" env:"GO_SECKILL_MYSQL_CHARSET" env-default:"utf8mb4"`
	ParseTime       bool          `yaml:"parse_time" env:"GO_SECKILL_MYSQL_PARSE_TIME" env-default:"true"`
	Loc             string        `yaml:"loc" env:"GO_SECKILL_MYSQL_LOC" env-default:"Local"`
	MaxOpenConns    int           `yaml:"max_open_conns" env:"GO_SECKILL_MYSQL_MAX_OPEN_CONNS" env-default:"20"`
	MaxIdleConns    int           `yaml:"max_idle_conns" env:"GO_SECKILL_MYSQL_MAX_IDLE_CONNS" env-default:"10"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime" env:"GO_SECKILL_MYSQL_CONN_MAX_LIFETIME" env-default:"30m"`
}

func (c MySQLConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		c.User,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.Charset,
		c.ParseTime,
		c.Loc,
	)
}

type RedisConfig struct {
	Addr         string `yaml:"addr" env:"GO_SECKILL_REDIS_ADDR" env-default:"127.0.0.1:6379"`
	Password     string `yaml:"password" env:"GO_SECKILL_REDIS_PASSWORD"`
	DB           int    `yaml:"db" env:"GO_SECKILL_REDIS_DB" env-default:"0"`
	PoolSize     int    `yaml:"pool_size" env:"GO_SECKILL_REDIS_POOL_SIZE" env-default:"20"`
	MinIdleConns int    `yaml:"min_idle_conns" env:"GO_SECKILL_REDIS_MIN_IDLE_CONNS" env-default:"5"`
}

// Load 负责按“配置文件 -> 环境变量覆盖”的顺序加载配置。
// 这样做的原因是：
// 1. 本地开发时可以先写一个易读的 yaml
// 2. 部署或测试时又可以用环境变量覆盖少量差异项
func Load(path string) (*Config, error) {
	cfg := &Config{}

	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := cleanenv.ReadConfig(path, cfg); err != nil {
				return nil, fmt.Errorf("read config file: %w", err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("stat config file: %w", err)
		}
	}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("read config from env: %w", err)
	}

	return cfg, nil
}
