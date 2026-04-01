package recommendations

import (
	"bytes"
	"fmt"
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
	article, err := readability.FromURL(rawURL, 30*time.Second)

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
