package axis

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type Service struct {
	configPath  string
	config      *Config
	layout      *RuntimeLayout
	state       *AppState
	store       *Store
	auth        *AuthContext
	controller  *MihomoControllerAdapter
	host        HostIntegration
	configMTime time.Time
}

func NewService(configPath string) (*Service, error) {
	if configPath == "" {
		resolvedPath, err := EnsureConfigPath("")
		if err != nil {
			return nil, err
		}
		configPath = resolvedPath
	}
	service := &Service{configPath: configPath}
	if err := service.LoadAll(); err != nil {
		return nil, err
	}
	return service, nil
}

func createEmptyState() *AppState {
	return &AppState{
		StartedAt:       nowISO(),
		Runtime:         RuntimeState{Mode: "bootstrap", LastApplyStatus: "idle"},
		Providers:       map[string]ProviderRecord{},
		GroupSelections: map[string]string{},
		Groups:          map[string]GroupState{},
		TransitRoutes:   map[string]TransitRouteState{},
		Host:            HostState{},
		Events:          []EventEntry{},
	}
}

func (s *Service) LoadAll() error {
	config, err := LoadConfig(s.configPath)
	if err != nil {
		return err
	}
	layout, err := EnsureRuntimeLayout(config, s.configPath)
	if err != nil {
		return err
	}
	store, err := NewStore(layout.DatabasePath)
	if err != nil {
		return err
	}
	if s.store != nil {
		_ = s.store.Close()
	}
	s.store = store
	s.config = config
	s.layout = layout
	s.auth = NewAuthContext(config.Admin.Username, config.Admin.PasswordHash, config.Admin.Password, config.Admin.SessionSecret, config.Admin.SessionTTLHours)
	s.controller = NewMihomoControllerAdapter(config.Runtime)
	s.host = NewNoopHostIntegration()

	if err := s.loadState(); err != nil {
		return err
	}
	s.detectRuntime()
	if _, err := s.renderRuntimeConfig(); err != nil {
		return err
	}
	s.detectController(false)
	if err := s.captureConfigModTime(); err != nil {
		return err
	}
	return s.persistState()
}

func (s *Service) loadState() error {
	state, err := s.store.LoadState()
	if err != nil {
		return err
	}
	if state != nil {
		s.state = normalizeState(state)
		s.state.Events, _ = s.store.ListEvents(100)
		return nil
	}

	if fileExists(s.layout.StatePath) {
		raw, err := os.ReadFile(s.layout.StatePath)
		if err == nil {
			var loaded AppState
			if json.Unmarshal(raw, &loaded) == nil {
				s.state = normalizeState(&loaded)
				s.state.Events, _ = s.store.ListEvents(100)
				return nil
			}
		}
	}

	s.state = normalizeState(createEmptyState())
	return nil
}

func (s *Service) persistState() error {
	s.state.Events, _ = s.store.ListEvents(100)
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.layout.StatePath, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return s.store.SaveState(s.state)
}

func (s *Service) pushEvent(level, scope, message string) {
	event := EventEntry{
		ID:      createNodeID(scope, nowISO()+message),
		Level:   level,
		Scope:   scope,
		Message: message,
		At:      nowISO(),
	}
	_ = s.store.AppendEvent(event, 100)
	s.state.Events, _ = s.store.ListEvents(100)
}

func (s *Service) detectRuntime() {
	binaryPath := pathOrResolved(s.configPath, s.config.Runtime.MihomoBinary)
	binaryFound := false
	if binaryPath != "" {
		if stat, err := os.Stat(binaryPath); err == nil && stat.Mode().IsRegular() {
			binaryFound = true
		}
	}
	s.state.Runtime = RuntimeState{
		Mode:                ternaryString(s.config.Runtime.RenderOnly, "render-only", "managed"),
		MihomoBinary:        firstNonEmpty(binaryPath, s.config.Runtime.MihomoBinary),
		MihomoBinaryFound:   binaryFound,
		Controller:          s.config.Runtime.ExternalController,
		ControllerReachable: false,
		ConfigPath:          s.layout.MihomoConfigPath,
		LastRenderAt:        s.state.Runtime.LastRenderAt,
		LastApplyAt:         s.state.Runtime.LastApplyAt,
		LastApplyStatus:     firstNonEmpty(s.state.Runtime.LastApplyStatus, "idle"),
		LastApplyMessage:    s.state.Runtime.LastApplyMessage,
	}
	if !binaryFound {
		s.pushEvent("warn", "runtime", fmt.Sprintf("未找到代理核心程序：%s", s.config.Runtime.MihomoBinary))
	}
}

