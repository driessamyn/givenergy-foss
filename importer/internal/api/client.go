package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

const (
	baseURL            = "https://api.givenergy.cloud/v1/"
	defaultRate        = 250 // requests per minute
	maxRetries         = 3
	maxRateLimitWaits  = 10
	retryBaseDelay     = 500 * time.Millisecond
)

// Client is a GivEnergy API client.
type Client struct {
	token      string
	httpClient *http.Client
	limiter    *rate.Limiter
}

// NewClient creates a new GivEnergy API client.
// rateLimit is max requests per minute; pass 0 to use the default (250).
func NewClient(token string, rateLimit int) *Client {
	if rateLimit <= 0 {
		rateLimit = defaultRate
	}
	// Token bucket: rateLimit requests per minute, burst of 1.
	r := rate.Every(time.Minute / time.Duration(rateLimit))
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    rate.NewLimiter(r, 1),
	}
}

// drainAndClose reads and discards the remaining body so the underlying
// TCP connection can be reused by the HTTP client pool, then closes it.
func drainAndClose(body io.ReadCloser) {
	io.Copy(io.Discard, body)
	body.Close()
}

// do executes an HTTP GET request with auth, rate limiting, and retry logic.
func (c *Client) do(ctx context.Context, url string) (*http.Response, error) {
	var lastErr error
	delay := retryBaseDelay
	rateLimitWaits := 0

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
			delay *= 2
		}

		if err := c.limiter.Wait(ctx); err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check X-RateLimit-Remaining — if 0, pause until reset.
		if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining == "0" {
			drainAndClose(resp.Body)
			rateLimitWaits++
			if rateLimitWaits >= maxRateLimitWaits {
				return nil, fmt.Errorf("rate limit exhausted after %d pauses", rateLimitWaits)
			}
			resetHeader := resp.Header.Get("X-RateLimit-Reset")
			pauseDuration := time.Second // default pause
			if resetHeader != "" {
				if resetTS, err := strconv.ParseInt(resetHeader, 10, 64); err == nil {
					if wait := time.Until(time.Unix(resetTS, 0)); wait > 0 {
						pauseDuration = wait
					}
				}
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(pauseDuration):
			}
			attempt-- // retry without consuming a retry slot
			continue
		}

		// Handle authentication and authorisation errors with clear messages.
		if resp.StatusCode == http.StatusUnauthorized {
			drainAndClose(resp.Body)
			return nil, fmt.Errorf("authentication failed (HTTP 401): check your API token")
		}
		if resp.StatusCode == http.StatusForbidden {
			drainAndClose(resp.Body)
			return nil, fmt.Errorf("access denied (HTTP 403): token may lack required scopes")
		}

		// Retry on 429 or server errors.
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			drainAndClose(resp.Body)
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}

		// Any other non-2xx status is a non-retryable error.
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			drainAndClose(resp.Body)
			return nil, fmt.Errorf("HTTP %d for %s", resp.StatusCode, req.URL.Path)
		}

		return resp, nil
	}

	if lastErr != nil {
		return nil, fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
	}
	return nil, fmt.Errorf("request failed after %d retries", maxRetries)
}

// getJSON fetches a URL and decodes the JSON response into v.
func (c *Client) getJSON(ctx context.Context, url string, v any) error {
	resp, err := c.do(ctx, url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(v)
}

// DiscoverInverters returns all communication devices (and their inverters).
func (c *Client) DiscoverInverters(ctx context.Context) ([]CommunicationDevice, error) {
	url := baseURL + "communication-device"
	var page CommunicationDeviceResponse
	if err := c.getJSON(ctx, url, &page); err != nil {
		return nil, fmt.Errorf("DiscoverInverters: %w", err)
	}

	all := make([]CommunicationDevice, 0, page.Meta.Total)
	all = append(all, page.Data...)

	for p := 2; p <= page.Meta.LastPage; p++ {
		var next CommunicationDeviceResponse
		if err := c.getJSON(ctx, fmt.Sprintf("%s?page=%d", url, p), &next); err != nil {
			return nil, fmt.Errorf("DiscoverInverters page %d: %w", p, err)
		}
		all = append(all, next.Data...)
	}

	return all, nil
}

// GetDataPoints fetches all data points for an inverter on a given date.
// date should be formatted as YYYY-MM-DD.
// Returns all data points and the total count.
func (c *Client) GetDataPoints(ctx context.Context, serial string, date string, pageSize int) ([]DataPoint, int, error) {
	basePageURL := fmt.Sprintf("%sinverter/%s/data-points/%s?pageSize=%d", baseURL, serial, date, pageSize)

	var first DataPointResponse
	if err := c.getJSON(ctx, basePageURL+"&page=1", &first); err != nil {
		return nil, 0, fmt.Errorf("GetDataPoints: %w", err)
	}

	all := make([]DataPoint, 0, first.Meta.Total)
	all = append(all, first.Data...)

	for p := 2; p <= first.Meta.LastPage; p++ {
		var next DataPointResponse
		url := fmt.Sprintf("%s&page=%d", basePageURL, p)
		if err := c.getJSON(ctx, url, &next); err != nil {
			return nil, 0, fmt.Errorf("GetDataPoints page %d: %w", p, err)
		}
		all = append(all, next.Data...)
	}

	return all, first.Meta.Total, nil
}

// GetEvents fetches all events for an inverter.
func (c *Client) GetEvents(ctx context.Context, serial string, pageSize int) ([]Event, error) {
	basePageURL := fmt.Sprintf("%sinverter/%s/events?pageSize=%d", baseURL, serial, pageSize)

	var first EventResponse
	if err := c.getJSON(ctx, basePageURL+"&page=1", &first); err != nil {
		return nil, fmt.Errorf("GetEvents: %w", err)
	}

	all := make([]Event, 0, first.Meta.Total)
	all = append(all, first.Data...)

	for p := 2; p <= first.Meta.LastPage; p++ {
		var next EventResponse
		url := fmt.Sprintf("%s&page=%d", basePageURL, p)
		if err := c.getJSON(ctx, url, &next); err != nil {
			return nil, fmt.Errorf("GetEvents page %d: %w", p, err)
		}
		all = append(all, next.Data...)
	}

	return all, nil
}
