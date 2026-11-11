package api

import (
	_ "embed"
	"net/http"
	"strconv"
)

// Public branding only. Private web and API routes still pass through the verifier.
const logoPath = "/brand/minder-icon.png"

//go:embed assets/minder-icon.png
var appIcon []byte

func serveLogo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Length", strconv.Itoa(len(appIcon)))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		w.Write(appIcon)
	}
}
