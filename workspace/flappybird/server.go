// Package flappybird implements a complete, thread-safe Flappy Bird game engine.
// It includes physics simulation, state management via a Finite State Machine (FSM),
// procedural pipe generation, collision detection (AABB), scoring logic, high score persistence,
// and audio event triggers.
package flappybird

import (
	"embed"
	"io"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

// Handler returns an http.Handler that maps routes to serve the embedded cyberpunk game client.
func Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "/index.html" {
			serveFile(w, "static/index.html", "text/html; charset=utf-8")
			return
		}
		if path == "/style.css" {
			serveFile(w, "static/style.css", "text/css; charset=utf-8")
			return
		}
		if path == "/app.js" {
			serveFile(w, "static/app.js", "application/javascript; charset=utf-8")
			return
		}
		http.NotFound(w, r)
	})

	return mux
}

// serveFile reads a file from the embedded static assets FS and writes it to the response.
func serveFile(w http.ResponseWriter, filePath string, contentType string) {
	file, err := staticFS.Open(filePath)
	if err != nil {
		http.Error(w, "Requested static resource was not found.", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	
	_, _ = io.Copy(w, file)
}
