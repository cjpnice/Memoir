package httpapi

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"memoir/internal/media"
)

func TestCORSMiddlewareDefaultsToWildcardForLocalDevelopment(t *testing.T) {
	handler := newTestHandler(t, ServerOptions{})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "http://localhost:3000")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected wildcard CORS origin, got %q", got)
	}
}

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	const origin = "https://memoir.example.com"
	handler := newTestHandler(t, ServerOptions{AllowedOrigins: []string{origin}})

	request := httptest.NewRequest(http.MethodOptions, "/api/v1/projects", nil)
	request.Header.Set("Origin", origin)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != origin {
		t.Fatalf("expected configured CORS origin %q, got %q", origin, got)
	}
	if !strings.Contains(response.Header().Get("Vary"), "Origin") {
		t.Fatalf("expected Vary header to include Origin, got %q", response.Header().Get("Vary"))
	}
}

func TestCORSMiddlewareRejectsUnknownOriginWhenConfigured(t *testing.T) {
	handler := newTestHandler(t, ServerOptions{AllowedOrigins: []string{"https://memoir.example.com"}})

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	request.Header.Set("Origin", "https://not-allowed.example.com")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", response.Code)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no CORS allow origin for rejected request, got %q", got)
	}
}

func TestUploadImagesRejectsTooManyFiles(t *testing.T) {
	handler := newTestHandler(t, ServerOptions{MaxUploadFiles: 1})

	request := newMultipartRequest(t, "/api/v1/projects/prj_1/images",
		map[string][]byte{
			"a.jpg": smallJPEGFixture(t),
			"b.jpg": smallJPEGFixture(t),
		},
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for too many files, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "一次最多导入 1 张照片") {
		t.Fatalf("unexpected error body: %s", response.Body.String())
	}
}

func TestUploadImagesRejectsOversizedBody(t *testing.T) {
	handler := newTestHandler(t, ServerOptions{MaxUploadBytes: 1024})

	request := newMultipartRequest(t, "/api/v1/projects/prj_1/images",
		map[string][]byte{
			"a.jpg": bytes.Repeat([]byte("x"), 2048),
		},
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 for oversized upload, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "上传内容超过") {
		t.Fatalf("unexpected error body: %s", response.Body.String())
	}
}

func TestEmbeddedWebAssetsServeIndexAndStaticFiles(t *testing.T) {
	handler := newTestHandler(t, ServerOptions{
		WebAssets: fstest.MapFS{
			"index.html":               {Data: []byte("<html>Memoir UI</html>")},
			"_next/static/app/main.js": {Data: []byte("console.log('memoir')")},
		},
	})

	for _, target := range []string{"/", "/index.html", "/projects/prj_1"} {
		request := httptest.NewRequest(http.MethodGet, target, nil)
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("expected status 200 for %s, got %d", target, response.Code)
		}
		if !strings.Contains(response.Body.String(), "Memoir UI") {
			t.Fatalf("expected index body for %s, got %q", target, response.Body.String())
		}
	}

	request := httptest.NewRequest(http.MethodGet, "/_next/static/app/main.js", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200 for static asset, got %d", response.Code)
	}
	if got := response.Body.String(); got != "console.log('memoir')" {
		t.Fatalf("unexpected static asset body: %q", got)
	}
}

func TestEmbeddedWebAssetsDoNotHandleMissingAPIRoutes(t *testing.T) {
	handler := newTestHandler(t, ServerOptions{
		WebAssets: fstest.MapFS{
			"index.html": {Data: []byte("<html>Memoir UI</html>")},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/missing", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for missing API route, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), "Memoir UI") {
		t.Fatalf("missing API route should not fall back to web UI: %q", response.Body.String())
	}
}

func TestEmbeddedWebAssetsReportMissingIndex(t *testing.T) {
	handler := newTestHandler(t, ServerOptions{
		WebAssets: fstest.MapFS{
			"_next/static/app/main.js": {Data: []byte("console.log('memoir')")},
		},
	})

	request := httptest.NewRequest(http.MethodGet, "/workspace", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 for missing embedded index, got %d", response.Code)
	}
	if !strings.Contains(response.Body.String(), "missing index.html") {
		t.Fatalf("expected missing index hint, got %q", response.Body.String())
	}
}

func newTestHandler(t *testing.T, options ServerOptions) http.Handler {
	t.Helper()
	manager, err := media.NewManager(t.TempDir())
	if err != nil {
		t.Fatalf("new media manager: %v", err)
	}
	return NewServer(nil, manager, options).Handler()
}

func newMultipartRequest(t *testing.T, target string, files map[string][]byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, data := range files {
		part, err := writer.CreateFormFile("files", name)
		if err != nil {
			t.Fatalf("create form file: %v", err)
		}
		if _, err := part.Write(data); err != nil {
			t.Fatalf("write form file: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func smallJPEGFixture(t *testing.T) []byte {
	t.Helper()
	// A minimal JPEG byte slice that is large enough for multipart handling tests.
	return []byte{
		0xff, 0xd8, 0xff, 0xdb, 0x00, 0x43, 0x00, 0x08, 0x06, 0x06, 0x07, 0x06,
		0x05, 0x08, 0x07, 0x07, 0x07, 0x09, 0x09, 0x08, 0x0a, 0x0c, 0x14, 0x0d,
		0x0c, 0x0b, 0x0b, 0x0c, 0x19, 0x12, 0x13, 0x0f, 0x14, 0x1d, 0x19, 0x1f,
		0x1e, 0x1d, 0x19, 0x1c, 0x1c, 0x20, 0x24, 0x2e, 0x27, 0x20, 0x22, 0x2c,
		0x23, 0x1c, 0x1c, 0x28, 0x37, 0x29, 0x2c, 0x30, 0x31, 0x34, 0x34, 0x34,
		0x1f, 0x27, 0x39, 0x3d, 0x38, 0x32, 0x3c, 0x2e, 0x33, 0x34, 0x32, 0xff,
		0xc0, 0x00, 0x11, 0x08, 0x00, 0x01, 0x00, 0x01, 0x03, 0x01, 0x11, 0x00,
		0x02, 0x11, 0x01, 0x03, 0x11, 0x01, 0xff, 0xc4, 0x00, 0x14, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0xff, 0xc4, 0x00, 0x14, 0x10, 0x01, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0xff, 0xda, 0x00, 0x08, 0x01, 0x01, 0x00, 0x00, 0x3f, 0x00,
		0xff, 0xd9,
	}
}
