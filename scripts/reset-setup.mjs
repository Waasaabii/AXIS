import { spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { resolveDevConfigPath, resolveDevHome } from "./axis-paths.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");

function getGoCommand() {
  return process.platform === "win32" ? "go.exe" : "go";
}

async function main() {
  const child = spawn(getGoCommand(), ["run", "./cmd/axis", "reset-setup", resolveDevConfigPath()], {
    cwd: repoRoot,
    env: {
      ...process.env,
      AXIS_HOME: resolveDevHome(),
    },
    stdio: "inherit",
  });

  child.on("exit", (code) => {
    process.exit(code ?? 0);
  });
  child.on("error", (error) => {
    console.error("[reset-setup] 启动失败:", error.message);
    process.exit(1);
  });
}

main().catch((error) => {
  console.error("[reset-setup] 失败:", error instanceof Error ? error.message : error);
  process.exit(1);
});
