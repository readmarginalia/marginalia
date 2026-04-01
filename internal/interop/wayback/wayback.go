package wayback

import (
	"fmt"
	"io"
	"log"
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

func (c *WaybackClient) RequestSave(targetURL string) error {
	escapedUrl := url.PathEscape(targetURL)
	u := c.baseURL
	u.Path = path.Join(u.Path, "save", escapedUrl)

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		log.Printf("wayback: request error for %s: %v", targetURL, err)
		return err
	}
	req.Header.Set("User-Agent", "Marginalia/1.0")

	resp, err := c.client.Do(req)
	if err != nil {
		log.Printf("wayback: save failed for %s: %v", targetURL, err)
		return err
	}
	defer resp.Body.Close()

	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		log.Printf("wayback: save returned %d for %s", resp.StatusCode, targetURL)
	}
	return nil
}

// URL returns a Wayback Machine archive URL for the given target,
// using the supplied timestamp to build the path.
func URL(t time.Time, targetURL string) string {
	ts := t.UTC().Format("20060102150405")
	return fmt.Sprintf("https://web.archive.org/web/%s/%s", ts, targetURL)
}
