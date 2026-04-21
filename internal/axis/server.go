package axis

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httputil"
	neturl "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	frontendassets "github.com/Waasaabii/AXIS/frontend"
)

type Server struct {
	service      *Service
	engine       *EngineFacade
	host         *HostFacade
	dynamicProxy *DynamicProxyService
	uiDevProxy   *httputil.ReverseProxy
	staticFS     fs.FS
	publicDir    string
}

func NewServer(service *Service) *Server {
	publicDir := resolvePublicDir(service)
	staticFS, _, err := frontendassets.StaticFS()
	if err != nil {
		staticFS = nil
	}
	return &Server{
		service:      service,
		engine:       NewEngineFacade(service),
		host:         NewHostFacade(service),
		dynamicProxy: NewDynamicProxyService(service),
		uiDevProxy:   newUIDevProxy(os.Getenv("AXIS_UI_DEV_URL")),
		staticFS:     staticFS,
		publicDir:    publicDir,
	}
}

func resolvePublicDir(service *Service) string {
	if envPublicDir := os.Getenv("AXIS_PUBLIC_DIR"); envPublicDir != "" {
		return envPublicDir
	}

	candidates := []string{}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "frontend", "dist"))
	}
	if executable, err := os.Executable(); err == nil {
		executableRoot := filepath.Dir(filepath.Dir(executable))
		candidates = append(candidates, filepath.Join(executableRoot, "frontend", "dist"))
	}
	if service != nil && service.layout != nil && service.layout.RootDir != "" {
		candidates = append(candidates, filepath.Join(filepath.Dir(service.layout.RootDir), "frontend", "dist"))
	}

	for _, candidate := range candidates {
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
	}

	if len(candidates) > 0 {
		return candidates[0]
	}
	return filepath.Join("frontend", "dist")
}

func newUIDevProxy(rawURL string) *httputil.ReverseProxy {
	value := strings.TrimSpace(rawURL)
	if value == "" {
		return nil
	}
	target, err := neturl.Parse(value)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) {
		originalDirector(r)
		r.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, _ *http.Request, proxyErr error) {
		http.Error(w, fmt.Sprintf("前端开发服务器不可用: %v", proxyErr), http.StatusBadGateway)
	}
	return proxy
}

func normalizeStaticTarget(urlPath string) string {
	target := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if target == "" || target == "." {
		return "index.html"
	}
	return target
}

func cloneRequestWithPath(r *http.Request, pathname string) *http.Request {
	clone := r.Clone(r.Context())
	if r.URL != nil {
		urlCopy := *r.URL
		clone.URL = &urlCopy
		clone.URL.Path = pathname
	}
	clone.RequestURI = pathname
	return clone
}

func staticPathExists(root fs.FS, target string) bool {
	info, err := fs.Stat(root, target)
	return err == nil && !info.IsDir()
}

func readJSONBody(r *http.Request) (map[string]any, error) {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeText(w http.ResponseWriter, status int, contentType, payload string) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, _ = io.WriteString(w, payload)
}

