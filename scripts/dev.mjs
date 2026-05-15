import { spawn } from "node:child_process";
import path from "node:path";
import readline from "node:readline";
import { fileURLToPath } from "node:url";

import { writeDevConfig } from "./dev-config.mjs";
import { getGoCommand, resolveGoEnv } from "./go-env.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");

const args = new Set(process.argv.slice(2));
const renderOnlyMode = args.has("--render-only");

const uiHost = process.env.AXIS_UI_HOST || "127.0.0.1";
const uiPort = Number(process.env.AXIS_UI_PORT || 5173);

const processes = [];
let shuttingDown = false;

function log(message) {
  console.log(`[dev] ${message}`);
}

function getPnpmCommand() {
  return process.platform === "win32" ? "pnpm.cmd" : "pnpm";
}

const frontendWorkspaceName = "@axis/frontend";

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

async function main() {
  process.on("SIGINT", () => {
    shutdown("收到 SIGINT，正在停止开发环境...").catch(() => process.exit(1));
  });
  process.on("SIGTERM", () => {
    shutdown("收到 SIGTERM，正在停止开发环境...").catch(() => process.exit(1));
  });

  const profile = await writeDevConfig({ renderOnlyMode });

  const backendEnv = {
    AXIS_HOME: profile.axisHome,
    PROXYRELAY_CONFIG: profile.configPath,
    AXIS_UI_DEV_URL: `http://${uiHost}:${uiPort}`,
  };
  const frontendEnv = {
    AXIS_SERVER_HOST: profile.serverHost,
    AXIS_SERVER_PORT: String(profile.serverPort),
    AXIS_UI_HOST: uiHost,
    AXIS_UI_PORT: String(uiPort),
  };

  spawnProcess("axis", getGoCommand(), ["run", "./cmd/axis", "serve"], resolveGoEnv(backendEnv));
  spawnProcess(
    "ui",
    getPnpmCommand(),
    ["--filter", frontendWorkspaceName, "dev", "--host", uiHost, "--port", String(uiPort)],
    frontendEnv,
  );

  log(`React 开发服务器: http://${uiHost}:${uiPort}`);
  log(`AXIS API 地址: http://${profile.serverHost}:${profile.serverPort}`);
  log(`AXIS 控制台入口: http://${profile.serverHost}:${profile.serverPort}`);
  if (renderOnlyMode) {
    log("当前只保存配置，不会自动应用到代理核心。");
  } else {
    log("当前为托管运行态；代理核心状态请在 AXIS 控制台内查看和维护。");
  }
}

main().catch((error) => {
  console.error("[dev] 启动失败:", error instanceof Error ? error.message : error);
  process.exit(1);
});
