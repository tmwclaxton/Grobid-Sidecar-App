package store

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"simple-go-app/internal/logging"
	"testing"
)

// test adding a log entry
func TestStore_AddLog(t *testing.T) {
	dbHost := "localhost"
	dbPort := "3306"
	dbUser := "sail"
	dbPassword := "password"
	dbName := "rapid_research"

	db, err := sql.Open("mysql", dbUser+":"+dbPassword+"@tcp("+dbHost+":"+dbPort+")/"+dbName)
	if err != nil {
		logging.ErrorLogger.Println("Error opening database:", err)
	}
	s := New(db)

	err = s.SaveLog(Log{
		Level:       "info",
		UserMessage: "test",
		FullLog:     "test",
		Stage:       "test",
		UserID:      1,
		ScreenID:    1,
	})
	if err != nil {
		return
	}
	if err != nil {
		t.Errorf("Error adding log entry: %v", err)
	}
}
