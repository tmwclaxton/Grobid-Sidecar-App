package store

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

func (s *Store) GetPaperBySlug(slug string) (*Paper, error) {
	paper := Paper{}
	err := s.db.QueryRow("SELECT * FROM papers WHERE slug = ?", slug).Scan(&paper.ID, &paper.Slug, &paper.CustomKey, &paper.ISSN, &paper.DOI, &paper.UserID, &paper.ScreenID, &paper.Title, &paper.Abstract, &paper.Journal, &paper.Year, &paper.Notes, &paper.CreatedAt, &paper.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &paper, nil
}

func (s *Store) GetPaperByID(id int64) (*Paper, error) {
	paper := Paper{}
	err := s.db.QueryRow("SELECT * FROM papers WHERE id = ?", id).Scan(&paper.ID, &paper.Slug, &paper.CustomKey, &paper.ISSN, &paper.DOI, &paper.UserID, &paper.ScreenID, &paper.Title, &paper.Abstract, &paper.Journal, &paper.Year, &paper.Notes, &paper.CreatedAt, &paper.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &paper, nil
}

func (s *Store) CreatePaper(paper *Paper) (int64, error) {
	result, err := s.db.Exec("INSERT INTO papers (slug, custom_key, issn, doi, user_id, screen_id, title, abstract, journal, year, notes, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", paper.Slug, paper.CustomKey, paper.ISSN, paper.DOI, paper.UserID, paper.ScreenID, paper.Title, paper.Abstract, paper.Journal, paper.Year, paper.Notes, paper.CreatedAt, paper.UpdatedAt)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}
