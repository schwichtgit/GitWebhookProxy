package assets

import (
	"bytes"
	"io/fs"
	"testing"
)

const logo = "gitwebhookproxy-round-100px.png"

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

func TestGetWebAssetData(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{name: "embedded logo", file: logo},
		{name: "missing file", file: "missing.png", wantErr: true},
		{name: "directory traversal", file: "../assets.go", wantErr: true},
		{name: "empty name", file: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			data, err := GetWebAssetData(tt.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetWebAssetData(%q) error = %v, wantErr %v", tt.file, err, tt.wantErr)
			}
			if !tt.wantErr && !bytes.HasPrefix(data, pngSignature) {
				t.Errorf("GetWebAssetData(%q) did not return PNG data", tt.file)
			}
		})
	}
}

func TestGetWebAssets(t *testing.T) {
	t.Parallel()
	webFS := GetWebAssets()
	if webFS == nil {
		t.Fatal("GetWebAssets() returned nil")
	}

	data, err := fs.ReadFile(webFS, logo)
	if err != nil {
		t.Fatalf("reading %s: %v", logo, err)
	}
	direct, err := GetWebAssetData(logo)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, direct) {
		t.Error("GetWebAssets and GetWebAssetData returned different content")
	}

	if _, err := fs.Stat(webFS, "web/"+logo); err == nil {
		t.Error("GetWebAssets should be rooted at the web directory")
	}
}
