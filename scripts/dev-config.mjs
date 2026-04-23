import { access, mkdir, readFile, writeFile } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import path from "node:path";
import YAML from "yaml";

import { resolveDevConfigPath, resolveDevHome, resolveDevRuntimeDir } from "./axis-paths.mjs";
import { readRepoConfigTemplate } from "./config-template.mjs";

function envString(name, fallback = "") {
  const value = process.env[name];
  if (value === undefined || value === null) {
    return fallback;
  }
  const trimmed = String(value).trim();
  return trimmed ? trimmed : fallback;
}

function envNumber(name, fallback) {
  const raw = process.env[name];
  if (raw === undefined || raw === null) {
    return fallback;
  }
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? parsed : fallback;
}

async function fileExists(targetPath) {
  try {
    await access(targetPath, fsConstants.F_OK);
    return true;
  } catch {
    return false;
  }
}

async function readSourceConfig(configPath) {
  if (await fileExists(configPath)) {
    return YAML.parse(await readFile(configPath, "utf8")) || {};
  }
  return readRepoConfigTemplate();
}

export function resolveDevProfile() {
  const axisHome = resolveDevHome();
  const runtimeDir = resolveDevRuntimeDir();
  const configPath = resolveDevConfigPath();
  return { axisHome, runtimeDir, configPath };
}

export async function writeDevConfig({ renderOnlyMode } = {}) {
  const { axisHome, runtimeDir, configPath } = resolveDevProfile();
  const managedMode = !renderOnlyMode;

  const serverHost = envString("AXIS_SERVER_HOST", "127.0.0.1");
  const serverPort = envNumber("AXIS_SERVER_PORT", 8787);
  const controllerHost = envString("AXIS_CONTROLLER_HOST", "127.0.0.1");
  const controllerPort = envNumber("AXIS_CONTROLLER_PORT", 11235);
  const controllerSecret = envString("AXIS_CONTROLLER_SECRET", "proxyrelay-local-secret");
  const mihomoBinary = envString("AXIS_MIHOMO_BIN", envString("MIHOMO_BIN", "mihomo"));

  const source = await readSourceConfig(configPath);
  const config = JSON.parse(JSON.stringify(source));

  config.server = {
    ...(config.server || {}),
    host: serverHost,
    port: serverPort,
  };
  config.runtime = {
    ...(config.runtime || {}),
    workdir: "../runtime",
    render_only: !managedMode,
    mihomo_binary: mihomoBinary,
    external_controller: `http://${controllerHost}:${controllerPort}`,
    external_secret: controllerSecret,
  };

  await mkdir(path.dirname(configPath), { recursive: true });
  await mkdir(runtimeDir, { recursive: true });
  await writeFile(configPath, YAML.stringify(config), "utf8");

  return {
    axisHome,
    runtimeDir,
    configPath,
    serverHost,
    serverPort,
    controllerHost,
    controllerPort,
    controllerSecret,
    mihomoBinary,
    config,
    managedMode,
  };
}

