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
	Context     string    `json:"context"`
	SourceType  string    `json:"source_type"`
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
		embeddings BLOB,
		context TEXT,
		source_type TEXT
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

// GetDB returns the underlying *sql.DB connection.
func (s *Storage) GetDB() *sql.DB {
	return s.db
}

// UpsertDocument stores a document in the database.
// The URL is used as the key.
func (s *Storage) UpsertDocument(doc *Document) error {
	// Marshal embeddings to JSON for storage in BLOB column
	embeddingsJSON, err := json.Marshal(doc.Embeddings)
	if err != nil {
		return fmt.Errorf("failed to marshal embeddings: %v", err)
	}

	stmt, err := s.db.Prepare(`
		INSERT OR REPLACE INTO documents (url, title, description, content, checksum, embeddings, context, source_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare upsert statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(doc.URL, doc.Title, doc.Description, doc.Content, doc.Checksum, embeddingsJSON, doc.Context, doc.SourceType)
	if err != nil {
		return fmt.Errorf("failed to execute upsert statement: %v", err)
	}
	return nil
}



// DeleteDocumentsByPrefix deletes all documents with a URL starting with the given prefix, optionally filtered by context.
func (s *Storage) DeleteDocumentsByPrefix(prefix, context string) error {
	query := "DELETE FROM documents WHERE url LIKE ? || '%'"
	args := []interface{}{prefix}

	if context != "" {
		query += " AND context = ?"
		args = append(args, context)
	}

	stmt, err := s.db.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(args...)
	if err != nil {
		return fmt.Errorf("failed to execute delete statement: %v", err)
	}
	return nil
}

// Clean deletes all documents from the database.
func (s *Storage) Clean() error {
	_, err := s.db.Exec("DELETE FROM documents")
	if err != nil {
		return fmt.Errorf("failed to clean documents table: %v", err)
	}
	return nil
}

// GetDocument retrieves a document by its URL, optionally filtered by context.
func (s *Storage) GetDocument(url, context string) (*Document, error) {
	var row *sql.Row
	query := "SELECT url, title, description, content, checksum, embeddings, context, source_type FROM documents WHERE url = ?"
	args := []interface{}{url}

	if context != "" {
		query += " AND context = ?"
		args = append(args, context)
	}

	row = s.db.QueryRow(query, args...)

	var doc Document
	var embeddingsJSON []byte
	err := row.Scan(&doc.URL, &doc.Title, &doc.Description, &doc.Content, &doc.Checksum, &embeddingsJSON, &doc.Context, &doc.SourceType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("document not found")
		}
		return nil, fmt.Errorf("failed to scan document: %v", err)
	}

	// Unmarshal embeddings from JSON
	if err := json.Unmarshal(embeddingsJSON, &doc.Embeddings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embeddings: %v", err)
	}

	return &doc, nil
}

// ListDocuments retrieves documents from the store, optionally filtered by context, with a limit.
func (s *Storage) ListDocuments(context string, limit int) ([]*Document, error) {
	var rows *sql.Rows
	var err error

	query := "SELECT url, title, description, content, checksum, embeddings, context, source_type FROM documents"
	args := []interface{}{}

	if context != "" {
		query += " WHERE context = ?"
		args = append(args, context)
	}

	query += " LIMIT ?"
	args = append(args, limit)

	rows, err = s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %v", err)
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var doc Document
		var embeddingsJSON []byte
		if err := rows.Scan(&doc.URL, &doc.Title, &doc.Description, &doc.Content, &doc.Checksum, &embeddingsJSON, &doc.Context, &doc.SourceType); err != nil {
			return nil, fmt.Errorf("failed to scan document row: %v", err)
		}

		// Unmarshal embeddings from JSON
		if err := json.Unmarshal(embeddingsJSON, &doc.Embeddings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embeddings: %v", err)
		}
		docs = append(docs, &doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating rows: %v", err)
	}

	return docs, nil
}

// ListAllDocuments retrieves all documents from the store, optionally filtered by context.
func (s *Storage) ListAllDocuments(context string) ([]*Document, error) {
	var rows *sql.Rows
	var err error

	query := "SELECT url, title, description, content, checksum, embeddings, context, source_type FROM documents"
	args := []interface{}{}

	if context != "" {
		query += " WHERE context = ?"
		args = append(args, context)
	}

	rows, err = s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %v", err)
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var doc Document
		var embeddingsJSON []byte
		if err := rows.Scan(&doc.URL, &doc.Title, &doc.Description, &doc.Content, &doc.Checksum, &embeddingsJSON, &doc.Context, &doc.SourceType); err != nil {
			return nil, fmt.Errorf("failed to scan document row: %v", err)
		}

		// Unmarshal embeddings from JSON
		if err := json.Unmarshal(embeddingsJSON, &doc.Embeddings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embeddings: %v", err)
		}
		docs = append(docs, &doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating rows: %v", err)
	}

	return docs, nil
}

// SearchDocChunks searches for documents based on a query and optional context.
func (s *Storage) SearchDocChunks(query string, context string) ([]*Document, error) {
	// This is a placeholder. Actual implementation will involve vector search
	// and filtering by context. For now, it will just return all documents
	// that match the context (if provided).
	var rows *sql.Rows
	var err error

	baseQuery := "SELECT url, title, description, content, checksum, embeddings, context, source_type FROM documents"
	args := []interface{}{}

	if context != "" {
		baseQuery += " WHERE context = ?"
		args = append(args, context)
	}

	// For now, without actual vector search, we'll just return all documents
	// that match the context. In a real scenario, the 'query' would be used
	// to perform a similarity search on the 'embeddings' field.
	rows, err = s.db.Query(baseQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents for search: %v", err)
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var doc Document
		var embeddingsJSON []byte
		if err := rows.Scan(&doc.URL, &doc.Title, &doc.Description, &doc.Content, &doc.Checksum, &embeddingsJSON, &doc.Context, &doc.SourceType); err != nil {
			return nil, fmt.Errorf("failed to scan document row during search: %v", err)
		}

		if err := json.Unmarshal(embeddingsJSON, &doc.Embeddings); err != nil {
			return nil, fmt.Errorf("failed to unmarshal embeddings during search: %v", err)
		}
		docs = append(docs, &doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating rows during search: %v", err)
	}

	return docs, nil
}

// GetContexts retrieves a list of unique contexts from the database.
func (s *Storage) GetContexts() ([]string, error) {
	rows, err := s.db.Query("SELECT DISTINCT context FROM documents WHERE context IS NOT NULL AND context != ''")
	if err != nil {
		return nil, fmt.Errorf("failed to query distinct contexts: %v", err)
	}
	defer rows.Close()

	var contexts []string
	for rows.Next() {
		var context string
		if err := rows.Scan(&context); err != nil {
			return nil, fmt.Errorf("failed to scan context row: %v", err)
		}
		contexts = append(contexts, context)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating context rows: %v", err)
	}

	return contexts, nil
}
