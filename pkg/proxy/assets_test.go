package proxy

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/schwichtgit/GitWebhookProxy/pkg/assets"
)

func TestProxy_serveAssets(t *testing.T) {
	t.Parallel()
	logo, err := assets.GetWebAssetData("gitwebhookproxy-round-100px.png")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name            string
		path            string
		wantStatus      int
		wantContentType string
		wantBody        []byte
	}{
		{
			name:            "embedded png",
			path:            "/assets/gitwebhookproxy-round-100px.png",
			wantStatus:      http.StatusOK,
			wantContentType: "image/png",
			wantBody:        logo,
		},
		{name: "missing file", path: "/assets/missing.png", wantStatus: http.StatusNotFound},
		{name: "directory traversal", path: "/assets/../proxy.go", wantStatus: http.StatusBadRequest},
		{name: "traversal at end of path", path: "/assets/web/..", wantStatus: http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			rr := httptest.NewRecorder()
			(&Proxy{}).serveAssets(rr, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
			if tt.wantBody == nil {
				return
			}
			if got := rr.Header().Get("Content-Type"); got != tt.wantContentType {
				t.Errorf("Content-Type = %q, want %q", got, tt.wantContentType)
			}
			if got := rr.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
			}
			if got := rr.Header().Get("Content-Length"); got != strconv.Itoa(len(tt.wantBody)) {
				t.Errorf("Content-Length = %q, want %d", got, len(tt.wantBody))
			}
			if rr.Body.String() != string(tt.wantBody) {
				t.Error("body does not match the embedded asset")
			}
		})
	}
}

func TestAssetContentType(t *testing.T) {
	t.Parallel()
	tests := map[string]string{
		"logo.png":    "image/png",
		"photo.jpg":   "image/jpeg",
		"photo.jpeg":  "image/jpeg",
		"anim.gif":    "image/gif",
		"site.css":    "text/css",
		"app.js":      "application/javascript",
		"index.html":  "text/html",
		"archive.zip": "application/octet-stream",
		"noext":       "application/octet-stream",
		"LOGO.PNG":    "application/octet-stream",
	}
	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := assetContentType(name); got != want {
				t.Errorf("assetContentType(%q) = %q, want %q", name, got, want)
			}
		})
	}
}
