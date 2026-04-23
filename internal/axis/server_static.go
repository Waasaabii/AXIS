package axis

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	neturl "net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func resolvePublicDir() string {
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
