package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Waasaabii/AXIS/internal/axis"
)

const (
	updaterRoleEnv        = "AXIS_DESKTOP_ROLE"
	updaterRoleValue      = "updater"
	updaterStatePathEnv   = "AXIS_UPDATER_STATE_PATH"
	updaterCommandPathEnv = "AXIS_UPDATER_COMMAND_PATH"
	updaterVersionEnv     = "AXIS_UPDATER_CURRENT_VERSION"
	updaterRepoEnv        = "AXIS_UPDATER_RELEASE_REPO"
	defaultReleaseRepo    = "Waasaabii/AXIS"
)

type updaterCommand struct {
	Action      string `json:"action"`
	RequestedAt string `json:"requestedAt"`
}

type updaterReleaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type UpdaterManager struct {
	mu          sync.RWMutex
	runtimeDir  string
	statePath   string
	commandPath string
	releaseRepo string
	version     string
	process     *exec.Cmd
	status      axis.UpdaterStatus
}

func maybeRunUpdaterRole() bool {
	if os.Getenv(updaterRoleEnv) != updaterRoleValue {
		return false
	}
	if err := runUpdaterRole(); err != nil {
		fmt.Fprintf(os.Stderr, "[axis-updater] %v\n", err)
		os.Exit(1)
	}
	os.Exit(0)
	return true
}

func NewUpdaterManager(runtimeDir, currentVersion string) (*UpdaterManager, error) {
	if strings.TrimSpace(runtimeDir) == "" {
		return &UpdaterManager{
			runtimeDir: runtimeDir,
			status: axis.UpdaterStatus{
				Mode:    "desktop-daemon",
				Running: false,
				State:   "idle",
				Message: "未配置可用运行时目录，桌面更新服务未启动",
			},
		}, nil
	}

	baseDir := filepath.Join(runtimeDir, "desktop-updater")
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}

	manager := &UpdaterManager{
		runtimeDir:  runtimeDir,
		statePath:   filepath.Join(baseDir, "state.json"),
		commandPath: filepath.Join(baseDir, "command.json"),
		releaseRepo: firstNonEmpty(os.Getenv(updaterRepoEnv), defaultReleaseRepo),
		version:     firstNonEmpty(currentVersion, "0.1.0"),
		status: axis.UpdaterStatus{
			Mode:            "desktop-daemon",
			Running:         false,
			State:           "starting",
			CurrentVersion:  firstNonEmpty(currentVersion, "0.1.0"),
			UpdateAvailable: false,
			CanAutoApply:    false,
			Message:         "桌面更新服务等待启动",
		},
	}
	return manager, nil
}

func (m *UpdaterManager) Start() error {
	if m == nil || strings.TrimSpace(m.runtimeDir) == "" {
		return nil
	}
	if m.process != nil && m.process.Process != nil {
		return nil
	}

	if err := m.writeState(m.status); err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	logPath := filepath.Join(filepath.Dir(m.statePath), "updater.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	cmd := exec.Command(executable)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("%s=%s", updaterRoleEnv, updaterRoleValue),
		fmt.Sprintf("%s=%s", updaterStatePathEnv, m.statePath),
		fmt.Sprintf("%s=%s", updaterCommandPathEnv, m.commandPath),
		fmt.Sprintf("%s=%s", updaterVersionEnv, m.version),
		fmt.Sprintf("%s=%s", updaterRepoEnv, m.releaseRepo),
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return err
	}

	m.mu.Lock()
	m.process = cmd
	m.status.Running = true
	m.status.State = "starting"
	m.status.Message = "桌面更新服务已启动"
	m.mu.Unlock()

	go func() {
		err := cmd.Wait()
		_ = logFile.Close()

		m.mu.Lock()
		defer m.mu.Unlock()

		current := m.readStateLocked()
		current.Running = false
		if err != nil && current.State != "idle" {
			current.State = "error"
			current.Message = "桌面更新服务已退出"
		}
		m.status = current
		m.process = nil
		_ = m.writeState(current)
	}()

	return nil
}

