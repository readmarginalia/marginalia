package feed

import (
	"crypto/sha256"
	"log/slog"
	"marginalia/internal/common"
	"marginalia/internal/recommendations"
	"time"

	"encoding/hex"
	"encoding/xml"
)

type Service struct {
	recommendations *recommendations.Service
}

func NewService(recommendations *recommendations.Service) *Service {
	return &Service{recommendations: recommendations}
}

type RSS struct {
	XMLName   xml.Name `xml:"rss"`
	Version   string   `xml:"version,attr"`
	ContentNS string   `xml:"xmlns:content,attr"`
	Channel   Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title          string `xml:"title"`
	Link           string `xml:"link"`
	Description    string `xml:"description"`
	ContentEncoded string `xml:"content:encoded"`
	Author         string `xml:"author,omitempty"`
	PubDate        string `xml:"pubDate"`
	GUID           string `xml:"guid"`
}

type RssOutput struct {
	Content      []byte
	ETag         string
	LastModified time.Time
}

func (s *Service) RenderRss(owner string) (*RssOutput, error) {
	recs, err := s.recommendations.All()
	if err != nil {
		return nil, err
	}

	items := make([]Item, len(recs))
	for i, r := range recs {
		items[i] = Item{
			Title:          r.Title,
			Link:           r.URL,
			Description:    r.Excerpt,
			ContentEncoded: r.Content,
			Author:         r.Byline,
			PubDate:        time.Unix(r.AddedAt, 0).UTC().Format(time.RFC1123Z),
			GUID:           r.URL,
		}
	}

	title := "Marginalia"
	desc := "Articles worth reading"
	if owner != "" {
		if owner[len(owner)-1] == 's' || owner[len(owner)-1] == 'S' {
			title = owner + "' Marginalia"
		} else {
			title = owner + "'s Marginalia"
		}
		desc = "Articles worth reading, recommended by " + owner
	}

	rss := RSS{
		Version:   "2.0",
		ContentNS: "http://purl.org/rss/1.0/modules/content/",
		Channel: Channel{
			Title:       title,
			Description: desc,
			Items:       items,
		},
	}

	out, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		slog.Error("Error generating RSS feed", "error", err)
		return nil, &common.ServiceError{Reason: "rss generation error", Code: 500}
	}

	data := append([]byte(xml.Header), out...)

	hash := sha256.Sum256(data)
	etag := `"` + hex.EncodeToString(hash[:8]) + `"`

	// Use the most recent item's timestamp as Last-Modified
	var lastMod time.Time
	if len(recs) > 0 {
		lastMod = time.Unix(recs[0].AddedAt, 0).UTC()
	} else {
		lastMod = time.Now().UTC()
	}

	return &RssOutput{
		Content:      data,
		ETag:         etag,
		LastModified: lastMod,
	}, nil
}
