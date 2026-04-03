import { spawn } from "node:child_process";
import { access, mkdir, readFile, writeFile } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import path from "node:path";
import readline from "node:readline";
import { fileURLToPath } from "node:url";
import YAML from "yaml";
import { resolveDevConfigPath, resolveDevHome, resolveDevRuntimeDir } from "./axis-paths.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const axisHome = resolveDevHome();
const runtimeDir = resolveDevRuntimeDir();
const localConfigPath = resolveDevConfigPath();

const args = new Set(process.argv.slice(2));
const renderOnlyMode = args.has("--render-only");
const managedMode = !renderOnlyMode;

const serverHost = process.env.AXIS_SERVER_HOST || "127.0.0.1";
const serverPort = Number(process.env.AXIS_SERVER_PORT || 8787);
const uiHost = process.env.AXIS_UI_HOST || "127.0.0.1";
const uiPort = Number(process.env.AXIS_UI_PORT || 5173);
const controllerHost = process.env.AXIS_CONTROLLER_HOST || "127.0.0.1";
const controllerPort = Number(process.env.AXIS_CONTROLLER_PORT || 11235);
const controllerSecret = process.env.AXIS_CONTROLLER_SECRET || "proxyrelay-local-secret";
const mihomoBinary = process.env.AXIS_MIHOMO_BIN || process.env.MIHOMO_BIN || "mihomo";

const processes = [];
let shuttingDown = false;

function log(message) {
  console.log(`[dev] ${message}`);
}

function getPnpmCommand() {
  return process.platform === "win32" ? "pnpm.cmd" : "pnpm";
}

const frontendWorkspaceName = "@axis/frontend";

function getGoCommand() {
  return process.platform === "win32" ? "go.exe" : "go";
}

function getExecutableCandidates(binary) {
  if (path.isAbsolute(binary) || binary.includes(path.sep)) {
    return [binary];
  }

  const pathEntries = (process.env.PATH || "").split(path.delimiter).filter(Boolean);
  const extensions =
    process.platform === "win32"
      ? (process.env.PATHEXT || ".EXE;.CMD;.BAT;.COM")
          .split(";")
          .map((item) => item.toLowerCase())
      : [""];

  return pathEntries.flatMap((entry) =>
    extensions.map((extension) => path.join(entry, process.platform === "win32" ? `${binary}${extension}` : binary)),
  );
}

async function fileExists(targetPath) {
  try {
    await access(targetPath, fsConstants.F_OK);
    return true;
  } catch {
    return false;
  }
}

async function resolveExecutable(binary) {
  const candidates = getExecutableCandidates(binary);

  for (const candidate of candidates) {
    if (await fileExists(candidate)) {
      return candidate;
    }
  }

  return "";
}

function pipeStream(stream, prefix) {
  if (!stream) {
    return;
  }

  const rl = readline.createInterface({ input: stream });
  rl.on("line", (line) => {
    console.log(`${prefix} ${line}`);
  });
}

function spawnProcess(name, command, commandArgs, extraEnv = {}) {
  const child = spawn(command, commandArgs, {
    cwd: repoRoot,
    env: {
      ...process.env,
      ...extraEnv,
    },
    stdio: ["inherit", "pipe", "pipe"],
  });

  pipeStream(child.stdout, `[${name}]`);
  pipeStream(child.stderr, `[${name}]`);

  child.on("error", (error) => {
    if (shuttingDown) {
      return;
    }

    shutdown(`${name} 启动失败: ${error.message}`, 1).catch(() => process.exit(1));
  });

  child.on("exit", (code, signal) => {
    if (shuttingDown) {
      return;
    }

    const reason =
      signal !== null ? `${name} 已因信号 ${signal} 退出` : `${name} 已退出，退出码 ${code ?? "unknown"}`;
    shutdown(reason, code ?? 1).catch((error) => {
      console.error("[dev] 退出清理失败:", error);
      process.exit(code ?? 1);
    });
  });

  processes.push(child);
  return child;
}

