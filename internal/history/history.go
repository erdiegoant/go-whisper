package history

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Entry is one transcription record.
type Entry struct {
	ID            int64
	Timestamp     time.Time
	ModeName      string
	PromptUsed    string
	RawText       string
	ProcessedText string
	DurationMs    int64
	Language      string
}

// Log manages the SQLite history database.
type Log struct {
	db *sql.DB
}

// Open opens (or creates) the history database at dir/history.db.
func Open(dir string) (*Log, error) {
	path := filepath.Join(dir, "history.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("history: open: %w", err)
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("history: migrate: %w", err)
	}
	return &Log{db: db}, nil
}

// Add writes a new entry. ProcessedText may equal RawText when no LLM ran.
func (l *Log) Add(e Entry) error {
	_, err := l.db.Exec(`
		INSERT INTO transcriptions
			(timestamp, mode_name, prompt_used, raw_text, processed_text, duration_ms, language)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.Timestamp.UTC().Format(time.RFC3339),
		e.ModeName,
		e.PromptUsed,
		e.RawText,
		e.ProcessedText,
		e.DurationMs,
		e.Language,
	)
	return err
}

// Recent returns the last n entries, newest first.
func (l *Log) Recent(n int) ([]Entry, error) {
	rows, err := l.db.Query(`
		SELECT id, timestamp, mode_name, prompt_used, raw_text, processed_text, duration_ms, language
		FROM transcriptions
		ORDER BY id DESC
		LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var ts string
		if err := rows.Scan(&e.ID, &ts, &e.ModeName, &e.PromptUsed,
			&e.RawText, &e.ProcessedText, &e.DurationMs, &e.Language); err != nil {
			return nil, err
		}
		e.Timestamp, _ = time.Parse(time.RFC3339, ts)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// Close closes the database.
func (l *Log) Close() error { return l.db.Close() }

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS transcriptions (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp      TEXT    NOT NULL,
			mode_name      TEXT    NOT NULL DEFAULT '',
			prompt_used    TEXT    NOT NULL DEFAULT '',
			raw_text       TEXT    NOT NULL DEFAULT '',
			processed_text TEXT    NOT NULL DEFAULT '',
			duration_ms    INTEGER NOT NULL DEFAULT 0,
			language       TEXT    NOT NULL DEFAULT ''
		)`)
	return err
}
