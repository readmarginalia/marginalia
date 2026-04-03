package wayback

import (
	"context"
	"fmt"
	"io"
	"marginalia/internal/telemetry/logging"
	"net/http"
	"net/url"
	"path"
	"time"
)

type WaybackClient struct {
	baseURL *url.URL
	client  *http.Client
}

func NewClient(baseURL string, timeout time.Duration) (*WaybackClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &WaybackClient{
		baseURL: u,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

func (c *WaybackClient) RequestSave(ctx context.Context, targetURL string) error {
	u := *c.baseURL
	u.Path = path.Join(u.Path, "save")
	u.Path = u.Path + "/" + targetURL

	logger := logging.FromContext(ctx)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		logger.ErrorContext(ctx, "wayback: request error", "url", targetURL, "error", err)
		return err
	}
	req.Header.Set("User-Agent", "Marginalia/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		logger.ErrorContext(ctx, "wayback: save failed", "url", targetURL, "error", err)
		return err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		logger.ErrorContext(ctx, "wayback: save returned error", "status", resp.StatusCode, "url", targetURL)
		return fmt.Errorf("wayback save failed with status %d", resp.StatusCode)
	}
	return nil
}

// URL returns a Wayback Machine archive URL for the given target,
// using the supplied timestamp to build the path.
func URL(t time.Time, targetURL string) string {
	ts := t.UTC().Format("20060102150405")
	return fmt.Sprintf("https://web.archive.org/web/%s/%s", ts, targetURL)
}
