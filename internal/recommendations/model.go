package recommendations

type Recommendation struct {
	ID       int64  `json:"id"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Byline   string `json:"byline"`
	Excerpt  string `json:"excerpt"`
	Content  string `json:"content"`
	SiteName string `json:"site_name"`
	AddedAt  int64  `json:"added_at"`
}
