package config

import (
	"path/filepath"

	"github.com/spf13/viper"
)

func BaseSettingsDir() string {
	currentConfig := viper.ConfigFileUsed()
	currentConfigDir := filepath.Dir(currentConfig)
	return currentConfigDir
}

func BuildSettingsPath(target string) string {
	return filepath.Join(BaseSettingsDir(), target)
}
