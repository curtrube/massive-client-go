package rest

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/curtrube/massive-client-go/rest/gen"
)

// DefaultBaseURL is the production Massive API endpoint.
const DefaultBaseURL = "https://api.massive.com"

type Client struct {
	*gen.ClientWithResponses
	httpClient *http.Client
	baseURL    string
	apiKey     string
	trace      bool
	pagination bool
}

type Option func(*Client)

func WithTrace(enabled bool) Option {
	return func(c *Client) { c.trace = enabled }
}

func WithPagination(enabled bool) Option {
	return func(c *Client) { c.pagination = enabled }
}

// WithHTTPClient overrides the internally created *http.Client.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) { c.httpClient = client }
}

// WithBaseURL overrides the default API base URL (DefaultBaseURL). Useful for
// pointing the client at a test server such as net/http/httptest.
func WithBaseURL(url string) Option {
	return func(c *Client) { c.baseURL = url }
}

// New is backward-compatible (no options = trace=false, pagination=true)
func New(apiKey string) *Client {
	return NewWithOptions(apiKey)
}

func NewWithOptions(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		apiKey = os.Getenv("MASSIVE_API_KEY")
	}
	if apiKey == "" {
		panic("MASSIVE_API_KEY is required")
	}

	c := &Client{
		apiKey:     apiKey,
		baseURL:    DefaultBaseURL,
		trace:      false,
		pagination: true,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		// Create transport (with debug if trace is on)
		var transport http.RoundTripper = http.DefaultTransport
		if c.trace {
			transport = &debugTransport{base: http.DefaultTransport}
		}

		// This http.Client is shared by the generated client AND the iterator
		c.httpClient = &http.Client{
			Timeout:   60 * time.Second,
			Transport: transport,
		}
	} else if c.trace {
		base := c.httpClient.Transport
		if base == nil {
			base = http.DefaultTransport
		}
		c.httpClient.Transport = &debugTransport{base: base}
	}

	var err error
	c.ClientWithResponses, err = gen.NewClientWithResponses(c.baseURL,
		gen.WithHTTPClient(c.httpClient), // ← THIS makes the FIRST request traced
		gen.WithRequestEditorFn(c.addHeaders),
	)
	if err != nil {
		panic(err)
	}

	return c
}

func (c *Client) addHeaders(_ context.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "massive-go-test")
	return nil
}

// === Pointer helpers ===
func String(v string) *string    { return &v }
func Int(v int) *int             { return &v }
func Int64(v int64) *int64       { return &v }
func Float64(v float64) *float64 { return &v }
func Bool(v bool) *bool          { return &v }

// Generic Ptr (used for everything else, including custom enums)
func Ptr[T any](v T) *T { return &v }

// debugTransport prints exactly the format you asked for
type debugTransport struct {
	base http.RoundTripper
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Printf("Request URL: %s\n", req.URL.String())

	// Redact Authorization for security
	h := req.Header.Clone()
	if auth := h.Get("Authorization"); auth != "" {
		h.Set("Authorization", "Bearer REDACTED")
	}
	fmt.Printf("Request Headers: %+v\n", h)

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Response Headers: %+v\n", resp.Header)

	return resp, nil
}

// CheckResponse turns any non-200 response into a clear error (including the raw body).
func CheckResponse(rsp any) error {
	if rsp == nil {
		return fmt.Errorf("nil response from server")
	}

	rv := reflect.ValueOf(rsp)
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return fmt.Errorf("nil response from server")
		}
		rv = rv.Elem()
	}

	// Extract HTTPResponse and Body (present on every generated *Response type)
	httpField := rv.FieldByName("HTTPResponse")
	bodyField := rv.FieldByName("Body")

	if !httpField.IsValid() || httpField.IsNil() {
		return nil // no HTTP info → assume success
	}

	httpResp := httpField.Interface().(*http.Response)
	if httpResp.StatusCode == http.StatusOK {
		return nil
	}

	// Build nice error message
	bodyStr := ""
	if bodyField.IsValid() {
		bodyStr = string(bodyField.Bytes())
	}

	return fmt.Errorf("API error %s\nBody: %s", httpResp.Status, bodyStr)
}