async function stopProcess(child) {
  if (!child || child.exitCode !== null || child.killed) {
    return;
  }

  child.kill("SIGTERM");
  await new Promise((resolve) => setTimeout(resolve, 500));

  if (child.exitCode === null && !child.killed) {
    child.kill("SIGKILL");
  }
}

async function shutdown(reason, code = 0) {
  if (shuttingDown) {
    return;
  }

  shuttingDown = true;
  if (reason) {
    log(reason);
  }

  await Promise.all(processes.map((child) => stopProcess(child)));
  process.exit(code);
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

  return config;
}

async function runPreflight() {
  return new Promise((resolve, reject) => {
    const child = spawn(getGoCommand(), ["run", "./cmd/axis", "preflight"], {
      cwd: repoRoot,
      env: {
        ...process.env,
        AXIS_HOME: axisHome,
        PROXYRELAY_CONFIG: localConfigPath,
      },
      stdio: ["ignore", "pipe", "pipe"],
    });

    pipeStream(child.stdout, "[preflight]");
    pipeStream(child.stderr, "[preflight]");

    child.on("error", reject);
    child.on("exit", () => resolve());
  });
}

async function waitForController() {
  const url = `http://${controllerHost}:${controllerPort}/version`;

  for (let attempt = 0; attempt < 20; attempt += 1) {
    try {
      const response = await fetch(url, {
        headers: {
          Authorization: `Bearer ${controllerSecret}`,
        },
      });
      if (response.ok) {
        return;
      }
    } catch {
      // ignore
    }

    await new Promise((resolve) => setTimeout(resolve, 500));
  }

  throw new Error(`Mihomo controller 未在 ${url} 就绪`);
}

async function startManagedRuntime() {
  const executable = await resolveExecutable(mihomoBinary);
  if (!executable) {
    throw new Error(`未找到 mihomo 可执行文件: ${mihomoBinary}`);
  }

  await runPreflight();
  spawnProcess("mihomo", executable, ["-d", runtimeDir, "-f", path.join(runtimeDir, "mihomo.yaml")]);
  await waitForController();
}

async function main() {
  process.on("SIGINT", () => {
    shutdown("收到 SIGINT，正在停止开发环境...").catch(() => process.exit(1));
  });
  process.on("SIGTERM", () => {
    shutdown("收到 SIGTERM，正在停止开发环境...").catch(() => process.exit(1));
  });

  await writeDevConfig();

  if (managedMode) {
    await startManagedRuntime();
  }

  const backendEnv = {
    AXIS_HOME: axisHome,
    PROXYRELAY_CONFIG: localConfigPath,
    AXIS_UI_DEV_URL: `http://${uiHost}:${uiPort}`,
  };
  const frontendEnv = {
    AXIS_SERVER_HOST: serverHost,
    AXIS_SERVER_PORT: String(serverPort),
    AXIS_UI_HOST: uiHost,
    AXIS_UI_PORT: String(uiPort),
  };

  spawnProcess("axis", getGoCommand(), ["run", "./cmd/axis", "serve"], backendEnv);
  spawnProcess("ui", getPnpmCommand(), ["--filter", frontendWorkspaceName, "dev", "--host", uiHost, "--port", String(uiPort)], frontendEnv);

  log(`React 开发服务器: http://${uiHost}:${uiPort}`);
  log(`AXIS API 地址: http://${serverHost}:${serverPort}`);
  log(`AXIS 控制台入口: http://${serverHost}:${serverPort}`);
  if (managedMode) {
    log(`Mihomo Controller: http://${controllerHost}:${controllerPort}`);
  } else {
    log("当前为仅渲染配置模式，如需启用完整运行时能力请直接使用 pnpm dev");
  }
}

main().catch((error) => {
  console.error("[dev] 启动失败:", error instanceof Error ? error.message : error);
  process.exit(1);
});
