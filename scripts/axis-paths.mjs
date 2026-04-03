import os from "node:os";
import path from "node:path";

export function resolveAxisHome() {
  if (process.env.AXIS_HOME && process.env.AXIS_HOME.trim()) {
    return path.resolve(process.env.AXIS_HOME.trim());
  }

  if (process.platform === "darwin") {
    return path.join(os.homedir(), "Library", "Application Support", "AXIS");
  }
  if (process.platform === "win32") {
    const appData = process.env.APPDATA || path.join(os.homedir(), "AppData", "Roaming");
    return path.join(appData, "AXIS");
  }
  const xdgConfigHome = process.env.XDG_CONFIG_HOME || path.join(os.homedir(), ".config");
  return path.join(xdgConfigHome, "AXIS");
}

export function resolveDevHome() {
  return path.join(resolveAxisHome(), "dev");
}

export function resolveDevConfigPath() {
  return path.join(resolveDevHome(), "config", "proxyrelay.yaml");
}

export function resolveDevRuntimeDir() {
  return path.join(resolveDevHome(), "runtime");
}
