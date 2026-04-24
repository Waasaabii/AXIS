export const apiKeys = {
  bootstrapStatus: "/api/bootstrap/status",
  session: "/api/session",
  sessionPassword: "/api/session/password",
  setupState: "/api/setup-state",
  setupAdmin: "/api/setup/admin",
  status: "/api/status",
  config: "/api/config",
  nodeSources: "/api/node-sources",
  routes: "/api/routes",
  usage: "/api/usage",
  publications: "/api/publications",
  providers: "/api/providers",
  groups: "/api/groups",
  landingProxies: "/api/landing-proxies",
  transitRoutes: "/api/transit-routes",
  listeners: "/api/listeners",
  events: "/api/events",
  renderedConfig: "/api/rendered-config",
  controller: "/api/controller",
  runtimePreflight: "/api/runtime-preflight",
  controllerProbe: "/api/controller/probe",
  reloadRuntime: "/api/reload",
  subscriptions: "/api/subscriptions",
  egressGroups: "/api/egress-groups",
  mihomoVersions: "/api/mihomo/versions",
  hostStatus: "/api/host/status",
  updaterStatus: "/api/host/updater",
  updaterCheck: "/api/host/updater/check",
  hostAutostart: "/api/host/autostart",
  hostOpenBrowser: "/api/host/open-browser",
  openapiJson: "/api/openapi.json",
} as const

export type ApiKey = typeof apiKeys[keyof typeof apiKeys]

export const apiPath = {
  nodeSource: (name: string) => `${apiKeys.nodeSources}/${encodeURIComponent(name)}`,
  nodeSourceUpdate: (name: string) => `${apiKeys.nodeSources}/${encodeURIComponent(name)}/update`,
  nodeSourceCheck: (name: string) => `${apiKeys.nodeSources}/${encodeURIComponent(name)}/check`,
  nodeSourceConnection: (name: string) => `${apiKeys.nodeSources}/${encodeURIComponent(name)}/connection`,
  nodeSourceBaotaConfig: (name: string) => `${apiKeys.nodeSources}/${encodeURIComponent(name)}/baota-config`,
  route: (name: string) => `${apiKeys.routes}/${encodeURIComponent(name)}`,
  routeUpdate: (name: string) => `${apiKeys.routes}/${encodeURIComponent(name)}/update`,
  publication: (name: string) => `${apiKeys.publications}/${encodeURIComponent(name)}`,
  publicationUpdate: (name: string) => `${apiKeys.publications}/${encodeURIComponent(name)}/update`,

  providerRefresh: (name: string) => `${apiKeys.providers}/${encodeURIComponent(name)}/refresh`,

  groupSelect: (groupName: string) => `${apiKeys.groups}/${encodeURIComponent(groupName)}/select`,
  groupHealthcheck: (groupName: string) => `${apiKeys.groups}/${encodeURIComponent(groupName)}/healthcheck`,

  transitRoute: (name: string) => `${apiKeys.transitRoutes}/${encodeURIComponent(name)}`,
  transitRouteUpdate: (name: string) => `${apiKeys.transitRoutes}/${encodeURIComponent(name)}/update`,
  transitRouteHealthcheck: (name: string) => `${apiKeys.transitRoutes}/${encodeURIComponent(name)}/healthcheck`,

  landingProxy: (name: string) => `${apiKeys.landingProxies}/${encodeURIComponent(name)}`,
  landingProxyUpdate: (name: string) => `${apiKeys.landingProxies}/${encodeURIComponent(name)}/update`,

  listener: (name: string) => `${apiKeys.listeners}/${encodeURIComponent(name)}`,
  listenerUpdate: (name: string) => `${apiKeys.listeners}/${encodeURIComponent(name)}/update`,

  subscription: (name: string) => `${apiKeys.subscriptions}/${encodeURIComponent(name)}`,
  subscriptionUpdate: (name: string) => `${apiKeys.subscriptions}/${encodeURIComponent(name)}/update`,
  subscriptionToggle: (name: string) => `${apiKeys.subscriptions}/${encodeURIComponent(name)}/toggle`,

  egressGroup: (name: string) => `${apiKeys.egressGroups}/${encodeURIComponent(name)}`,
  egressGroupUpdate: (name: string) => `${apiKeys.egressGroups}/${encodeURIComponent(name)}/update`,

  mihomoVersionDownload: (version: string) => `${apiKeys.mihomoVersions}/${encodeURIComponent(version)}/download`,
  mihomoVersionInstall: (version: string) => `${apiKeys.mihomoVersions}/${encodeURIComponent(version)}/install`,
  mihomoVersionActivate: (version: string) => `${apiKeys.mihomoVersions}/${encodeURIComponent(version)}/activate`,
} as const
