import test from "node:test";
import assert from "node:assert/strict";
import os from "node:os";
import path from "node:path";
import { mkdtemp, mkdir, writeFile } from "node:fs/promises";
import { buildRuntimePreflightSnapshot } from "../src/services/runtime-preflight.js";

async function createRuntimeFixture() {
  const tempDir = await mkdtemp(path.join(os.tmpdir(), "proxyrelay-preflight-"));
  const runtimeDir = path.join(tempDir, "runtime");
  const providersDir = path.join(runtimeDir, "providers");
  const serviceDir = path.join(tempDir, "systemd");

  await mkdir(providersDir, { recursive: true });
  await mkdir(serviceDir, { recursive: true });
  await writeFile(path.join(runtimeDir, "mihomo.yaml"), "mode: rule\n", "utf8");
  await writeFile(path.join(runtimeDir, "mihomo.last-good.yaml"), "mode: rule\n", "utf8");
  await writeFile(path.join(runtimeDir, "control-state.json"), "{}\n", "utf8");
  await writeFile(path.join(serviceDir, "proxyrelayd.service"), "[Unit]\nDescription=ProxyRelay\n", "utf8");
  await writeFile(path.join(serviceDir, "mihomo.service"), "[Unit]\nDescription=Mihomo\n", "utf8");

  return {
    tempDir,
    runtimeDir,
    providersDir,
    serviceDir,
    layout: {
      runtimeDir,
      providersDir,
      mihomoConfigPath: path.join(runtimeDir, "mihomo.yaml"),
      lastGoodConfigPath: path.join(runtimeDir, "mihomo.last-good.yaml"),
      statePath: path.join(runtimeDir, "control-state.json")
    }
  };
}

test("runtime preflight reports ready snapshot for healthy managed runtime", async () => {
  const fixture = await createRuntimeFixture();
  const snapshot = await buildRuntimePreflightSnapshot({
    configPath: path.join(fixture.tempDir, "config", "proxyrelay.yaml"),
    config: {
      runtime: {
        render_only: false,
        external_controller: "http://127.0.0.1:11235",
        external_secret: "controller-secret"
      }
    },
    layout: fixture.layout,
    state: {
      runtime: {
        mihomoBinaryFound: true,
        mihomoBinary: "/usr/local/bin/mihomo",
        lastApplyStatus: "success",
        lastApplyMessage: "运行态已加载配置"
      }
    },
    controller: {
      reachable: true,
      message: "controller 可达"
    },
    serviceDir: fixture.serviceDir,
    expectedSystemRuntimeDir: fixture.runtimeDir
  });

  assert.equal(snapshot.ready, true);
  assert.equal(snapshot.status, "ok");
  assert.equal(snapshot.summary.errors, 0);
  assert.match(snapshot.checks.find((item) => item.key === "mihomo-binary").summary, /已检测到 Mihomo/);
});

test("runtime preflight highlights Ubuntu runtime misalignment", async () => {
  const fixture = await createRuntimeFixture();
  const snapshot = await buildRuntimePreflightSnapshot({
    configPath: "/etc/proxyrelay/proxyrelay.yaml",
    config: {
      runtime: {
        render_only: false,
        external_controller: "http://127.0.0.1:11235",
        external_secret: ""
      }
    },
    layout: fixture.layout,
    state: {
      runtime: {
        mihomoBinaryFound: false,
        mihomoBinary: "mihomo",
        lastApplyStatus: "failed",
        lastApplyMessage: "controller 返回 404"
      }
    },
    controller: {
      reachable: false,
      message: "connect ECONNREFUSED 127.0.0.1:11235"
    },
    serviceDir: path.join(fixture.tempDir, "missing-systemd")
  });

  assert.equal(snapshot.ready, false);
  assert.equal(snapshot.status, "fail");
  assert.ok(snapshot.recommendations.some((item) => item.includes("/var/lib/proxyrelay/runtime")));
  assert.ok(snapshot.recommendations.some((item) => item.includes("Mihomo")));
  assert.equal(snapshot.checks.find((item) => item.key === "runtime-path").level, "error");
});
