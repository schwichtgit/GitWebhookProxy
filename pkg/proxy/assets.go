package proxy

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/stakater/GitWebhookProxy/pkg/assets"
)

// serveAssets serves static assets from the embedded filesystem
func (p *Proxy) serveAssets(w http.ResponseWriter, r *http.Request) {
	// Extract the filename from the URL path
	filename := strings.TrimPrefix(r.URL.Path, "/assets/")

	// Prevent directory traversal attacks
	if strings.Contains(filename, "..") {
		http.Error(w, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Get the file data
	data, err := assets.GetWebAssetData(filename)
	if err != nil {
		log.Printf("Error reading asset file %s: %v", strconv.Quote(filename), err)
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", assetContentType(filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// data comes from the embedded asset filesystem, not from the request.
	if _, err := w.Write(data); err != nil { //nolint:gosec // G705: embedded content
		log.Printf("Error writing response: %v", err)
	}
}

// assetContentType returns the Content-Type for an asset based on its file
// extension.
func assetContentType(filename string) string {
	switch filepath.Ext(filename) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".css":
		return "text/css"
	case ".js":
		return "application/javascript"
	case ".html":
		return "text/html"
	}
	return "application/octet-stream"
}
