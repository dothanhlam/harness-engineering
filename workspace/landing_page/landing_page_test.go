package landing_page

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// TestHandler_Index verifies that the root page serves the index.html with correct headers and key page markings.
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

	// Read body and verify critical section keywords are present
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(resp.Body)
	bodyStr := buf.String()

	keywords := []string{
		"<!DOCTYPE html>",
		"HARNESS",
		"Autonomous Systems.",
		"Pipeline Status:",
		"Corporate Email Address",
		"Submit Inquiry Profile",
	}

	for _, kw := range keywords {
		if !strings.Contains(bodyStr, kw) {
			t.Errorf("Expected HTML to contain %q, but it was missing", kw)
		}
	}
}

// TestHandler_CSS verifies that style.css is served with correct status and mime-types.
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

// TestHandler_JS verifies that app.js is served with correct status and mime-types.
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

// TestHandler_NotFound verifies that unrecognized URL routes produce standard 404 responses.
func TestHandler_NotFound(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/nonexistent_resource.ico")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 Not Found, got %d", resp.StatusCode)
	}
}

// TestHandler_Contact_Success_JSON verifies successful JSON inquiry form posts.
func TestHandler_Contact_Success_JSON(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	payload := ContactSubmission{
		Name:    "Dennis Ritchie",
		Email:   "dmr@bell-labs.com",
		Message: "Interested in high-scale C validation engines.",
	}
	jsonBytes, _ := json.Marshal(payload)

	resp, err := http.Post(server.URL+"/api/contact", "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		t.Fatalf("Failed to execute POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	var contactResp ContactResponse
	if err := json.NewDecoder(resp.Body).Decode(&contactResp); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if !contactResp.Success {
		t.Errorf("Expected Success to be true, got false")
	}

	if !strings.Contains(contactResp.Message, "securely transmitted") {
		t.Errorf("Expected success message, got %q", contactResp.Message)
	}
}

// TestHandler_Contact_Success_Form verifies successful URL-encoded form posts.
func TestHandler_Contact_Success_Form(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	formData := url.Values{}
	formData.Set("name", "Ada Lovelace")
	formData.Set("email", "ada@analytical-engine.org")
	formData.Set("message", "Requesting a review of mechanical computing QA suites.")

	resp, err := http.PostForm(server.URL+"/api/contact", formData)
	if err != nil {
		t.Fatalf("Failed to execute POST request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	var contactResp ContactResponse
	if err := json.NewDecoder(resp.Body).Decode(&contactResp); err != nil {
		t.Fatalf("Failed to decode JSON response: %v", err)
	}

	if !contactResp.Success {
		t.Errorf("Expected Success to be true, got false")
	}
}

// TestHandler_Contact_InvalidMethod verifies that GET requests on API contact return StatusMethodNotAllowed.
func TestHandler_Contact_InvalidMethod(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/contact")
	if err != nil {
		t.Fatalf("Failed to execute GET request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 Method Not Allowed, got %d", resp.StatusCode)
	}
}

// TestHandler_Contact_Validation errors verifies correct 400 Bad Request triggers on invalid input bounds.
func TestHandler_Contact_Validation(t *testing.T) {
	handler := Handler()
	server := httptest.NewServer(handler)
	defer server.Close()

	tests := []struct {
		name          string
		payload       ContactSubmission
		expectedStatus int
		expectMsgPart string
	}{
		{
			name: "Missing Name",
			payload: ContactSubmission{
				Name:    "",
				Email:   "test@email.com",
				Message: "This is a valid long enough message.",
			},
			expectedStatus: http.StatusBadRequest,
			expectMsgPart:  "required",
		},
		{
			name: "Missing Email",
			payload: ContactSubmission{
				Name:    "Test User",
				Email:   "",
				Message: "This is a valid long enough message.",
			},
			expectedStatus: http.StatusBadRequest,
			expectMsgPart:  "required",
		},
		{
			name: "Missing Message",
			payload: ContactSubmission{
				Name:    "Test User",
				Email:   "test@email.com",
				Message: "",
			},
			expectedStatus: http.StatusBadRequest,
			expectMsgPart:  "required",
		},
		{
			name: "Invalid Email Format",
			payload: ContactSubmission{
				Name:    "Test User",
				Email:   "testemailcom",
				Message: "This is a valid long enough message.",
			},
			expectedStatus: http.StatusBadRequest,
			expectMsgPart:  "incorrectly",
		},
		{
			name: "Message Too Short",
			payload: ContactSubmission{
				Name:    "Test User",
				Email:   "test@email.com",
				Message: "Short",
			},
			expectedStatus: http.StatusBadRequest,
			expectMsgPart:  "at least 10 characters",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jsonBytes, _ := json.Marshal(tc.payload)
			resp, err := http.Post(server.URL+"/api/contact", "application/json", bytes.NewBuffer(jsonBytes))
			if err != nil {
				t.Fatalf("Failed to execute POST request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			var contactResp ContactResponse
			_ = json.NewDecoder(resp.Body).Decode(&contactResp)
			if contactResp.Success {
				t.Errorf("Expected Success to be false, got true")
			}
			if !strings.Contains(contactResp.Message, tc.expectMsgPart) {
				t.Errorf("Expected error message to contain %q, got %q", tc.expectMsgPart, contactResp.Message)
			}
		})
	}
}
