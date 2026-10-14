package esi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const esiBaseURL = "https://esi.evetech.net"

// ErrForbidden means ESI refused the request with a 403. It usually means the
// token is missing the scope, the character lacks the corporation role, or
// (for structures) the character isn't on the structure's access list.
var ErrForbidden = errors.New("esi: forbidden")

// ErrNotFound means ESI answered 404.
var ErrNotFound = errors.New("esi: not found")

// Client makes ESI calls. The zero value is ready to use.
type Client struct {
	HTTPClient *http.Client
	BaseURL    string // defaults to esiBaseURL; tests point it at a fake server
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c *Client) url(format string, args ...any) string {
	base := c.BaseURL
	if base == "" {
		base = esiBaseURL
	}
	return base + fmt.Sprintf(format, args...)
}

// getPages fetches every page of a paginated ESI route, using the X-Pages
// header from each response to know when to stop. url may already have a
// query string.
func getPages[T any](ctx context.Context, c *Client, url, accessToken string) ([]T, error) {
	sep := "?"
	if strings.Contains(url, "?") {
		sep = "&"
	}

	all := make([]T, 0)
	maxPages := 1 // will be overwritten using data from first page
	for onPage := 1; onPage <= maxPages; onPage++ {
		var page []T
		header, err := c.doWithRetry(ctx, "GET", fmt.Sprintf("%s%spage=%d", url, sep, onPage), accessToken, nil, &page)
		if err != nil {
			return nil, err
		}
		all = append(all, page...)

		if totalPagesStr := header.Get("X-Pages"); totalPagesStr != "" {
			maxPages, err = strconv.Atoi(totalPagesStr)
			if err != nil {
				return nil, fmt.Errorf("invalid X-Pages header %q from %s: %w", totalPagesStr, url, err)
			}
		}
	}
	return all, nil
}

// get fetches a route that isn't paginated.
func get[T any](ctx context.Context, c *Client, url, accessToken string) (T, error) {
	var out T
	_, err := c.doWithRetry(ctx, "GET", url, accessToken, nil, &out)
	return out, err
}

// errRateLimited carries how long ESI asked us to wait.
type errRateLimited struct {
	wait time.Duration
	err  error
}

func (e *errRateLimited) Error() string { return e.err.Error() }
func (e *errRateLimited) Unwrap() error { return e.err }

// doWithRetry is do, trying again after an error other than a 403 or 404.
// Large corporations' journals and assets run to many pages, and one failed
// page would otherwise lose them all. When ESI rate limits the request it
// waits as long as ESI asks.
func (c *Client) doWithRetry(ctx context.Context, method, url, accessToken string, body any, out any) (http.Header, error) {
	const attempts = 3
	for attempt := 1; ; attempt++ {
		header, err := c.do(ctx, method, url, accessToken, body, out)
		if err == nil || attempt == attempts || errors.Is(err, ErrForbidden) || errors.Is(err, ErrNotFound) {
			return header, err
		}
		wait := time.Duration(attempt) * time.Second
		var limited *errRateLimited
		if errors.As(err, &limited) && limited.wait > wait {
			wait = limited.wait
		}
		log.Printf("esi: %v; trying again in %s", err, wait)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
	}
}

// do sends one ESI request, sending body as JSON when it isn't nil, and
// parses the JSON response into out. 403 and 404 come back wrapped in
// ErrForbidden and ErrNotFound so callers can tell them apart.
func (c *Client) do(ctx context.Context, method, url, accessToken string, body any, out any) (http.Header, error) {
	var reqBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	log.Println("Querying:", method, url)

	response, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden:
		return nil, fmt.Errorf("%w: %w", ErrForbidden, esiStatusError(url, response.Status, respBody))
	case http.StatusNotFound:
		return nil, fmt.Errorf("%w: %w", ErrNotFound, esiStatusError(url, response.Status, respBody))
	case http.StatusTooManyRequests, 420: // 420 is ESI's error-limit response
		seconds, _ := strconv.Atoi(response.Header.Get("Retry-After"))
		if seconds == 0 {
			seconds, _ = strconv.Atoi(response.Header.Get("X-ESI-Error-Limit-Reset"))
		}
		return nil, &errRateLimited{wait: time.Duration(seconds) * time.Second, err: esiStatusError(url, response.Status, respBody)}
	default:
		return nil, esiStatusError(url, response.Status, respBody)
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return nil, fmt.Errorf("could not parse response from %s: %w", url, err)
	}
	return response.Header, nil
}

// esiStatusError builds an error for a non-200 ESI response, including the
// response body since ESI puts its error message there.
func esiStatusError(url, status string, body []byte) error {
	return fmt.Errorf("esi: %s returned %s: %s", url, status, string(body))
}
