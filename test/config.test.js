import test from "node:test";
import assert from "node:assert/strict";
import os from "node:os";
import path from "node:path";
import { mkdtemp, readFile } from "node:fs/promises";
import { loadConfig, resolveDefaultRuntimeWorkdir, writeConfig } from "../src/lib/config.js";

test("selects environment-aware default runtime workdir", () => {
  assert.equal(resolveDefaultRuntimeWorkdir("/etc/proxyrelay/proxyrelay.yaml"), "/var/lib/proxyrelay/runtime");
  assert.equal(resolveDefaultRuntimeWorkdir("/opt/proxyrelay/config/proxyrelay.yaml"), "../runtime");
});

test("can write normalized config back to yaml", async () => {
  const tempDir = await mkdtemp(path.join(os.tmpdir(), "proxyrelay-config-"));
  const filePath = path.join(tempDir, "proxyrelay.yaml");

  const config = await loadConfig(new URL("../config/proxyrelay.yaml", import.meta.url));
  config.server.port = 9988;
  config.admin.password = "updated-pass";

  const saved = await writeConfig(filePath, config);
  const raw = await readFile(filePath, "utf8");

  assert.equal(saved.server.port, 9988);
  assert.match(raw, /port: 9988/);
  assert.match(raw, /password: updated-pass/);
});
