import { access, readFile, writeFile } from "node:fs/promises";
import path from "node:path";
import { constants as fsConstants } from "node:fs";
import { spawnSync } from "node:child_process";
import { createAuthContext } from "../lib/auth.js";
import { ensureRuntimeLayout, loadConfig, persistJson, writeConfig } from "../lib/config.js";
import { compilePattern } from "../lib/pattern.js";
import { createMihomoControllerAdapter } from "./mihomo-controller.js";
import { renderMihomoConfig } from "./mihomo-renderer.js";
import { buildRuntimePreflightSnapshot } from "./runtime-preflight.js";
import { fetchProviderSnapshot } from "./subscription-service.js";

const DEFAULT_CONFIG_PATH = path.resolve(process.cwd(), "config/proxyrelay.yaml");

function nowIso() {
  return new Date().toISOString();
}

function timestampAgeMs(timestamp) {
  if (!timestamp) {
    return Number.POSITIVE_INFINITY;
  }

  const value = Date.parse(timestamp);
  if (Number.isNaN(value)) {
    return Number.POSITIVE_INFINITY;
  }

  return Date.now() - value;
}

function hasControllerStateChanged(previous, next) {
  if (!previous) {
    return true;
  }

  return (
    previous.reachable !== next.reachable ||
    previous.mode !== next.mode ||
    previous.version !== next.version ||
    previous.message !== next.message
  );
}

function readControllerResultMessage(result, fallback) {
  const payload = result?.payload;
  if (typeof payload === "string" && payload.trim()) {
    return payload.trim();
  }

  if (payload && typeof payload === "object") {
    if (typeof payload.message === "string" && payload.message.trim()) {
      return payload.message.trim();
    }

    if (typeof payload.error === "string" && payload.error.trim()) {
      return payload.error.trim();
    }
  }

  return fallback;
}

function createEmptyState() {
  return {
    startedAt: nowIso(),
    runtime: {
      mode: "bootstrap",
      mihomoBinaryFound: false,
      controllerReachable: false,
      lastRenderAt: null,
      lastApplyAt: null,
      lastApplyStatus: "idle",
      lastApplyMessage: null
    },
    providers: {},
    groupSelections: {},
    events: []
  };
}

function pushEvent(state, level, scope, message) {
  state.events.unshift({
    id: `${Date.now()}-${Math.random().toString(16).slice(2, 8)}`,
    level,
    scope,
    message,
    at: nowIso()
  });
  state.events = state.events.slice(0, 100);
}

function matchNode(group, node) {
  const include = group.filter ? compilePattern(group.filter).test(node.name) : true;
  const exclude = group.exclude_filter ? compilePattern(group.exclude_filter).test(node.name) : false;
  return include && !exclude;
}

