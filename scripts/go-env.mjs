import { mkdirSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const tmpRoot = path.join(repoRoot, ".tmp");

function getGoCommand() {
  return process.platform === "win32" ? "go.exe" : "go";
}

function resolveGoProxy() {
  const candidates = [process.env.GOPROXY, process.env.AXIS_GOPROXY];
  for (const candidate of candidates) {
    const normalized = normalizeGoList(candidate);
    if (normalized) {
      return normalized;
    }
  }

  // 默认优先国内可用代理，并在网络错误时回退（使用 `|` 分隔）。
  // 需要自定义时：设置 `AXIS_GOPROXY` 或直接设置 `GOPROXY`。
  return "https://goproxy.cn|https://goproxy.io|direct";
}

function resolveGoSumDB() {
  const candidates = [process.env.GOSUMDB, process.env.AXIS_GOSUMDB];
  for (const candidate of candidates) {
    const value = (candidate || "").trim();
    if (value) {
      return value;
    }
  }

  // 在国内环境下 `sum.golang.org` 经常不可达；这个镜像通常更稳。
  return "sum.golang.google.cn";
}

function normalizeGoList(value) {
  const raw = (value || "").trim();
  if (!raw) {
    return "";
  }

  // 保留 `,`/`|` 的语义（`,` 仅在 404/410 回退，`|` 在任意错误回退），只做轻量清洗。
  return raw
    .split(",")
    .map((item) =>
      item
        .split("|")
        .map((part) => part.trim())
        .filter(Boolean)
        .join("|"),
    )
    .map((item) => item.trim())
    .filter(Boolean)
    .join(",");
}

function resolveGoEnv(extraEnv = {}) {
  const env = {
    ...process.env,
    ...extraEnv,
  };

  if (!env.GOPROXY || !env.GOPROXY.trim()) {
    env.GOPROXY = resolveGoProxy();
  }

  if (!env.GOSUMDB || !env.GOSUMDB.trim()) {
    env.GOSUMDB = resolveGoSumDB();
  }

  // 让 `pnpm` 脚本跑 Go 时把缓存固定到仓库内，避免污染全局环境。
  // `.tmp/` 会被 gitignore，清理也更可控。
  env.GOCACHE = env.GOCACHE || path.join(tmpRoot, "go-build");
  env.GOMODCACHE = env.GOMODCACHE || path.join(tmpRoot, "go-mod");
  env.GOTMPDIR = env.GOTMPDIR || path.join(tmpRoot, "go-tmp");

  // Go 会要求 `GOTMPDIR` 存在，否则直接报错。
  mkdirSync(tmpRoot, { recursive: true });
  mkdirSync(env.GOCACHE, { recursive: true });
  mkdirSync(env.GOMODCACHE, { recursive: true });
  mkdirSync(env.GOTMPDIR, { recursive: true });

  return env;
}

export {
  getGoCommand,
  resolveGoEnv,
  resolveGoProxy,
};
