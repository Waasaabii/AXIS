import YAML from "yaml";

function buildProvider(subscription, providersDir) {
  return {
    type: "http",
    url: subscription.url,
    path: `${providersDir}/${subscription.name}.yaml`,
    interval: Number(subscription.interval || 3600),
    "health-check": {
      enable: true,
      url: subscription.health_check_url || "https://www.gstatic.com/generate_204",
      interval: Number(subscription.health_check_interval || 300),
      timeout: 5000,
      lazy: true,
      "expected-status": 204
    }
  };
}

function buildGroup(group) {
  const mode = group.mode || "manual";
  const base = {
    name: group.name,
    type: mode === "manual" ? "select" : mode === "fallback" ? "fallback" : "url-test",
    use: [group.provider]
  };

  if (group.filter) {
    base.filter = group.filter;
  }

  if (group.exclude_filter) {
    base["exclude-filter"] = group.exclude_filter;
  }

  if (mode !== "manual") {
    base.url = group.health_check_url || "https://www.gstatic.com/generate_204";
    base.interval = Number(group.interval || 300);
    base.lazy = true;
    base.timeout = 5000;
  }

  return base;
}

function buildListener(listener) {
  return {
    name: listener.name,
    type: listener.type || "socks",
    listen: listener.listen || "0.0.0.0",
    port: Number(listener.port),
    udp: listener.udp !== false,
    users: Array.isArray(listener.users) ? listener.users : [],
    proxy: listener.egress_group
  };
}

export function renderMihomoConfig(config, { providersDir }) {
  const enabledProviderNames = new Set(
    config.subscriptions.filter((s) => s.enabled !== false).map((s) => s.name)
  );
  const validGroupNames = new Set(
    config.egress_groups.filter((g) => enabledProviderNames.has(g.provider)).map((g) => g.name)
  );
  const document = {
    mode: "rule",
    "log-level": "info",
    ipv6: false,
    "external-controller": config.runtime.external_controller.replace(/^https?:\/\//, ""),
    secret: config.runtime.external_secret || undefined,
    profile: {
      "store-selected": true,
      "store-fake-ip": false
    },
    "proxy-providers": Object.fromEntries(
      config.subscriptions
        .filter((subscription) => subscription.enabled !== false)
        .map((subscription) => [subscription.name, buildProvider(subscription, "providers")])
    ),
    "proxy-groups": config.egress_groups
      .filter((group) => enabledProviderNames.has(group.provider))
      .map(buildGroup),
    listeners: config.listeners
      .filter((listener) => listener.enabled !== false)
      .filter((listener) => validGroupNames.has(listener.egress_group))
      .map(buildListener)
  };

  return YAML.stringify(document);
}