func (s *Server) sessionFromRequest(r *http.Request) (bool, string) {
	if s == nil || s.service == nil || s.service.auth == nil {
		return false, ""
	}
	cookies := ParseCookies(r.Header.Get("Cookie"))
	return s.service.auth.ReadSession(cookies["proxyrelay_session"])
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathname := r.URL.Path
	method := r.Method
	if pathname == "/api/openapi.json" && method == http.MethodGet {
		writeJSON(w, 200, BuildOpenAPISpec())
		return
	}
	if pathname == "/api/health" && method == http.MethodGet {
		writeJSON(w, 200, s.dynamicProxy.GetHealth(r))
		return
	}
	if pathname == "/api/proxy/next" && method == http.MethodGet {
		result := s.dynamicProxy.GetNextProxy(r)
		if result.RequestID != "" {
			w.Header().Set("X-Request-Id", result.RequestID)
		}
		if !result.OK {
			writeJSON(w, result.Status, map[string]any{
				"code":       result.Code,
				"message":    firstNonEmpty(result.Message, "internal server error"),
				"request_id": firstNonEmpty(result.RequestID, "unknown"),
			})
			return
		}
		if result.Format == "json" {
			writeJSON(w, result.Status, result.Body)
			return
		}
		writeText(w, result.Status, firstNonEmpty(result.ContentType, "text/plain; charset=utf-8"), fmt.Sprint(result.Body))
		return
	}

	validSession, username := s.sessionFromRequest(r)
	if pathname == "/api/bootstrap/status" && method == http.MethodGet {
		writeJSON(w, 200, s.service.GetBootstrapStatus(validSession, username))
		return
	}
	if pathname == "/api/session" && method == http.MethodGet {
		writeJSON(w, 200, s.service.SessionStatus(validSession, username))
		return
	}
	if pathname == "/api/session" && method == http.MethodPost {
		payload, err := readJSONBody(r)
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		if s.service.GetSetupState().NeedsPasswordReset {
			writeJSON(w, 409, map[string]any{"ok": false, "error": "首次初始化还没完成，请先创建管理员账号和密码"})
			return
		}
		response, status := s.service.Login(stringValue(payload["username"]), stringValue(payload["password"]))
		if status == 200 {
			cookie, err := s.service.auth.CreateSessionCookie(stringValue(payload["username"]))
			if err == nil {
				writeSetCookie(w, cookie)
			}
		}
		writeJSON(w, status, response)
		return
	}
	if pathname == "/api/session" && method == http.MethodDelete {
		writeSetCookie(w, s.service.auth.ClearSessionCookie())
		writeJSON(w, 200, map[string]any{"ok": true})
		return
	}
	if pathname == "/api/session/password" && method == http.MethodPut {
		if !validSession {
			writeJSON(w, 401, map[string]any{"ok": false, "error": "未登录或会话已过期"})
			return
		}
		payload, err := readJSONBody(r)
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		response, status := s.service.UpdatePassword(stringValue(payload["password"]))
		writeJSON(w, status, response)
		return
	}
	if pathname == "/api/setup-state" && method == http.MethodGet {
		writeJSON(w, 200, s.service.GetSetupState())
		return
	}
	if pathname == "/api/setup/admin" && method == http.MethodPut {
		payload, err := readJSONBody(r)
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		response, status := s.service.BootstrapAdmin(stringValue(payload["username"]), stringValue(payload["password"]))
		writeJSON(w, status, response)
		return
	}

	if strings.HasPrefix(pathname, "/api/") {
		if !validSession {
			writeJSON(w, 401, map[string]any{"ok": false, "error": "未登录或会话已过期"})
			return
		}

		switch {
		case pathname == "/api/status" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetStatus())
			return
		case pathname == "/api/host/status" && method == http.MethodGet:
			status, err := s.host.GetStatus()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, status)
			return
		case pathname == "/api/host/updater" && method == http.MethodGet:
			status, err := s.host.GetUpdaterStatus()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, status)
			return
		case pathname == "/api/host/updater/check" && method == http.MethodPost:
			response, err := s.host.CheckForUpdates()
			if err != nil {
				writeJSON(w, 400, response)
				return
			}
			writeJSON(w, 200, response)
			return
		case pathname == "/api/host/open-browser" && method == http.MethodPost:
			response, err := s.host.OpenControlCenter(firstNonEmpty(stringValue(r.URL.Query().Get("url")), ""))
			if err != nil {
				writeJSON(w, 400, response)
				return
			}
			writeJSON(w, 200, response)
			return
		case pathname == "/api/host/autostart" && method == http.MethodPut:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, setErr := s.host.SetAutostart(boolValue(payload["enabled"], false))
			if setErr != nil {
				writeJSON(w, 400, response)
				return
			}
			writeJSON(w, 200, response)
			return
		case pathname == "/api/config" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetConfig())
			return
		case pathname == "/api/config" && method == http.MethodPut:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			configPayload := payload
			if nested, ok := payload["config"].(map[string]any); ok {
				configPayload = nested
			}
			config, err := decodeIntoConfig(configPayload)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.SaveConfig(config)
			writeJSON(w, status, response)
			return
		case pathname == "/api/providers" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetProviders())
			return
		case pathname == "/api/groups" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetGroups())
			return
		case pathname == "/api/transit-routes" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetTransitRoutes())
			return
		case pathname == "/api/landing-proxies" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetLandingProxies())
			return
		case pathname == "/api/listeners" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetListeners())
			return
		case pathname == "/api/events" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetEvents())
			return
		case pathname == "/api/rendered-config" && method == http.MethodGet:
			response, err := s.service.GetRenderedConfig()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, response)
			return
		case pathname == "/api/subscription-test" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.TestSubscription(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/controller" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetControllerStatus())
			return
		case pathname == "/api/runtime-preflight" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetRuntimePreflight())
			return
		case pathname == "/api/controller/probe" && method == http.MethodPost:
			writeJSON(w, 200, s.service.ProbeController())
			return
		case pathname == "/api/reload" && method == http.MethodPost:
			writeJSON(w, 200, s.service.ReloadConfig())
			return
		case pathname == "/api/subscriptions" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.AddSubscription(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/egress-groups" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.AddEgressGroup(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/transit-routes" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.AddTransitRoute(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/landing-proxies" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.AddLandingProxy(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/listeners" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.AddListener(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/mihomo/versions" && method == http.MethodGet:
			response, err := s.service.MihomoVersions()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, response)
			return
		case pathname == "/api/mihomo/version-state" && method == http.MethodGet:
			response, err := s.service.MihomoVersionState()
			if err != nil {
				writeJSON(w, 500, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, 200, response)
			return
		}

		if strings.HasPrefix(pathname, "/api/providers/") && strings.HasSuffix(pathname, "/refresh") && method == http.MethodPost {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/providers/"), "/refresh")
			writeJSON(w, 200, s.service.RefreshProvider(name))
			return
		}
		if strings.HasPrefix(pathname, "/api/groups/") && strings.HasSuffix(pathname, "/select") && method == http.MethodPost {
			groupName := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/groups/"), "/select")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.SelectGroup(groupName, stringValue(payload["proxyName"]))
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/groups/") && strings.HasSuffix(pathname, "/healthcheck") && method == http.MethodPost {
			groupName := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/groups/"), "/healthcheck")
			response, status := s.service.RunHealthcheck(groupName)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/transit-routes/") && strings.HasSuffix(pathname, "/healthcheck") && method == http.MethodPost {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/transit-routes/"), "/healthcheck")
			response, status := s.service.RunTransitRouteHealthcheck(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/subscriptions/") && strings.HasSuffix(pathname, "/toggle") && method == http.MethodPost {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/subscriptions/"), "/toggle")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.ToggleSubscription(name, boolValue(payload["enabled"], false))
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/subscriptions/") && strings.HasSuffix(pathname, "/update") && method == http.MethodPut {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/subscriptions/"), "/update")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdateSubscription(name, payload)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/subscriptions/") && method == http.MethodDelete {
			name := strings.TrimPrefix(pathname, "/api/subscriptions/")
			response, status := s.service.RemoveSubscription(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/egress-groups/") && strings.HasSuffix(pathname, "/update") && method == http.MethodPut {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/egress-groups/"), "/update")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdateEgressGroup(name, payload)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/transit-routes/") && strings.HasSuffix(pathname, "/update") && method == http.MethodPut {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/transit-routes/"), "/update")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdateTransitRoute(name, payload)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/egress-groups/") && method == http.MethodDelete {
			name := strings.TrimPrefix(pathname, "/api/egress-groups/")
			response, status := s.service.RemoveEgressGroup(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/transit-routes/") && method == http.MethodDelete {
			name := strings.TrimPrefix(pathname, "/api/transit-routes/")
			response, status := s.service.RemoveTransitRoute(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/landing-proxies/") && strings.HasSuffix(pathname, "/update") && method == http.MethodPut {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/landing-proxies/"), "/update")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdateLandingProxy(name, payload)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/landing-proxies/") && method == http.MethodDelete {
			name := strings.TrimPrefix(pathname, "/api/landing-proxies/")
			response, status := s.service.RemoveLandingProxy(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/listeners/") && strings.HasSuffix(pathname, "/update") && method == http.MethodPut {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/listeners/"), "/update")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdateListener(name, payload)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/listeners/") && method == http.MethodDelete {
			name := strings.TrimPrefix(pathname, "/api/listeners/")
			response, status := s.service.RemoveListener(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/mihomo/versions/") && strings.HasSuffix(pathname, "/download") && method == http.MethodPost {
			version := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/mihomo/versions/"), "/download")
			response, status := s.service.DownloadMihomoVersion(version)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/mihomo/versions/") && strings.HasSuffix(pathname, "/install") && method == http.MethodPost {
			version := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/mihomo/versions/"), "/install")
			response, status := s.service.InstallMihomoVersion(version)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/mihomo/versions/") && strings.HasSuffix(pathname, "/activate") && method == http.MethodPost {
			version := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/mihomo/versions/"), "/activate")
			response, status := s.service.ActivateMihomoVersion(version)
			writeJSON(w, status, response)
			return
		}

		writeJSON(w, 404, map[string]any{"ok": false, "error": "请求的接口不存在"})
		return
	}

	s.serveStatic(w, r)
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if s.uiDevProxy != nil {
		s.uiDevProxy.ServeHTTP(w, r)
		return
	}

	target := normalizeStaticTarget(r.URL.Path)
	if s.staticFS != nil {
		fileServer := http.FileServerFS(s.staticFS)
		if staticPathExists(s.staticFS, target) {
			servePath := "/" + target
			if target == "index.html" {
				servePath = "/"
			}
			fileServer.ServeHTTP(w, cloneRequestWithPath(r, servePath))
			return
		}
		if staticPathExists(s.staticFS, "index.html") {
			fileServer.ServeHTTP(w, cloneRequestWithPath(r, "/"))
			return
		}
	}

	if stat, err := os.Stat(s.publicDir); err != nil || !stat.IsDir() {
		http.Error(w, "前端资源不可用，请先执行 pnpm build:ui 或启动 pnpm dev", http.StatusServiceUnavailable)
		return
	}

	filePath := filepath.Join(s.publicDir, target)
	if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
		http.ServeFile(w, r, filePath)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.publicDir, "index.html"))
}
