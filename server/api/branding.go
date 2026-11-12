package api

import (
	"net/http"
	"path/filepath"
	"strings"
)

// brandingFiles lists the only files served under /branding/.
var brandingFiles = map[string]string{
	"":             "index.html",
	"app-icon.svg": "app-icon.svg",
	"app-icon.png": "app-icon.png",
}

// serveBranding serves the brand guide. It runs after sign-in, so the guide is as private as the app.
func (s *Server) serveBranding(w http.ResponseWriter, r *http.Request) {
	if s.Branding == "" {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path == "/branding" {
		http.Redirect(w, r, "/branding/", http.StatusMovedPermanently)
		return
	}
	file, ok := brandingFiles[strings.TrimPrefix(r.URL.Path, "/branding/")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeFile(w, r, filepath.Join(s.Branding, file))
}
