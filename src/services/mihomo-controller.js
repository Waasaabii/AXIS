function trimSlash(input) {
  return input.replace(/\/+$/, "");
}

export function createMihomoControllerAdapter(runtimeConfig) {
  return new MihomoControllerAdapter(runtimeConfig);
}

class MihomoControllerAdapter {
  constructor(runtimeConfig) {
    this.baseUrl = trimSlash(runtimeConfig.external_controller || "http://127.0.0.1:9090");
    this.secret = runtimeConfig.external_secret || "";
    this.renderOnly = runtimeConfig.render_only ?? true;
  }

  headers() {
    return {
      Accept: "application/json",
      ...(this.secret ? { Authorization: `Bearer ${this.secret}` } : {})
    };
  }

  async request(pathname, options = {}) {
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), options.timeoutMs || 3000);

    try {
      const response = await fetch(`${this.baseUrl}${pathname}`, {
        ...options,
        headers: {
          ...this.headers(),
          ...(options.body ? { "Content-Type": "application/json" } : {}),
          ...(options.headers || {})
        },
        signal: controller.signal
      });

      const rawText = await response.text();
      let payload = rawText;
      if (rawText) {
        try {
          payload = JSON.parse(rawText);
        } catch {
          payload = rawText;
        }
      }

      return {
        ok: response.ok,
        status: response.status,
        payload
      };
    } finally {
      clearTimeout(timeout);
    }
  }

  async probe() {
    if (this.renderOnly) {
      return {
        ok: false,
        reachable: false,
        mode: "render-only",
        message: "当前为 render-only 模式，未接入运行态 controller。"
      };
    }

    try {
      const versionResult = await this.request("/version");
      if (!versionResult.ok) {
        return {
          ok: false,
          reachable: false,
          mode: "managed",
          message: `controller 返回 ${versionResult.status}`
        };
      }

      const version =
        typeof versionResult.payload === "string"
          ? versionResult.payload
          : versionResult.payload?.version || versionResult.payload?.meta || "unknown";

      return {
        ok: true,
        reachable: true,
        mode: "managed",
        version,
        message: "controller 可达"
      };
    } catch (error) {
      return {
        ok: false,
        reachable: false,
        mode: "managed",
        message: error instanceof Error ? error.message : "controller 不可达"
      };
    }
  }

  async refreshProvider(providerName) {
    if (this.renderOnly) {
      return { ok: false, deferred: true, message: "render-only 模式下跳过 provider 运行态刷新" };
    }

    return this.request(`/providers/proxies/${encodeURIComponent(providerName)}`, {
      method: "PUT"
    });
  }

  async selectProxy(groupName, proxyName) {
    if (this.renderOnly) {
      return { ok: false, deferred: true, message: "render-only 模式下仅记录期望选择" };
    }

    return this.request(`/proxies/${encodeURIComponent(groupName)}`, {
      method: "PUT",
      body: JSON.stringify({ name: proxyName })
    });
  }

  async reloadConfig(configPath) {
    if (this.renderOnly) {
      return { ok: false, deferred: true, message: "render-only 模式下仅生成配置文件" };
    }

    return this.request("/configs?force=true", {
      method: "PUT",
      body: JSON.stringify(
        configPath
          ? {
              path: configPath
            }
          : {}
      ),
      timeoutMs: 5000
    });
  }
}
