package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// MockOpenAIServer implements an OpenAI-compatible /v1/chat/completions endpoint
type MockOpenAIServer struct {
	mu           sync.Mutex
	expectations []*Expectation
	requests     []CapturedRequest
	listener     net.Listener
	server       *http.Server
	fallback     *Response
}

// CapturedRequest stores the full request for assertions
type CapturedRequest struct {
	Raw       ChatCompletionRequest
	Timestamp time.Time
	Matched   bool
	MatchedBy string // Expectation name
}

// Expectation links a matcher to a response
type Expectation struct {
	Name     string
	Matcher  RequestMatcher
	Response *Response
	Times    int // 0 = unlimited
	matched  int
}

// Response defines what to return
type Response struct {
	Body       *ChatCompletionResponse
	Error      *APIError
	StatusCode int           // Defaults to 200
	Delay      time.Duration // Simulate latency
}

// APIError represents an OpenAI API error response
type APIError struct {
	Error APIErrorDetail `json:"error"`
}

// APIErrorDetail contains the error details
type APIErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code,omitempty"`
}

// NewMockOpenAIServer creates a new mock server (not started)
func NewMockOpenAIServer() *MockOpenAIServer {
	return &MockOpenAIServer{
		expectations: make([]*Expectation, 0),
		requests:     make([]CapturedRequest, 0),
	}
}

// Start starts the server on a random available port and returns the base URL
func (s *MockOpenAIServer) Start() (string, error) {
	// Listen on a random available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("failed to listen on random port: %w", err)
	}
	s.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", s.handleChatCompletions)

	s.server = &http.Server{
		Handler: mux,
	}

	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("OpenAI mock server error: %v\n", err)
		}
	}()

	return s.URL(), nil
}

// Stop gracefully stops the server with a timeout
func (s *MockOpenAIServer) Stop() error {
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// URL returns the server's base URL including /v1 path (e.g., "http://127.0.0.1:12345/v1")
// This is the format expected by OpenAI clients which append /chat/completions to the base URL.
func (s *MockOpenAIServer) URL() string {
	if s.listener == nil {
		return ""
	}
	return fmt.Sprintf("http://%s/v1", s.listener.Addr().String())
}

// Expect adds an expectation to the server
func (s *MockOpenAIServer) Expect(e *Expectation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expectations = append(s.expectations, e)
}

// SetFallback sets the response when no expectation matches
func (s *MockOpenAIServer) SetFallback(r *Response) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallback = r
}

// Requests returns all captured requests
func (s *MockOpenAIServer) Requests() []CapturedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]CapturedRequest, len(s.requests))
	copy(result, s.requests)
	return result
}

// RequestCount returns the number of captured requests
func (s *MockOpenAIServer) RequestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

// LastRequest returns the most recent captured request, or nil if none
func (s *MockOpenAIServer) LastRequest() *CapturedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.requests) == 0 {
		return nil
	}
	req := s.requests[len(s.requests)-1]
	return &req
}

// Reset clears all expectations and captured requests
func (s *MockOpenAIServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.expectations = make([]*Expectation, 0)
	s.requests = make([]CapturedRequest, 0)
	s.fallback = nil
}

// handleChatCompletions handles POST /v1/chat/completions
func (s *MockOpenAIServer) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request
	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid_request_error", "Invalid JSON: "+err.Error())
		return
	}

	// Capture the request
	captured := CapturedRequest{
		Raw:       req,
		Timestamp: time.Now(),
	}

	// Find a matching expectation
	s.mu.Lock()
	var response *Response
	for _, exp := range s.expectations {
		// Skip if this expectation has been used up
		if exp.Times > 0 && exp.matched >= exp.Times {
			continue
		}

		if exp.Matcher.Matches(&req) {
			exp.matched++
			response = exp.Response
			captured.Matched = true
			captured.MatchedBy = exp.Name
			break
		}
	}

	// Use fallback if no match
	if response == nil && s.fallback != nil {
		response = s.fallback
		captured.Matched = true
		captured.MatchedBy = "_fallback"
	}

	s.requests = append(s.requests, captured)
	s.mu.Unlock()

	// No matching expectation
	if response == nil {
		s.writeError(w, http.StatusInternalServerError, "server_error",
			"No matching expectation found for request")
		return
	}

	// Apply delay if configured
	if response.Delay > 0 {
		time.Sleep(response.Delay)
	}

	// Return error response if configured
	if response.Error != nil {
		statusCode := response.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusInternalServerError
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response.Error)
		return
	}

	// Return success response
	if response.Body != nil {
		w.Header().Set("Content-Type", "application/json")
		statusCode := response.StatusCode
		if statusCode == 0 {
			statusCode = http.StatusOK
		}
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response.Body)
		return
	}

	// Neither error nor body configured - this is a configuration error
	s.writeError(w, http.StatusInternalServerError, "server_error",
		"Expectation matched but no response configured")
}

// writeError writes an OpenAI-style error response
func (s *MockOpenAIServer) writeError(w http.ResponseWriter, statusCode int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIError{
		Error: APIErrorDetail{
			Message: message,
			Type:    errType,
		},
	})
}
