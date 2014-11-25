package model

type Item struct {
	Id          string  `json:"id"`
	Score       string  `json:"score,omitempty"`
	Link        string  `json:"link,omitempty"`
	Image       string  `json:"image,omitempty"`
	Title       string  `json:"title,omitempty"`
	Description string  `json:"description,omitempty"`
	Categories  string  `json:"categories,omitempty"`
	Price       float64 `json:"price,omitempty"`
	Currency    string  `json:"currency,omitempty"`
	Stars       float64 `json:"starts,omitempty"`

	// metadata
	ScrapUrl  string `json:"scrapUrl,omitempty"`
	ScrapTags string `json:"scrapTags,omitempty"`
	Version   int    `json:"version,omitempty"`
	Index     string `json:"index,omitempty"`
	LastScrap string `json:"lastScrap,omitempty"`
}
