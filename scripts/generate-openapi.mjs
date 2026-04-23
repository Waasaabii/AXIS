import { mkdir, writeFile } from "node:fs/promises";
import { spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { getGoCommand, resolveGoEnv } from "./go-env.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const generatedDir = path.join(repoRoot, "frontend", "src", "generated");
const schemaJsonPath = path.join(generatedDir, "openapi.json");
const schemaTypesPath = path.join(generatedDir, "openapi.ts");

function getPnpmCommand() {
  return process.platform === "win32" ? "pnpm.cmd" : "pnpm";
}

async function capture(command, args) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: repoRoot,
      env: command === getGoCommand() ? resolveGoEnv() : process.env,
      stdio: ["ignore", "pipe", "pipe"],
    });

    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => {
      stdout += chunk.toString();
    });
    child.stderr.on("data", (chunk) => {
      stderr += chunk.toString();
    });
    child.on("error", reject);
    child.on("exit", (code) => {
      if (code === 0) {
        resolve(stdout);
        return;
      }
      reject(new Error(stderr || `${command} ${args.join(" ")} 失败，退出码 ${code}`));
    });
  });
}

async function main() {
  await mkdir(generatedDir, { recursive: true });

  const schemaJson = await capture(getGoCommand(), ["run", "./cmd/axis", "openapi"]);
  await writeFile(schemaJsonPath, schemaJson, "utf8");

  await capture(getPnpmCommand(), [
    "--filter",
    "@axis/frontend",
    "exec",
    "openapi-typescript",
    "./src/generated/openapi.json",
    "-o",
    "./src/generated/openapi.ts",
  ]);

  console.log(`OpenAPI 已生成: ${path.relative(repoRoot, schemaTypesPath)}`);
}

main().catch((error) => {
  console.error("[openapi] 生成失败:", error instanceof Error ? error.message : error);
  process.exit(1);
});
