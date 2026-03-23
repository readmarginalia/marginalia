package feed

import (
	"encoding/xml"
	"time"

	"marginalia/db"
)

type RSS struct {
	XMLName    xml.Name `xml:"rss"`
	Version    string   `xml:"version,attr"`
	ContentNS  string   `xml:"xmlns:content,attr"`
	Channel    Channel  `xml:"channel"`
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

func Render(recs []db.Recommendation) ([]byte, error) {
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

	rss := RSS{
		Version:   "2.0",
		ContentNS: "http://purl.org/rss/1.0/modules/content/",
		Channel: Channel{
			Title:       "Marginalia",
			Description: "Articles worth reading",
			Items:       items,
		},
	}

	out, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}
