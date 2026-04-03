export namespace axis {
	
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

}

