package recommendations

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
)

type Article struct {
	Title    string
	Byline   string
	Excerpt  string
	Content  string
	SiteName string
}

func extractFromURL(rawURL string) (*Article, error) {
	article, err := readability.FromURL(rawURL, 30*time.Second, func(r *http.Request) {
		r.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Marginalia/1.0)")
	})

	if err != nil {
		return nil, fmt.Errorf("extract article: %w", err)
	}

	var buf bytes.Buffer
	if err := article.RenderHTML(&buf); err != nil {
		return nil, fmt.Errorf("render html: %w", err)
	}

	return &Article{
		Title:    article.Title(),
		Byline:   article.Byline(),
		Excerpt:  article.Excerpt(),
		Content:  buf.String(),
		SiteName: article.SiteName(),
	}, nil
}
