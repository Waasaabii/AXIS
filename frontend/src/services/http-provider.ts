import type {
  AddEgressGroupRequest,
  AddLandingProxyRequest,
  AddListenerRequest,
  AddSubscriptionRequest,
  AddTransitRouteRequest,
  AxisAPI,
  BootstrapStatus,
  BootstrapAdminRequest,
  ConfigEnvelope,
  ControllerStatusResponse,
  EventEntry,
  GroupMutationResponse,
  GroupView,
  HostStatus,
  LandingProxyView,
  ListenerView,
  LoginRequest,
  LoginResponse,
  MihomoVersionActionResponse,
  MihomoVersionsResponse,
  ProbeControllerResponse,
  ProviderListItem,
  ProviderRefreshResponse,
  ReloadResponse,
  RenderedConfigResponse,
  RuntimePreflightSnapshot,
  SaveConfigRequest,
  SaveConfigResponse,
  SelectGroupRequest,
  SessionStatusResponse,
  SetupStateResponse,
  SimpleOkResponse,
  StatusResponse,
  ToggleSubscriptionRequest,
  TransitRouteMutationResponse,
  TransitRouteView,
  UpdaterStatus,
  UpdateLandingProxyRequest,
  UpdateEgressGroupRequest,
  UpdateListenerRequest,
  UpdatePasswordRequest,
  UpdateSubscriptionRequest,
  UpdateTransitRouteRequest,
} from "./provider-types"
import { ApiError } from "./provider-types"

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null
}

function readErrorMessage(payload: unknown, fallback: string) {
  if (isRecord(payload) && typeof payload.error === "string" && payload.error.trim()) {
    return payload.error
  }
  if (isRecord(payload) && typeof payload.message === "string" && payload.message.trim()) {
    return payload.message
  }
  return fallback
}

async function requestJson<T>(url: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json")
  }

  const response = await fetch(url, {
    ...init,
    headers,
  })

  const payload = await response.json().catch(() => undefined)
  if (!response.ok) {
    throw new ApiError(readErrorMessage(payload, `请求失败 (${response.status})`), response.status, payload)
  }

  return payload as T
}

