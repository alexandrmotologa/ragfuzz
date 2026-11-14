package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// DistHandler returns an http.Handler that serves embedded React static assets with SPA fallback.
func DistHandler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		// Fallback empty handler
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "embedded UI assets not available", http.StatusNotFound)
		})
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Check if file exists in embedded filesystem
		f, err := sub.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA Fallback: serve index.html for unknown client routes
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
