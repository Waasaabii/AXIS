package axis

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ResetSetupResult struct {
	ConfigPath string   `json:"configPath"`
	RuntimeDir string   `json:"runtimeDir"`
	Removed    []string `json:"removed"`
}

func ResetSetup(configPath string) (ResetSetupResult, error) {
	result := ResetSetupResult{}

	resolvedConfig, err := EnsureConfigPath(configPath)
	if err != nil {
		return result, err
	}

	config, err := LoadConfig(resolvedConfig)
	if err != nil {
		return result, err
	}

	runtimeDir := ResolveRuntimeDir(resolvedConfig, config.Runtime.Workdir)
	resolvedRuntimeDir, err := filepath.Abs(runtimeDir)
	if err != nil {
		return result, err
	}
	if !isSafeResetDirectory(resolvedRuntimeDir) {
		return result, fmt.Errorf("拒绝清理危险目录: %s", resolvedRuntimeDir)
	}

	resetConfig := &Config{
		Server: config.Server,
		Admin: AdminConfig{
			Username:        firstNonEmpty(config.Admin.Username, "admin"),
			Password:        "",
			PasswordHash:    "",
			SessionSecret:   "",
			SessionTTLHours: max(config.Admin.SessionTTLHours, 12),
		},
		Runtime:        config.Runtime,
		Subscriptions:  []Subscription{},
		LandingProxies: []LandingProxy{},
		EgressGroups:   []EgressGroup{},
		TransitRoutes:  []TransitRoute{},
		Listeners:      []Listener{},
	}

	if _, err := WriteConfig(resolvedConfig, resetConfig); err != nil {
		return result, err
	}
	if err := os.RemoveAll(resolvedRuntimeDir); err != nil {
		return result, err
	}
	if err := os.MkdirAll(resolvedRuntimeDir, 0o755); err != nil {
		return result, err
	}

	result.ConfigPath = resolvedConfig
	result.RuntimeDir = resolvedRuntimeDir
	result.Removed = []string{resolvedRuntimeDir}
	return result, nil
}

func isSafeResetDirectory(targetPath string) bool {
	cleaned := filepath.Clean(targetPath)
	if cleaned == "" || cleaned == "." || cleaned == string(filepath.Separator) {
		return false
	}
	base := strings.ToLower(filepath.Base(cleaned))
	switch base {
	case "runtime", "dev-runtime":
		return true
	}
	return strings.Contains(strings.ToLower(cleaned), string(filepath.Separator)+"axis"+string(filepath.Separator))
}
