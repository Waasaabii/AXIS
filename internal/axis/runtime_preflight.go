package axis

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	defaultSystemConfigDir = "/etc/proxyrelay"
	defaultSystemdDir      = "/etc/systemd/system"
)

func createCheck(key, title, level, summary, recommendation string) RuntimeCheck {
	return RuntimeCheck{
		Key:            key,
		Title:          title,
		Level:          level,
		OK:             level != "error",
		Summary:        summary,
		Recommendation: recommendation,
	}
}

func BuildRuntimePreflightSnapshot(configPath string, config *Config, layout *RuntimeLayout, state *AppState, controller *ControllerState) RuntimePreflightSnapshot {
	systemConfig := filepath.Clean(filepath.Dir(configPath)) == defaultSystemConfigDir
	runtimeAligned := !systemConfig || filepath.Clean(layout.RuntimeDir) == defaultSystemRuntimeWorkdir
	systemctlAvailable := exec.Command("which", "systemctl").Run() == nil
	proxyrelayServicePath := filepath.Join(defaultSystemdDir, "proxyrelayd.service")
	mihomoServicePath := filepath.Join(defaultSystemdDir, "mihomo.service")
	controllerSecretConfigured := config.Runtime.ExternalSecret != ""
	renderOnly := config.Runtime.RenderOnly

	checks := []RuntimeCheck{
		createCheck("runtime-path", "运行目录对齐", ternaryLevel(runtimeAligned, "success", "error"),
			ternaryString(runtimeAligned, fmt.Sprintf("当前工作目录是 %s", layout.RuntimeDir), fmt.Sprintf("当前工作目录 %s 与推荐目录 %s 不一致", layout.RuntimeDir, defaultSystemRuntimeWorkdir)),
			ternaryString(runtimeAligned, "", "建议把工作目录调整为 /var/lib/proxyrelay/runtime。")),
		createCheck("runtime-permissions", "运行目录权限", ternaryLevel(isWritable(layout.RuntimeDir) && isWritable(layout.ProvidersDir), "success", "error"),
			ternaryString(isWritable(layout.RuntimeDir) && isWritable(layout.ProvidersDir), "工作目录和订阅目录都可以正常写入", "工作目录或订阅目录不可写，程序无法持续保存配置和状态"),
			ternaryString(isWritable(layout.RuntimeDir) && isWritable(layout.ProvidersDir), "", "请检查 /var/lib/proxyrelay/runtime 的目录权限，并确认运行用户有写入权限。")),
		createCheck("rendered-config", "渲染产物", ternaryLevel(fileExists(layout.MihomoConfigPath) && fileExists(layout.LastGoodConfigPath) && fileExists(layout.StatePath), "success", "error"),
			ternaryString(fileExists(layout.MihomoConfigPath) && fileExists(layout.LastGoodConfigPath) && fileExists(layout.StatePath), "代理核心配置、最近一次可用配置和状态文件都已生成", "程序需要的关键文件还没生成完整"),
			ternaryString(fileExists(layout.MihomoConfigPath) && fileExists(layout.LastGoodConfigPath) && fileExists(layout.StatePath), "", "请先启动服务，让程序生成工作目录中的关键文件。")),
	}

	binaryFound := state.Runtime.MihomoBinaryFound
	controllerReachable := controller != nil && controller.Reachable
	applyStatus := state.Runtime.LastApplyStatus
	applyMessage := firstNonEmpty(state.Runtime.LastApplyMessage, "尚未应用配置")

	mihomoLevel := "success"
	if !binaryFound {
		if renderOnly {
			mihomoLevel = "warn"
		} else {
			mihomoLevel = "error"
		}
	}
	checks = append(checks, createCheck("mihomo-binary", "代理核心程序", mihomoLevel,
		ternaryString(binaryFound, fmt.Sprintf("已检测到代理核心程序：%s", firstNonEmpty(state.Runtime.MihomoBinary, config.Runtime.MihomoBinary)), ternaryString(renderOnly, "当前只保存配置，暂时还没有安装代理核心程序", "已开启自动接管，但还没有找到代理核心程序")),
		ternaryString(binaryFound || renderOnly, "", fmt.Sprintf("请安装 Mihomo，并确认 %s 可以直接执行。", config.Runtime.MihomoBinary))))

	secretLevel := "success"
	if !controllerSecretConfigured {
		if renderOnly {
			secretLevel = "info"
		} else {
			secretLevel = "warn"
		}
	}
	checks = append(checks, createCheck("controller-secret", "代理核心访问密钥", secretLevel,
		ternaryString(controllerSecretConfigured, "已配置代理核心访问密钥", "当前还没有配置代理核心访问密钥"),
		ternaryString(controllerSecretConfigured || renderOnly, "", "请在 Mihomo 中设置访问密钥，并同步填写到 runtime.external_secret。")))

	controllerLevel := "success"
	if renderOnly {
		controllerLevel = "info"
	} else if !controllerReachable {
		controllerLevel = "error"
	}
	checks = append(checks, createCheck("controller-reachability", "代理核心连接", controllerLevel,
		ternaryString(renderOnly, "当前只保存配置，暂不要求连上代理核心", ternaryString(controllerReachable, "AXIS 已成功连上代理核心", fmt.Sprintf("AXIS 还没连上代理核心：%s", firstNonEmpty(controller.Message, "未知错误")))),
		ternaryString(renderOnly || controllerReachable, "", "请确认 Mihomo 已启动、连接地址填写正确，且访问密钥保持一致。")))

	applyLevel := "warn"
	if renderOnly {
		applyLevel = "info"
	} else if applyStatus == "success" {
		applyLevel = "success"
	} else if applyStatus == "failed" {
		applyLevel = "error"
	}
	checks = append(checks, createCheck("runtime-apply", "配置应用结果", applyLevel,
		ternaryString(renderOnly, "当前只生成配置文件，暂不会自动发送到代理核心", applyMessage),
		ternaryString(renderOnly || applyStatus == "success", "", "保存配置或点击应用后，请确认代理核心已经加载最新配置。")))

	systemdLevel := "info"
	if systemConfig {
		if systemctlAvailable && fileExists(proxyrelayServicePath) && fileExists(mihomoServicePath) {
			systemdLevel = "success"
		} else {
			systemdLevel = "error"
		}
	}
	checks = append(checks, createCheck("systemd-units", "开机自启动配置", systemdLevel,
		func() string {
			if !systemConfig {
				return "当前使用的是仓库内配置，暂不检查系统级常驻服务。"
			}
			if systemctlAvailable && fileExists(proxyrelayServicePath) && fileExists(mihomoServicePath) {
				return "AXIS 和 Mihomo 的系统服务文件都已就位。"
			}
			return "系统服务文件还不完整，当前还没有完成常驻部署。"
		}(),
		func() string {
			if systemConfig && (!systemctlAvailable || !fileExists(proxyrelayServicePath) || !fileExists(mihomoServicePath)) {
				return "请执行 deploy/install-ubuntu.sh，或手动安装 AXIS 与 Mihomo 的系统服务文件。"
			}
			return ""
		}()))

	recommendations := []string{}
	for _, check := range checks {
		if check.Recommendation != "" && !containsString(recommendations, check.Recommendation) {
			recommendations = append(recommendations, check.Recommendation)
		}
	}

	summary := RuntimePreflightSummary{Ready: true, Total: len(checks)}
	for _, check := range checks {
		switch check.Level {
		case "success":
			summary.Passed++
		case "warn":
			summary.Warnings++
		case "error":
			summary.Errors++
			summary.Ready = false
		case "info":
			summary.Info++
		}
	}
	status := "ok"
	if summary.Errors > 0 {
		status = "fail"
	} else if summary.Warnings > 0 {
		status = "warn"
	}

	return RuntimePreflightSnapshot{
		GeneratedAt: nowISO(),
		Mode:        ternaryString(renderOnly, "render-only", "managed"),
		Status:      status,
		Ready:       summary.Ready,
		Summary:     summary,
		Paths: RuntimePreflightPaths{
			ConfigPath:            configPath,
			RuntimeDir:            layout.RuntimeDir,
			ProvidersDir:          layout.ProvidersDir,
			MihomoConfigPath:      layout.MihomoConfigPath,
			LastGoodConfigPath:    layout.LastGoodConfigPath,
			StatePath:             layout.StatePath,
			ProxyrelayServicePath: proxyrelayServicePath,
			MihomoServicePath:     mihomoServicePath,
		},
		Checks:          checks,
		Recommendations: recommendations,
	}
}

func isWritable(dir string) bool {
	file, err := os.CreateTemp(dir, "axis-write-test-*")
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(file.Name())
	return true
}

func ternaryString(condition bool, left, right string) string {
	if condition {
		return left
	}
	return right
}

func ternaryLevel(condition bool, left, right string) string {
	if condition {
		return left
	}
	return right
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
