package axis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const appName = "AXIS"

func ResolveAXISHome() (string, error) {
	if override := strings.TrimSpace(os.Getenv("AXIS_HOME")); override != "" {
		return filepath.Abs(override)
	}

	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, appName), nil
}

func DefaultConfigPath() (string, error) {
	homeDir, err := ResolveAXISHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, "config", "proxyrelay.yaml"), nil
}

func EnsureConfigPath(configPath string) (string, error) {
	targetPath := strings.TrimSpace(configPath)
	if targetPath == "" {
		var err error
		targetPath, err = DefaultConfigPath()
		if err != nil {
			return "", err
		}
	}

	resolved, err := filepath.Abs(targetPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o755); err != nil {
		return "", err
	}
	if _, err := os.Stat(resolved); err == nil {
		return resolved, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}

	if _, err := WriteConfig(resolved, BuildDefaultConfig(resolved)); err != nil {
		return "", fmt.Errorf("初始化默认配置失败: %w", err)
	}
	return resolved, nil
}
