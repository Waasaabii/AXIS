package axis

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"

	frontendassets "github.com/Waasaabii/AXIS/frontend"
)

type Server struct {
	service      *Service
	dynamicProxy *DynamicProxyService
	uiDevProxy   *httputil.ReverseProxy
	staticFS     fs.FS
	publicDir    string
}

func NewServer(service *Service) *Server {
	publicDir := resolvePublicDir()
	staticFS, _, err := frontendassets.StaticFS()
	if err != nil {
		staticFS = nil
	}
	return &Server{
		service:      service,
		dynamicProxy: NewDynamicProxyService(service),
		uiDevProxy:   newUIDevProxy(os.Getenv("AXIS_UI_DEV_URL")),
		staticFS:     staticFS,
		publicDir:    publicDir,
	}
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
		if s.dynamicProxy == nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": "dynamic proxy 未初始化"})
			return
		}
		writeJSON(w, 200, s.dynamicProxy.GetHealth(r))
		return
	}
	if pathname == "/api/proxy/next" && method == http.MethodGet {
		if s.dynamicProxy == nil {
			writeJSON(w, 500, map[string]any{"ok": false, "error": "dynamic proxy 未初始化"})
			return
		}
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

	if strings.HasPrefix(pathname, "/api/") && (s == nil || s.service == nil) {
		writeJSON(w, 500, map[string]any{"ok": false, "error": "service 未初始化"})
		return
	}
	if pathname == "/sub" && method == http.MethodGet {
		result, err := s.service.BuildPublishedSubscription(r.URL.Query().Get("token"))
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", result.Name+".yaml"))
		_, _ = w.Write([]byte(result.Content))
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
		case pathname == "/api/node-sources" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetNodeSources())
			return
		case pathname == "/api/node-sources" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.AddNodeSource(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/routes" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetRoutes())
			return
		case pathname == "/api/routes" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.AddRoute(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/usage" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetUsage())
			return
		case pathname == "/api/usage" && method == http.MethodPut:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdateUsage(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/publications" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetPublications())
			return
		case pathname == "/api/publications" && method == http.MethodPost:
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.AddPublication(payload)
			writeJSON(w, status, response)
			return
		case pathname == "/api/host/status" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetHostStatus())
			return
		case pathname == "/api/host/updater" && method == http.MethodGet:
			writeJSON(w, 200, s.service.GetUpdaterStatus())
			return
		case pathname == "/api/host/updater/check" && method == http.MethodPost:
			response, err := s.service.CheckForUpdates()
			if err != nil {
				writeJSON(w, 400, response)
				return
			}
			writeJSON(w, 200, response)
			return
		case pathname == "/api/host/open-browser" && method == http.MethodPost:
			response, err := s.service.OpenHostControlCenter(firstNonEmpty(stringValue(r.URL.Query().Get("url")), ""))
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
			response, setErr := s.service.SetHostAutostart(boolValue(payload["enabled"], false))
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
			config, err := DecodeConfigPayload(payload)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.SaveConfigFromAPI(config)
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
		if strings.HasPrefix(pathname, "/api/node-sources/") && strings.HasSuffix(pathname, "/update") && method == http.MethodPut {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/node-sources/"), "/update")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdateNodeSource(name, payload)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/node-sources/") && strings.HasSuffix(pathname, "/check") && method == http.MethodGet {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/node-sources/"), "/check")
			response, status := s.service.CheckLocalNode(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/node-sources/") && strings.HasSuffix(pathname, "/connection") && method == http.MethodGet {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/node-sources/"), "/connection")
			response, status := s.service.GetLocalNodeConnection(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/node-sources/") && strings.HasSuffix(pathname, "/baota-config") && method == http.MethodGet {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/node-sources/"), "/baota-config")
			response, status := s.service.GetLocalNodeBaotaConfig(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/node-sources/") && method == http.MethodDelete {
			name := strings.TrimPrefix(pathname, "/api/node-sources/")
			response, status := s.service.RemoveNodeSource(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/routes/") && strings.HasSuffix(pathname, "/update") && method == http.MethodPut {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/routes/"), "/update")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdateRoute(name, payload)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/routes/") && method == http.MethodDelete {
			name := strings.TrimPrefix(pathname, "/api/routes/")
			response, status := s.service.RemoveRoute(name)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/publications/") && strings.HasSuffix(pathname, "/update") && method == http.MethodPut {
			name := strings.TrimSuffix(strings.TrimPrefix(pathname, "/api/publications/"), "/update")
			payload, err := readJSONBody(r)
			if err != nil {
				writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			response, status := s.service.UpdatePublication(name, payload)
			writeJSON(w, status, response)
			return
		}
		if strings.HasPrefix(pathname, "/api/publications/") && method == http.MethodDelete {
			name := strings.TrimPrefix(pathname, "/api/publications/")
			response, status := s.service.RemovePublication(name)
			writeJSON(w, status, response)
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
