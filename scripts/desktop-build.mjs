import { spawn } from "node:child_process";
import { access, mkdir, readdir, rm } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const desktopDir = path.join(repoRoot, "desktop");
const desktopBuildBinDir = path.join(desktopDir, "build", "bin");
const desktopTmpDir = path.join(repoRoot, ".tmp", "desktop");

function getGoCommand() {
  return process.platform === "win32" ? "go.exe" : "go";
}

function getPnpmCommand() {
  return process.platform === "win32" ? "pnpm.cmd" : "pnpm";
}

async function fileExists(targetPath) {
  try {
    await access(targetPath, fsConstants.F_OK);
    return true;
  } catch {
    return false;
  }
}

function run(command, args, cwd, extraEnv = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd,
      env: {
        ...process.env,
        ...extraEnv,
      },
      stdio: "inherit",
    });

    child.on("error", reject);
    child.on("exit", (code) => {
      if (code === 0) {
        resolve();
        return;
      }
      reject(new Error(`${command} ${args.join(" ")} 失败，退出码 ${code}`));
    });
  });
}

async function resolveBuildConfigPath() {
  const candidates = [
    path.join(repoRoot, "config", "proxyrelay.yaml"),
    path.join(repoRoot, "config", "proxyrelay.example.yaml"),
  ];

  for (const candidate of candidates) {
    if (await fileExists(candidate)) {
      return candidate;
    }
  }

  throw new Error("未找到桌面构建所需配置：config/proxyrelay.yaml 或 config/proxyrelay.example.yaml");
}

async function findMacAppBundle() {
  const entries = await readdir(desktopBuildBinDir, { withFileTypes: true }).catch(() => []);
  const appEntry = entries.find((entry) => entry.isDirectory() && entry.name.endsWith(".app"));
  return appEntry ? path.join(desktopBuildBinDir, appEntry.name) : "";
}

async function postProcessMacBundle() {
  if (process.platform !== "darwin") {
    return;
  }

  const appBundle = await findMacAppBundle();
  if (!appBundle) {
    console.warn("[desktop-build] 未找到 macOS .app 产物，跳过后处理");
    return;
  }

  await run("xattr", ["-cr", appBundle], repoRoot);

  const identity = (process.env.AXIS_MAC_SIGN_IDENTITY || "-").trim() || "-";
  const notaryProfile = (process.env.AXIS_MAC_NOTARY_PROFILE || "").trim();
  const codesignArgs = ["--force", "--deep", "--sign", identity];

  if (identity !== "-") {
    codesignArgs.push("--options", "runtime", "--timestamp");
  }

  codesignArgs.push(appBundle);

  await run("codesign", codesignArgs, repoRoot);
  await run("codesign", ["--verify", "--deep", "--strict", "--verbose=2", appBundle], repoRoot);

  if (identity === "-") {
    console.warn("[desktop-build] 当前只做了 ad-hoc 签名，适合本机自用调试；要分发给其他机器，请设置 AXIS_MAC_SIGN_IDENTITY 并配合公证。");
    return;
  }

  if (!notaryProfile) {
    console.warn("[desktop-build] 已完成 Developer ID 签名，但未配置 AXIS_MAC_NOTARY_PROFILE；互联网分发前建议继续做 notarization。");
    return;
  }

  await mkdir(desktopTmpDir, { recursive: true });
  const zipPath = path.join(desktopTmpDir, `${path.basename(appBundle, ".app")}.zip`);
  await rm(zipPath, { force: true });
  await run("ditto", ["-c", "-k", "--sequesterRsrc", "--keepParent", appBundle, zipPath], repoRoot);
  await run("xcrun", ["notarytool", "submit", zipPath, "--wait", "--keychain-profile", notaryProfile], repoRoot);
  await run("xcrun", ["stapler", "staple", appBundle], repoRoot);
  await run("spctl", ["--assess", "--type", "execute", "--verbose=4", appBundle], repoRoot);
}

async function main() {
  const args = ["run", "github.com/wailsapp/wails/v2/cmd/wails@v2.11.0", "build", "-tags", "embedui"];
  const customPlatform = process.env.AXIS_WAILS_PLATFORM || "";
  const debugBuild = process.env.AXIS_WAILS_DEBUG === "1";
  const dryRun = process.env.AXIS_WAILS_DRYRUN === "1";
  const buildConfigPath = await resolveBuildConfigPath();

  if (customPlatform) {
    args.push("-platform", customPlatform);
  }
  if (debugBuild) {
    args.push("-debug");
  }
  if (dryRun) {
    args.push("-dryrun");
  }

  await run(getPnpmCommand(), ["run", "openapi:generate"], repoRoot);
  await run(getGoCommand(), args, desktopDir, {
    PROXYRELAY_CONFIG: buildConfigPath,
  });
  if (!dryRun) {
    await postProcessMacBundle();
  }
}

main().catch((error) => {
  console.error("[desktop-build] 失败:", error instanceof Error ? error.message : error);
  process.exit(1);
});
