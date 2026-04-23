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

async function main() {
  const profile = await writeDevConfig({ renderOnlyMode });

  const child = spawn(getGoCommand(), ["run", "github.com/wailsapp/wails/v2/cmd/wails@v2.11.0", "dev"], {
    cwd: desktopDir,
    env: resolveGoEnv({
      AXIS_HOME: profile.axisHome,
      PROXYRELAY_CONFIG: profile.configPath,
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
