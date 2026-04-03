import { spawn } from "node:child_process";
import { access, mkdir, readFile, writeFile } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import YAML from "yaml";
import { resolveDevConfigPath, resolveDevHome, resolveDevRuntimeDir } from "./axis-paths.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const desktopDir = path.join(repoRoot, "desktop");
const axisHome = resolveDevHome();
const runtimeDir = resolveDevRuntimeDir();
const localConfigPath = resolveDevConfigPath();
const args = new Set(process.argv.slice(2));
const renderOnlyMode = args.has("--render-only");
const managedMode = !renderOnlyMode;

const serverHost = process.env.AXIS_SERVER_HOST || "127.0.0.1";
const serverPort = Number(process.env.AXIS_SERVER_PORT || 8787);
const controllerHost = process.env.AXIS_CONTROLLER_HOST || "127.0.0.1";
const controllerPort = Number(process.env.AXIS_CONTROLLER_PORT || 11235);
const controllerSecret = process.env.AXIS_CONTROLLER_SECRET || "proxyrelay-local-secret";
const mihomoBinary = process.env.AXIS_MIHOMO_BIN || process.env.MIHOMO_BIN || "mihomo";

function getGoCommand() {
  return process.platform === "win32" ? "go.exe" : "go";
}

async function fileExists(targetPath) {
  try {
    await access(targetPath, fsConstants.F_OK);
    return true;
  } catch {
    return false;
  }
}

async function readSourceConfig() {
  const candidates = [
    localConfigPath,
    path.join(repoRoot, "config", "proxyrelay.yaml"),
    path.join(repoRoot, "config", "proxyrelay.example.yaml"),
  ];

  for (const candidate of candidates) {
    if (await fileExists(candidate)) {
      return YAML.parse(await readFile(candidate, "utf8")) || {};
    }
  }

  throw new Error("未找到 config/proxyrelay.yaml 或 config/proxyrelay.example.yaml");
}

async function writeDevConfig() {
  const source = await readSourceConfig();
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

  await mkdir(path.dirname(localConfigPath), { recursive: true });
  await mkdir(runtimeDir, { recursive: true });
  await writeFile(localConfigPath, YAML.stringify(config), "utf8");
}

async function main() {
  await writeDevConfig();

  const child = spawn(getGoCommand(), ["run", "github.com/wailsapp/wails/v2/cmd/wails@v2.11.0", "dev"], {
    cwd: desktopDir,
    env: {
      ...process.env,
      AXIS_HOME: axisHome,
      PROXYRELAY_CONFIG: localConfigPath,
    },
    stdio: "inherit",
  });

  child.on("exit", (code) => {
    process.exit(code ?? 0);
  });
  child.on("error", (error) => {
    console.error("[desktop-dev] 启动失败:", error.message);
    process.exit(1);
  });

  process.on("SIGINT", () => child.kill("SIGINT"));
  process.on("SIGTERM", () => child.kill("SIGTERM"));
}

main().catch((error) => {
  console.error("[desktop-dev] 失败:", error instanceof Error ? error.message : error);
  process.exit(1);
});
