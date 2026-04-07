package http_server

import (
	"net/http"
	"os"
	"path/filepath"
	"roboserver/shared"

	"github.com/go-chi/chi/v5"
)

func (h *HTTPServer_t) PluginRoutes(r chi.Router) {
	// Serve compiled handler frontend assets from handlers/{type}/dist/
	basePath := shared.AppConfig.Handlers.BasePath
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		shared.DebugPrint("Failed to resolve handlers base path: %v", err)
		return
	}

	// List available handler types
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		entries, err := os.ReadDir(absBase)
		if err != nil {
			http.Error(w, "Failed to list plugins", http.StatusInternalServerError)
			return
		}
		types := make([]string, 0)
		for _, e := range entries {
			if e.IsDir() && e.Name() != "_template" {
				distPath := filepath.Join(absBase, e.Name(), "dist")
				if _, err := os.Stat(distPath); err == nil {
					types = append(types, e.Name())
				}
			}
		}
		sendResponseAsJSON(w, types, http.StatusOK)
	})

	// Serve static assets: /plugins/{type}/{file}
	// Maps to: handlers/{type}/dist/{file}
	r.Get("/{type}/*", func(w http.ResponseWriter, r *http.Request) {
		robotType := chi.URLParam(r, "type")
		// Get the wildcard path after /{type}/
		filePath := chi.URLParam(r, "*")
		if filePath == "" {
			filePath = "index.js"
		}

		fullPath := filepath.Join(absBase, robotType, "dist", filePath)
		fullPath = filepath.Clean(fullPath)

		// Security: ensure we're still within the handlers directory
		if !isSubpath(absBase, fullPath) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// Set appropriate content types for JS modules
		if filepath.Ext(fullPath) == ".js" {
			w.Header().Set("Content-Type", "application/javascript")
		}
		w.Header().Set("Cache-Control", "public, max-age=3600")

		http.ServeFile(w, r, fullPath)
	})
}

// isSubpath checks if child is under parent directory, resolving symlinks.
func isSubpath(parent, child string) bool {
	// Resolve symlinks to prevent traversal via symlinked paths
	realParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return false
	}
	realChild, err := filepath.EvalSymlinks(child)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(realParent, realChild)
	if err != nil {
		return false
	}
	return len(rel) > 0 && rel[0] != '.'
}
