package store

type PDFDTO struct {
	Title    string   `json:"title"`
	Abstract string   `json:"abstract"`
	Year     string   `json:"year"`
	DOI      string   `json:"doi"`
	Authors  []string `json:"authors"`
	Sections []string `json:"sections"`
}

type Paper struct {
	ID        int64  `json:"id"`
	Slug      string `json:"slug"`
	CustomKey string `json:"custom_key"`
	ISSN      string `json:"issn"`
	DOI       string `json:"doi"`
	UserID    int64  `json:"user_id"`
	ScreenID  int64  `json:"screen_id"`
	Title     string `json:"title"`
	Abstract  string `json:"abstract"`
	Journal   string `json:"journal"`
	Year      int64  `json:"year"`
	Notes     string `json:"notes"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Section struct {
	ID        int64  `json:"id"`
	PaperID   int64  `json:"paper_id"`
	Order     int64  `json:"order"`
	Header    string `json:"header"`
	Text      string `json:"text"`
	Embedding string `json:"embedding"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
