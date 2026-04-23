import { access, readFile } from "node:fs/promises";
import { constants as fsConstants } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import YAML from "yaml";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, "..");
const repoConfigTemplatePath = path.join(repoRoot, "config", "proxyrelay.example.yaml");

async function fileExists(targetPath) {
  try {
    await access(targetPath, fsConstants.F_OK);
    return true;
  } catch {
    return false;
  }
}

export function resolveRepoConfigTemplatePath() {
  return repoConfigTemplatePath;
}

export async function readRepoConfigTemplate() {
  if (!(await fileExists(repoConfigTemplatePath))) {
    throw new Error("未找到仓库配置模板：config/proxyrelay.example.yaml");
  }
  return YAML.parse(await readFile(repoConfigTemplatePath, "utf8")) || {};
}
