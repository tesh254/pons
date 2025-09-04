package storage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.etcd.io/bbolt"
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

// Storage manages the bbolt database.
type Storage struct {
	db *bbolt.DB
}

// NewStorage creates or opens a bbolt database.
func NewStorage(dbPath string) (*Storage, error) {
	// Ensure the directory exists.
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	db, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open db: %v", err)
	}
	return &Storage{db: db}, nil
}

// Close closes the database connection.
func (s *Storage) Close() {
	s.db.Close()
}

// bucketName is the name of the bucket where documents are stored.
var bucketName = []byte("documents")

// UpsertDocument stores a document in the database.
// The URL is used as the key.
func (s *Storage) UpsertDocument(doc *Document) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(bucketName)
		if err != nil {
			return fmt.Errorf("failed to create bucket: %v", err)
		}

		encoded, err := json.Marshal(doc)
		if err != nil {
			return fmt.Errorf("failed to marshal document: %v", err)
		}

		return b.Put([]byte(doc.URL), encoded)
	})
}

// GetDocument retrieves a document by its URL.
func (s *Storage) GetDocument(url string) (*Document, error) {
	var doc Document
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return fmt.Errorf("bucket not found")
		}

		v := b.Get([]byte(url))
		if v == nil {
			return fmt.Errorf("document not found")
		}

		return json.Unmarshal(v, &doc)
	})
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// DeleteDocumentsByPrefix deletes all documents with a URL starting with the given prefix.
func (s *Storage) DeleteDocumentsByPrefix(prefix string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			return nil // Bucket does not exist, nothing to delete
		}

		c := b.Cursor()
		prefixBytes := []byte(prefix)

		for k, _ := c.Seek(prefixBytes); k != nil && bytes.HasPrefix(k, prefixBytes); k, _ = c.Next() {
			if err := b.Delete(k); err != nil {
				// Handle the error, maybe log it or return it
				return fmt.Errorf("failed to delete key %s: %v", k, err)
			}
		}

		return nil
	})
}

// ListDocuments retrieves all documents from the store.
func (s *Storage) ListDocuments() ([]*Document, error) {
	var docs []*Document
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(bucketName)
		if b == nil {
			// If the bucket doesn't exist, there are no documents.
			return nil
		}

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var doc Document
			if err := json.Unmarshal(v, &doc); err != nil {
				// Decide how to handle corrupted data. For now, we'll just skip it.
				// You might want to log this error.
				continue
			}
			docs = append(docs, &doc)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list documents: %v", err)
	}
	return docs, nil
}
