package config

import (
	"fmt"
	"os"
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
	OAuth           *OAuthConfig   `mapstructure:"oauth"`
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
	ServerModeHTTP  ServerMode = "http"
)

type ServerConfig struct {
	Port    int        `mapstructure:"port"`
	Host    string     `mapstructure:"host"`
	Timeout string     `mapstructure:"timeout"`
	Mode    ServerMode `mapstructure:"mode"`
	Name    string     `mapstructure:"name"`
	Version string     `mapstructure:"version"`
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

type OAuthConfig struct {
	Enabled      bool     `mapstructure:"enabled" `
	Provider     string   `mapstructure:"provider"` // internal, oauth2, github, google, etc.
	ClientID     string   `mapstructure:"client_id"`
	ClientSecret string   `mapstructure:"client_secret"`
	Scopes       string   `mapstructure:"scopes"`
	BaseURL      string   `mapstructure:"base_url"` // Base URL for OAuth endpoints
	Host         string   `mapstructure:"host"`     // Server host (defaults to server.host)
	Port         int      `mapstructure:"port"`     // Server port (defaults to server.port) // Server port (defaults to server.port)
	AllowOrigins []string `mapstructure:"allow_origins"`
}

// InitFlags initializes command line flags (without parsing)
func InitFlags() {
	pflag.String("mode", string(ServerModeSTDIO), "Server mode (stdio|sse|http)")
	pflag.String("swagger-file", "", "Path to the swagger file")
	pflag.String("adjustments-file", "", "Path to the adjustments file")
	// Note: no pflag.Parse() here as it's called in main.go
}

func Load() (*Config, error) {
	viper.Reset() // Ensure clean state

	viper.SetEnvPrefix("AUTO_MCP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	viper.AutomaticEnv()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		return nil, err
	}

	// Load ./config.yaml first
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	viper.AddConfigPath("/etc/auto-mcp")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	//Loading additionals config files
	if _, err := os.Stat("/config/config.yaml"); err == nil {
		viper.SetConfigFile("/config/config.yaml")
		// Merge /config/config.yaml (overrides overlapping keys)
		if err := viper.MergeInConfig(); err != nil {
			// It's OK if this file doesn't exist, only error if it's another problem
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, err
			}
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}
	// Set server mode from flag
	if mode := viper.GetString("mode"); mode != "" {
		switch ServerMode(mode) {
		case ServerModeSSE, ServerModeSTDIO, ServerModeHTTP:
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

	// If OAuth is enabled, inherit server settings if not specified
	if config.OAuth != nil && config.OAuth.Enabled {
		if config.OAuth.BaseURL == "" {
			return nil, fmt.Errorf("oauth.base_url is required, please adjust the config or pass --oauth.base_url or AUTO_MCP_OAUTH_BASE_URL environment variable")
		}
	}

	return &config, nil
}
