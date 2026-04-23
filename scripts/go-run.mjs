import { spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { getGoCommand, resolveGoEnv } from "./go-env.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const args = process.argv.slice(2);

if (args.length === 0) {
  console.error("[go-run] 缺少 Go 子命令参数");
  process.exit(1);
}

const child = spawn(getGoCommand(), args, {
  cwd: repoRoot,
  env: resolveGoEnv(),
  stdio: "inherit",
});

child.on("exit", (code) => {
  process.exit(code ?? 0);
});

child.on("error", (error) => {
  console.error("[go-run] 启动失败:", error.message);
  process.exit(1);
});
