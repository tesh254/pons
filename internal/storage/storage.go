package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// Document represents the data to be stored.
type Document struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Checksum    string    `json:"checksum"`
	Embeddings  []float32 `json:"embeddings"`
}

// Storage manages the SQLite database.
type Storage struct {
	db *sql.DB
}

// NewStorage creates or opens an SQLite database.
func NewStorage(dbPath string) (*Storage, error) {
	// Ensure the directory exists.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Enable WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %v", err)
	}

	// Create documents table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS documents (
		url TEXT PRIMARY KEY,
		title TEXT,
		description TEXT,
		content TEXT,
		checksum TEXT,
		embeddings BLOB
	);`
	_, err = db.Exec(createTableSQL)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create documents table: %v", err)
	}

	return &Storage{db: db}, nil
}

// Close closes the database connection.
func (s *Storage) Close() {
	s.db.Close()
}



// UpsertDocument stores a document in the database.
// The URL is used as the key.
func (s *Storage) UpsertDocument(doc *Document) error {
	embeddingsJSON, err := json.Marshal(doc.Embeddings)
	if err != nil {
		return fmt.Errorf("failed to marshal embeddings: %v", err)
	}

	stmt, err := s.db.Prepare(`
		INSERT OR REPLACE INTO documents (url, title, description, content, checksum, embeddings)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare upsert statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(doc.URL, doc.Title, doc.Description, doc.Content, doc.Checksum, embeddingsJSON)
	if err != nil {
		return fmt.Errorf("failed to execute upsert statement: %v", err)
	}
	return nil
}

// GetDocument retrieves a document by its URL.
func (s *Storage) GetDocument(url string) (*Document, error) {
	row := s.db.QueryRow("SELECT url, title, description, content, checksum, embeddings FROM documents WHERE url = ?", url)

	var doc Document
	var embeddingsJSON []byte
	err := row.Scan(&doc.URL, &doc.Title, &doc.Description, &doc.Content, &doc.Checksum, &embeddingsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("failed to scan document: %v", err)
	}

	err = json.Unmarshal(embeddingsJSON, &doc.Embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal embeddings: %v", err)
	}

	return &doc, nil
}

// DeleteDocumentsByPrefix deletes all documents with a URL starting with the given prefix.
func (s *Storage) DeleteDocumentsByPrefix(prefix string) error {
	stmt, err := s.db.Prepare("DELETE FROM documents WHERE url LIKE ? || '%' ")
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(prefix)
	if err != nil {
		return fmt.Errorf("failed to execute delete statement: %v", err)
	}
	return nil
}

// ListDocuments retrieves all documents from the store.
func (s *Storage) ListDocuments() ([]*Document, error) {
	rows, err := s.db.Query("SELECT url, title, description, content, checksum, embeddings FROM documents")
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %v", err)
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var doc Document
		var embeddingsJSON []byte
		if err := rows.Scan(&doc.URL, &doc.Title, &doc.Description, &doc.Content, &doc.Checksum, &embeddingsJSON); err != nil {
			return nil, fmt.Errorf("failed to scan document row: %v", err)
		}

		if err := json.Unmarshal(embeddingsJSON, &doc.Embeddings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embeddings for document %s: %v", doc.URL, err)
		}
		docs = append(docs, &doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating rows: %v", err)
	}

	return docs, nil
}
