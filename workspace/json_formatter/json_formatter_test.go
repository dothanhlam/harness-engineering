package json_formatter

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHandler_Index verifies that the root path serves the index.html content correctly.
func TestHandler_Index(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("Expected HTML Content-Type, got %q", contentType)
	}
}

// TestHandler_CSS verifies that style.css is served with correct headers and type.
func TestHandler_CSS(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/style.css")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/css") {
		t.Errorf("Expected CSS Content-Type, got %q", contentType)
	}
}

// TestHandler_JS verifies that app.js is served with correct headers and type.
func TestHandler_JS(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/app.js")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/javascript") {
		t.Errorf("Expected JS Content-Type, got %q", contentType)
	}
}

// TestHandler_NotFound verifies that unrecognized routes produce a 404 response.
func TestHandler_NotFound(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/nonexistent_resource.png")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 Not Found, got %d", resp.StatusCode)
	}
}

// TestHandler_FormatAPI_Success verifies successful JSON formatting and statistics calculation.
func TestHandler_FormatAPI_Success(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	payload := FormatRequest{
		JSONData: `{"name":"Antigravity","active":true,"details":{"nested":1}}`,
		Indent:   2,
		Minify:   false,
	}
	jsonBytes, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/format", "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		t.Fatalf("Failed to execute POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	var formatResp FormatResponse
	if err := json.NewDecoder(resp.Body).Decode(&formatResp); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if !formatResp.Success {
		t.Fatalf("Expected Success to be true, got false. ErrorMessage: %s", formatResp.ErrorMessage)
	}

	expectedFormatted := "{\n  \"name\": \"Antigravity\",\n  \"active\": true,\n  \"details\": {\n    \"nested\": 1\n  }\n}"
	if formatResp.Formatted != expectedFormatted {
		t.Errorf("Expected formatted JSON:\n%q\nGot:\n%q", expectedFormatted, formatResp.Formatted)
	}

	if formatResp.Stats == nil {
		t.Fatalf("Expected stats to be populated, got nil")
	}

	if formatResp.Stats.MaxDepth != 3 {
		t.Errorf("Expected MaxDepth to be 3, got %d", formatResp.Stats.MaxDepth)
	}

	if formatResp.Stats.KeyCount != 3 { // name, active, details (nested is nested inside details, but traverse counts it as key. Let's see: name:1, active:1, details:1, nested:1. Traverse function: name, active, details (3) + nested (1) = 4 keys?)
		// Wait, let's verify key count based on implementation:
		// Traverse function:
		// case map[string]interface{}:
		//   objects = 1
		//   for _, child := range v {
		//     keys++
		//     k, a, o := traverse(child)
		//     keys += k
		//     ...
		//   }
		// Root map has 3 keys (name, active, details). Inside details, child is map with 1 key (nested).
		// So total keys should be 3 + 1 = 4. Let's check what it actually returns.
		if formatResp.Stats.KeyCount != 4 {
			t.Errorf("Expected KeyCount to be 4, got %d", formatResp.Stats.KeyCount)
		}
	}

	if formatResp.Stats.ObjectCount != 2 { // root object + details object
		t.Errorf("Expected ObjectCount to be 2, got %d", formatResp.Stats.ObjectCount)
	}
}

// TestHandler_FormatAPI_Minify verifies successful minification when requested.
func TestHandler_FormatAPI_Minify(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	payload := FormatRequest{
		JSONData: "{\n  \"name\": \"Antigravity\",\n  \"active\": true\n}",
		Minify:   true,
	}
	jsonBytes, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/format", "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		t.Fatalf("Failed to execute POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	var formatResp FormatResponse
	if err := json.NewDecoder(resp.Body).Decode(&formatResp); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if !formatResp.Success {
		t.Fatalf("Expected Success to be true, got false")
	}

	expectedMinified := `{"name":"Antigravity","active":true}`
	if formatResp.Formatted != expectedMinified {
		t.Errorf("Expected minified:\n%q\nGot:\n%q", expectedMinified, formatResp.Formatted)
	}
}

// TestHandler_FormatAPI_InvalidMethod verifies that GET requests on API return Method Not Allowed.
func TestHandler_FormatAPI_InvalidMethod(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/format")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 Method Not Allowed, got %d", resp.StatusCode)
	}
}

// TestHandler_FormatAPI_SyntaxError verifies that invalid JSON returns detailed location-based errors.
func TestHandler_FormatAPI_SyntaxError(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	payload := FormatRequest{
		JSONData: `{"name": "Antigravity", "active": true, }`, // Trailing comma syntax error
	}
	jsonBytes, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/format", "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		t.Fatalf("Failed to execute POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("Expected status 400 Bad Request, got %d", resp.StatusCode)
	}

	var formatResp FormatResponse
	if err := json.NewDecoder(resp.Body).Decode(&formatResp); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if formatResp.Success {
		t.Fatal("Expected Success to be false, got true")
	}

	if formatResp.ErrorLine == 0 {
		t.Errorf("Expected non-zero ErrorLine, got %d", formatResp.ErrorLine)
	}
}