func (m *UpdaterManager) Status() axis.UpdaterStatus {
	if m == nil {
		return axis.UpdaterStatus{
			Mode:    "desktop-daemon",
			Running: false,
			State:   "idle",
			Message: "桌面更新服务未初始化",
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readStateLocked()
}

func (m *UpdaterManager) CheckForUpdates() error {
	if m == nil || strings.TrimSpace(m.commandPath) == "" {
		return fmt.Errorf("桌面更新服务未初始化")
	}
	command := updaterCommand{
		Action:      "check",
		RequestedAt: nowISO(),
	}
	return writeJSONFile(m.commandPath, command)
}

func (m *UpdaterManager) Stop() error {
	if m == nil {
		return nil
	}
	_ = writeJSONFile(m.commandPath, updaterCommand{Action: "shutdown", RequestedAt: nowISO()})

	m.mu.RLock()
	cmd := m.process
	m.mu.RUnlock()
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		for range time.Tick(150 * time.Millisecond) {
			m.mu.RLock()
			running := m.process != nil
			m.mu.RUnlock()
			if !running {
				close(done)
				return
			}
		}
	}()

	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second):
		return cmd.Process.Kill()
	}
}

func (m *UpdaterManager) writeState(status axis.UpdaterStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = status
	return m.writeStateLocked(status)
}

func (m *UpdaterManager) readStateLocked() axis.UpdaterStatus {
	state, err := readUpdaterState(m.statePath)
	if err == nil {
		if state.Mode == "" {
			state.Mode = "desktop-daemon"
		}
		m.status = state
		return state
	}
	return m.status
}

func (m *UpdaterManager) writeStateLocked(status axis.UpdaterStatus) error {
	if status.Mode == "" {
		status.Mode = "desktop-daemon"
	}
	m.status = status
	return writeJSONFile(m.statePath, status)
}

func runUpdaterRole() error {
	statePath := strings.TrimSpace(os.Getenv(updaterStatePathEnv))
	commandPath := strings.TrimSpace(os.Getenv(updaterCommandPathEnv))
	currentVersion := firstNonEmpty(os.Getenv(updaterVersionEnv), "0.1.0")
	releaseRepo := firstNonEmpty(os.Getenv(updaterRepoEnv), defaultReleaseRepo)
	if statePath == "" || commandPath == "" {
		return fmt.Errorf("缺少 updater 状态文件配置")
	}

	status := axis.UpdaterStatus{
		Mode:            "desktop-daemon",
		Running:         true,
		State:           "starting",
		CurrentVersion:  currentVersion,
		UpdateAvailable: false,
		CanAutoApply:    false,
		Message:         "桌面更新服务正在初始化",
	}
	if err := writeJSONFile(statePath, status); err != nil {
		return err
	}

	if checked, err := checkLatestRelease(releaseRepo, status); err == nil {
		status = checked
	} else {
		status.State = "error"
		status.Message = err.Error()
		status.CheckedAt = nowISO()
	}
	if err := writeJSONFile(statePath, status); err != nil {
		return err
	}

	checkTicker := time.NewTicker(6 * time.Hour)
	commandTicker := time.NewTicker(2 * time.Second)
	defer checkTicker.Stop()
	defer commandTicker.Stop()

	for {
		select {
		case <-checkTicker.C:
			next, err := checkLatestRelease(releaseRepo, status)
			if err != nil {
				status.State = "error"
				status.Message = err.Error()
				status.CheckedAt = nowISO()
			} else {
				status = next
			}
			if err := writeJSONFile(statePath, status); err != nil {
				return err
			}
		case <-commandTicker.C:
			command, ok, err := readUpdaterCommand(commandPath)
			if err != nil {
				status.State = "error"
				status.Message = err.Error()
				status.CheckedAt = nowISO()
				if writeErr := writeJSONFile(statePath, status); writeErr != nil {
					return writeErr
				}
				continue
			}
			if !ok {
				continue
			}

			switch command.Action {
			case "shutdown":
				status.Running = false
				status.State = "idle"
				status.Message = "桌面更新服务已停止"
				status.CheckedAt = nowISO()
				return writeJSONFile(statePath, status)
			case "check":
				next, err := checkLatestRelease(releaseRepo, status)
				if err != nil {
					status.State = "error"
					status.Message = err.Error()
					status.CheckedAt = nowISO()
				} else {
					status = next
				}
				if err := writeJSONFile(statePath, status); err != nil {
					return err
				}
			}
		}
	}
}

