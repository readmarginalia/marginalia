package feed

import (
	"crypto/sha256"
	"html"
	"log"
	"marginalia/internal/common"
	"marginalia/internal/interop/wayback"
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

func (s *Service) RenderRss(owner string) (*RssOutput, error) {
	recs, err := s.recommendations.All()
	if err != nil {
		return nil, common.ServiceError{Reason: "failed to fetch recommendations", Code: 500}
	}

	items := make([]Item, len(recs))
	for i, r := range recs {
		addedAt := time.Unix(r.AddedAt, 0).UTC()
		cacheURL := wayback.URL(addedAt, r.URL)
		items[i] = Item{
			Title:          r.Title,
			Link:           r.URL,
			Description:    r.Excerpt,
			ContentEncoded: r.Content + `<br><hr><p><i><a href="` + html.EscapeString(cacheURL) + `">View Archived Snapshot</a></i></p>`,
			Author:         r.Byline,
			PubDate:        addedAt.Format(time.RFC1123Z),
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
		log.Printf("Error generating RSS feed: %v", err)
		return nil, common.ServiceError{Reason: "rss generation error", Code: 500}
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
