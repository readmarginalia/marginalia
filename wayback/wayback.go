package wayback

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 15 * time.Second}

// ArchiveURL returns an optimistically predicted Wayback Machine archive URL
// using the current UTC timestamp.
func ArchiveURL(targetURL string) string {
	ts := time.Now().UTC().Format("20060102150405")
	return fmt.Sprintf("https://web.archive.org/web/%s/%s", ts, targetURL)
}

// RequestSave triggers an asynchronous snapshot on the Wayback Machine.
// It returns immediately; the actual save happens in a background goroutine.
func RequestSave(targetURL string) {
	go func() {
		req, err := http.NewRequest(http.MethodGet, "https://web.archive.org/save/"+targetURL, nil)
		if err != nil {
			log.Printf("wayback: request error for %s: %v", targetURL, err)
			return
		}
		req.Header.Set("User-Agent", "Marginalia/1.0")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("wayback: save failed for %s: %v", targetURL, err)
			return
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)

		if resp.StatusCode >= 400 {
			log.Printf("wayback: save returned %d for %s", resp.StatusCode, targetURL)
		}
	}()
}
