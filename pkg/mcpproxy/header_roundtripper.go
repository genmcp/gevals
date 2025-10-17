package mcpproxy

import (
	"net/http"
)

// HeaderRoundTripper wraps an http.RoundTripper and adds custom headers to every request.
type HeaderRoundTripper struct {
	// Headers is the map of headers to add to each request
	Headers map[string]string
	// Transport is the underlying RoundTripper to use for the actual request
	Transport http.RoundTripper
}

// NewHeaderRoundTripper creates a new HeaderRoundTripper with the given headers.
// If transport is nil, http.DefaultTransport is used.
func NewHeaderRoundTripper(headers map[string]string, transport http.RoundTripper) *HeaderRoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &HeaderRoundTripper{
		Headers:   headers,
		Transport: transport,
	}
}

// RoundTrip implements the http.RoundTripper interface.
// It adds the configured headers to the request before passing it to the underlying transport.
func (h *HeaderRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Add all configured headers to the request
	for key, value := range h.Headers {
		req.Header.Set(key, value)
	}

	// Pass the request to the underlying transport
	return h.Transport.RoundTrip(req)
}
