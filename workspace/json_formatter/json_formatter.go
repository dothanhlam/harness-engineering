// Package json_formatter implements a high-performance JSON formatter, validator, and visualizer.
// It serves a responsive, premium web interface and provides endpoints for asynchronous JSON formatting.
package json_formatter

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

// FormatRequest defines the payload for formatting JSON via the API.
type FormatRequest struct {
	JSONData  string `json:"jsonData"`
	Indent    int    `json:"indent"` // Number of spaces (0 for tab, -1 for minify)
	Minify    bool   `json:"minify"`
}

// FormatResponse represents the structured JSON response returned by the formatter API.
type FormatResponse struct {
	Success      bool           `json:"success"`
	Formatted    string         `json:"formatted,omitempty"`
	Stats        *JSONStats     `json:"stats,omitempty"`
	ErrorMessage string         `json:"errorMessage,omitempty"`
	ErrorOffset  int64          `json:"errorOffset,omitempty"`
	ErrorLine    int            `json:"errorLine,omitempty"`
	ErrorCol     int            `json:"errorCol,omitempty"`
}

// JSONStats represents structural metrics computed from the JSON document.
type JSONStats struct {
	Size         int  `json:"size"`
	MaxDepth     int  `json:"maxDepth"`
	KeyCount     int  `json:"keyCount"`
	ArrayCount   int  `json:"arrayCount"`
	ObjectCount  int  `json:"objectCount"`
}

// Handler returns a self-contained http.Handler mapping requests to the embedded static web UI
// and the JSON processing API endpoints.
func Handler() http.Handler {
	mux := http.NewServeMux()

	// Static files handler (Index, CSS, JS)
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

	// JSON Formatter API Handler
	mux.HandleFunc("/api/format", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			_ = json.NewEncoder(w).Encode(FormatResponse{
				Success:      false,
				ErrorMessage: "HTTP method not allowed; use POST.",
			})
			return
		}

		var req FormatRequest
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(FormatResponse{
				Success:      false,
				ErrorMessage: "Failed to read request body.",
			})
			return
		}
		defer r.Body.Close()

		if err := json.Unmarshal(body, &req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(FormatResponse{
				Success:      false,
				ErrorMessage: "Invalid request payload. Ensure request body is JSON.",
			})
			return
		}

		inputJSON := strings.TrimSpace(req.JSONData)
		if inputJSON == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(FormatResponse{
				Success:      false,
				ErrorMessage: "JSON data cannot be empty.",
			})
			return
		}

		// Validate JSON and compute structural statistics
		var temp interface{}
		dec := json.NewDecoder(strings.NewReader(inputJSON))
		if err := dec.Decode(&temp); err != nil {
			line, col, offset := locateSyntaxError(inputJSON, err)
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(FormatResponse{
				Success:      false,
				ErrorMessage: err.Error(),
				ErrorOffset:  offset,
				ErrorLine:    line,
				ErrorCol:     col,
			})
			return
		}

		// Compute metrics
		stats := computeJSONStats(temp, len(inputJSON))

		// Process Formatting / Minification
		var formattedData []byte
		if req.Minify {
			var compact bytes.Buffer
			if err := json.Compact(&compact, []byte(inputJSON)); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(FormatResponse{
					Success:      false,
					ErrorMessage: fmt.Sprintf("Failed to minify: %v", err),
				})
				return
			}
			formattedData = compact.Bytes()
		} else {
			indentPrefix := ""
			indentStr := "  " // Default 2 spaces
			if req.Indent == 0 {
				indentStr = "\t"
			} else if req.Indent > 0 {
				indentStr = strings.Repeat(" ", req.Indent)
			}

			var pretty bytes.Buffer
			if err := json.Indent(&pretty, []byte(inputJSON), indentPrefix, indentStr); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(FormatResponse{
					Success:      false,
					ErrorMessage: fmt.Sprintf("Failed to format: %v", err),
				})
				return
			}
			formattedData = pretty.Bytes()
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(FormatResponse{
			Success:   true,
			Formatted: string(formattedData),
			Stats:     stats,
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

// locateSyntaxError returns line number, column number, and character offset for a syntax error
func locateSyntaxError(input string, err error) (int, int, int64) {
	syntaxErr, ok := err.(*json.SyntaxError)
	if !ok {
		return 1, 1, 0
	}

	offset := syntaxErr.Offset
	if offset < 0 {
		offset = 0
	}
	if offset > int64(len(input)) {
		offset = int64(len(input))
	}

	line := 1
	col := 1
	for i := int64(0); i < offset; i++ {
		if input[i] == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return line, col, offset
}

// computeJSONStats recursively analyzes raw interface value to compute JSON stats
func computeJSONStats(val interface{}, rawSize int) *JSONStats {
	stats := &JSONStats{Size: rawSize}
	stats.MaxDepth = getDepth(val)
	stats.KeyCount, stats.ArrayCount, stats.ObjectCount = traverse(val)
	return stats
}

func getDepth(val interface{}) int {
	switch v := val.(type) {
	case map[string]interface{}:
		maxChildDepth := 0
		for _, child := range v {
			d := getDepth(child)
			if d > maxChildDepth {
				maxChildDepth = d
			}
		}
		return maxChildDepth + 1
	case []interface{}:
		maxChildDepth := 0
		for _, child := range v {
			d := getDepth(child)
			if d > maxChildDepth {
				maxChildDepth = d
			}
		}
		return maxChildDepth + 1
	default:
		return 1
	}
}

func traverse(val interface{}) (keys, arrays, objects int) {
	switch v := val.(type) {
	case map[string]interface{}:
		objects = 1
		for _, child := range v {
			keys++
			k, a, o := traverse(child)
			keys += k
			arrays += a
			objects += o
		}
	case []interface{}:
		arrays = 1
		for _, child := range v {
			k, a, o := traverse(child)
			keys += k
			arrays += a
			objects += o
		}
	}
	return
}
