import test from "node:test";
import assert from "node:assert/strict";
import { parseSubscriptionPayload } from "../src/services/subscription-service.js";

test("can parse vmess and vless subscription payloads", () => {
  const vmessJson = Buffer.from(
    JSON.stringify({
      ps: "HK-01",
      add: "hk.example.com",
      port: "443",
      net: "ws",
      tls: "tls"
    }),
    "utf8"
  ).toString("base64");

  const raw = `${Buffer.from(
    [`vmess://${vmessJson}`, "vless://uuid@example.com:443?type=ws&security=tls#JP-01"].join("\n"),
    "utf8"
  ).toString("base64")}\n`;

  const nodes = parseSubscriptionPayload(raw, "airport-main");

  assert.equal(nodes.length, 2);
  assert.equal(nodes[0].name, "HK-01");
  assert.equal(nodes[1].type, "vless");
});

test("can parse clash yaml payload", () => {
  const raw = `
proxies:
  - name: US-01
    type: ss
    server: us.example.com
    port: 8388
`;

  const nodes = parseSubscriptionPayload(raw, "airport-main");
  assert.equal(nodes.length, 1);
  assert.equal(nodes[0].name, "US-01");
});
