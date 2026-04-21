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

type WailsBindingMethod = (...args: unknown[]) => Promise<unknown>
type WailsBindingGroup = Record<string, WailsBindingMethod>

declare global {
  interface Window {
    go?: {
      main?: Record<string, WailsBindingGroup>
    }
  }
}

function readWailsMethod(bindingName: string, methodName: string): WailsBindingMethod | undefined {
  if (typeof window === "undefined") {
    return undefined
  }
  return window.go?.main?.[bindingName]?.[methodName]
}

async function callBinding<T>(bindingName: string, methodName: string, ...args: unknown[]): Promise<T> {
  const target = readWailsMethod(bindingName, methodName)
  if (typeof target !== "function") {
    throw new ApiError(`Wails 绑定缺失: ${bindingName}.${methodName}`, 500, null)
  }
  try {
    return (await target(...args)) as T
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error)
    throw new ApiError(message || "Wails 调用失败", 500, error)
  }
}

export function hasWailsRuntime() {
  return typeof window !== "undefined" && typeof readWailsMethod("EngineBindings", "GetStatus") === "function"
}

export const wailsAPI: AxisAPI = {
  getBootstrapStatus: () => callBinding<BootstrapStatus>("EngineBindings", "GetBootstrapStatus"),
  getSession: () => callBinding<SessionStatusResponse>("EngineBindings", "GetSession"),
  login: (body: LoginRequest) => callBinding<LoginResponse>("EngineBindings", "Login", body),
  logout: () => callBinding<SimpleOkResponse>("EngineBindings", "Logout"),
  updatePassword: (body: UpdatePasswordRequest) => callBinding<SimpleOkResponse>("EngineBindings", "UpdatePassword", body),
  bootstrapAdmin: (body: BootstrapAdminRequest) => callBinding<SimpleOkResponse>("EngineBindings", "BootstrapAdmin", body),
  getStatus: () => callBinding<StatusResponse>("EngineBindings", "GetStatus"),
  getConfig: () => callBinding<ConfigEnvelope>("EngineBindings", "GetConfig"),
  saveConfig: (body: SaveConfigRequest) => callBinding<SaveConfigResponse>("EngineBindings", "SaveConfig", body),
  getProviders: () => callBinding<ProviderListItem[]>("EngineBindings", "GetProviders"),
  refreshProvider: (name: string) => callBinding<ProviderRefreshResponse>("EngineBindings", "RefreshProvider", name),
  getGroups: () => callBinding<GroupView[]>("EngineBindings", "GetGroups"),
  getTransitRoutes: () => callBinding<TransitRouteView[]>("EngineBindings", "GetTransitRoutes"),
  getLandingProxies: () => callBinding<LandingProxyView[]>("EngineBindings", "GetLandingProxies"),
  selectGroup: (groupName: string, body: SelectGroupRequest) => callBinding<GroupMutationResponse>("EngineBindings", "SelectGroup", groupName, body),
  healthcheckGroup: (groupName: string) => callBinding<GroupMutationResponse>("EngineBindings", "HealthcheckGroup", groupName),
  addTransitRoute: (body: AddTransitRouteRequest) => callBinding<SaveConfigResponse>("EngineBindings", "AddTransitRoute", body),
  updateTransitRoute: (name: string, body: UpdateTransitRouteRequest) => callBinding<SaveConfigResponse>("EngineBindings", "UpdateTransitRoute", name, body),
  deleteTransitRoute: (name: string) => callBinding<SaveConfigResponse>("EngineBindings", "DeleteTransitRoute", name),
  healthcheckTransitRoute: (name: string) => callBinding<TransitRouteMutationResponse>("EngineBindings", "HealthcheckTransitRoute", name),
  addLandingProxy: (body: AddLandingProxyRequest) => callBinding<SaveConfigResponse>("EngineBindings", "AddLandingProxy", body),
  updateLandingProxy: (name: string, body: UpdateLandingProxyRequest) => callBinding<SaveConfigResponse>("EngineBindings", "UpdateLandingProxy", name, body),
  deleteLandingProxy: (name: string) => callBinding<SaveConfigResponse>("EngineBindings", "DeleteLandingProxy", name),
  getListeners: () => callBinding<ListenerView[]>("EngineBindings", "GetListeners"),
  addListener: (body: AddListenerRequest) => callBinding<SaveConfigResponse>("EngineBindings", "AddListener", body),
  updateListener: (name: string, body: UpdateListenerRequest) => callBinding<SaveConfigResponse>("EngineBindings", "UpdateListener", name, body),
  deleteListener: (name: string) => callBinding<SaveConfigResponse>("EngineBindings", "DeleteListener", name),
  getEvents: () => callBinding<EventEntry[]>("EngineBindings", "GetEvents"),
  getRenderedConfig: () => callBinding<RenderedConfigResponse>("EngineBindings", "GetRenderedConfig"),
  getController: () => callBinding<ControllerStatusResponse>("EngineBindings", "GetController"),
  getSetupState: () => callBinding<SetupStateResponse>("EngineBindings", "GetSetupState"),
  getRuntimePreflight: () => callBinding<RuntimePreflightSnapshot>("EngineBindings", "GetRuntimePreflight"),
  probeController: () => callBinding<ProbeControllerResponse>("EngineBindings", "ProbeController"),
  reloadRuntime: () => callBinding<ReloadResponse>("EngineBindings", "ReloadRuntime"),
  addSubscription: (body: AddSubscriptionRequest) => callBinding<SaveConfigResponse>("EngineBindings", "AddSubscription", body),
  updateSubscription: (name: string, body: UpdateSubscriptionRequest) => callBinding<SaveConfigResponse>("EngineBindings", "UpdateSubscription", name, body),
  toggleSubscription: (name: string, body: ToggleSubscriptionRequest) => callBinding<SaveConfigResponse>("EngineBindings", "ToggleSubscription", name, body),
  deleteSubscription: (name: string) => callBinding<SaveConfigResponse>("EngineBindings", "DeleteSubscription", name),
  addEgressGroup: (body: AddEgressGroupRequest) => callBinding<SaveConfigResponse>("EngineBindings", "AddEgressGroup", body),
  updateEgressGroup: (name: string, body: UpdateEgressGroupRequest) => callBinding<SaveConfigResponse>("EngineBindings", "UpdateEgressGroup", name, body),
  deleteEgressGroup: (name: string) => callBinding<SaveConfigResponse>("EngineBindings", "DeleteEgressGroup", name),
  getMihomoVersions: () => callBinding<MihomoVersionsResponse>("EngineBindings", "GetMihomoVersions"),
  downloadMihomoVersion: (version: string) => callBinding<MihomoVersionActionResponse>("EngineBindings", "DownloadMihomoVersion", version),
  installMihomoVersion: (version: string) => callBinding<MihomoVersionActionResponse>("EngineBindings", "InstallMihomoVersion", version),
  activateMihomoVersion: (version: string) => callBinding<MihomoVersionActionResponse>("EngineBindings", "ActivateMihomoVersion", version),
  getHostStatus: () => callBinding<HostStatus>("HostBindings", "GetStatus"),
  getUpdaterStatus: () => callBinding<UpdaterStatus>("HostBindings", "GetUpdaterStatus"),
  checkForUpdates: () => callBinding<SimpleOkResponse>("HostBindings", "CheckForUpdates"),
  setHostAutostart: (enabled: boolean) => callBinding<SimpleOkResponse>("HostBindings", "SetAutostart", enabled),
  openHostControlCenter: () => callBinding<SimpleOkResponse>("HostBindings", "OpenControlCenter"),
}
