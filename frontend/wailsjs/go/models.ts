export namespace axis {
	
	export class AxisConnectionConfig {
	    url?: string;
	    username?: string;
	    password?: string;
	    token?: string;
	
	    static createFrom(source: any = {}) {
	        return new AxisConnectionConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.token = source["token"];
	    }
	}
	export class BaotaConfigResponse {
	    ok: boolean;
	    name: string;
	    protocol: string;
	    mode: string;
	    target: string;
	    snippet: string;
	    steps: string[];
	    warnings?: string[];
	    applySupported: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BaotaConfigResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.mode = source["mode"];
	        this.target = source["target"];
	        this.snippet = source["snippet"];
	        this.steps = source["steps"];
	        this.warnings = source["warnings"];
	        this.applySupported = source["applySupported"];
	    }
	}
	export class BootstrapAuthStatus {
	    authenticated: boolean;
	    username?: string;
	    requiresPasswordReset: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapAuthStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.authenticated = source["authenticated"];
	        this.username = source["username"];
	        this.requiresPasswordReset = source["requiresPasswordReset"];
	    }
	}
	export class SetupCheck {
	    key: string;
	    title: string;
	    ready: boolean;
	    summary: string;
	    action?: string;
	
	    static createFrom(source: any = {}) {
	        return new SetupCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.title = source["title"];
	        this.ready = source["ready"];
	        this.summary = source["summary"];
	        this.action = source["action"];
	    }
	}
	export class SetupState {
	    required: boolean;
	    needsPasswordReset: boolean;
	    adminUsername: string;
	    hasNodeSources: boolean;
	    hasRoutes: boolean;
	    hasUsage: boolean;
	    hasPublications: boolean;
	    hasSubscriptions: boolean;
	    hasRealSubscriptions: boolean;
	    hasEgressGroups: boolean;
	    hasListeners: boolean;
	    checks: SetupCheck[];
	    reasons: string[];
	
	    static createFrom(source: any = {}) {
	        return new SetupState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.required = source["required"];
	        this.needsPasswordReset = source["needsPasswordReset"];
	        this.adminUsername = source["adminUsername"];
	        this.hasNodeSources = source["hasNodeSources"];
	        this.hasRoutes = source["hasRoutes"];
	        this.hasUsage = source["hasUsage"];
	        this.hasPublications = source["hasPublications"];
	        this.hasSubscriptions = source["hasSubscriptions"];
	        this.hasRealSubscriptions = source["hasRealSubscriptions"];
	        this.hasEgressGroups = source["hasEgressGroups"];
	        this.hasListeners = source["hasListeners"];
	        this.checks = this.convertValues(source["checks"], SetupCheck);
	        this.reasons = source["reasons"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class UpdaterStatus {
	    mode: string;
	    running: boolean;
	    state: string;
	    currentVersion?: string;
	    latestVersion?: string;
	    updateAvailable: boolean;
	    canAutoApply: boolean;
	    releaseUrl?: string;
	    assetName?: string;
	    assetUrl?: string;
	    checkedAt?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdaterStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.running = source["running"];
	        this.state = source["state"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.updateAvailable = source["updateAvailable"];
	        this.canAutoApply = source["canAutoApply"];
	        this.releaseUrl = source["releaseUrl"];
	        this.assetName = source["assetName"];
	        this.assetUrl = source["assetUrl"];
	        this.checkedAt = source["checkedAt"];
	        this.message = source["message"];
	    }
	}
	export class MainServiceStatus {
	    ready: boolean;
	    state: string;
	    mode: string;
	    controller?: string;
	    configPath?: string;
	    blockingReason?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new MainServiceStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.state = source["state"];
	        this.mode = source["mode"];
	        this.controller = source["controller"];
	        this.configPath = source["configPath"];
	        this.blockingReason = source["blockingReason"];
	        this.message = source["message"];
	    }
	}
	export class HostStatus {
	    mode: string;
	    platform?: string;
	    desktopMode: boolean;
	    autostartEnabled: boolean;
	    autostartManaged: boolean;
	    configPath?: string;
	    runtimeDir?: string;
	    listenAddress?: string;
	    logs?: string[];
	
	    static createFrom(source: any = {}) {
	        return new HostStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mode = source["mode"];
	        this.platform = source["platform"];
	        this.desktopMode = source["desktopMode"];
	        this.autostartEnabled = source["autostartEnabled"];
	        this.autostartManaged = source["autostartManaged"];
	        this.configPath = source["configPath"];
	        this.runtimeDir = source["runtimeDir"];
	        this.listenAddress = source["listenAddress"];
	        this.logs = source["logs"];
	    }
	}
	export class BootstrapStatus {
	    host: HostStatus;
	    mainService: MainServiceStatus;
	    updater: UpdaterStatus;
	    setup: SetupState;
	    auth: BootstrapAuthStatus;
	    nextStep: string;
	    blockingReason?: string;
	
	    static createFrom(source: any = {}) {
	        return new BootstrapStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.host = this.convertValues(source["host"], HostStatus);
	        this.mainService = this.convertValues(source["mainService"], MainServiceStatus);
	        this.updater = this.convertValues(source["updater"], UpdaterStatus);
	        this.setup = this.convertValues(source["setup"], SetupState);
	        this.auth = this.convertValues(source["auth"], BootstrapAuthStatus);
	        this.nextStep = source["nextStep"];
	        this.blockingReason = source["blockingReason"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CapabilityLayer {
	    protocol: string;
	    import: string;
	    preserve: string;
	    publish: string;
	    usage: string;
	    create: string;
	    diagnostics: string;
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new CapabilityLayer(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.protocol = source["protocol"];
	        this.import = source["import"];
	        this.preserve = source["preserve"];
	        this.publish = source["publish"];
	        this.usage = source["usage"];
	        this.create = source["create"];
	        this.diagnostics = source["diagnostics"];
	        this.warnings = source["warnings"];
	    }
	}
	export class CoreCapabilitiesResponse {
	    coreName: string;
	    coreVersion?: string;
	    coreDetected: boolean;
	    checkedAt: string;
	    protocols: CapabilityLayer[];
	    axisTemplates: string[];
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new CoreCapabilitiesResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.coreName = source["coreName"];
	        this.coreVersion = source["coreVersion"];
	        this.coreDetected = source["coreDetected"];
	        this.checkedAt = source["checkedAt"];
	        this.protocols = this.convertValues(source["protocols"], CapabilityLayer);
	        this.axisTemplates = source["axisTemplates"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class EventEntry {
	    id: string;
	    level: string;
	    scope: string;
	    message: string;
	    at: string;
	
	    static createFrom(source: any = {}) {
	        return new EventEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.level = source["level"];
	        this.scope = source["scope"];
	        this.message = source["message"];
	        this.at = source["at"];
	    }
	}
	export class GroupCandidate {
	    id: string;
	    name: string;
	    type: string;
	    server: string;
	    port: number;
	    nodeName?: string;
	
	    static createFrom(source: any = {}) {
	        return new GroupCandidate(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.server = source["server"];
	        this.port = source["port"];
	        this.nodeName = source["nodeName"];
	    }
	}
	export class GroupView {
	    name: string;
	    mode: string;
	    provider: string;
	    filter: string;
	    excludeFilter?: string;
	    candidateCount: number;
	    current?: string;
	    currentValue?: string;
	    candidates: GroupCandidate[];
	    proxyOrder?: string[];
	    lastHealthcheckAt?: string;
	    landingProxy?: string;
	    healthCheckURL?: string;
	    healthCheckInterval?: number;
	    routeSummary?: string;
	    providerMissing: boolean;
	    providerDisabled: boolean;
	    landingMissing: boolean;
	    landingDisabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new GroupView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.mode = source["mode"];
	        this.provider = source["provider"];
	        this.filter = source["filter"];
	        this.excludeFilter = source["excludeFilter"];
	        this.candidateCount = source["candidateCount"];
	        this.current = source["current"];
	        this.currentValue = source["currentValue"];
	        this.candidates = this.convertValues(source["candidates"], GroupCandidate);
	        this.proxyOrder = source["proxyOrder"];
	        this.lastHealthcheckAt = source["lastHealthcheckAt"];
	        this.landingProxy = source["landingProxy"];
	        this.healthCheckURL = source["healthCheckURL"];
	        this.healthCheckInterval = source["healthCheckInterval"];
	        this.routeSummary = source["routeSummary"];
	        this.providerMissing = source["providerMissing"];
	        this.providerDisabled = source["providerDisabled"];
	        this.landingMissing = source["landingMissing"];
	        this.landingDisabled = source["landingDisabled"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class LLMModelInfo {
	    id: string;
	    name?: string;
	    displayName?: string;
	    ownedBy?: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMModelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.displayName = source["displayName"];
	        this.ownedBy = source["ownedBy"];
	    }
	}
	export class LLMModelsResponse {
	    ok: boolean;
	    enabled: boolean;
	    models: LLMModelInfo[];
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMModelsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.enabled = source["enabled"];
	        this.models = this.convertValues(source["models"], LLMModelInfo);
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LLMProposalStep {
	    title: string;
	    status: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMProposalStep(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class LLMProposalResponse {
	    ok: boolean;
	    enabled: boolean;
	    mode: string;
	    kind: string;
	    status: string;
	    steps: LLMProposalStep[];
	    proposal: Record<string, any>;
	    warnings?: string[];
	    needsTest: boolean;
	    needsConfirm: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMProposalResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.enabled = source["enabled"];
	        this.mode = source["mode"];
	        this.kind = source["kind"];
	        this.status = source["status"];
	        this.steps = this.convertValues(source["steps"], LLMProposalStep);
	        this.proposal = source["proposal"];
	        this.warnings = source["warnings"];
	        this.needsTest = source["needsTest"];
	        this.needsConfirm = source["needsConfirm"];
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class LLMTestResponse {
	    ok: boolean;
	    endpoint: string;
	    model: string;
	    content?: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LLMTestResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.endpoint = source["endpoint"];
	        this.model = source["model"];
	        this.content = source["content"];
	        this.message = source["message"];
	    }
	}
	export class LandingProxyView {
	    name: string;
	    type: string;
	    server: string;
	    port: number;
	    username?: string;
	    password?: string;
	    tls: boolean;
	    sni?: string;
	    skipCertVerify: boolean;
	    enabled: boolean;
	    inUseBy: string[];
	    routeCount: number;
	
	    static createFrom(source: any = {}) {
	        return new LandingProxyView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.server = source["server"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.tls = source["tls"];
	        this.sni = source["sni"];
	        this.skipCertVerify = source["skipCertVerify"];
	        this.enabled = source["enabled"];
	        this.inUseBy = source["inUseBy"];
	        this.routeCount = source["routeCount"];
	    }
	}
	export class ListenerUser {
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new ListenerUser(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
	export class ListenerView {
	    name: string;
	    type: string;
	    listen: string;
	    port: number;
	    udp: boolean;
	    enabled: boolean;
	    users: ListenerUser[];
	    userCount: number;
	    routeMode: string;
	    egressGroup: string;
	    transitRoute?: string;
	    targetName?: string;
	    currentProxy?: string;
	    routeSummary?: string;
	    status: string;
	    groupMissing: boolean;
	    providerMissing: boolean;
	    providerDisabled: boolean;
	    landingMissing: boolean;
	    landingDisabled: boolean;
	    transitMissing: boolean;
	    transitDisabled: boolean;
	    transitProxyMissing: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ListenerView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.listen = source["listen"];
	        this.port = source["port"];
	        this.udp = source["udp"];
	        this.enabled = source["enabled"];
	        this.users = this.convertValues(source["users"], ListenerUser);
	        this.userCount = source["userCount"];
	        this.routeMode = source["routeMode"];
	        this.egressGroup = source["egressGroup"];
	        this.transitRoute = source["transitRoute"];
	        this.targetName = source["targetName"];
	        this.currentProxy = source["currentProxy"];
	        this.routeSummary = source["routeSummary"];
	        this.status = source["status"];
	        this.groupMissing = source["groupMissing"];
	        this.providerMissing = source["providerMissing"];
	        this.providerDisabled = source["providerDisabled"];
	        this.landingMissing = source["landingMissing"];
	        this.landingDisabled = source["landingDisabled"];
	        this.transitMissing = source["transitMissing"];
	        this.transitDisabled = source["transitDisabled"];
	        this.transitProxyMissing = source["transitProxyMissing"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LocalNodeCheckItem {
	    name: string;
	    ok: boolean;
	    message: string;
	    action?: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalNodeCheckItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.action = source["action"];
	    }
	}
	export class LocalNodeCheckResponse {
	    ok: boolean;
	    name: string;
	    checks: LocalNodeCheckItem[];
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalNodeCheckResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.name = source["name"];
	        this.checks = this.convertValues(source["checks"], LocalNodeCheckItem);
	        this.message = source["message"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LocalNodeConfig {
	    access_mode?: string;
	    listen?: string;
	    port?: number;
	    external_host?: string;
	    external_port?: number;
	    users?: ListenerUser[];
	    route?: string;
	    certificate?: string;
	    private_key?: string;
	    tls?: boolean;
	    sni?: string;
	    skip_cert_verify?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new LocalNodeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.access_mode = source["access_mode"];
	        this.listen = source["listen"];
	        this.port = source["port"];
	        this.external_host = source["external_host"];
	        this.external_port = source["external_port"];
	        this.users = this.convertValues(source["users"], ListenerUser);
	        this.route = source["route"];
	        this.certificate = source["certificate"];
	        this.private_key = source["private_key"];
	        this.tls = source["tls"];
	        this.sni = source["sni"];
	        this.skip_cert_verify = source["skip_cert_verify"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LocalNodeConnection {
	    username?: string;
	    password?: string;
	    address: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new LocalNodeConnection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	        this.address = source["address"];
	        this.url = source["url"];
	    }
	}
	export class LocalNodeConnectionResponse {
	    ok: boolean;
	    name: string;
	    protocol: string;
	    host: string;
	    port: number;
	    connections: LocalNodeConnection[];
	    warnings?: string[];
	
	    static createFrom(source: any = {}) {
	        return new LocalNodeConnectionResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.connections = this.convertValues(source["connections"], LocalNodeConnection);
	        this.warnings = source["warnings"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class LocalProxyUsage {
	    enabled: boolean;
	    type: string;
	    listen: string;
	    port: number;
	    users?: ListenerUser[];
	
	    static createFrom(source: any = {}) {
	        return new LocalProxyUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.type = source["type"];
	        this.listen = source["listen"];
	        this.port = source["port"];
	        this.users = this.convertValues(source["users"], ListenerUser);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class MihomoVersionRecord {
	    version: string;
	    status: string;
	    archivePath?: string;
	    binaryPath?: string;
	    downloadedAt?: string;
	    installedAt?: string;
	    activatedAt?: string;
	    lastError?: string;
	
	    static createFrom(source: any = {}) {
	        return new MihomoVersionRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.status = source["status"];
	        this.archivePath = source["archivePath"];
	        this.binaryPath = source["binaryPath"];
	        this.downloadedAt = source["downloadedAt"];
	        this.installedAt = source["installedAt"];
	        this.activatedAt = source["activatedAt"];
	        this.lastError = source["lastError"];
	    }
	}
	export class MihomoVersionSupport {
	    version: string;
	    recommended: boolean;
	    supported: boolean;
	    compatibility: string;
	    releaseUrl: string;
	    assetUrl?: string;
	    assetName?: string;
	    currentPlatform: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MihomoVersionSupport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.recommended = source["recommended"];
	        this.supported = source["supported"];
	        this.compatibility = source["compatibility"];
	        this.releaseUrl = source["releaseUrl"];
	        this.assetUrl = source["assetUrl"];
	        this.assetName = source["assetName"];
	        this.currentPlatform = source["currentPlatform"];
	    }
	}
	export class MihomoVersionsResponse {
	    platform: Record<string, string>;
	    recommended: string;
	    activeVersion?: string;
	    configuredBinary: string;
	    supportMatrix: MihomoVersionSupport[];
	    installedVersions: MihomoVersionRecord[];
	
	    static createFrom(source: any = {}) {
	        return new MihomoVersionsResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.platform = source["platform"];
	        this.recommended = source["recommended"];
	        this.activeVersion = source["activeVersion"];
	        this.configuredBinary = source["configuredBinary"];
	        this.supportMatrix = this.convertValues(source["supportMatrix"], MihomoVersionSupport);
	        this.installedVersions = this.convertValues(source["installedVersions"], MihomoVersionRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class NodeSourceEndpoint {
	    server?: string;
	    port?: number;
	    username?: string;
	    password?: string;
	    tls?: boolean;
	    sni?: string;
	    skip_cert_verify?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NodeSourceEndpoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.server = source["server"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.password = source["password"];
	        this.tls = source["tls"];
	        this.sni = source["sni"];
	        this.skip_cert_verify = source["skip_cert_verify"];
	    }
	}
	export class NodeSourceSubscription {
	    url?: string;
	    interval?: number;
	    health_check_url?: string;
	    health_check_interval?: number;
	    headers?: Record<string, string>;
	    via?: string;
	
	    static createFrom(source: any = {}) {
	        return new NodeSourceSubscription(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.interval = source["interval"];
	        this.health_check_url = source["health_check_url"];
	        this.health_check_interval = source["health_check_interval"];
	        this.headers = source["headers"];
	        this.via = source["via"];
	    }
	}
	export class NodeSource {
	    name: string;
	    type: string;
	    enabled: boolean;
	    protocol?: string;
	    subscription?: NodeSourceSubscription;
	    endpoint?: NodeSourceEndpoint;
	    axis?: AxisConnectionConfig;
	    local_node?: LocalNodeConfig;
	    notes?: string;
	
	    static createFrom(source: any = {}) {
	        return new NodeSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.enabled = source["enabled"];
	        this.protocol = source["protocol"];
	        this.subscription = this.convertValues(source["subscription"], NodeSourceSubscription);
	        this.endpoint = this.convertValues(source["endpoint"], NodeSourceEndpoint);
	        this.axis = this.convertValues(source["axis"], AxisConnectionConfig);
	        this.local_node = this.convertValues(source["local_node"], LocalNodeConfig);
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class PublicationAuth {
	    username?: string;
	    password?: string;
	    token?: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicationAuth(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.username = source["username"];
	        this.password = source["password"];
	        this.token = source["token"];
	    }
	}
	export class PublicationConfig {
	    name: string;
	    type: string;
	    enabled: boolean;
	    route: string;
	    listen?: string;
	    port?: number;
	    auth: PublicationAuth;
	    accessScope?: string;
	    format?: string;
	
	    static createFrom(source: any = {}) {
	        return new PublicationConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.enabled = source["enabled"];
	        this.route = source["route"];
	        this.listen = source["listen"];
	        this.port = source["port"];
	        this.auth = this.convertValues(source["auth"], PublicationAuth);
	        this.accessScope = source["accessScope"];
	        this.format = source["format"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class RouteHealthCheck {
	    url?: string;
	    interval?: number;
	
	    static createFrom(source: any = {}) {
	        return new RouteHealthCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.interval = source["interval"];
	    }
	}
	export class RouteEndpointRef {
	    source?: string;
	    node?: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteEndpointRef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source = source["source"];
	        this.node = source["node"];
	    }
	}
	export class RouteConfig {
	    name: string;
	    enabled: boolean;
	    entry?: RouteEndpointRef;
	    strategy?: string;
	    landing?: RouteEndpointRef;
	    health_check?: RouteHealthCheck;
	    notes?: string;
	
	    static createFrom(source: any = {}) {
	        return new RouteConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.entry = this.convertValues(source["entry"], RouteEndpointRef);
	        this.strategy = source["strategy"];
	        this.landing = this.convertValues(source["landing"], RouteEndpointRef);
	        this.health_check = this.convertValues(source["health_check"], RouteHealthCheck);
	        this.notes = source["notes"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	export class RuntimeCheck {
	    key: string;
	    title: string;
	    ok: boolean;
	    level: string;
	    summary: string;
	    recommendation?: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimeCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.title = source["title"];
	        this.ok = source["ok"];
	        this.level = source["level"];
	        this.summary = source["summary"];
	        this.recommendation = source["recommendation"];
	    }
	}
	export class RuntimePreflightPaths {
	    configPath: string;
	    runtimeDir: string;
	    providersDir: string;
	    mihomoConfigPath: string;
	    lastGoodConfigPath: string;
	    statePath: string;
	    proxyrelayServicePath: string;
	    mihomoServicePath: string;
	
	    static createFrom(source: any = {}) {
	        return new RuntimePreflightPaths(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.configPath = source["configPath"];
	        this.runtimeDir = source["runtimeDir"];
	        this.providersDir = source["providersDir"];
	        this.mihomoConfigPath = source["mihomoConfigPath"];
	        this.lastGoodConfigPath = source["lastGoodConfigPath"];
	        this.statePath = source["statePath"];
	        this.proxyrelayServicePath = source["proxyrelayServicePath"];
	        this.mihomoServicePath = source["mihomoServicePath"];
	    }
	}
	export class RuntimePreflightSummary {
	    ready: boolean;
	    total: number;
	    passed: number;
	    warnings: number;
	    errors: number;
	    info: number;
	
	    static createFrom(source: any = {}) {
	        return new RuntimePreflightSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.total = source["total"];
	        this.passed = source["passed"];
	        this.warnings = source["warnings"];
	        this.errors = source["errors"];
	        this.info = source["info"];
	    }
	}
	export class RuntimePreflightSnapshot {
	    generatedAt: string;
	    mode: string;
	    status: string;
	    ready: boolean;
	    summary: RuntimePreflightSummary;
	    paths: RuntimePreflightPaths;
	    checks: RuntimeCheck[];
	    recommendations: string[];
	
	    static createFrom(source: any = {}) {
	        return new RuntimePreflightSnapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generatedAt = source["generatedAt"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.ready = source["ready"];
	        this.summary = this.convertValues(source["summary"], RuntimePreflightSummary);
	        this.paths = this.convertValues(source["paths"], RuntimePreflightPaths);
	        this.checks = this.convertValues(source["checks"], RuntimeCheck);
	        this.recommendations = source["recommendations"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	
	export class TransitRouteView {
	    name: string;
	    enabled: boolean;
	    upstreamProvider: string;
	    upstreamProxyName: string;
	    egressGroup: string;
	    notes?: string;
	    currentProxy?: string;
	    candidateCount: number;
	    egressGroupMode?: string;
	    runtimeGroupName?: string;
	    routeSummary?: string;
	    lastTestedAt?: string;
	    lastTestStatus?: string;
	    lastTestMessage?: string;
	    lastTestDelay?: number;
	    lastTestUrl?: string;
	    status: string;
	    providerMissing: boolean;
	    providerDisabled: boolean;
	    transitProxyMissing: boolean;
	    egressGroupMissing: boolean;
	    egressProviderMissing: boolean;
	    egressProviderDisabled: boolean;
	    landingMissing: boolean;
	    landingDisabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TransitRouteView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.enabled = source["enabled"];
	        this.upstreamProvider = source["upstreamProvider"];
	        this.upstreamProxyName = source["upstreamProxyName"];
	        this.egressGroup = source["egressGroup"];
	        this.notes = source["notes"];
	        this.currentProxy = source["currentProxy"];
	        this.candidateCount = source["candidateCount"];
	        this.egressGroupMode = source["egressGroupMode"];
	        this.runtimeGroupName = source["runtimeGroupName"];
	        this.routeSummary = source["routeSummary"];
	        this.lastTestedAt = source["lastTestedAt"];
	        this.lastTestStatus = source["lastTestStatus"];
	        this.lastTestMessage = source["lastTestMessage"];
	        this.lastTestDelay = source["lastTestDelay"];
	        this.lastTestUrl = source["lastTestUrl"];
	        this.status = source["status"];
	        this.providerMissing = source["providerMissing"];
	        this.providerDisabled = source["providerDisabled"];
	        this.transitProxyMissing = source["transitProxyMissing"];
	        this.egressGroupMissing = source["egressGroupMissing"];
	        this.egressProviderMissing = source["egressProviderMissing"];
	        this.egressProviderDisabled = source["egressProviderDisabled"];
	        this.landingMissing = source["landingMissing"];
	        this.landingDisabled = source["landingDisabled"];
	    }
	}
	
	export class VirtualInterfaceUsage {
	    enabled: boolean;
	    mode?: string;
	    status?: string;
	    message?: string;
	
	    static createFrom(source: any = {}) {
	        return new VirtualInterfaceUsage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.mode = source["mode"];
	        this.status = source["status"];
	        this.message = source["message"];
	    }
	}
	export class UsageView {
	    selectedRoute?: string;
	    localProxy: LocalProxyUsage;
	    virtualInterface: VirtualInterfaceUsage;
	    currentRoute?: RouteConfig;
	    ready: boolean;
	    missingSteps: string[];
	    lastAppliedAt?: string;
	    lastApplyStatus?: string;
	    lastApplyMessage?: string;
	
	    static createFrom(source: any = {}) {
	        return new UsageView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.selectedRoute = source["selectedRoute"];
	        this.localProxy = this.convertValues(source["localProxy"], LocalProxyUsage);
	        this.virtualInterface = this.convertValues(source["virtualInterface"], VirtualInterfaceUsage);
	        this.currentRoute = this.convertValues(source["currentRoute"], RouteConfig);
	        this.ready = source["ready"];
	        this.missingSteps = source["missingSteps"];
	        this.lastAppliedAt = source["lastAppliedAt"];
	        this.lastApplyStatus = source["lastApplyStatus"];
	        this.lastApplyMessage = source["lastApplyMessage"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

