import crypto from "node:crypto";
import YAML from "yaml";

function createNodeId(source, canonical) {
  return crypto.createHash("sha1").update(`${source}:${canonical}`).digest("hex").slice(0, 12);
}

function tryDecodeBase64Text(input) {
  const normalized = input.replace(/\s+/g, "");
  if (!/^[A-Za-z0-9+/=_-]+$/.test(normalized) || normalized.length < 16) {
    return null;
  }

  try {
    const decoded = Buffer.from(normalized, "base64").toString("utf8");
    if (decoded.includes("://") || decoded.includes("proxies:")) {
      return decoded;
    }
  } catch {
    return null;
  }

  return null;
}

function normalizeName(name, fallback) {
  return decodeURIComponent((name || fallback || "未命名节点").trim());
}

function parseUrlProtocol(protocol, line) {
  const url = new URL(line);
  return {
    type: protocol,
    name: normalizeName(url.hash.slice(1), `${protocol}-${url.hostname}:${url.port}`),
    server: url.hostname,
    port: Number(url.port || 0),
    network: url.searchParams.get("type") || "",
    tls: url.searchParams.get("security") || "",
    source: line
  };
}

function parseVmess(line) {
  const raw = line.replace(/^vmess:\/\//, "");
  const decoded = Buffer.from(raw, "base64").toString("utf8");
  const payload = JSON.parse(decoded);

  return {
    type: "vmess",
    name: normalizeName(payload.ps, `vmess-${payload.add}:${payload.port}`),
    server: payload.add || "",
    port: Number(payload.port || 0),
    network: payload.net || "",
    tls: payload.tls || "",
    source: line
  };
}

function parseSs(line) {
  const [prefixPart, fragment = ""] = line.split("#");
  const encoded = prefixPart.replace(/^ss:\/\//, "");
  let decoded = encoded;

  if (!encoded.includes("@")) {
    decoded = Buffer.from(encoded, "base64").toString("utf8");
  }

  const [credentialPart, serverPart] = decoded.split("@");
  const [cipher] = credentialPart.split(":");
  const [server, port] = (serverPart || "").split(":");

  return {
    type: "shadowsocks",
    name: normalizeName(fragment, `ss-${server}:${port}`),
    server: server || "",
    port: Number(port || 0),
    network: "",
    tls: cipher || "",
    source: line
  };
}

function parseClashYaml(text) {
  try {
    const parsed = YAML.parse(text);
    if (!Array.isArray(parsed?.proxies)) {
      return [];
    }

    return parsed.proxies.map((proxy) => ({
      type: proxy.type || "unknown",
      name: normalizeName(proxy.name, `${proxy.server}:${proxy.port}`),
      server: proxy.server || "",
      port: Number(proxy.port || 0),
      network: proxy.network || proxy["ws-opts"]?.path || "",
      tls: proxy.tls ? "tls" : "",
      source: "clash-yaml"
    }));
  } catch {
    return [];
  }
}

export function parseSubscriptionPayload(rawBody, sourceName) {
  const trimmed = rawBody.trim();
  const clashNodes = parseClashYaml(trimmed);
  if (clashNodes.length > 0) {
    return clashNodes.map((node) => ({
      ...node,
      id: createNodeId(sourceName, `${node.type}:${node.name}:${node.server}:${node.port}`)
    }));
  }

  const decoded = tryDecodeBase64Text(trimmed);
  const text = decoded || trimmed;
  const lines = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#"));

  const nodes = [];
  for (const line of lines) {
    try {
      let parsed;
      if (line.startsWith("vmess://")) {
        parsed = parseVmess(line);
      } else if (line.startsWith("vless://")) {
        parsed = parseUrlProtocol("vless", line);
      } else if (line.startsWith("trojan://")) {
        parsed = parseUrlProtocol("trojan", line);
      } else if (line.startsWith("hysteria2://")) {
        parsed = parseUrlProtocol("hysteria2", line);
      } else if (line.startsWith("ss://")) {
        parsed = parseSs(line);
      } else {
        continue;
      }

      nodes.push({
        ...parsed,
        id: createNodeId(sourceName, line)
      });
    } catch {
      // 某一条节点解析失败时直接跳过，避免整份订阅不可用。
    }
  }

  return nodes;
}

function maskUrl(url) {
  try {
    const parsed = new URL(url);
    return `${parsed.protocol}//${parsed.host}${parsed.pathname}`;
  } catch {
    return url;
  }
}

function userAgentForType(type) {
  if (type === "clash-http") {
    return "clash-verge/v2.2.3";
  }
  return "clash-meta/v1.19.0";
}

export async function fetchProviderSnapshot(subscription) {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 15000);

  try {
    const response = await fetch(subscription.url, {
      signal: controller.signal,
      headers: {
        "User-Agent": userAgentForType(subscription.type)
      }
    });
    const body = await response.text();
    const nodes = parseSubscriptionPayload(body, subscription.name);

    return {
      ok: response.ok,
      status: response.status,
      statusText: response.statusText,
      provider: subscription.name,
      urlMasked: maskUrl(subscription.url),
      refreshedAt: new Date().toISOString(),
      nodeCount: nodes.length,
      nodes: nodes.map((node) => ({
        id: node.id,
        name: node.name,
        type: node.type,
        server: node.server,
        port: node.port,
        network: node.network,
        tls: node.tls
      }))
    };
  } finally {
    clearTimeout(timeout);
  }
}
