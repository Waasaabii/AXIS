import { access } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";

const DEFAULT_SYSTEM_CONFIG_DIR = "/etc/proxyrelay";
const DEFAULT_SYSTEM_RUNTIME_DIR = "/var/lib/proxyrelay/runtime";
const DEFAULT_SYSTEMD_DIR = "/etc/systemd/system";

function normalizePath(input) {
  return input instanceof URL ? fileURLToPath(input) : String(input);
}

async function hasAccess(targetPath, mode = fsConstants.F_OK) {
  try {
    await access(targetPath, mode);
    return true;
  } catch {
    return false;
  }
}

function findBinaryInPath(binary) {
  const result = spawnSync("which", [binary], { encoding: "utf8" });
  if (result.status === 0) {
    return result.stdout.trim();
  }

  return "";
}

function createCheck(key, title, level, summary, recommendation = "") {
  return {
    key,
    title,
    ok: level !== "error",
    level,
    summary,
    recommendation
  };
}

function dedupeRecommendations(checks) {
  return [...new Set(checks.map((check) => check.recommendation).filter(Boolean))];
}

export async function buildRuntimePreflightSnapshot({
  configPath,
  config,
  layout,
  state,
  controller,
  serviceDir = DEFAULT_SYSTEMD_DIR,
  expectedSystemRuntimeDir = DEFAULT_SYSTEM_RUNTIME_DIR
}) {
  const normalizedConfigPath = normalizePath(configPath);
  const configDir = path.resolve(path.dirname(normalizedConfigPath));
  const systemConfig = configDir === DEFAULT_SYSTEM_CONFIG_DIR;
  const runtimeDir = path.resolve(layout.runtimeDir);
  const providersDir = path.resolve(layout.providersDir);
  const runtimeAligned = !systemConfig || runtimeDir === expectedSystemRuntimeDir;
  const runtimeWritable = await hasAccess(runtimeDir, fsConstants.W_OK);
  const providersWritable = await hasAccess(providersDir, fsConstants.W_OK);
  const renderedConfigExists = await hasAccess(layout.mihomoConfigPath);
  const lastGoodConfigExists = await hasAccess(layout.lastGoodConfigPath);
  const stateFileExists = await hasAccess(layout.statePath);
  const systemctlAvailable = Boolean(findBinaryInPath("systemctl"));
  const proxyrelayUnitPath = path.join(serviceDir, "proxyrelayd.service");
  const mihomoUnitPath = path.join(serviceDir, "mihomo.service");
  const proxyrelayUnitExists = await hasAccess(proxyrelayUnitPath);
  const mihomoUnitExists = await hasAccess(mihomoUnitPath);
  const renderOnly = Boolean(config.runtime.render_only);
  const binaryFound = Boolean(state.runtime?.mihomoBinaryFound);
  const controllerReachable = Boolean(controller?.reachable);
  const controllerSecretConfigured = Boolean(config.runtime.external_secret);
  const applyStatus = state.runtime?.lastApplyStatus || "idle";
  const applyMessage = state.runtime?.lastApplyMessage || "尚未执行运行态下发";

  const checks = [
    createCheck(
      "runtime-path",
      "运行目录对齐",
      runtimeAligned ? "success" : "error",
      runtimeAligned
        ? `当前运行目录为 ${runtimeDir}`
        : `当前运行目录 ${runtimeDir} 与 Ubuntu 部署目标 ${expectedSystemRuntimeDir} 不一致`,
      runtimeAligned ? "" : "将 runtime.workdir 调整为 /var/lib/proxyrelay/runtime"
    ),
    createCheck(
      "runtime-permissions",
      "运行目录权限",
      runtimeWritable && providersWritable ? "success" : "error",
      runtimeWritable && providersWritable
        ? "runtime 与 providers 目录可写"
        : "runtime 或 providers 目录不可写，控制面无法持续生成配置与状态文件",
      runtimeWritable && providersWritable ? "" : "检查 /var/lib/proxyrelay/runtime 目录权限并确保 proxyrelay 用户可写"
    ),
    createCheck(
      "rendered-config",
      "渲染产物",
      renderedConfigExists && lastGoodConfigExists && stateFileExists ? "success" : "error",
      renderedConfigExists && lastGoodConfigExists && stateFileExists
        ? "mihomo.yaml、last-good 与 control-state 已生成"
        : "渲染产物不完整，控制面还未形成稳定运行态",
      renderedConfigExists && lastGoodConfigExists && stateFileExists ? "" : "先启动控制面生成 runtime 目录下的三类核心文件"
    ),
    createCheck(
      "mihomo-binary",
      "Mihomo 二进制",
      binaryFound ? "success" : renderOnly ? "warn" : "error",
      binaryFound
        ? `已检测到 Mihomo 二进制：${state.runtime?.mihomoBinary || config.runtime.mihomo_binary}`
        : renderOnly
        ? "当前为 render-only 模式，允许暂时未安装 Mihomo"
        : "managed 模式下未检测到 Mihomo 二进制",
      binaryFound || renderOnly ? "" : `安装 Mihomo 并确认 ${config.runtime.mihomo_binary} 可执行`
    ),
    createCheck(
      "controller-secret",
      "Controller 密钥",
      controllerSecretConfigured ? "success" : renderOnly ? "info" : "warn",
      controllerSecretConfigured ? "已配置 external controller 密钥" : "当前未配置 external controller 密钥",
      controllerSecretConfigured || renderOnly ? "" : "为 Mihomo external-controller 配置 secret，并同步到 runtime.external_secret"
    ),
    createCheck(
      "controller-reachability",
      "Controller 连通性",
      renderOnly ? "info" : controllerReachable ? "success" : "error",
      renderOnly
        ? "当前为 render-only 模式，未要求 controller 可达"
        : controllerReachable
        ? "external controller 可达"
        : `external controller 不可达：${controller?.message || "未知错误"}`,
      renderOnly || controllerReachable ? "" : "确认 mihomo 已启动、external-controller 地址正确且 secret 一致"
    ),
    createCheck(
      "runtime-apply",
      "运行态下发",
      renderOnly ? "info" : applyStatus === "success" ? "success" : applyStatus === "failed" ? "error" : "warn",
      renderOnly ? "当前为 render-only 模式，仅生成配置文件" : applyMessage,
      renderOnly || applyStatus === "success" ? "" : "保存配置或执行重载后，检查 controller 是否真正加载了最新 mihomo.yaml"
    ),
    createCheck(
      "systemd-units",
      "systemd 服务",
      systemConfig
        ? systemctlAvailable && proxyrelayUnitExists && mihomoUnitExists
          ? "success"
          : "error"
        : "info",
      systemConfig
        ? systemctlAvailable && proxyrelayUnitExists && mihomoUnitExists
          ? "proxyrelayd 与 mihomo 的 service 文件已就位"
          : "systemd service 文件缺失，Ubuntu 常驻部署尚未完成"
        : "当前使用仓库内配置，跳过系统级 service 检查",
      systemConfig && (!systemctlAvailable || !proxyrelayUnitExists || !mihomoUnitExists)
        ? "执行 deploy/install-ubuntu.sh，或手动安装 proxyrelayd.service 与 mihomo.service"
        : ""
    )
  ];

  const ready = checks.every((check) => check.level !== "error");
  const recommendations = dedupeRecommendations(checks);
  const errors = checks.filter((check) => check.level === "error").length;
  const warnings = checks.filter((check) => check.level === "warn").length;
  const status = errors > 0 ? "fail" : warnings > 0 ? "warn" : "ok";

  return {
    generatedAt: new Date().toISOString(),
    mode: renderOnly ? "render-only" : "managed",
    status,
    ready,
    summary: {
      ready,
      total: checks.length,
      passed: checks.filter((check) => check.level === "success").length,
      warnings,
      errors,
      info: checks.filter((check) => check.level === "info").length
    },
    paths: {
      configPath: normalizedConfigPath,
      runtimeDir,
      providersDir,
      mihomoConfigPath: layout.mihomoConfigPath,
      lastGoodConfigPath: layout.lastGoodConfigPath,
      statePath: layout.statePath,
      proxyrelayServicePath: proxyrelayUnitPath,
      mihomoServicePath: mihomoUnitPath
    },
    checks,
    recommendations
  };
}

export function formatRuntimePreflightForCli(snapshot) {
  const lines = [
    "ProxyRelay Runtime Preflight",
    `模式: ${snapshot.mode}`,
    `就绪状态: ${snapshot.ready ? "通过" : "未通过"}`,
    `统计: 成功 ${snapshot.summary.passed} / 警告 ${snapshot.summary.warnings} / 错误 ${snapshot.summary.errors}`
  ];

  for (const check of snapshot.checks) {
    const icon =
      check.level === "success" ? "OK" : check.level === "warn" ? "WARN" : check.level === "error" ? "ERR" : "INFO";
    lines.push(`${icon} ${check.title}: ${check.summary}`);
  }

  if (snapshot.recommendations.length) {
    lines.push("建议操作:");
    for (const recommendation of snapshot.recommendations) {
      lines.push(`- ${recommendation}`);
    }
  }

  return `${lines.join("\n")}\n`;
}