func (s *Service) detectController(recordEvent bool) {
	probe := s.controller.Probe()
	probe.CheckedAt = nowISO()
	previous := s.state.Controller
	s.state.Controller = &probe
	s.state.Runtime.ControllerReachable = probe.Reachable
	if recordEvent && (previous == nil || previous.Reachable != probe.Reachable || previous.Message != probe.Message || previous.Version != probe.Version) {
		level := "warn"
		if probe.Reachable {
			level = "info"
		}
		s.pushEvent(level, "controller", probe.Message)
	}
}

func (s *Service) renderRuntimeConfig() (string, error) {
	rendered, err := RenderMihomoConfig(s.config)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(s.layout.MihomoConfigPath, []byte(rendered), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(s.layout.LastGoodConfigPath, []byte(rendered), 0o644); err != nil {
		return "", err
	}
	if err := s.writeManualProviderFiles(); err != nil {
		return "", err
	}
	s.state.Runtime.LastRenderAt = nowISO()
	s.pushEvent("info", "render", "最新代理核心配置已生成")
	return rendered, nil
}

func (s *Service) writeManualProviderFiles() error {
	for _, subscription := range s.config.Subscriptions {
		if !subscription.Enabled || !isManualProviderType(subscription.Type) {
			continue
		}
		content, err := buildManualProviderFileContent(subscription)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(s.layout.ProvidersDir, providerFileName(subscription.Name))
		if err := os.WriteFile(targetPath, content, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) removeManualProviderFile(name string) error {
	if strings.TrimSpace(name) == "" {
		return nil
	}
	targetPath := filepath.Join(s.layout.ProvidersDir, providerFileName(name))
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Service) Login(username, password string) (map[string]any, int) {
	if s.config.Admin.RequiresPasswordReset {
		return map[string]any{"ok": false, "error": "首次初始化还没完成，请先创建管理员账号和密码"}, 409
	}
	if !s.auth.VerifyCredentials(username, password) {
		s.pushEvent("warn", "auth", fmt.Sprintf("登录失败: %s", firstNonEmpty(username, "unknown")))
		_ = s.persistState()
		return map[string]any{"ok": false, "error": "账号或密码错误"}, 401
	}
	setupState := s.GetSetupState()
	s.pushEvent("info", "auth", fmt.Sprintf("登录成功: %s", username))
	_ = s.persistState()
	return map[string]any{
		"ok":                    true,
		"requiresPasswordReset": s.config.Admin.RequiresPasswordReset,
		"setupRequired":         setupState.Required,
		"user":                  map[string]string{"username": username},
	}, 200
}

func (s *Service) SessionStatus(valid bool, username string) map[string]any {
	if !valid {
		return map[string]any{"authenticated": false}
	}
	setupState := s.GetSetupState()
	return map[string]any{
		"authenticated":         true,
		"requiresPasswordReset": s.config.Admin.RequiresPasswordReset,
		"setupRequired":         setupState.Required,
		"user":                  map[string]string{"username": username},
	}
}

func isPlaceholderSubscriptionURL(value string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return true
	}
	return strings.Contains(trimmed, "example.com") || strings.Contains(trimmed, "your-subscription-url")
}

func hasUsableSubscriptionSource(subscription Subscription) bool {
	if isManualProviderType(subscription.Type) {
		return strings.TrimSpace(subscription.Server) != "" && subscription.Port > 0
	}
	return !isPlaceholderSubscriptionURL(subscription.URL)
}

func BuildSetupState(config *Config) SetupState {
	hasNodeSources := len(config.NodeSources) > 0
	hasRoutes := len(config.Routes) > 0
	hasUsage := config.Usage.SelectedRoute != "" && (config.Usage.LocalProxy.Enabled || config.Usage.VirtualInterface.Enabled)
	hasPublications := len(config.Publications) > 0

	checks := []SetupCheck{
		{
			Key:     "password",
			Title:   "管理员密码",
			Ready:   !config.Admin.RequiresPasswordReset,
			Summary: ternaryString(config.Admin.RequiresPasswordReset, "请先设置管理员密码，避免他人直接进入控制台。", "管理员密码已设置。"),
			Action:  ternaryString(config.Admin.RequiresPasswordReset, "去设置密码", ""),
		},
		{
			Key:     "node-sources",
			Title:   "节点来源",
			Ready:   hasNodeSources,
			Summary: ternaryString(hasNodeSources, "已添加节点来源。", "还没有可用的节点来源。"),
			Action:  ternaryString(hasNodeSources, "", "添加节点来源"),
		},
		{
			Key:     "routes",
			Title:   "线路",
			Ready:   hasRoutes,
			Summary: ternaryString(hasRoutes, "已创建可使用的线路。", "还没有线路，节点还不能被这台设备使用或发布。"),
			Action:  ternaryString(hasRoutes, "", "创建线路"),
		},
		{
			Key:     "usage",
			Title:   "本机使用",
			Ready:   hasUsage,
			Summary: ternaryString(hasUsage, "这台设备已选择使用方式。", "还没有选择这台设备要怎么使用代理。"),
			Action:  ternaryString(hasUsage, "", "设置使用方式"),
		},
	}

	reasons := []string{}
	if config.Admin.RequiresPasswordReset {
		reasons = append(reasons, "管理员密码还没设置。")
	}
	if !hasNodeSources {
		reasons = append(reasons, "还没有添加节点来源。")
	}
	if !hasRoutes {
		reasons = append(reasons, "还没有创建线路。")
	}
	if !hasUsage {
		reasons = append(reasons, "还没有选择这台设备的使用方式。")
	}

	return SetupState{
		Required:             len(reasons) > 0,
		NeedsPasswordReset:   config.Admin.RequiresPasswordReset,
		AdminUsername:        config.Admin.Username,
		HasNodeSources:       hasNodeSources,
		HasRoutes:            hasRoutes,
		HasUsage:             hasUsage,
		HasPublications:      hasPublications,
		HasSubscriptions:     hasNodeSources,
		HasRealSubscriptions: hasNodeSources,
		HasEgressGroups:      hasRoutes,
		HasListeners:         hasUsage,
		Checks:               checks,
		Reasons:              reasons,
	}
}

func (s *Service) GetSetupState() SetupState {
	return BuildSetupState(s.config)
}

func (s *Service) UpdatePassword(password string) (map[string]any, int) {
	if password == "" {
		return map[string]any{"ok": false, "error": "新密码不能为空"}, 400
	}
	hash, err := CreatePasswordHash(password)
	if err != nil {
		return map[string]any{"ok": false, "error": "修改密码失败: " + err.Error()}, 500
	}
	s.config.Admin.PasswordHash = hash
	s.config.Admin.Password = ""
	s.config.Admin.RequiresPasswordReset = false
	if _, err := WriteConfig(s.configPath, s.config); err != nil {
		return map[string]any{"ok": false, "error": "修改密码失败: " + err.Error()}, 500
	}
	if err := s.LoadAll(); err != nil {
		return map[string]any{"ok": false, "error": "修改密码失败: " + err.Error()}, 500
	}
	return map[string]any{"ok": true}, 200
}

func (s *Service) BootstrapAdmin(username, password string) (map[string]any, int) {
	username = strings.TrimSpace(username)
	if !s.config.Admin.RequiresPasswordReset {
		return map[string]any{"ok": false, "error": "当前不是首次初始化状态，无需再创建管理员账号"}, 409
	}
	if username == "" {
		return map[string]any{"ok": false, "error": "管理员账号不能为空"}, 400
	}
	if password == "" {
		return map[string]any{"ok": false, "error": "管理员密码不能为空"}, 400
	}

	hash, err := CreatePasswordHash(password)
	if err != nil {
		return map[string]any{"ok": false, "error": "创建管理员账号失败: " + err.Error()}, 500
	}

	s.config.Admin.Username = username
	s.config.Admin.PasswordHash = hash
	s.config.Admin.Password = ""
	s.config.Admin.RequiresPasswordReset = false
	if _, err := WriteConfig(s.configPath, s.config); err != nil {
		return map[string]any{"ok": false, "error": "创建管理员账号失败: " + err.Error()}, 500
	}
	if err := s.LoadAll(); err != nil {
		return map[string]any{"ok": false, "error": "创建管理员账号失败: " + err.Error()}, 500
	}
	return map[string]any{"ok": true}, 200
}

func (s *Service) GetStatus() map[string]any {
	s.detectController(false)
	return map[string]any{
		"app": map[string]any{
			"name":       "AXIS",
			"mode":       s.state.Runtime.Mode,
			"startedAt":  s.state.StartedAt,
			"renderOnly": s.config.Runtime.RenderOnly,
		},
		"runtime":    s.state.Runtime,
		"controller": s.state.Controller,
		"counts": map[string]int{
			"nodeSources":  len(s.config.NodeSources),
			"routes":       len(s.config.Routes),
			"publications": len(s.config.Publications),
			"nodes":        len(s.config.NodeSources),
		},
		"usage":        s.GetUsage(),
		"warnings":     s.buildWarnings(),
		"recentEvents": s.state.Events[:minInt(len(s.state.Events), 12)],
	}
}

func sumProviderNodes(providers []map[string]any) int {
	total := 0
	for _, item := range providers {
		if value, ok := item["nodeCount"].(int); ok {
			total += value
		}
	}
	return total
}

func (s *Service) buildWarnings() []string {
	warnings := []string{}
	if s.config.Admin.Password != "" {
		warnings = append(warnings, "管理员密码仍以明文方式写在配置里，建议改成加密保存。")
	}
	if !s.state.Runtime.MihomoBinaryFound {
		warnings = append(warnings, "还没找到代理核心程序，所以当前只能保存配置，不能直接接管运行。")
	}
	if !s.config.Runtime.RenderOnly && !s.state.Runtime.ControllerReachable {
		warnings = append(warnings, "当前已开启自动接管，但还连不上代理核心，请检查程序是否已启动。")
	}
	return warnings
}

func (s *Service) GetEvents() []EventEntry {
	s.state.Events, _ = s.store.ListEvents(100)
	return s.state.Events
}

func (s *Service) MihomoVersions() (MihomoVersionsResponse, error) {
	records, err := s.store.ListVersionRecords()
	if err != nil {
		return MihomoVersionsResponse{}, err
	}
	response := MihomoVersionsResponse{
		Platform: map[string]string{
			"os":   runtime.GOOS,
			"arch": runtime.GOARCH,
		},
		Recommended:       "v1.19.21",
		ConfiguredBinary:  s.config.Runtime.MihomoBinary,
		SupportMatrix:     defaultMihomoSupportMatrix(),
		InstalledVersions: records,
	}
	for _, record := range records {
		if record.ActivatedAt != "" {
			response.ActiveVersion = record.Version
			break
		}
	}
	return response, nil
}

func (s *Service) MihomoVersionState() (map[string]any, error) {
	records, err := s.store.ListVersionRecords()
	if err != nil {
		return nil, err
	}
	return map[string]any{"records": records}, nil
}

func (s *Service) findSupportedVersion(version string) (MihomoVersionSupport, bool) {
	for _, item := range defaultMihomoSupportMatrix() {
		if item.Version == version {
			return item, true
		}
	}
	return MihomoVersionSupport{}, false
}

func (s *Service) DownloadMihomoVersion(version string) (map[string]any, int) {
	support, ok := s.findSupportedVersion(version)
	if !ok || !support.Supported || support.AssetURL == "" {
		return map[string]any{"ok": false, "error": fmt.Sprintf("版本 %s 不在当前支持列表中", version)}, 400
	}
	versionDir := filepath.Join(s.layout.MihomoVersionsDir, version)
	archivePath := filepath.Join(versionDir, support.AssetName)
	if err := downloadFile(archivePath, support.AssetURL); err != nil {
		record := MihomoVersionRecord{Version: version, Status: "failed", ArchivePath: archivePath, LastError: err.Error()}
		_ = s.store.UpsertVersionRecord(record)
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	record := MihomoVersionRecord{Version: version, Status: "downloaded", ArchivePath: archivePath, DownloadedAt: nowISO()}
	_ = s.store.UpsertVersionRecord(record)
	s.pushEvent("info", "mihomo", fmt.Sprintf("Mihomo %s 已下载", version))
	_ = s.persistState()
	return map[string]any{"ok": true, "record": record}, 200
}

func (s *Service) InstallMihomoVersion(version string) (map[string]any, int) {
	records, err := s.store.ListVersionRecords()
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	var record *MihomoVersionRecord
	for index := range records {
		if records[index].Version == version {
			record = &records[index]
			break
		}
	}
	if record == nil || record.ArchivePath == "" {
		return map[string]any{"ok": false, "error": "请先下载该版本"}, 400
	}
	binaryName := "mihomo"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(s.layout.MihomoVersionsDir, version, binaryName)
	if strings.HasSuffix(record.ArchivePath, ".gz") {
		if err := extractGzipArchive(record.ArchivePath, binaryPath); err != nil {
			record.Status = "failed"
			record.LastError = err.Error()
			_ = s.store.UpsertVersionRecord(*record)
			return map[string]any{"ok": false, "error": err.Error()}, 500
		}
	} else {
		return map[string]any{"ok": false, "error": "当前仅支持 .gz 制品自动安装"}, 400
	}
	record.Status = "installed"
	record.BinaryPath = binaryPath
	record.InstalledAt = nowISO()
	record.LastError = ""
	_ = s.store.UpsertVersionRecord(*record)
	s.pushEvent("info", "mihomo", fmt.Sprintf("Mihomo %s 已安装", version))
	_ = s.persistState()
	return map[string]any{"ok": true, "record": record}, 200
}

func (s *Service) ActivateMihomoVersion(version string) (map[string]any, int) {
	records, err := s.store.ListVersionRecords()
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}, 500
	}
	targetIndex := -1
	for index := range records {
		if records[index].Version == version {
			targetIndex = index
		} else if records[index].ActivatedAt != "" {
			records[index].ActivatedAt = ""
			_ = s.store.UpsertVersionRecord(records[index])
		}
	}
	if targetIndex < 0 || records[targetIndex].BinaryPath == "" {
		return map[string]any{"ok": false, "error": "目标版本尚未安装"}, 400
	}

	next := mustJSONClone(*s.config)
	next.Runtime.MihomoBinary = records[targetIndex].BinaryPath
	result, status := s.SaveConfig(&next)
	if status != 200 {
		return result, status
	}

	records[targetIndex].ActivatedAt = nowISO()
	records[targetIndex].Status = "active"
	_ = s.store.UpsertVersionRecord(records[targetIndex])
	s.pushEvent("info", "mihomo", fmt.Sprintf("Mihomo 已切换到 %s", version))
	_ = s.persistState()
	return map[string]any{"ok": true, "record": records[targetIndex]}, 200
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func (s *Service) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}

func (s *Service) Config() *Config {
	return s.config
}

func (s *Service) AuthFingerprint() string {
	if s == nil || s.config == nil || s.auth == nil {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.Join([]string{
		s.config.Admin.Username,
		s.config.Admin.PasswordHash,
		s.config.Admin.Password,
		s.auth.secret,
	}, "|")))
	return fmt.Sprintf("%x", sum[:])
}
