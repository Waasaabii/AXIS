package axis

import (
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"
	"testing/fstest"
)

func TestServeStaticFallsBackToEmbeddedIndex(t *testing.T) {
	server := &Server{
		staticFS: fstest.MapFS{
			"index.html":    &fstest.MapFile{Data: []byte("<html>embedded</html>")},
			"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://axis.local/listeners", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "embedded") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestServeStaticServesEmbeddedAsset(t *testing.T) {
	server := &Server{
		staticFS: fstest.MapFS{
			"index.html":    &fstest.MapFile{Data: []byte("<html>embedded</html>")},
			"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://axis.local/assets/app.js", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "console.log") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestServeStaticUsesUIDevProxy(t *testing.T) {
	vite := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("vite-dev:" + r.URL.Path))
	}))
	defer vite.Close()

	target, err := neturl.Parse(vite.URL)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	server := &Server{
		uiDevProxy: newUIDevProxy(target.String()),
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://axis.local/@vite/client", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "vite-dev:/@vite/client") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}
