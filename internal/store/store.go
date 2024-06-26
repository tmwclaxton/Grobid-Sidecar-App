package store

import (
	"database/sql"
	"errors"
	"github.com/uniplaces/carbon"
	"log"
	"simple-go-app/internal/helpers"
	"simple-go-app/internal/logging"
	"simple-go-app/internal/parsing"
	"strconv"
)

// Store is the concrete implementation of the Store interface of the mysql package
type Store struct {
	db *sql.DB
}

type Paper struct {
	ID        int64   `json:"id"`
	Slug      string  `json:"slug"`
	CustomKey *string `json:"custom_key,omitempty"`
	ISSN      *string `json:"issn,omitempty"`
	DOI       *string `json:"doi,omitempty"`
	UserID    int64   `json:"user_id"`
	ScreenID  int64   `json:"screen_id"`
	PubMedID  *int    `json:"pubmed_id,omitempty"`
	Title     string  `json:"title"`
	Abstract  string  `json:"abstract"`
	Journal   *string `json:"journal,omitempty"`
	Year      *string `json:"year,omitempty"`
	// Issue     *uint16 `json:"issue,omitempty"`
	Notes     *string `json:"notes,omitempty"`
	Keywords  *string `json:"keywords,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type Section struct {
	ID        int64   `json:"id"`
	PaperID   int64   `json:"paper_id"`
	Order     int64   `json:"order"`
	Header    string  `json:"header"`
	Text      string  `json:"text"`
	Embedding *string `json:"embedding,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
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

// Log represents a log entry to be saved in the database
type Log struct {
	Level       string
	UserMessage string
	FullLog     string
	Stage       string
	UserID      int64
	ScreenID    int64
}

// New creates a new Store instance
func New(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetDB returns the underlying sql.DB instance
func (store *Store) GetDB() *sql.DB {
	return store.db
}

func (store *Store) FindDOIFromPaperRepository(pdfdto *parsing.PDFDTO, screenID int64) {
	paper, err := store.FindPaperByTitleAndAbstract(screenID, pdfdto.Title, pdfdto.Abstract)
	if err != nil {
		log.Println("Error finding paper by title and abstract:", err)
	} else if paper.ID == 0 {
		log.Println("Paper not found by title and abstract")
	} else {
		log.Printf("Found paper by title and abstract: %v\n", paper.ID)
		pdfdto.DOI = *paper.DOI
	}

	if pdfdto.DOI == "" {
		paper, err = store.FindPaperByTitle(screenID, pdfdto.Title)
		if err != nil {
			log.Println("Error finding paper by title:", err)
		} else if paper.ID == 0 {
			log.Println("Paper not found by title")
		} else {
			log.Printf("Found paper by title: %v\n", paper.ID)
			pdfdto.DOI = *paper.DOI
			pdfdto.Abstract = paper.Abstract
		}
	}
}

func (store *Store) FindPaperByTitleAndAbstract(screenID int64, title string, abstract string) (Paper, error) {
	var papers []Paper

	// there should only be one paper with the same title and abstract but we will handle the case where there are multiple and log it
	rows, err := store.db.Query("SELECT * FROM papers WHERE screen_id = ? AND title = ? AND abstract = ?", screenID, title, abstract)
	if err != nil {
		return Paper{}, err
	}

	for rows.Next() {
		var paper Paper
		err = rows.Scan(
			&paper.ID,
			&paper.Slug,
			&paper.CustomKey,
			&paper.ISSN,
			&paper.DOI,
			&paper.UserID,
			&paper.ScreenID,
			&paper.Title,
			&paper.Abstract,
			&paper.Journal,
			&paper.Year,
			&paper.Notes,
			&paper.PubMedID,
			&paper.CreatedAt,
			&paper.UpdatedAt)
		if err != nil {
			return Paper{}, err
		}
		papers = append(papers, paper)
	}

	if len(papers) == 0 {
		return Paper{}, nil
	}

	if len(papers) > 1 {
		logging.InfoLogger.Println("Multiple papers found with the same title and abstract.")
		for _, paper := range papers {
			log.Printf("Paper id: %v\n", paper.ID)
		}
	}

	return papers[0], nil
}

func (store *Store) FindPaperByTitle(screenID int64, title string) (Paper, error) {
	var papers []Paper

	// there should only be one paper with the same title and abstract but we will handle the case where there are multiple and log it
	rows, err := store.db.Query("SELECT * FROM papers WHERE screen_id = ? AND title = ?", screenID, title)
	if err != nil {
		return Paper{}, err
	}

	for rows.Next() {
		var paper Paper
		err = rows.Scan(&paper.ID, &paper.Slug, &paper.CustomKey, &paper.ISSN, &paper.DOI, &paper.UserID, &paper.ScreenID, &paper.Title, &paper.Abstract, &paper.Journal, &paper.Year, &paper.Notes, &paper.PubMedID, &paper.CreatedAt, &paper.UpdatedAt)
		if err != nil {
			return Paper{}, err
		}
		papers = append(papers, paper)
	}

	if len(papers) == 0 {
		return Paper{}, nil
	}

	if len(papers) > 1 {
		// log this
		logging.InfoLogger.Println("Multiple papers found with the same title.")
		for _, paper := range papers {
			log.Printf("Paper id: %v\n", paper.ID)
		}
	}
	return papers[0], nil
}

func (store *Store) FindPaperByDOI(id int64, doi string) (Paper, error) {
	var paper Paper
	err := store.db.QueryRow("SELECT * FROM papers WHERE screen_id = ? AND doi = ?", id, doi).Scan(&paper.ID, &paper.Slug, &paper.CustomKey, &paper.ISSN, &paper.DOI, &paper.UserID, &paper.ScreenID, &paper.Title, &paper.Abstract, &paper.Journal, &paper.Year, &paper.Notes, &paper.PubMedID, &paper.CreatedAt, &paper.UpdatedAt)
	if err != nil {
		return Paper{}, err
	}

	return paper, nil
}

func (store *Store) CreatePaper(dto *parsing.PDFDTO, userID int64, screenID int64) (Paper, error) {

	// if user_id, screen_id, return error
	if userID == 0 || screenID == 0 {
		return Paper{}, errors.New("CreatePaper: missing required fields: userID: " + strconv.FormatInt(userID, 10) + ", screenID: " + strconv.FormatInt(screenID, 10) + ", title: " + dto.Title + ", abstract: " + dto.Abstract)
	}

	// if dto.Title || dto.Abstract are missing, log it
	if dto.Title == "" || dto.Abstract == "" {
		logging.WarningLogger.Println("Title or Abstract empty - UserID: " + strconv.FormatInt(userID, 10) + ", ScreenID: " + strconv.FormatInt(screenID, 10) + ", Title: " + dto.Title + ", Abstract: " + dto.Abstract)
	}

	// if title and doi are empty, return error
	if dto.Title == "" && dto.DOI == "" {
		return Paper{}, errors.New("CreatePaper: missing required fields: userID: " + strconv.FormatInt(userID, 10) + ", screenID: " + strconv.FormatInt(screenID, 10) + ", title: " + dto.Title + ", abstract: " + dto.Abstract)
	}

	// create slug
	slug := helpers.GenerateRandomString(14)

	// create paper
	_, err := store.db.Exec("INSERT INTO papers (slug, user_id, screen_id, pubmed_id, title, issn, abstract, year, doi, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		slug, userID, screenID, dto.PubMedID, dto.Title, dto.ISSN, dto.Abstract, dto.Year, dto.DOI, carbon.Now().DateTimeString(), carbon.Now().DateTimeString())
	if err != nil {
		return Paper{}, err
	}
	// grab paper by doi
	paper, _ := store.FindPaperByDOI(screenID, dto.DOI)

	return paper, nil
}

func (store *Store) GetNextSectionOrder(paperID int64) (int, interface{}) {
	var order int
	err := store.db.QueryRow("SELECT COALESCE(MAX(`order`), -1) + 1 FROM sections WHERE paper_id = ?", paperID).Scan(&order)
	if err != nil {
		return 0, err
	}
	return order, nil
}

func (store *Store) CreateSection(paperID int64, header string, text string, order int) (interface{}, interface{}) {

	// validate inputs
	if header == "" {
		logging.WarningLogger.Println("Header empty - PaperID: " + strconv.FormatInt(paperID, 10) + ", Header: " + header + ", Text: " + text)
	}

	if paperID == 0 || text == "" {
		return nil, errors.New("CreateSection: missing required fields: paperID: " + strconv.FormatInt(paperID, 10) + ", header: " + header + ", text: " + text)
	}

	var section Section
	// check if section already exists
	section, err := store.FindSectionByHeaderAndText(paperID, header, text)
	if section.ID != 0 {
		//log.Printf("Section already exists: %v\n", section.ID)
		return section, nil
	}

	_, err = store.db.Exec("INSERT INTO sections (paper_id, header, text, `order`, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)", paperID, header, text, order, carbon.Now().DateTimeString(), carbon.Now().DateTimeString())
	if err != nil {
		return section, err
	}

	section, _ = store.FindSectionByPaperAndPosition(paperID, order)

	return section, nil
}

func (store *Store) FindSectionByPaperAndPosition(paperID int64, position int) (Section, interface{}) {
	var section Section
	err := store.db.QueryRow("SELECT * FROM sections WHERE paper_id = ? AND `order` = ?", paperID, position).Scan(&section.ID, &section.PaperID, &section.Order, &section.Header, &section.Text, &section.Embedding, &section.CreatedAt, &section.UpdatedAt)
	if err != nil {
		return Section{}, err
	}
	return section, nil
}

func (store *Store) FindSectionByHeaderAndText(paperID int64, header string, text string) (Section, interface{}) {
	var section Section
	err := store.db.QueryRow("SELECT * FROM sections WHERE paper_id = ? AND header = ? AND text = ?", paperID, header, text).Scan(&section.ID, &section.PaperID, &section.Order, &section.Header, &section.Text, &section.Embedding, &section.CreatedAt, &section.UpdatedAt)
	if err != nil {
		//log.Printf("Error finding section by header and text: %v\n", err)
		return Section{}, err
	}

	if section.ID == 0 {
		//log.Printf("Section not found: %v\n", section.ID)
		return Section{}, nil
	}

	//log.Printf("Section found: %v\n", section.ID)
	return section, nil
}

// SaveLog saves a log entry to the database
func (store *Store) SaveLog(logEntry Log) error {
	_, err := store.db.Exec(`
		INSERT INTO logs (level, user_message, full_log, stage, user_id, screen_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())`,
		logEntry.Level, logEntry.UserMessage, logEntry.FullLog, logEntry.Stage, logEntry.UserID, logEntry.ScreenID)

	if err != nil {
		log.Println("Error saving log entry:", err)
		return err
	}

	return nil
}
