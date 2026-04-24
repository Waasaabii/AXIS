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
import { apiKeys, apiPath } from "./api-keys"
import type { BaotaConfigResponse, CoreCapabilitiesResponse, LLMModelsResponse, LLMProposalRequest, LLMProposalResponse, LLMTestRequest, LLMTestResponse, LocalNodeCheckResponse, LocalNodeConnectionResponse, NodeSource, PublicationConfig, RouteConfig, UsageView } from "./product-types"

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
  getBootstrapStatus: () => requestJson<BootstrapStatus>(apiKeys.bootstrapStatus),
  getSession: () => requestJson<SessionStatusResponse>(apiKeys.session),
  login: (body: LoginRequest) => requestJson<LoginResponse>(apiKeys.session, { method: "POST", body: JSON.stringify(body) }),
  logout: () => requestJson<SimpleOkResponse>(apiKeys.session, { method: "DELETE" }),
  updatePassword: (body: UpdatePasswordRequest) => requestJson<SimpleOkResponse>(apiKeys.sessionPassword, { method: "PUT", body: JSON.stringify(body) }),
  bootstrapAdmin: (body: BootstrapAdminRequest) => requestJson<SimpleOkResponse>(apiKeys.setupAdmin, { method: "PUT", body: JSON.stringify(body) }),
  getStatus: () => requestJson<StatusResponse>(apiKeys.status),
  getCoreCapabilities: () => requestJson<CoreCapabilitiesResponse>(apiKeys.coreCapabilities),
  getLLMModels: (body?: LLMTestRequest) => body ? requestJson<LLMModelsResponse>(apiKeys.llmModels, { method: "POST", body: JSON.stringify(body) }) : requestJson<LLMModelsResponse>(apiKeys.llmModels),
  testLLM: (body: LLMTestRequest) => requestJson<LLMTestResponse>(apiKeys.llmTest, { method: "POST", body: JSON.stringify(body) }),
  createLLMProposal: (body: LLMProposalRequest) => requestJson<LLMProposalResponse>(apiKeys.llmProposals, { method: "POST", body: JSON.stringify(body) }),
  getConfig: () => requestJson<ConfigEnvelope>(apiKeys.config),
  saveConfig: (body: SaveConfigRequest) => requestJson<SaveConfigResponse>(apiKeys.config, { method: "PUT", body: JSON.stringify(body) }),
  getNodeSources: () => requestJson<NodeSource[]>(apiKeys.nodeSources),
  addNodeSource: (body: Partial<NodeSource>) => requestJson<SaveConfigResponse>(apiKeys.nodeSources, { method: "POST", body: JSON.stringify(body) }),
  updateNodeSource: (name: string, body: Partial<NodeSource>) => requestJson<SaveConfigResponse>(apiPath.nodeSourceUpdate(name), { method: "PUT", body: JSON.stringify(body) }),
  deleteNodeSource: (name: string) => requestJson<SaveConfigResponse>(apiPath.nodeSource(name), { method: "DELETE" }),
  checkLocalNode: (name: string) => requestJson<LocalNodeCheckResponse>(apiPath.nodeSourceCheck(name)),
  getLocalNodeConnection: (name: string) => requestJson<LocalNodeConnectionResponse>(apiPath.nodeSourceConnection(name)),
  getLocalNodeBaotaConfig: (name: string) => requestJson<BaotaConfigResponse>(apiPath.nodeSourceBaotaConfig(name)),
  getRoutes: () => requestJson<RouteConfig[]>(apiKeys.routes),
  addRoute: (body: Partial<RouteConfig>) => requestJson<SaveConfigResponse>(apiKeys.routes, { method: "POST", body: JSON.stringify(body) }),
  updateRoute: (name: string, body: Partial<RouteConfig>) => requestJson<SaveConfigResponse>(apiPath.routeUpdate(name), { method: "PUT", body: JSON.stringify(body) }),
  deleteRoute: (name: string) => requestJson<SaveConfigResponse>(apiPath.route(name), { method: "DELETE" }),
  getUsage: () => requestJson<UsageView>(apiKeys.usage),
  updateUsage: (body: Partial<UsageView>) => requestJson<SaveConfigResponse>(apiKeys.usage, { method: "PUT", body: JSON.stringify(body) }),
  getPublications: () => requestJson<PublicationConfig[]>(apiKeys.publications),
  addPublication: (body: Partial<PublicationConfig>) => requestJson<SaveConfigResponse>(apiKeys.publications, { method: "POST", body: JSON.stringify(body) }),
  updatePublication: (name: string, body: Partial<PublicationConfig>) => requestJson<SaveConfigResponse>(apiPath.publicationUpdate(name), { method: "PUT", body: JSON.stringify(body) }),
  deletePublication: (name: string) => requestJson<SaveConfigResponse>(apiPath.publication(name), { method: "DELETE" }),
  getProviders: () => requestJson<ProviderListItem[]>(apiKeys.providers),
  refreshProvider: (name: string) => requestJson<ProviderRefreshResponse>(apiPath.providerRefresh(name), { method: "POST" }),
  getGroups: () => requestJson<GroupView[]>(apiKeys.groups),
  getTransitRoutes: () => requestJson<TransitRouteView[]>(apiKeys.transitRoutes),
  getLandingProxies: () => requestJson<LandingProxyView[]>(apiKeys.landingProxies),
  selectGroup: (groupName: string, body: SelectGroupRequest) => requestJson<GroupMutationResponse>(apiPath.groupSelect(groupName), { method: "POST", body: JSON.stringify(body) }),
  healthcheckGroup: (groupName: string) => requestJson<GroupMutationResponse>(apiPath.groupHealthcheck(groupName), { method: "POST" }),
  addTransitRoute: (body: AddTransitRouteRequest) => requestJson<SaveConfigResponse>(apiKeys.transitRoutes, { method: "POST", body: JSON.stringify(body) }),
  updateTransitRoute: (name: string, body: UpdateTransitRouteRequest) => requestJson<SaveConfigResponse>(apiPath.transitRouteUpdate(name), { method: "PUT", body: JSON.stringify(body) }),
  deleteTransitRoute: (name: string) => requestJson<SaveConfigResponse>(apiPath.transitRoute(name), { method: "DELETE" }),
  healthcheckTransitRoute: (name: string) => requestJson<TransitRouteMutationResponse>(apiPath.transitRouteHealthcheck(name), { method: "POST" }),
  addLandingProxy: (body: AddLandingProxyRequest) => requestJson<SaveConfigResponse>(apiKeys.landingProxies, { method: "POST", body: JSON.stringify(body) }),
  updateLandingProxy: (name: string, body: UpdateLandingProxyRequest) => requestJson<SaveConfigResponse>(apiPath.landingProxyUpdate(name), { method: "PUT", body: JSON.stringify(body) }),
  deleteLandingProxy: (name: string) => requestJson<SaveConfigResponse>(apiPath.landingProxy(name), { method: "DELETE" }),
  getListeners: () => requestJson<ListenerView[]>(apiKeys.listeners),
  addListener: (body: AddListenerRequest) => requestJson<SaveConfigResponse>(apiKeys.listeners, { method: "POST", body: JSON.stringify(body) }),
  updateListener: (name: string, body: UpdateListenerRequest) => requestJson<SaveConfigResponse>(apiPath.listenerUpdate(name), { method: "PUT", body: JSON.stringify(body) }),
  deleteListener: (name: string) => requestJson<SaveConfigResponse>(apiPath.listener(name), { method: "DELETE" }),
  getEvents: () => requestJson<EventEntry[]>(apiKeys.events),
  getRenderedConfig: () => requestJson<RenderedConfigResponse>(apiKeys.renderedConfig),
  getController: () => requestJson<ControllerStatusResponse>(apiKeys.controller),
  getSetupState: () => requestJson<SetupStateResponse>(apiKeys.setupState),
  getRuntimePreflight: () => requestJson<RuntimePreflightSnapshot>(apiKeys.runtimePreflight),
  probeController: () => requestJson<ProbeControllerResponse>(apiKeys.controllerProbe, { method: "POST" }),
  reloadRuntime: () => requestJson<ReloadResponse>(apiKeys.reloadRuntime, { method: "POST" }),
  addSubscription: (body: AddSubscriptionRequest) => requestJson<SaveConfigResponse>(apiKeys.subscriptions, { method: "POST", body: JSON.stringify(body) }),
  updateSubscription: (name: string, body: UpdateSubscriptionRequest) => requestJson<SaveConfigResponse>(apiPath.subscriptionUpdate(name), { method: "PUT", body: JSON.stringify(body) }),
  toggleSubscription: (name: string, body: ToggleSubscriptionRequest) => requestJson<SaveConfigResponse>(apiPath.subscriptionToggle(name), { method: "POST", body: JSON.stringify(body) }),
  deleteSubscription: (name: string) => requestJson<SaveConfigResponse>(apiPath.subscription(name), { method: "DELETE" }),
  addEgressGroup: (body: AddEgressGroupRequest) => requestJson<SaveConfigResponse>(apiKeys.egressGroups, { method: "POST", body: JSON.stringify(body) }),
  updateEgressGroup: (name: string, body: UpdateEgressGroupRequest) => requestJson<SaveConfigResponse>(apiPath.egressGroupUpdate(name), { method: "PUT", body: JSON.stringify(body) }),
  deleteEgressGroup: (name: string) => requestJson<SaveConfigResponse>(apiPath.egressGroup(name), { method: "DELETE" }),
  getMihomoVersions: () => requestJson<MihomoVersionsResponse>(apiKeys.mihomoVersions),
  downloadMihomoVersion: (version: string) => requestJson<MihomoVersionActionResponse>(apiPath.mihomoVersionDownload(version), { method: "POST" }),
  installMihomoVersion: (version: string) => requestJson<MihomoVersionActionResponse>(apiPath.mihomoVersionInstall(version), { method: "POST" }),
  activateMihomoVersion: (version: string) => requestJson<MihomoVersionActionResponse>(apiPath.mihomoVersionActivate(version), { method: "POST" }),
  getHostStatus: () => requestJson<HostStatus>(apiKeys.hostStatus),
  getUpdaterStatus: () => requestJson<UpdaterStatus>(apiKeys.updaterStatus),
  checkForUpdates: () => requestJson<SimpleOkResponse>(apiKeys.updaterCheck, { method: "POST" }),
  setHostAutostart: (enabled: boolean) => requestJson<SimpleOkResponse>(apiKeys.hostAutostart, { method: "PUT", body: JSON.stringify({ enabled }) }),
  openHostControlCenter: () => requestJson<SimpleOkResponse>(apiKeys.hostOpenBrowser, { method: "POST" }),
}