async function fileExists(filePath) {
  try {
    await access(filePath, fsConstants.F_OK);
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

export async function createControlPlane(options = {}) {
  const configPath = process.env.PROXYRELAY_CONFIG || DEFAULT_CONFIG_PATH;
  const controlPlane = new ControlPlane(configPath, options);
  await controlPlane.initialize();
  return controlPlane;
}

class ControlPlane {
  constructor(configPath, { skipStartupRefresh = false } = {}) {
    this.configPath = configPath;
    this.skipStartupRefresh = skipStartupRefresh;
    this.config = null;
    this.layout = null;
    this.state = createEmptyState();
    this.authContext = null;
    this.controller = null;
    this.controllerProbePromise = null;
  }

  async initialize() {
    await this.loadAll();

    if (!this.skipStartupRefresh && this.config.server.startup_refresh) {
      for (const subscription of this.config.subscriptions.filter((item) => item.enabled !== false)) {
        await this.refreshProvider(subscription.name);
      }
    }
  }

  async loadAll() {
    this.config = await loadConfig(this.configPath);
    this.layout = await ensureRuntimeLayout(this.config, this.configPath);
    await this.loadState();
    await this.detectRuntime();
    await this.renderRuntimeConfig();
    this.authContext = createAuthContext({
      username: this.config.admin.username,
      password: this.config.admin.password,
      passwordHash: this.config.admin.password_hash,
      sessionSecret: this.config.admin.session_secret,
      sessionTtlHours: this.config.admin.session_ttl_hours
    });
    this.controller = createMihomoControllerAdapter(this.config.runtime);
    await this.detectController({ recordEvent: true });
    await this.persistState();
  }

  async loadState() {
    if (!(await fileExists(this.layout.statePath))) {
      this.state = createEmptyState();
      await this.persistState();
      return;
    }

    const raw = await readFile(this.layout.statePath, "utf8");
    this.state = JSON.parse(raw);
  }

  async persistState() {
    await persistJson(this.layout.statePath, this.state);
  }

  async detectRuntime() {
    const binary = this.config.runtime.mihomo_binary;
    const binaryPath =
      binary.includes("/") || binary.startsWith(".")
        ? path.resolve(path.dirname(this.configPath), binary)
        : findBinaryInPath(binary);
    const binaryFound = binaryPath ? await fileExists(binaryPath) : false;

    this.state.runtime = {
      mode: this.config.runtime.render_only ? "render-only" : "managed",
      mihomoBinary: binaryPath || binary,
      mihomoBinaryFound: binaryFound,
      controller: this.config.runtime.external_controller,
      controllerReachable: false,
      configPath: this.layout.mihomoConfigPath,
      lastRenderAt: this.state.runtime?.lastRenderAt || null,
      lastApplyAt: this.state.runtime?.lastApplyAt || null,
      lastApplyStatus: this.state.runtime?.lastApplyStatus || "idle",
      lastApplyMessage: this.state.runtime?.lastApplyMessage || null
    };

    if (!binaryFound) {
      pushEvent(this.state, "warn", "runtime", `未找到 mihomo 二进制: ${binary}`);
    }
  }

  async detectController({ recordEvent = true, persist = false } = {}) {
    const previous = this.state.controller || null;
    const probe = await this.controller.probe();
    const nextState = {
      reachable: probe.reachable,
      mode: probe.mode,
      version: probe.version || null,
      message: probe.message,
      checkedAt: nowIso()
    };
    this.state.controller = nextState;
    this.state.runtime.controllerReachable = probe.reachable;
    const changed = hasControllerStateChanged(previous, nextState);

    if (recordEvent && changed) {
      pushEvent(this.state, probe.reachable ? "info" : "warn", "controller", probe.message);
    }

    if (persist) {
      await this.persistState();
    }

    return nextState;
  }

  async ensureFreshControllerStatus({ maxAgeMs = 15000, recordEvent = false, persist = false } = {}) {
    if (timestampAgeMs(this.state.controller?.checkedAt) <= maxAgeMs) {
      return this.state.controller;
    }

    if (!this.controllerProbePromise) {
      this.controllerProbePromise = this.detectController({ recordEvent, persist }).finally(() => {
        this.controllerProbePromise = null;
      });
    }

    return this.controllerProbePromise;
  }

  async renderRuntimeConfig() {
    const rendered = renderMihomoConfig(this.config, { providersDir: "providers" });
    await writeFile(this.layout.mihomoConfigPath, rendered, "utf8");
    await writeFile(this.layout.lastGoodConfigPath, rendered, "utf8");
    this.state.runtime.lastRenderAt = nowIso();
    pushEvent(this.state, "info", "render", "已生成 runtime/mihomo.yaml");
    await this.persistState();
    return rendered;
  }

  getServerConfig() {
    return this.config.server;
  }

  getAuthConfig() {
    return {
      username: this.config.admin.username,
      password: this.config.admin.password,
      passwordHash: this.config.admin.password_hash,
      sessionSecret: this.config.admin.session_secret,
      sessionTtlHours: this.config.admin.session_ttl_hours
    };
  }

  getAuthContext() {
    return this.authContext;
  }

  async login(username, password) {
    if (!this.authContext.verifyCredentials(username, password)) {
      pushEvent(this.state, "warn", "auth", `登录失败: ${username || "unknown"}`);
      await this.persistState();
      return { ok: false, error: "账号或密码错误" };
    }

    pushEvent(this.state, "info", "auth", `登录成功: ${username}`);
    await this.persistState();
    return { ok: true, user: { username } };
  }

  getProviderRecord(name) {
    return this.state.providers[name] || {
      provider: name,
      refreshedAt: null,
      nodeCount: 0,
      nodes: [],
      lastError: null
    };
  }

  buildGroupView(group) {
    const providerRecord = this.getProviderRecord(group.provider);
    const candidates = (providerRecord.nodes || []).filter((node) => matchNode(group, node));
    const selection = this.state.groupSelections[group.name] || candidates[0]?.name || null;

    return {
      name: group.name,
      mode: group.mode || "manual",
      provider: group.provider,
      filter: group.filter || "",
      candidateCount: candidates.length,
      current: selection,
      candidates: candidates.map((node) => ({
        id: node.id,
        name: node.name,
        type: node.type,
        server: node.server,
        port: node.port
      })),
      lastHealthcheckAt: this.state.groups?.[group.name]?.lastHealthcheckAt || null
    };
  }

  async getStatus() {
    await this.ensureFreshControllerStatus();
    const providers = await this.getProviders();
    const groups = await this.getGroups();
    const listeners = await this.getListeners();

    return {
      app: {
        name: "ProxyRelay",
        mode: this.state.runtime.mode,
        startedAt: this.state.startedAt,
        renderOnly: this.config.runtime.render_only
      },
      runtime: this.state.runtime,
      controller: this.state.controller,
      counts: {
        providers: providers.length,
        groups: groups.length,
        listeners: listeners.length,
        nodes: providers.reduce((sum, item) => sum + item.nodeCount, 0)
      },
      warnings: this.buildWarnings(),
      recentEvents: this.state.events.slice(0, 12)
    };
  }

  buildWarnings() {
    const warnings = [];

    if (this.config.admin.password) {
      warnings.push("当前 admin 使用明文密码，建议改成 password_hash。");
    }

    if (!this.state.runtime.mihomoBinaryFound) {
      warnings.push("当前机器未检测到 mihomo 二进制，控制面处于 render-only 状态。");
    }

    if (!this.config.runtime.render_only && !this.state.runtime.controllerReachable) {
      warnings.push("控制面已设置为 managed，但当前无法连通 mihomo controller。");
    }

    return warnings;
  }

  async getConfig() {
    return {
      path: this.configPath,
      config: this.config
    };
  }

  async saveConfig(nextConfig) {
    await writeConfig(this.configPath, nextConfig);
    await this.loadAll();
    pushEvent(this.state, "info", "config", "配置已保存并重新加载");
    await this.persistState();
    const runtimeApply = await this.applyRuntimeConfig("配置保存");
    return {
      ok: true,
      path: this.configPath,
      config: this.config,
      runtimeApply
    };
  }

  async getProviders() {
    return this.config.subscriptions.map((subscription) => {
      const record = this.getProviderRecord(subscription.name);
      return {
        name: subscription.name,
        type: subscription.type || "mihomo-http",
        urlMasked: record.urlMasked || subscription.url,
        interval: Number(subscription.interval || 3600),
        enabled: subscription.enabled !== false,
        refreshedAt: record.refreshedAt,
        nodeCount: record.nodeCount || 0,
        lastError: record.lastError || null,
        nodes: record.nodes || []
      };
    });
  }

  async getGroups() {
    const providerNames = new Set(this.config.subscriptions.map((s) => s.name));
    const enabledProviders = new Set(
      this.config.subscriptions.filter((s) => s.enabled !== false).map((s) => s.name)
    );
    return this.config.egress_groups.map((group) => {
      const view = this.buildGroupView(group);
      const providerExists = providerNames.has(group.provider);
      const providerEnabled = enabledProviders.has(group.provider);
      return {
        ...view,
        providerMissing: !providerExists,
        providerDisabled: providerExists && !providerEnabled
      };
    });
  }

  async getListeners() {
    const groups = new Map((await this.getGroups()).map((group) => [group.name, group]));
    const groupNames = new Set(this.config.egress_groups.map((g) => g.name));
    return this.config.listeners.map((listener) => {
      const group = groups.get(listener.egress_group);
      const groupMissing = !groupNames.has(listener.egress_group);
      return {
        name: listener.name,
        type: listener.type || "socks",
        listen: listener.listen || "0.0.0.0",
        port: Number(listener.port),
        udp: listener.udp !== false,
        users: Array.isArray(listener.users) ? listener.users : [],
        userCount: Array.isArray(listener.users) ? listener.users.length : 0,
        egressGroup: listener.egress_group,
        currentProxy: group?.current || null,
        status: groupMissing ? "orphaned" : (group?.providerMissing || group?.providerDisabled) ? "degraded" : "configured",
        groupMissing,
        providerMissing: group?.providerMissing || false,
        providerDisabled: group?.providerDisabled || false
      };
    });
  }

  async getEvents() {
    return this.state.events;
  }

  async getRenderedConfig() {
    const content = await readFile(this.layout.mihomoConfigPath, "utf8");
    return {
      path: this.layout.mihomoConfigPath,
      updatedAt: this.state.runtime.lastRenderAt,
      content
    };
  }

  async testSubscription({ name, type, url }) {
    if (!url) {
      return { ok: false, error: "订阅 URL 不能为空" };
    }

    try {
      const result = await fetchProviderSnapshot({
        name: name || "test-provider",
        type: type || "mihomo-http",
        url
      });
      pushEvent(this.state, result.ok ? "info" : "warn", "provider-test", `临时订阅测试完成: ${name || "test-provider"}`);
      await this.persistState();
      return {
        ok: result.ok,
        result: {
          ...result,
          previewNodes: result.nodes.slice(0, 20)
        }
      };
    } catch (error) {
      pushEvent(this.state, "error", "provider-test", "临时订阅测试失败");
      await this.persistState();
      return {
        ok: false,
        error: error instanceof Error ? error.message : "订阅测试失败"
      };
    }
  }

  async getRuntimePreflight() {
    await this.ensureFreshControllerStatus();
    return buildRuntimePreflightSnapshot({
      configPath: this.configPath,
      config: this.config,
      layout: this.layout,
      state: this.state,
      controller: this.state.controller
    });
  }

  async getControllerStatus() {
    await this.ensureFreshControllerStatus();
    return {
      baseUrl: this.config.runtime.external_controller,
      secretConfigured: Boolean(this.config.runtime.external_secret),
      renderOnly: this.config.runtime.render_only,
      ...(this.state.controller || {
        reachable: false,
        mode: this.config.runtime.render_only ? "render-only" : "managed",
        version: null,
        message: "尚未探测"
      })
    };
  }

  async probeController() {
    await this.detectController({ recordEvent: true, persist: true });
    return {
      ok: this.state.controller?.reachable ?? false,
      controller: await this.getControllerStatus()
    };
  }

  async reloadConfig() {
    await this.loadAll();
    pushEvent(this.state, "info", "reload", "配置已重新加载并渲染");
    await this.persistState();
    const runtimeApply = await this.applyRuntimeConfig("配置重载");
    return {
      ok: true,
      message: runtimeApply.ok
        ? "配置已重新加载并下发到运行态"
        : runtimeApply.deferred
        ? `配置已重新加载：${runtimeApply.message}`
        : `配置已重新加载，但运行态下发失败：${runtimeApply.message}`,
      runtime: this.state.runtime,
      runtimeApply
    };
  }

  async applyRuntimeConfig(reason) {
    const appliedAt = nowIso();

    if (this.config.runtime.render_only) {
      const message = "当前为 render-only 模式，仅完成配置渲染";
      this.state.runtime.lastApplyAt = appliedAt;
      this.state.runtime.lastApplyStatus = "deferred";
      this.state.runtime.lastApplyMessage = message;
      await this.persistState();
      return {
        ok: false,
        deferred: true,
        message
      };
    }

    await this.ensureFreshControllerStatus({ maxAgeMs: 0 });

    try {
      const result = await this.controller.reloadConfig(this.layout.mihomoConfigPath);
      const message = result.ok
        ? `运行态已加载 ${this.layout.mihomoConfigPath}`
        : readControllerResultMessage(result, `controller 返回 ${result.status}`);
      this.state.runtime.lastApplyAt = appliedAt;
      this.state.runtime.lastApplyStatus = result.ok ? "success" : "failed";
      this.state.runtime.lastApplyMessage = message;
      pushEvent(this.state, result.ok ? "info" : "warn", "runtime-apply", `${reason}: ${message}`);
      await this.persistState();
      return {
        ...result,
        message
      };
    } catch (error) {
      const message = error instanceof Error ? error.message : "运行态下发失败";
      this.state.runtime.lastApplyAt = appliedAt;
      this.state.runtime.lastApplyStatus = "failed";
      this.state.runtime.lastApplyMessage = message;
      pushEvent(this.state, "error", "runtime-apply", `${reason}: ${message}`);
      await this.persistState();
      return {
        ok: false,
        message
      };
    }
  }

  async refreshProvider(providerName) {
    const subscription = this.config.subscriptions.find((item) => item.name === providerName);
    if (!subscription) {
      return { ok: false, error: `未找到 provider: ${providerName}` };
    }

    try {
      const snapshot = await fetchProviderSnapshot(subscription);
      this.state.providers[providerName] = {
        ...snapshot,
        lastError: snapshot.ok ? null : `${snapshot.status} ${snapshot.statusText}`
      };
      let runtimeRefresh;
      try {
        runtimeRefresh = await this.controller.refreshProvider(providerName);
      } catch (runtimeError) {
        runtimeRefresh = {
          ok: false,
          error: runtimeError instanceof Error ? runtimeError.message : "运行态刷新失败"
        };
      }
      pushEvent(
        this.state,
        snapshot.ok ? "info" : "warn",
        "provider",
        `${providerName} 刷新完成，节点数 ${snapshot.nodeCount}`
      );
      await this.persistState();
      return {
        ok: snapshot.ok,
        provider: this.state.providers[providerName],
        runtime: runtimeRefresh
      };
    } catch (error) {
      this.state.providers[providerName] = {
        ...this.getProviderRecord(providerName),
        refreshedAt: nowIso(),
        lastError: error instanceof Error ? error.message : "刷新失败"
      };
      pushEvent(this.state, "error", "provider", `${providerName} 刷新失败`);
      await this.persistState();
      return {
        ok: false,
        error: error instanceof Error ? error.message : "刷新失败",
        provider: this.state.providers[providerName]
      };
    }
  }

  async selectGroup(groupName, proxyName) {
    const group = this.config.egress_groups.find((item) => item.name === groupName);
    if (!group) {
      return { ok: false, error: `未找到出口组: ${groupName}` };
    }

    const view = this.buildGroupView(group);
    const match = view.candidates.find((candidate) => candidate.name === proxyName);
    if (!match) {
      return { ok: false, error: `节点不属于出口组 ${groupName}` };
    }

    this.state.groupSelections[groupName] = proxyName;
    let runtimeSelect;
    try {
      runtimeSelect = await this.controller.selectProxy(groupName, proxyName);
    } catch (runtimeError) {
      runtimeSelect = {
        ok: false,
        error: runtimeError instanceof Error ? runtimeError.message : "运行态切换失败"
      };
    }
    pushEvent(this.state, "info", "group", `出口组 ${groupName} 已切换到 ${proxyName}`);
    await this.persistState();
    return {
      ok: true,
      group: this.buildGroupView(group),
      runtime: runtimeSelect
    };
  }

  async addSubscription(data) {
    if (!data.name || !data.url) {
      return { ok: false, error: "订阅名称和 URL 不能为空" };
    }
    if (this.config.subscriptions.some((s) => s.name === data.name)) {
      return { ok: false, error: `订阅 ${data.name} 已存在` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.subscriptions.push({
      name: data.name,
      type: data.type || "mihomo-http",
      url: data.url,
      interval: Number(data.interval || 3600),
      enabled: data.enabled !== false,
      health_check_url: data.health_check_url || "https://www.gstatic.com/generate_204",
      health_check_interval: Number(data.health_check_interval || 300)
    });
    const result = await this.saveConfig(nextConfig);
    if (result.ok) {
      await this.refreshProvider(data.name).catch(() => {});
    }
    return result;
  }

  async removeSubscription(name) {
    if (!this.config.subscriptions.some((s) => s.name === name)) {
      return { ok: false, error: `订阅 ${name} 不存在` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.subscriptions = nextConfig.subscriptions.filter((s) => s.name !== name);
    delete this.state.providers[name];
    return this.saveConfig(nextConfig);
  }

  async toggleSubscription(name, enabled) {
    const idx = this.config.subscriptions.findIndex((s) => s.name === name);
    if (idx < 0) {
      return { ok: false, error: `订阅 ${name} 不存在` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.subscriptions[idx].enabled = enabled;
    return this.saveConfig(nextConfig);
  }

  async addEgressGroup(data) {
    if (!data.name || !data.provider) {
      return { ok: false, error: "出口组名称和 provider 不能为空" };
    }
    if (this.config.egress_groups.some((g) => g.name === data.name)) {
      return { ok: false, error: `出口组 ${data.name} 已存在` };
    }
    if (!this.config.subscriptions.some((s) => s.name === data.provider)) {
      return { ok: false, error: `provider ${data.provider} 不存在` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.egress_groups.push({
      name: data.name,
      provider: data.provider,
      mode: data.mode || "manual",
      filter: data.filter || "",
      exclude_filter: data.exclude_filter || ""
    });
    return this.saveConfig(nextConfig);
  }

  async updateEgressGroup(name, updates) {
    const idx = this.config.egress_groups.findIndex((g) => g.name === name);
    if (idx < 0) {
      return { ok: false, error: `出口组 ${name} 不存在` };
    }
    if (updates.provider && !this.config.subscriptions.some((s) => s.name === updates.provider)) {
      return { ok: false, error: `provider ${updates.provider} 不存在` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.egress_groups[idx] = { ...nextConfig.egress_groups[idx], ...updates };
    return this.saveConfig(nextConfig);
  }

  async removeEgressGroup(name) {
    if (!this.config.egress_groups.some((g) => g.name === name)) {
      return { ok: false, error: `出口组 ${name} 不存在` };
    }
    const referencingListeners = this.config.listeners.filter((l) => l.egress_group === name);
    if (referencingListeners.length > 0) {
      return { ok: false, error: `出口组 ${name} 仍被监听接口引用，请先删除相关接口` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.egress_groups = nextConfig.egress_groups.filter((g) => g.name !== name);
    return this.saveConfig(nextConfig);
  }

  async addListener(data) {
    if (!data.name || !data.port || !data.egress_group) {
      return { ok: false, error: "监听名称、端口和出口组不能为空" };
    }
    if (this.config.listeners.some((l) => l.name === data.name)) {
      return { ok: false, error: `监听 ${data.name} 已存在` };
    }
    if (this.config.listeners.some((l) => l.port === Number(data.port))) {
      return { ok: false, error: `端口 ${data.port} 已被占用` };
    }
    if (!this.config.egress_groups.some((g) => g.name === data.egress_group)) {
      return { ok: false, error: `出口组 ${data.egress_group} 不存在` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.listeners.push({
      name: data.name,
      type: data.type || "socks",
      listen: data.listen || "0.0.0.0",
      port: Number(data.port),
      udp: data.udp !== false,
      enabled: data.enabled !== false,
      users: Array.isArray(data.users) ? data.users : [],
      egress_group: data.egress_group
    });
    return this.saveConfig(nextConfig);
  }

  async updateListener(name, updates) {
    const idx = this.config.listeners.findIndex((l) => l.name === name);
    if (idx < 0) {
      return { ok: false, error: `监听 ${name} 不存在` };
    }
    if (updates.port && updates.port !== this.config.listeners[idx].port) {
      if (this.config.listeners.some((l, i) => i !== idx && l.port === Number(updates.port))) {
        return { ok: false, error: `端口 ${updates.port} 已被占用` };
      }
    }
    if (updates.egress_group && !this.config.egress_groups.some((g) => g.name === updates.egress_group)) {
      return { ok: false, error: `出口组 ${updates.egress_group} 不存在` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.listeners[idx] = { ...nextConfig.listeners[idx], ...updates };
    return this.saveConfig(nextConfig);
  }

  async removeListener(name) {
    if (!this.config.listeners.some((l) => l.name === name)) {
      return { ok: false, error: `监听 ${name} 不存在` };
    }
    const nextConfig = JSON.parse(JSON.stringify(this.config));
    nextConfig.listeners = nextConfig.listeners.filter((l) => l.name !== name);
    return this.saveConfig(nextConfig);
  }

  async runHealthcheck(groupName) {
    const group = this.config.egress_groups.find((item) => item.name === groupName);
    if (!group) {
      return { ok: false, error: `未找到出口组: ${groupName}` };
    }

    this.state.groups = this.state.groups || {};
    this.state.groups[groupName] = {
      ...(this.state.groups[groupName] || {}),
      lastHealthcheckAt: nowIso()
    };
    pushEvent(this.state, "info", "healthcheck", `已执行出口组健康检查: ${groupName}`);
    await this.persistState();
    return { ok: true, group: this.buildGroupView(group) };
  }
}
