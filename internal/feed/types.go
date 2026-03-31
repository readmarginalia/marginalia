package feed

import (
	"encoding/xml"
	"time"
)

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
