package config

import "time"

type Config struct {
	App      AppConfig
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	Tracer   TracerConfig
}

type AppConfig struct {
	AppName         string        `env:"APP_NAME" env-default:"url-shortener"`
	Environment     string        `env:"ENVIRONMENT" env-default:"test"`
	MetricsEnable   bool          `env:"ENABLE_METRICS" env-default:"false"`
	LogLevel        string        `env:"LOG_LEVEL" env-default:"info"`
}

type ServerConfig struct {
	Port            string        `env:"PORT" env-default:"8080"`
	NodeID          int64         `env:"NODE_ID" env-required:"true"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"5s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	ShutdownTimeout time.Duration `env:"SERVER_SHUTDOWN_TIMEOUT" env-default:"2s"`
}

type DatabaseConfig struct {
	DbTTL             time.Duration `env:"DATABASE_TTL" env-default:"336h"`
	ConnString        string        `env:"CONNECTION_STRING" env-required:"true"`
	MaxConns          int32         `env:"MAX_CONNS" env-default:"25"`
	MinConns          int32         `env:"MIN_CONNS" env-default:"5"`
	MaxConnLifeTime   time.Duration `env:"MAX_CONN_LIFETIME" env-default:"1h"`
	MaxConnIdleTime   time.Duration `env:"MAX_CONN_IDLETIME" env-default:"30m"`
	HealthCheckPeriod time.Duration `env:"HEALTH_CHECK_PERIOD" env-default:"1m"`
	ConnectionTimeout time.Duration `env:"CONNECTION_TIMEOUT" env-default:"5s"`
}

type CacheConfig struct {
	RedisAddrs      []string      `env:"REDIS_ADDRS" env-default:"localhost:6379" env-separator:","`
	RedisMasterName string        `env:"REDIS_MASTER_NAME" env-default:""`
	RedisPassword   string        `env:"REDIS_PASSWORD" env-default:""`
	CacheTTL        time.Duration `env:"CACHE_TTL" env-default:"24h"`
}

type TracerConfig struct {
	TracesEnable  bool    `env:"ENABLE_TRACES" env-default:"false"`
	CollectorAddr string  `env:"COLLECTOR_ADDR" env-default:""`
	Ratio         float64 `env:"TRACER_RATIO" env-default:"1"`
}
