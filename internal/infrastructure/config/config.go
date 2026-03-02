package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server  ServerConfig
	Mock    MockConfig
	Log     LogConfig
	Metrics MetricsConfig
	Auth    AuthConfig
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// MockConfig holds mock behaviour settings.
type MockConfig struct {
	ScenariosPath    string
	DefaultLatencyMs int
	ErrorRate        float64
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string
}

// MetricsConfig holds Prometheus settings.
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// AuthConfig holds auth mock settings.
type AuthConfig struct {
	MockEnabled bool
}

// Load reads configuration from file and environment variables.
// Environment variable prefix is MOCK_ (e.g. MOCK_SERVER_PORT=9090).
func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	// Defaults
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")
	v.SetDefault("server.shutdown_timeout", "10s")
	v.SetDefault("mock.scenarios_path", "config/scenarios/default.json")
	v.SetDefault("mock.default_latency_ms", 0)
	v.SetDefault("mock.error_rate", 0.0)
	v.SetDefault("log.level", "info")
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
	v.SetDefault("auth.mock_enabled", false)

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("default")
		v.SetConfigType("yaml")
		v.AddConfigPath("config")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("MOCK")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Ignore missing file — rely on defaults + env.
	_ = v.ReadInConfig()

	cfg := &Config{}
	cfg.Server.Port = v.GetInt("server.port")
	cfg.Server.ReadTimeout = v.GetDuration("server.read_timeout")
	cfg.Server.WriteTimeout = v.GetDuration("server.write_timeout")
	cfg.Server.ShutdownTimeout = v.GetDuration("server.shutdown_timeout")
	cfg.Mock.ScenariosPath = v.GetString("mock.scenarios_path")
	cfg.Mock.DefaultLatencyMs = v.GetInt("mock.default_latency_ms")
	cfg.Mock.ErrorRate = v.GetFloat64("mock.error_rate")
	cfg.Log.Level = v.GetString("log.level")
	cfg.Metrics.Enabled = v.GetBool("metrics.enabled")
	cfg.Metrics.Path = v.GetString("metrics.path")
	cfg.Auth.MockEnabled = v.GetBool("auth.mock_enabled")

	return cfg, nil
}
