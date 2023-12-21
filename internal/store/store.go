package store

import (
	"database/sql"
	"log"
	"simple-go-app/internal/helpers"
	"simple-go-app/internal/parsing"
)

// Store is the concrete implementation of the Store interface of the mysql package
type Store struct {
	db *sql.DB
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

type Screen struct {
	ID        int64  `json:"id"`
	Slug      string `json:"slug"`
	UserID    int64  `json:"user_id"`
	Title     string `json:"title"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// New creates a new Store instance
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetDB returns the underlying sql.DB instance
func (store *Store) GetDB() *sql.DB {
	return store.db
}

func (store *Store) fetchRecord(dest interface{}, query string, args ...interface{}) error {
	return store.db.QueryRow(query, args...).Scan(dest)
}

func handleQueryResult(err error) {
	if err != nil {
		log.Println(err)
	}
}

func (store *Store) FindDOIFromPaperRepository(dto *parsing.PDFDTO, id int64) {
	screen := store.FindScreenByID(id)
	if screen == nil {
		return
	}

	if paper := store.FindPaperByTitleAndAbstract(screen.ID, dto.Title, dto.Abstract); paper != nil {
		dto.DOI = paper.DOI
	} else if info := store.FindPaperByTitle(screen.ID, dto.Title); info != nil {
		dto.DOI = info.DOI
		if info.Abstract != "" {
			dto.Abstract = info.Abstract
		}
	}
}

func (store *Store) FindScreenByID(id int64) *Screen {
	var screen Screen
	err := store.fetchRecord(&screen, "SELECT * FROM screens WHERE id = ?", id)
	handleQueryResult(err)
	return &screen
}

func (store *Store) FindPaperByTitleAndAbstract(screenID int64, title, abstract string) *Paper {
	var paper Paper
	err := store.fetchRecord(&paper, "SELECT * FROM papers WHERE screen_id = ? AND title = ? AND abstract = ?", screenID, title, abstract)
	handleQueryResult(err)
	return &paper
}

func (store *Store) FindPaperByTitle(screenID int64, title string) *Paper {
	var paper Paper
	err := store.fetchRecord(&paper, "SELECT * FROM papers WHERE screen_id = ? AND title = ?", screenID, title)
	handleQueryResult(err)
	return &paper
}

// FindPaperByDOI finds a paper by its DOI
func (store *Store) FindPaperByDOI(screenID int64, doi string) (*Paper, error) {
	var paper Paper
	err := store.fetchRecord(&paper, "SELECT * FROM papers WHERE screen_id = ? AND doi = ?", screenID, doi)
	if err != nil {
		return &paper, err
	}
	return &paper, nil
}

func (store *Store) CreatePaper(dto *parsing.PDFDTO, userID string, screenID int64) (*Paper, error) {
	log.Println("Creating paper")
	// Execute the SQL query for paper creation
	_, err := store.db.Exec(`
    INSERT INTO papers (
        slug, custom_key, issn, doi, user_id, screen_id, title, abstract, journal, year, notes
    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		helpers.GenerateRandomString(10), dto.CustomKey, dto.ISSN, dto.DOI, userID, screenID, dto.Title, dto.Abstract, dto.Journal, dto.Year, dto.Notes)

	if err != nil {
		log.Println("Error creating paper:", err)
	}

	// Fetch the created paper from the database and return the result directly
	return store.FindPaperByDOI(screenID, dto.DOI)
}

func (store *Store) GetNextSectionOrder(paperID int64) int64 {
	var order int64

	err := store.fetchRecord(&order, "SELECT COALESCE(MAX(`order`), -1) + 1 FROM sections WHERE paper_id = ?", paperID)
	handleQueryResult(err)

	return order
}

func (store *Store) CreateSection(paperID int64, header, text string, order int) (*Section, error) {
	log.Println("Creating section")
	// first or create
	_, err := store.db.Exec(`
		INSERT INTO sections (paper_id, header, text, `+"`order`"+`
		) VALUES (?, ?, ?, ?)`,
		paperID, header, text, order)

	if err != nil {
		return nil, err
	}

	// Fetch the created section from the database and return the result directly
	return store.FindSectionByOrder(paperID, order)
}

func (store *Store) FindSectionByOrder(paperID int64, order int) (*Section, error) {
	var section Section
	err := store.fetchRecord(&section, "SELECT * FROM sections WHERE paper_id = ? AND `order` = ?", paperID, order)
	if err != nil {
		return nil, err
	}
	return &section, nil
}
