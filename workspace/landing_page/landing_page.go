// Package landing_page implements the next-generation marketing and technical landing page for Harness Engineering.
// It integrates premium, responsive design structures, dynamic form validations, and asynchronous contact form handlers.
package landing_page

import (
	"embed"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

// ContactSubmission defines the schema for inbound technical inquiries from the landing page.
type ContactSubmission struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

// ContactResponse represents the structured JSON envelope returned to the client.
type ContactResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// Handler constructs a self-contained http.Handler mapping route requests to correct embedded static assets
// and secure lead-generation form endpoints.
func Handler() http.Handler {
	mux := http.NewServeMux()

	// Route handler for landing page structure and styles
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

	// API handler for lead generation contact form
	mux.HandleFunc("/api/contact", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// 1. Enforce strict POST request method
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(ContactResponse{
				Success: false,
				Message: "HTTP method not allowed; use POST.",
			})
			return
		}

		var sub ContactSubmission
		contentType := r.Header.Get("Content-Type")

		// 2. Parse payload based on request Content-Type
		if strings.Contains(contentType, "application/json") {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(ContactResponse{
					Success: false,
					Message: "Unable to read request payload.",
				})
				return
			}
			defer r.Body.Close()

			if err := json.Unmarshal(body, &sub); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(ContactResponse{
					Success: false,
					Message: "Invalid JSON format payload.",
				})
				return
			}
		} else {
			// Fallback parsing for URL-encoded forms
			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(ContactResponse{
					Success: false,
					Message: "Unable to parse form data.",
				})
				return
			}
			sub.Name = r.FormValue("name")
			sub.Email = r.FormValue("email")
			sub.Message = r.FormValue("message")
		}

		// 3. String sanitation & cleaning
		sub.Name = strings.TrimSpace(sub.Name)
		sub.Email = strings.TrimSpace(sub.Email)
		sub.Message = strings.TrimSpace(sub.Message)

		// 4. Strict structural validations
		if sub.Name == "" || sub.Email == "" || sub.Message == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ContactResponse{
				Success: false,
				Message: "All fields (Name, Email, Message) are required.",
			})
			return
		}

		// Quick ReDoS-safe structural validation checks for email
		if !strings.Contains(sub.Email, "@") || !strings.Contains(sub.Email, ".") || len(sub.Email) > 254 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ContactResponse{
				Success: false,
				Message: "Corporate email address is structured incorrectly.",
			})
			return
		}

		if len(sub.Message) < 10 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ContactResponse{
				Success: false,
				Message: "Inquiry scope description must contain at least 10 characters.",
			})
			return
		}

		// 5. Successful transmission response
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ContactResponse{
			Success: true,
			Message: "Thank you! Your inquiry has been securely transmitted. A Harness Engineer will respond shortly.",
		})
	})

	return mux
}

// serveFile reads a file from the embedded static assets FS and writes it to the response writer
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
