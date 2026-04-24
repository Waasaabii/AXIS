export interface ListenerUser {
  username: string
  password: string
}

export interface NodeSource {
  name: string
  type: 'subscription' | 'proxy' | 'axis' | 'local_node' | string
  enabled: boolean
  protocol?: string
  subscription?: {
    url?: string
    interval?: number
    health_check_url?: string
    health_check_interval?: number
  }
  endpoint?: {
    server?: string
    port?: number
    username?: string
    password?: string
    tls?: boolean
    sni?: string
    skip_cert_verify?: boolean
  }
  axis?: {
    url?: string
    username?: string
    password?: string
    token?: string
  }
  local_node?: {
    access_mode?: string
    listen?: string
    port?: number
    external_host?: string
    external_port?: number
    users?: ListenerUser[]
    route?: string
    certificate?: string
    private_key?: string
    tls?: boolean
    sni?: string
    skip_cert_verify?: boolean
  }
  notes?: string
}

export interface RouteConfig {
  name: string
  enabled: boolean
  entry?: { source?: string; node?: string }
  strategy?: string
  landing?: { source?: string; node?: string }
  health_check?: { url?: string; interval?: number }
  notes?: string
}

export interface LocalProxyUsage {
  enabled: boolean
  type: string
  listen: string
  port: number
  users?: ListenerUser[]
}

export interface VirtualInterfaceUsage {
  enabled: boolean
  mode?: string
  status?: string
  message?: string
}

export interface UsageView {
  selectedRoute?: string
  localProxy: LocalProxyUsage
  virtualInterface: VirtualInterfaceUsage
  currentRoute?: RouteConfig
  ready: boolean
  missingSteps: string[]
  lastAppliedAt?: string
  lastApplyStatus?: string
  lastApplyMessage?: string
}

export interface PublicationConfig {
  name: string
  type: 'http_proxy' | 'subscription' | 'axis_connection' | string
  enabled: boolean
  route: string
  listen?: string
  port?: number
  auth: {
    username?: string
    password?: string
    token?: string
  }
  accessScope?: string
  access_scope?: string
  format?: string
}

export interface SaveConfigResponse {
  ok: boolean
  config?: unknown
  path?: string
  message?: string
}

export interface LocalNodeCheckResponse {
  ok: boolean
  name: string
  message: string
  checks: Array<{ name: string; ok: boolean; message: string; action?: string }>
}

export interface LocalNodeConnectionResponse {
  ok: boolean
  name: string
  protocol: string
  host: string
  port: number
  connections: Array<{ username?: string; password?: string; address: string; url: string }>
  warnings?: string[]
}

export interface BaotaConfigResponse {
  ok: boolean
  name: string
  protocol: string
  mode: string
  target: string
  snippet: string
  steps: string[]
  warnings?: string[]
  applySupported: boolean
}


export interface CoreCapabilityLayer {
  protocol: string
  import: string
  preserve: string
  publish: string
  usage: string
  create: string
  diagnostics: string
  warnings?: string[]
}

export interface CoreCapabilitiesResponse {
  coreName: string
  coreVersion?: string
  coreDetected: boolean
  checkedAt: string
  protocols: CoreCapabilityLayer[]
  axisTemplates: string[]
  message: string
}

export interface LLMModelInfo {
  id: string
  name?: string
  displayName?: string
  ownedBy?: string
}

export interface LLMModelsResponse {
  ok: boolean
  enabled: boolean
  models: LLMModelInfo[]
  message: string
}

export interface LLMTestRequest {
  base_url?: string
  api_key?: string
  model?: string
  endpoint?: string
}

export interface LLMTestResponse {
  ok: boolean
  endpoint: string
  model: string
  content?: string
  message: string
}

export interface LLMProposalRequest {
  kind: string
  goal: string
  target?: string
  context?: Record<string, unknown>
  stream?: boolean
}

export interface LLMProposalResponse {
  ok: boolean
  enabled: boolean
  mode: string
  kind: string
  status: string
  steps: Array<{ title: string; status: string; message: string }>
  proposal: Record<string, unknown>
  warnings?: string[]
  needsTest: boolean
  needsConfirm: boolean
  message: string
}