export const httpAPI: AxisAPI = {
  getBootstrapStatus: () => requestJson<BootstrapStatus>("/api/bootstrap/status"),
  getSession: () => requestJson<SessionStatusResponse>("/api/session"),
  login: (body: LoginRequest) => requestJson<LoginResponse>("/api/session", { method: "POST", body: JSON.stringify(body) }),
  logout: () => requestJson<SimpleOkResponse>("/api/session", { method: "DELETE" }),
  updatePassword: (body: UpdatePasswordRequest) => requestJson<SimpleOkResponse>("/api/session/password", { method: "PUT", body: JSON.stringify(body) }),
  bootstrapAdmin: (body: BootstrapAdminRequest) => requestJson<SimpleOkResponse>("/api/setup/admin", { method: "PUT", body: JSON.stringify(body) }),
  getStatus: () => requestJson<StatusResponse>("/api/status"),
  getConfig: () => requestJson<ConfigEnvelope>("/api/config"),
  saveConfig: (body: SaveConfigRequest) => requestJson<SaveConfigResponse>("/api/config", { method: "PUT", body: JSON.stringify(body) }),
  getProviders: () => requestJson<ProviderListItem[]>("/api/providers"),
  refreshProvider: (name: string) => requestJson<ProviderRefreshResponse>(`/api/providers/${encodeURIComponent(name)}/refresh`, { method: "POST" }),
  getGroups: () => requestJson<GroupView[]>("/api/groups"),
  getTransitRoutes: () => requestJson<TransitRouteView[]>("/api/transit-routes"),
  getLandingProxies: () => requestJson<LandingProxyView[]>("/api/landing-proxies"),
  selectGroup: (groupName: string, body: SelectGroupRequest) => requestJson<GroupMutationResponse>(`/api/groups/${encodeURIComponent(groupName)}/select`, { method: "POST", body: JSON.stringify(body) }),
  healthcheckGroup: (groupName: string) => requestJson<GroupMutationResponse>(`/api/groups/${encodeURIComponent(groupName)}/healthcheck`, { method: "POST" }),
  addTransitRoute: (body: AddTransitRouteRequest) => requestJson<SaveConfigResponse>("/api/transit-routes", { method: "POST", body: JSON.stringify(body) }),
  updateTransitRoute: (name: string, body: UpdateTransitRouteRequest) => requestJson<SaveConfigResponse>(`/api/transit-routes/${encodeURIComponent(name)}/update`, { method: "PUT", body: JSON.stringify(body) }),
  deleteTransitRoute: (name: string) => requestJson<SaveConfigResponse>(`/api/transit-routes/${encodeURIComponent(name)}`, { method: "DELETE" }),
  healthcheckTransitRoute: (name: string) => requestJson<TransitRouteMutationResponse>(`/api/transit-routes/${encodeURIComponent(name)}/healthcheck`, { method: "POST" }),
  addLandingProxy: (body: AddLandingProxyRequest) => requestJson<SaveConfigResponse>("/api/landing-proxies", { method: "POST", body: JSON.stringify(body) }),
  updateLandingProxy: (name: string, body: UpdateLandingProxyRequest) => requestJson<SaveConfigResponse>(`/api/landing-proxies/${encodeURIComponent(name)}/update`, { method: "PUT", body: JSON.stringify(body) }),
  deleteLandingProxy: (name: string) => requestJson<SaveConfigResponse>(`/api/landing-proxies/${encodeURIComponent(name)}`, { method: "DELETE" }),
  getListeners: () => requestJson<ListenerView[]>("/api/listeners"),
  addListener: (body: AddListenerRequest) => requestJson<SaveConfigResponse>("/api/listeners", { method: "POST", body: JSON.stringify(body) }),
  updateListener: (name: string, body: UpdateListenerRequest) => requestJson<SaveConfigResponse>(`/api/listeners/${encodeURIComponent(name)}/update`, { method: "PUT", body: JSON.stringify(body) }),
  deleteListener: (name: string) => requestJson<SaveConfigResponse>(`/api/listeners/${encodeURIComponent(name)}`, { method: "DELETE" }),
  getEvents: () => requestJson<EventEntry[]>("/api/events"),
  getRenderedConfig: () => requestJson<RenderedConfigResponse>("/api/rendered-config"),
  getController: () => requestJson<ControllerStatusResponse>("/api/controller"),
  getSetupState: () => requestJson<SetupStateResponse>("/api/setup-state"),
  getRuntimePreflight: () => requestJson<RuntimePreflightSnapshot>("/api/runtime-preflight"),
  probeController: () => requestJson<ProbeControllerResponse>("/api/controller/probe", { method: "POST" }),
  reloadRuntime: () => requestJson<ReloadResponse>("/api/reload", { method: "POST" }),
  addSubscription: (body: AddSubscriptionRequest) => requestJson<SaveConfigResponse>("/api/subscriptions", { method: "POST", body: JSON.stringify(body) }),
  updateSubscription: (name: string, body: UpdateSubscriptionRequest) => requestJson<SaveConfigResponse>(`/api/subscriptions/${encodeURIComponent(name)}/update`, { method: "PUT", body: JSON.stringify(body) }),
  toggleSubscription: (name: string, body: ToggleSubscriptionRequest) => requestJson<SaveConfigResponse>(`/api/subscriptions/${encodeURIComponent(name)}/toggle`, { method: "POST", body: JSON.stringify(body) }),
  deleteSubscription: (name: string) => requestJson<SaveConfigResponse>(`/api/subscriptions/${encodeURIComponent(name)}`, { method: "DELETE" }),
  addEgressGroup: (body: AddEgressGroupRequest) => requestJson<SaveConfigResponse>("/api/egress-groups", { method: "POST", body: JSON.stringify(body) }),
  updateEgressGroup: (name: string, body: UpdateEgressGroupRequest) => requestJson<SaveConfigResponse>(`/api/egress-groups/${encodeURIComponent(name)}/update`, { method: "PUT", body: JSON.stringify(body) }),
  deleteEgressGroup: (name: string) => requestJson<SaveConfigResponse>(`/api/egress-groups/${encodeURIComponent(name)}`, { method: "DELETE" }),
  getMihomoVersions: () => requestJson<MihomoVersionsResponse>("/api/mihomo/versions"),
  downloadMihomoVersion: (version: string) => requestJson<MihomoVersionActionResponse>(`/api/mihomo/versions/${encodeURIComponent(version)}/download`, { method: "POST" }),
  installMihomoVersion: (version: string) => requestJson<MihomoVersionActionResponse>(`/api/mihomo/versions/${encodeURIComponent(version)}/install`, { method: "POST" }),
  activateMihomoVersion: (version: string) => requestJson<MihomoVersionActionResponse>(`/api/mihomo/versions/${encodeURIComponent(version)}/activate`, { method: "POST" }),
  getHostStatus: () => requestJson<HostStatus>("/api/host/status"),
  getUpdaterStatus: () => requestJson<UpdaterStatus>("/api/host/updater"),
  checkForUpdates: () => requestJson<SimpleOkResponse>("/api/host/updater/check", { method: "POST" }),
  setHostAutostart: (enabled: boolean) => requestJson<SimpleOkResponse>("/api/host/autostart", { method: "PUT", body: JSON.stringify({ enabled }) }),
  openHostControlCenter: () => requestJson<SimpleOkResponse>("/api/host/open-browser", { method: "POST" }),
}
