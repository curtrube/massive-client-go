package rest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDefaultsUnchanged ensures existing behavior is preserved when the new
// options are not used.
func TestDefaultsUnchanged(t *testing.T) {
	c := New("test-key")
	assert.Equalf(t, DefaultBaseURL, c.baseURL, "baseURL=%v, want=%v", c.baseURL, DefaultBaseURL)
	assert.NotNilf(t, c.httpClient, "httpClient should be created by default")
	assert.Truef(t, c.pagination, "defaults should be pagination=true, got pagination=%v", c.pagination)
	assert.Falsef(t, c.trace, "defaults should be trace=false, got trace=%v", c.trace)
}

//func TestWithHTTPClientAndBaseURL(t *testing.T) {
//	var testServer *httptest.Server
//	var authHeaders []string
//
//	testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		authHeaders = append(authHeaders, r.Header.Get("Authorization"))
//		w.Header().Set("Content-Type", "application/json")
//	}))
//}
