package feed

import (
	"encoding/xml"
	"html"
	"time"

	"marginalia/db"
	"marginalia/wayback"
)

type RSS struct {
	XMLName    xml.Name `xml:"rss"`
	Version    string   `xml:"version,attr"`
	ContentNS  string   `xml:"xmlns:content,attr"`
	AtomNS     string   `xml:"xmlns:atom,attr"`
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
	CacheLink      *AtomLink  `xml:"atom:link,omitempty"`
}

type AtomLink struct {
	Rel  string `xml:"rel,attr"`
	Href string `xml:"href,attr"`
}

func Render(recs []db.Recommendation, owner string) ([]byte, error) {
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
			CacheLink:      &AtomLink{Rel: "related", Href: cacheURL},
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
		AtomNS:    "http://www.w3.org/2005/Atom",
		Channel: Channel{
			Title:       title,
			Description: desc,
			Items:       items,
		},
	}

	out, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), out...), nil
}
