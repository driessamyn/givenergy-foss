package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient creates a Client pointed at the given test server URL.
func newTestClient(t *testing.T, serverURL string, rateLimit int) *Client {
	t.Helper()
	c := NewClient("test-token", rateLimit)
	// Override the base URL by monkey-patching via a round-tripper that rewrites the host.
	c.httpClient = &http.Client{
		Timeout: 5 * time.Second,
		Transport: &prefixRewriter{
			prefix: serverURL,
			inner:  http.DefaultTransport,
		},
	}
	return c
}

// prefixRewriter rewrites requests so that any URL starting with baseURL goes to the test server.
type prefixRewriter struct {
	prefix string
	inner  http.RoundTripper
}

func (rw *prefixRewriter) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the scheme+host with the test server's scheme+host.
	cloned := req.Clone(req.Context())
	original := req.URL.String()
	// Strip the production base URL prefix and replace with test server.
	stripped := strings.TrimPrefix(original, baseURL)
	cloned.URL, _ = cloned.URL.Parse(rw.prefix + "/" + stripped)
	return rw.inner.RoundTrip(cloned)
}

func TestAuthHeaderSent(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		resp := CommunicationDeviceResponse{
			Meta: Meta{CurrentPage: 1, LastPage: 1, Total: 0},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, 600)
	_, err := c.DiscoverInverters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("Authorization header: got %q, want %q", gotAuth, "Bearer test-token")
	}
}

func TestPaginationDataPoints(t *testing.T) {
	// Mock server returns 3 pages of 1 data point each.
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := atomic.AddInt32(&callCount, 1)
		resp := DataPointResponse{
			Data: []DataPoint{
				{Status: fmt.Sprintf("page-%d", page)},
			},
			Meta: Meta{CurrentPage: int(page), LastPage: 3, Total: 3, PerPage: 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, 600)
	points, total, err := c.GetDataPoints(context.Background(), "ED123", "2026-04-01", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("total: got %d, want 3", total)
	}
	if len(points) != 3 {
		t.Errorf("points: got %d, want 3", len(points))
	}
	if int32(callCount) != 3 {
		t.Errorf("HTTP calls: got %d, want 3", callCount)
	}
	for i, dp := range points {
		want := fmt.Sprintf("page-%d", i+1)
		if dp.Status != want {
			t.Errorf("points[%d].Status: got %q, want %q", i, dp.Status, want)
		}
	}
}

func TestPaginationEvents(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := atomic.AddInt32(&callCount, 1)
		resp := EventResponse{
			Data: []Event{
				{Event: fmt.Sprintf("event-%d", page)},
			},
			Meta: Meta{CurrentPage: int(page), LastPage: 3, Total: 3, PerPage: 1},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, 600)
	events, err := c.GetEvents(context.Background(), "ED123", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("events: got %d, want 3", len(events))
	}
	if int32(callCount) != 3 {
		t.Errorf("HTTP calls: got %d, want 3", callCount)
	}
}

func TestRetryOn429(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		resp := CommunicationDeviceResponse{
			Meta: Meta{CurrentPage: 1, LastPage: 1, Total: 0},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, 600)
	// Speed up retries for the test.
	_, err := c.DiscoverInverters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int32(callCount) != 3 {
		t.Errorf("HTTP calls: got %d, want 3 (2 retries + success)", callCount)
	}
}

func TestRetryOn5xx(t *testing.T) {
	var callCount int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp := CommunicationDeviceResponse{
			Meta: Meta{CurrentPage: 1, LastPage: 1, Total: 0},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, 600)
	_, err := c.DiscoverInverters(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int32(callCount) != 2 {
		t.Errorf("HTTP calls: got %d, want 2 (1 retry + success)", callCount)
	}
}

func TestRetryExhausted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL, 600)
	_, err := c.DiscoverInverters(context.Background())
	if err == nil {
		t.Error("expected error after retries exhausted, got nil")
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	c := newTestClient(t, srv.URL, 600)
	_, err := c.DiscoverInverters(ctx)
	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}
}
