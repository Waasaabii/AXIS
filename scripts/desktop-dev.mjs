import net from "node:net";
import { spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { writeDevConfig } from "./dev-config.mjs";
import { getGoCommand, resolveGoEnv } from "./go-env.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const desktopDir = path.join(repoRoot, "desktop");
const args = new Set(process.argv.slice(2));
const renderOnlyMode = args.has("--render-only");

function isPortAvailable(port, host = "127.0.0.1") {
  return new Promise((resolve) => {
    const server = net.createServer();
    server.once("error", () => resolve(false));
    server.once("listening", () => {
      server.close(() => resolve(true));
    });
    server.listen(port, host);
  });
}

async function findAvailablePort(preferredPort, host = "127.0.0.1") {
  const start = Number(preferredPort);
  for (let port = start; port < start + 100; port += 1) {
    if (await isPortAvailable(port, host)) return port;
  }
  throw new Error(`没有找到可用端口：${start}-${start + 99}`);
}

async function main() {
  const profile = await writeDevConfig({ renderOnlyMode });
  const uiHost = process.env.AXIS_UI_HOST || "127.0.0.1";
  const uiPort = await findAvailablePort(process.env.AXIS_UI_PORT || 5173, uiHost);
  const devServerPort = await findAvailablePort(process.env.AXIS_WAILS_DEVSERVER_PORT || 34115, "127.0.0.1");
  const frontendURL = `http://${uiHost}:${uiPort}`;
  const devServerURL = `http://localhost:${devServerPort}`;
  console.log(`[desktop-dev] Frontend: ${frontendURL}`);
  console.log(`[desktop-dev] Wails DevServer: ${devServerURL}`);

  const child = spawn(getGoCommand(), [
    "run",
    "github.com/wailsapp/wails/v2/cmd/wails@v2.11.0",
    "dev",
    "-frontenddevserverurl",
    frontendURL,
    "-devserver",
    devServerURL,
  ], {
    cwd: desktopDir,
    env: resolveGoEnv({
      AXIS_HOME: profile.axisHome,
      PROXYRELAY_CONFIG: profile.configPath,
      AXIS_UI_HOST: uiHost,
      AXIS_UI_PORT: String(uiPort),
    }),
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
