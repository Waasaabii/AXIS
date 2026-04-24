import { mkdir, rm, cp, writeFile } from "node:fs/promises";
import { spawn } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { getGoCommand, resolveGoEnv } from "./go-env.mjs";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const args = process.argv.slice(2);

function argValue(name, fallback = "") {
  const index = args.indexOf(name);
  if (index >= 0 && args[index + 1]) return args[index + 1];
  const inline = args.find((item) => item.startsWith(`${name}=`));
  if (inline) return inline.slice(name.length + 1);
  return fallback;
}

function run(command, commandArgs, options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, commandArgs, {
      cwd: options.cwd || repoRoot,
      env: { ...process.env, ...(options.env || {}) },
      stdio: "inherit",
    });
    child.on("error", reject);
    child.on("exit", (code) => {
      if (code === 0) resolve();
      else reject(new Error(`${command} ${commandArgs.join(" ")} 失败，退出码 ${code}`));
    });
  });
}

function getPnpmCommand() {
  return process.platform === "win32" ? "pnpm.cmd" : "pnpm";
}

const version = argValue("--version", process.env.GITHUB_REF_NAME || "dev").replace(/^v/, "");
const goos = argValue("--goos", process.env.GOOS || "linux");
const goarch = argValue("--goarch", process.env.GOARCH || "amd64");
const distRoot = path.join(repoRoot, "dist", "release");
const packageName = `axis_${version}_${goos}_${goarch}`;
const packageDir = path.join(distRoot, packageName);
const binaryName = goos === "windows" ? "axis.exe" : "axis";

await rm(packageDir, { recursive: true, force: true });
await mkdir(path.join(packageDir, "bin"), { recursive: true });
await mkdir(path.join(packageDir, "config"), { recursive: true });
await mkdir(path.join(packageDir, "runtime"), { recursive: true });

await run(getPnpmCommand(), ["run", "build:ui"]);
await run(getGoCommand(), ["build", "-tags", "embedui", "-trimpath", "-o", path.join(packageDir, "bin", binaryName), "./cmd/axis"], {
  env: resolveGoEnv({ GOOS: goos, GOARCH: goarch, CGO_ENABLED: "0" }),
});
await cp(path.join(repoRoot, "config", "proxyrelay.example.yaml"), path.join(packageDir, "config", "proxyrelay.yaml"));
await writeFile(path.join(packageDir, "README.md"), `# AXIS ${version}\n\n## 启动\n\n\`\`\`bash\nPROXYRELAY_CONFIG=./config/proxyrelay.yaml ./bin/${binaryName} serve\n\`\`\`\n\n默认配置监听 \`127.0.0.1:8787\`。如果放到宝塔后面，请在配置里改成需要的本机端口，再由宝塔反代到 AXIS。\n`, "utf8");

const archiveName = `${packageName}.tar.gz`;
await rm(path.join(distRoot, archiveName), { force: true });
await run("tar", ["-czf", archiveName, packageName], { cwd: distRoot });
console.log(`Release 包已生成: ${path.relative(repoRoot, path.join(distRoot, archiveName))}`);
