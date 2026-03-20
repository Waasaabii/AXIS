import test from "node:test";
import assert from "node:assert/strict";
import { createMihomoControllerAdapter } from "../src/services/mihomo-controller.js";

test("controller adapter reports render-only mode clearly", async () => {
  const adapter = createMihomoControllerAdapter({
    external_controller: "http://127.0.0.1:9090",
    external_secret: "",
    render_only: true
  });

  const result = await adapter.probe();

  assert.equal(result.ok, false);
  assert.equal(result.reachable, false);
  assert.equal(result.mode, "render-only");
});

test("controller adapter can reload runtime config in managed mode", async () => {
  const originalFetch = global.fetch;
  const requests = [];

  global.fetch = async (url, options = {}) => {
    requests.push({
      url,
      method: options.method,
      headers: options.headers,
      body: options.body
    });

    return {
      ok: true,
      status: 200,
      async text() {
        return '{"message":"reloaded"}';
      }
    };
  };

  try {
    const adapter = createMihomoControllerAdapter({
      external_controller: "http://127.0.0.1:11235",
      external_secret: "controller-secret",
      render_only: false
    });

    const result = await adapter.reloadConfig("/var/lib/proxyrelay/runtime/mihomo.yaml");

    assert.equal(result.ok, true);
    assert.equal(requests.length, 1);
    assert.equal(requests[0].url, "http://127.0.0.1:11235/configs?force=true");
    assert.equal(requests[0].method, "PUT");
    assert.match(requests[0].headers.Authorization, /^Bearer controller-secret$/);
    assert.match(requests[0].body, /\/var\/lib\/proxyrelay\/runtime\/mihomo\.yaml/);
  } finally {
    global.fetch = originalFetch;
  }
});
