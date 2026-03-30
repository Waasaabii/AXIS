package axis

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(databasePath string) (*Store, error) {
	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if err := store.init(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) init() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS kv_state (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			level TEXT NOT NULL,
			scope TEXT NOT NULL,
			message TEXT NOT NULL,
			at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS mihomo_versions (
			version TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			archive_path TEXT DEFAULT '',
			binary_path TEXT DEFAULT '',
			downloaded_at TEXT DEFAULT '',
			installed_at TEXT DEFAULT '',
			activated_at TEXT DEFAULT '',
			last_error TEXT DEFAULT ''
		);`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) SaveState(state *AppState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`INSERT INTO kv_state (key, value, updated_at) VALUES ('app_state', ?, ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`, string(raw), nowISO())
	return err
}

func (s *Store) LoadState() (*AppState, error) {
	row := s.db.QueryRow(`SELECT value FROM kv_state WHERE key='app_state'`)
	var raw string
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	var state AppState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Store) AppendEvent(event EventEntry, limit int) error {
	if _, err := s.db.Exec(`INSERT INTO events (id, level, scope, message, at) VALUES (?, ?, ?, ?, ?)`, event.ID, event.Level, event.Scope, event.Message, event.At); err != nil {
		return err
	}

	if limit > 0 {
		_, err := s.db.Exec(`DELETE FROM events WHERE id NOT IN (SELECT id FROM events ORDER BY at DESC LIMIT ?)`, limit)
		return err
	}

	return nil
}

func (s *Store) ListEvents(limit int) ([]EventEntry, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.Query(`SELECT id, level, scope, message, at FROM events ORDER BY at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]EventEntry, 0, limit)
	for rows.Next() {
		var item EventEntry
		if err := rows.Scan(&item.ID, &item.Level, &item.Scope, &item.Message, &item.At); err != nil {
			return nil, err
		}
		events = append(events, item)
	}

	return events, rows.Err()
}

func (s *Store) UpsertVersionRecord(record MihomoVersionRecord) error {
	_, err := s.db.Exec(`INSERT INTO mihomo_versions (version, status, archive_path, binary_path, downloaded_at, installed_at, activated_at, last_error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(version) DO UPDATE SET
			status=excluded.status,
			archive_path=excluded.archive_path,
			binary_path=excluded.binary_path,
			downloaded_at=excluded.downloaded_at,
			installed_at=excluded.installed_at,
			activated_at=excluded.activated_at,
			last_error=excluded.last_error`,
		record.Version,
		record.Status,
		record.ArchivePath,
		record.BinaryPath,
		record.DownloadedAt,
		record.InstalledAt,
		record.ActivatedAt,
		record.LastError,
	)
	return err
}

func (s *Store) ListVersionRecords() ([]MihomoVersionRecord, error) {
	rows, err := s.db.Query(`SELECT version, status, archive_path, binary_path, downloaded_at, installed_at, activated_at, last_error FROM mihomo_versions ORDER BY version DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records := []MihomoVersionRecord{}
	for rows.Next() {
		var item MihomoVersionRecord
		if err := rows.Scan(&item.Version, &item.Status, &item.ArchivePath, &item.BinaryPath, &item.DownloadedAt, &item.InstalledAt, &item.ActivatedAt, &item.LastError); err != nil {
			return nil, err
		}
		records = append(records, item)
	}
	return records, rows.Err()
}

func withTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
