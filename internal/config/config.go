package config

import (
	"fmt"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Version information - set by GoReleaser during build
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// GetVersionInfo returns a formatted version string
func GetVersionInfo() string {
	return fmt.Sprintf("auto-mcp version %s, commit %s, built at %s", version, commit, date)
}

type Config struct {
	Server          ServerConfig   `mapstructure:"server"`
	Logging         LoggingConfig  `mapstructure:"logging"`
	EndpointConfig  EndpointConfig `mapstructure:"endpoint"`
	SwaggerFile     string         `mapstructure:"swagger_file"`
	AdjustmentsFile string         `mapstructure:"adjustments_file"`
}

// AuthType represents the type of authentication to use
type AuthType string

const (
	AuthTypeNone   AuthType = "none"
	AuthTypeBasic  AuthType = "basic"
	AuthTypeBearer AuthType = "bearer"
	AuthTypeAPIKey AuthType = "api_key"
	AuthTypeOAuth2 AuthType = "oauth2"
)

type EndpointConfig struct {
	BaseURL    string            `json:"base_url" mapstructure:"base_url"`
	AuthType   AuthType          `json:"auth_type" mapstructure:"auth_type"`
	AuthConfig map[string]string `json:"auth_config" mapstructure:"auth_config"`
	Headers    map[string]string `json:"headers" mapstructure:"headers"`
}

type ServerMode string

const (
	ServerModeSSE   ServerMode = "sse"
	ServerModeSTDIO ServerMode = "stdio"
)

type ServerConfig struct {
	Port    int        `mapstructure:"port"`
	Host    string     `mapstructure:"host"`
	Timeout string     `mapstructure:"timeout"`
	Mode    ServerMode `mapstructure:"mode"`
}

type LoggingConfig struct {
	Level             string `mapstructure:"level"`
	Format            string `mapstructure:"format"`
	Color             bool   `mapstructure:"color"`
	DisableStacktrace bool   `mapstructure:"disable_stacktrace"`
	OutputPath        string `mapstructure:"output_path"`
	AppendToFile      bool   `mapstructure:"append_to_file"`
	DisableConsole    bool   `mapstructure:"disable_console"`
}

// InitFlags initializes command line flags (without parsing)
func InitFlags() {
	pflag.String("mode", string(ServerModeSTDIO), "Server mode (stdio|sse)")
	pflag.String("swagger-file", "", "Path to the swagger file")
	pflag.String("adjustments-file", "", "Path to the adjustments file")
	// Note: no pflag.Parse() here as it's called in main.go
}

func Load() (*Config, error) {
	// Initialize viper
	viper.SetEnvPrefix("AUTO_MCP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return nil, err
	}

	// Load main config
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	// Follows Linux convention of checking the local directory first, then system-wide locations
	// This is needed for Docker container in Dockerfile.goreleaser where config is mounted at /etc/auto-mcp
	viper.AddConfigPath("/etc/auto-mcp")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	// Set server mode from flag
	if mode := viper.GetString("mode"); mode != "" {
		switch ServerMode(mode) {
		case ServerModeSSE, ServerModeSTDIO:
			config.Server.Mode = ServerMode(mode)
		}
	}

	// Set swagger file from flag or environment
	if swaggerFile := viper.GetString("swagger-file"); swaggerFile != "" {
		config.SwaggerFile = swaggerFile
	}

	// validate swagger file
	if config.SwaggerFile == "" {
		return nil, fmt.Errorf("swagger file is required, please adjust the config or pass --swagger-file or AUTO_MCP_SWAGGER_FILE environment variable")
	}

	// Set adjustments file from flag or environment
	if adjustmentsFile := viper.GetString("adjustments-file"); adjustmentsFile != "" {
		config.AdjustmentsFile = adjustmentsFile
	}

	return &config, nil
}
