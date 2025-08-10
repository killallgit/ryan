package config

import (
	"path/filepath"

	"github.com/spf13/viper"
)

func BaseSettingsDir() string {
	// Check if config.path is explicitly set (for testing)
	if configPath := viper.GetString("config.path"); configPath != "" {
		return configPath
	}

	currentConfig := viper.ConfigFileUsed()
	currentConfigDir := filepath.Dir(currentConfig)
	return currentConfigDir
}

func BuildSettingsPath(target string) string {
	return filepath.Join(BaseSettingsDir(), target)
}