func readUpdaterState(statePath string) (axis.UpdaterStatus, error) {
	var status axis.UpdaterStatus
	raw, err := os.ReadFile(statePath)
	if err != nil {
		return status, err
	}
	if err := json.Unmarshal(raw, &status); err != nil {
		return axis.UpdaterStatus{}, err
	}
	return status, nil
}

func readUpdaterCommand(commandPath string) (updaterCommand, bool, error) {
	var command updaterCommand
	raw, err := os.ReadFile(commandPath)
	if err != nil {
		if os.IsNotExist(err) {
			return command, false, nil
		}
		return command, false, err
	}
	if err := os.Remove(commandPath); err != nil && !os.IsNotExist(err) {
		return command, false, err
	}
	if err := json.Unmarshal(raw, &command); err != nil {
		return command, false, err
	}
	return command, true, nil
}

func writeJSONFile(targetPath string, payload any) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, append(raw, '\n'), 0o644)
}

func checkLatestRelease(releaseRepo string, status axis.UpdaterStatus) (axis.UpdaterStatus, error) {
	next := status
	next.Running = true
	next.State = "checking"
	next.Message = "正在检查 GitHub Releases"
	next.CheckedAt = nowISO()

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", releaseRepo), nil)
	if err != nil {
		return status, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "AXIS-Updater")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return status, fmt.Errorf("检查更新失败: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return status, fmt.Errorf("检查更新失败: GitHub 返回 %d %s", response.StatusCode, strings.TrimSpace(string(body)))
	}

	var release updaterReleaseResponse
	if err := json.NewDecoder(response.Body).Decode(&release); err != nil {
		return status, fmt.Errorf("解析更新信息失败: %w", err)
	}

	assetName, assetURL := pickReleaseAsset(release)
	next.State = "idle"
	next.LatestVersion = strings.TrimSpace(release.TagName)
	next.ReleaseURL = strings.TrimSpace(release.HTMLURL)
	next.AssetName = assetName
	next.AssetURL = assetURL
	next.UpdateAvailable = versionNewer(release.TagName, status.CurrentVersion)
	next.CanAutoApply = false
	if next.UpdateAvailable {
		next.Message = fmt.Sprintf("发现新版本 %s，可从 GitHub Releases 获取。", firstNonEmpty(next.LatestVersion, "unknown"))
	} else {
		next.Message = "当前已经是最新版本。"
	}
	next.CheckedAt = nowISO()
	return next, nil
}

func pickReleaseAsset(release updaterReleaseResponse) (string, string) {
	suffixes := platformAssetHints()
	for _, asset := range release.Assets {
		name := strings.ToLower(asset.Name)
		for _, suffix := range suffixes {
			if strings.Contains(name, suffix) {
				return asset.Name, asset.BrowserDownloadURL
			}
		}
	}
	return "", ""
}

func platformAssetHints() []string {
	goos := strings.ToLower(runtime.GOOS)
	goarch := strings.ToLower(runtime.GOARCH)
	return []string{
		goos + "-" + goarch,
		goos + "_" + goarch,
		goos + goarch,
		goos,
	}
}

func versionNewer(latest, current string) bool {
	latestParts := normalizeVersionParts(latest)
	currentParts := normalizeVersionParts(current)
	maxLen := len(latestParts)
	if len(currentParts) > maxLen {
		maxLen = len(currentParts)
	}
	for index := 0; index < maxLen; index++ {
		latestValue := partAt(latestParts, index)
		currentValue := partAt(currentParts, index)
		if latestValue == currentValue {
			continue
		}
		return latestValue > currentValue
	}
	return false
}

func normalizeVersionParts(value string) []int {
	cleaned := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(value, "v"), "V"))
	if cleaned == "" {
		return nil
	}
	parts := strings.Split(cleaned, ".")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		fragment := part
		for index, runeValue := range fragment {
			if runeValue < '0' || runeValue > '9' {
				fragment = fragment[:index]
				break
			}
		}
		number, _ := strconv.Atoi(fragment)
		result = append(result, number)
	}
	return result
}

func partAt(parts []int, index int) int {
	if index < 0 || index >= len(parts) {
		return 0
	}
	return parts[index]
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}
