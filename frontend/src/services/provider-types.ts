import type { components } from "@/generated/openapi"

type Schemas = components["schemas"]

export type ErrorResponse = Schemas["ErrorResponse"]
export type SimpleOkResponse = Schemas["SimpleOkResponse"]
export type LoginRequest = Schemas["LoginRequest"]
export type LoginResponse = Schemas["LoginResponse"]
export type UpdatePasswordRequest = Schemas["UpdatePasswordRequest"]
export type BootstrapAdminRequest = Schemas["BootstrapAdminRequest"]
export type SessionStatusResponse = Schemas["SessionStatusResponse"]
export type Config = Schemas["Config"]
export type ConfigEnvelope = Schemas["ConfigEnvelope"]
export type SaveConfigRequest = Schemas["SaveConfigRequest"]
export type SaveConfigResponse = Schemas["SaveConfigResponse"]
export type StatusResponse = Schemas["StatusResponse"]
export type ControllerStatusResponse = Schemas["ControllerStatusResponse"]
export type ProbeControllerResponse = Schemas["ProbeControllerResponse"]
export type ReloadResponse = Schemas["ReloadResponse"]
export type RuntimePreflightSnapshot = Schemas["RuntimePreflightSnapshot"]
export type ProviderListItem = Schemas["ProviderListItem"]
export type ProviderRefreshResponse = Schemas["ProviderRefreshResponse"]
export type GroupView = Schemas["GroupView"]
export type GroupMutationResponse = Schemas["GroupMutationResponse"]
export type TransitRouteView = Schemas["TransitRouteView"]
export type TransitRouteMutationResponse = Schemas["TransitRouteMutationResponse"]
export type AddTransitRouteRequest = Schemas["AddTransitRouteRequest"]
export type UpdateTransitRouteRequest = Schemas["UpdateTransitRouteRequest"]
export type SelectGroupRequest = Schemas["SelectGroupRequest"]
export type LandingProxyView = Schemas["LandingProxyView"]
export type AddLandingProxyRequest = Schemas["AddLandingProxyRequest"]
export type UpdateLandingProxyRequest = Schemas["UpdateLandingProxyRequest"]
export type ListenerUser = Schemas["ListenerUser"]
export type ListenerView = Schemas["ListenerView"]
export type AddListenerRequest = Schemas["AddListenerRequest"]
export type UpdateListenerRequest = Schemas["UpdateListenerRequest"]
export type EventEntry = Schemas["EventEntry"]
export type RenderedConfigResponse = Schemas["RenderedConfigResponse"]
export type AddSubscriptionRequest = Schemas["AddSubscriptionRequest"]
export type ToggleSubscriptionRequest = Schemas["ToggleSubscriptionRequest"]
export type AddEgressGroupRequest = Schemas["AddEgressGroupRequest"]
export type UpdateEgressGroupRequest = Schemas["UpdateEgressGroupRequest"]
export type MihomoVersionsResponse = Schemas["MihomoVersionsResponse"]
export type MihomoVersionActionResponse = Schemas["MihomoVersionActionResponse"]
export type SetupStateResponse = Schemas["SetupState"]
export type HostStatus = Schemas["HostStatus"]
export type MainServiceStatus = Schemas["MainServiceStatus"]
export type UpdaterStatus = Schemas["UpdaterStatus"]
export type BootstrapAuthStatus = Schemas["BootstrapAuthStatus"]
export type BootstrapStatus = Schemas["BootstrapStatus"]

export class ApiError extends Error {
  status: number
  info: unknown

  constructor(message: string, status: number, info: unknown) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.info = info
  }
}

export interface AxisAPI {
  getBootstrapStatus: () => Promise<BootstrapStatus>
  getSession: () => Promise<SessionStatusResponse>
  login: (body: LoginRequest) => Promise<LoginResponse>
  logout: () => Promise<SimpleOkResponse>
  updatePassword: (body: UpdatePasswordRequest) => Promise<SimpleOkResponse>
  bootstrapAdmin: (body: BootstrapAdminRequest) => Promise<SimpleOkResponse>
  getStatus: () => Promise<StatusResponse>
  getConfig: () => Promise<ConfigEnvelope>
  saveConfig: (body: SaveConfigRequest) => Promise<SaveConfigResponse>
  getProviders: () => Promise<ProviderListItem[]>
  refreshProvider: (name: string) => Promise<ProviderRefreshResponse>
  getGroups: () => Promise<GroupView[]>
  getTransitRoutes: () => Promise<TransitRouteView[]>
  getLandingProxies: () => Promise<LandingProxyView[]>
  selectGroup: (groupName: string, body: SelectGroupRequest) => Promise<GroupMutationResponse>
  healthcheckGroup: (groupName: string) => Promise<GroupMutationResponse>
  addTransitRoute: (body: AddTransitRouteRequest) => Promise<SaveConfigResponse>
  updateTransitRoute: (name: string, body: UpdateTransitRouteRequest) => Promise<SaveConfigResponse>
  deleteTransitRoute: (name: string) => Promise<SaveConfigResponse>
  healthcheckTransitRoute: (name: string) => Promise<TransitRouteMutationResponse>
  addLandingProxy: (body: AddLandingProxyRequest) => Promise<SaveConfigResponse>
  updateLandingProxy: (name: string, body: UpdateLandingProxyRequest) => Promise<SaveConfigResponse>
  deleteLandingProxy: (name: string) => Promise<SaveConfigResponse>
  getListeners: () => Promise<ListenerView[]>
  addListener: (body: AddListenerRequest) => Promise<SaveConfigResponse>
  updateListener: (name: string, body: UpdateListenerRequest) => Promise<SaveConfigResponse>
  deleteListener: (name: string) => Promise<SaveConfigResponse>
  getEvents: () => Promise<EventEntry[]>
  getRenderedConfig: () => Promise<RenderedConfigResponse>
  getController: () => Promise<ControllerStatusResponse>
  getSetupState: () => Promise<SetupStateResponse>
  getRuntimePreflight: () => Promise<RuntimePreflightSnapshot>
  probeController: () => Promise<ProbeControllerResponse>
  reloadRuntime: () => Promise<ReloadResponse>
  addSubscription: (body: AddSubscriptionRequest) => Promise<SaveConfigResponse>
  toggleSubscription: (name: string, body: ToggleSubscriptionRequest) => Promise<SaveConfigResponse>
  deleteSubscription: (name: string) => Promise<SaveConfigResponse>
  addEgressGroup: (body: AddEgressGroupRequest) => Promise<SaveConfigResponse>
  updateEgressGroup: (name: string, body: UpdateEgressGroupRequest) => Promise<SaveConfigResponse>
  deleteEgressGroup: (name: string) => Promise<SaveConfigResponse>
  getMihomoVersions: () => Promise<MihomoVersionsResponse>
  downloadMihomoVersion: (version: string) => Promise<MihomoVersionActionResponse>
  installMihomoVersion: (version: string) => Promise<MihomoVersionActionResponse>
  activateMihomoVersion: (version: string) => Promise<MihomoVersionActionResponse>
  getHostStatus: () => Promise<HostStatus>
  getUpdaterStatus: () => Promise<UpdaterStatus>
  checkForUpdates: () => Promise<SimpleOkResponse>
  setHostAutostart: (enabled: boolean) => Promise<SimpleOkResponse>
  openHostControlCenter: () => Promise<SimpleOkResponse>
}
