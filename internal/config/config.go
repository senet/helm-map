// Package config handles CLI flag parsing, environment variable binding,
// and configuration file loading for helm-map.
package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the merged configuration from flags, env vars, and config file.
type Config struct {
	Output      string `mapstructure:"output"`
	Depth       int    `mapstructure:"depth"`
	WithImages  bool   `mapstructure:"withImages"`
	DryRun      bool   `mapstructure:"dryRun"`
	Namespace   string `mapstructure:"namespace"`
	Kubeconfig  string `mapstructure:"kubeconfig"`
	KubeContext string `mapstructure:"kubeContext"`
}

// Defaults returns the default configuration.
func Defaults() Config {
	return Config{
		Output:    "terminal",
		Depth:     0,
		Namespace: os.Getenv("HELM_NAMESPACE"),
	}
}

// InitViper sets up Viper to read from environment variables and config file.
func InitViper() {
	viper.SetEnvPrefix("HELM_MAP")
	viper.AutomaticEnv()

	// Config file location: $HELM_DATA_HOME/helm-map/config.yaml
	dataHome := os.Getenv("HELM_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			dataHome = filepath.Join(home, ".local", "share", "helm")
		}
	}
	if dataHome != "" {
		viper.AddConfigPath(filepath.Join(dataHome, "helm-map"))
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		_ = viper.ReadInConfig() // Ignore error — config file is optional.
	}

	// Set defaults.
	viper.SetDefault("output", "terminal")
	viper.SetDefault("depth", 0)
	viper.SetDefault("withImages", false)
	viper.SetDefault("dryRun", false)
}

// Load reads the current Viper state into a Config struct.
func Load() Config {
	var cfg Config
	_ = viper.Unmarshal(&cfg) // Ignore unmarshal error; defaults are safe.

	if cfg.Namespace == "" {
		cfg.Namespace = os.Getenv("HELM_NAMESPACE")
	}
	return cfg
}
