import test from "node:test";
import assert from "node:assert/strict";
import { loadConfig } from "../src/lib/config.js";
import { renderMihomoConfig } from "../src/services/mihomo-renderer.js";

test("renders listeners and proxy groups into mihomo yaml", async () => {
  const config = await loadConfig(new URL("../config/proxyrelay.yaml", import.meta.url));
  const output = renderMihomoConfig(config, { providersDir: "providers" });

  assert.match(output, /proxy-providers:/);
  assert.match(output, /listeners:/);
  assert.match(output, /egress-hk-manual/);
  assert.match(output, /hk-socks/);
});
