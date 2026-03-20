import { mkdir, readFile, rename, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import YAML from "yaml";
import { compilePattern } from "./pattern.js";

const DEFAULT_LOCAL_RUNTIME_WORKDIR = "../runtime";
const DEFAULT_SYSTEM_RUNTIME_WORKDIR = "/var/lib/proxyrelay/runtime";

function ensureArray(value) {
  return Array.isArray(value) ? value : [];
}

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function toConfigFilePath(configPath) {
  return configPath instanceof URL ? fileURLToPath(configPath) : String(configPath);
}

export function resolveDefaultRuntimeWorkdir(configPath) {
  const configFilePath = toConfigFilePath(configPath);
  const configDir = path.resolve(path.dirname(configFilePath));

  if (configDir === "/etc/proxyrelay") {
    return DEFAULT_SYSTEM_RUNTIME_WORKDIR;
  }

  return DEFAULT_LOCAL_RUNTIME_WORKDIR;
}

export async function loadConfig(configPath) {
  const raw = await readFile(configPath, "utf8");
  const parsed = YAML.parse(raw) || {};
  const runtimeWorkdir = parsed.runtime?.workdir || resolveDefaultRuntimeWorkdir(configPath);

  const config = {
    server: {
      host: parsed.server?.host || "127.0.0.1",
      port: Number(parsed.server?.port || 8787),
      startup_refresh: parsed.server?.startup_refresh ?? false
    },
    admin: {
      username: parsed.admin?.username || "admin",
      password: parsed.admin?.password || "",
      password_hash: parsed.admin?.password_hash || "",
      session_secret: parsed.admin?.session_secret || "",
      session_ttl_hours: Number(parsed.admin?.session_ttl_hours || 12)
    },
    runtime: {
      workdir: runtimeWorkdir,
      mihomo_binary: parsed.runtime?.mihomo_binary || "mihomo",
      external_controller: parsed.runtime?.external_controller || "http://127.0.0.1:9090",
      external_secret: parsed.runtime?.external_secret || "",
      render_only: parsed.runtime?.render_only ?? true
    },
    subscriptions: ensureArray(parsed.subscriptions),
    egress_groups: ensureArray(parsed.egress_groups),
    listeners: ensureArray(parsed.listeners)
  };

  validateConfig(config);
  return config;
}

export function validateConfig(config) {
  assert(config.server.port > 0, "server.port 必须大于 0");
  assert(config.admin.username, "admin.username 不能为空");
  assert(config.admin.password || config.admin.password_hash, "admin.password 或 admin.password_hash 至少提供一个");

  const providerNames = new Set();
  for (const subscription of config.subscriptions) {
    assert(subscription.name, "subscriptions[].name 不能为空");
    assert(subscription.url, `订阅 ${subscription.name} 缺少 url`);
    if (providerNames.has(subscription.name)) {
      throw new Error(`重复的订阅名称: ${subscription.name}`);
    }
    providerNames.add(subscription.name);
  }

  const groupNames = new Set();
  for (const group of config.egress_groups) {
    assert(group.name, "egress_groups[].name 不能为空");
    assert(group.provider, `出口组 ${group.name} 缺少 provider`);
    if (group.filter) {
      compilePattern(group.filter);
    }
    if (group.exclude_filter) {
      compilePattern(group.exclude_filter);
    }
    if (groupNames.has(group.name)) {
      throw new Error(`重复的出口组名称: ${group.name}`);
    }
    groupNames.add(group.name);
  }

  const ports = new Set();
  for (const listener of config.listeners) {
    assert(listener.name, "listeners[].name 不能为空");
    assert(listener.port, `监听 ${listener.name} 缺少 port`);
    assert(listener.egress_group, `监听 ${listener.name} 缺少 egress_group`);
    if (ports.has(listener.port)) {
      throw new Error(`重复的监听端口: ${listener.port}`);
    }
    ports.add(listener.port);
  }
}

export async function ensureRuntimeLayout(config, configPath) {
  const configFilePath = toConfigFilePath(configPath);
  const rootDir = path.dirname(configFilePath);
  const runtimeDir = path.resolve(rootDir, config.runtime.workdir);
  const providersDir = path.join(runtimeDir, "providers");
  await mkdir(runtimeDir, { recursive: true });
  await mkdir(providersDir, { recursive: true });

  return {
    rootDir,
    runtimeDir,
    providersDir,
    mihomoConfigPath: path.join(runtimeDir, "mihomo.yaml"),
    lastGoodConfigPath: path.join(runtimeDir, "mihomo.last-good.yaml"),
    statePath: path.join(runtimeDir, "control-state.json")
  };
}

export async function persistJson(filePath, data) {
  await writeFile(filePath, `${JSON.stringify(data, null, 2)}\n`, "utf8");
}

function cloneSerializable(value) {
  return JSON.parse(JSON.stringify(value));
}

export function normalizeConfigForSave(config, configPath = "config/proxyrelay.yaml") {
  return cloneSerializable({
    server: {
      host: config.server?.host || "127.0.0.1",
      port: Number(config.server?.port || 8787),
      startup_refresh: Boolean(config.server?.startup_refresh)
    },
    admin: {
      username: config.admin?.username || "admin",
      password: config.admin?.password || "",
      password_hash: config.admin?.password_hash || "",
      session_secret: config.admin?.session_secret || "",
      session_ttl_hours: Number(config.admin?.session_ttl_hours || 12)
    },
    runtime: {
      workdir: config.runtime?.workdir || resolveDefaultRuntimeWorkdir(configPath),
      mihomo_binary: config.runtime?.mihomo_binary || "mihomo",
      external_controller: config.runtime?.external_controller || "http://127.0.0.1:9090",
      external_secret: config.runtime?.external_secret || "",
      render_only: config.runtime?.render_only ?? true
    },
    subscriptions: ensureArray(config.subscriptions).map((item) => ({
      name: item.name || "",
      type: item.type || "mihomo-http",
      url: item.url || "",
      interval: Number(item.interval || 3600),
      enabled: item.enabled !== false,
      health_check_url: item.health_check_url || "https://www.gstatic.com/generate_204",
      health_check_interval: Number(item.health_check_interval || 300),
      headers: item.headers || undefined,
      via: item.via || undefined
    })),
    egress_groups: ensureArray(config.egress_groups).map((item) => ({
      name: item.name || "",
      provider: item.provider || "",
      mode: item.mode || "manual",
      filter: item.filter || "",
      exclude_filter: item.exclude_filter || "",
      strategy: item.strategy || undefined,
      health_check_url: item.health_check_url || undefined,
      interval: item.interval ? Number(item.interval) : undefined,
      fallback: item.fallback || undefined
    })),
    listeners: ensureArray(config.listeners).map((item) => ({
      name: item.name || "",
      type: item.type || "socks",
      listen: item.listen || "0.0.0.0",
      port: Number(item.port || 0),
      udp: item.udp !== false,
      enabled: item.enabled !== false,
      users: ensureArray(item.users).map((user) => ({
        username: user.username || "",
        password: user.password || ""
      })),
      egress_group: item.egress_group || ""
    }))
  });
}

export async function writeConfig(configPath, config) {
  const normalized = normalizeConfigForSave(config, configPath);
  validateConfig({
    ...normalized,
    server: normalized.server,
    admin: normalized.admin,
    runtime: normalized.runtime
  });

  const tempPath = `${configPath}.tmp`;
  const body = YAML.stringify(normalized);
  await writeFile(tempPath, body, "utf8");
  await rename(tempPath, configPath);
  return normalized;
}
