// api.go
package api

import (
	"fmt"
	"log"
	"math"

	"github.com/tesh254/pons/internal/llm"
	"github.com/tesh254/pons/internal/storage"
)

// API provides methods to interact with the document storage.
type API struct {
	storage *storage.Storage
	llm     *llm.Embeddings
}

// NewAPI creates a new API instance.
func NewAPI(storage *storage.Storage, llm *llm.Embeddings) *API {
	return &API{
		storage: storage,
		llm:     llm,
	}
}

// Llm returns the llm instance.
func (a *API) Llm() *llm.Embeddings {
	return a.llm
}

// UpsertDocument stores a new document or updates an existing one.
func (a *API) UpsertDocument(baseURL, url, content, checksum string, embeddings []float32) error {
	doc := &storage.Document{
		URL:        baseURL + url,
		Content:    content,
		Checksum:   checksum,
		Embeddings: embeddings,
	}
	return a.storage.UpsertDocument(doc)
}

// GetDocument retrieves a document by URL.
func (a *API) GetDocument(url string) (*storage.Document, error) {
	return a.storage.GetDocument(url)
}

// DeleteDocument deletes a document by URL.
func (a *API) DeleteDocument(url string) error {
	return a.storage.DeleteDocumentsByPrefix(url)
}

type Doc struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

type SearchResult struct {
	Doc   *Doc
	Score float64
}

// Search finds the most similar document to a query embedding.
func (a *API) Search(queryEmbedding []float32) (*storage.Document, float64, error) {
	docs, err := a.storage.ListDocuments()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list documents for search: %v", err)
	}

	if len(docs) == 0 {
		return nil, 0, fmt.Errorf("no documents in storage to search")
	}

	var bestDoc *storage.Document
	maxSimilarity := -1.0

	for _, doc := range docs {
		if len(doc.Embeddings) == 0 {
			log.Printf("Skipping document %s due to empty embeddings", doc.URL)
			continue // Skip documents without embeddings
		}
		similarity, err := cosineSimilarity(queryEmbedding, doc.Embeddings)
		if err != nil {
			log.Printf("Error calculating cosine similarity for document %s: %v (queryEmbedding length: %d, doc.Embeddings length: %d)", doc.URL, err, len(queryEmbedding), len(doc.Embeddings))
			continue
		}
		log.Printf("Document %s similarity: %f", doc.URL, similarity)

		if similarity > maxSimilarity {
			maxSimilarity = similarity
			bestDoc = doc
		}
	}

	if bestDoc == nil {
		return nil, 0, fmt.Errorf("could not find a suitable document")
	}

	return bestDoc, maxSimilarity, nil
}

// cosineSimilarity computes the cosine similarity between two vectors.
// This is a helper function, as the one in the llm package is a method on the Embeddings struct.
// A standalone function here avoids circular dependencies if llm needed to use the api package.
func cosineSimilarity(a, b []float32) (float64, error) {
	if len(a) != len(b) {
		return 0, fmt.Errorf("vectors must have the same length")
	}

	var dotProduct, aMagnitude, bMagnitude float64
	for i := 0; i < len(a); i++ {
		dotProduct += float64(a[i] * b[i])
		aMagnitude += float64(a[i] * a[i])
		bMagnitude += float64(b[i] * b[i])
	}

	if aMagnitude == 0 || bMagnitude == 0 {
		return 0, nil
	}

	return dotProduct / (math.Sqrt(aMagnitude) * math.Sqrt(bMagnitude)), nil
}

// UpsertDirect upserts a document directly.
func (a *API) UpsertDirect(doc *storage.Document) error {
	return a.storage.UpsertDocument(doc)
}

// ListDocuments lists all documents.
func (a *API) ListDocuments() ([]*storage.Document, error) {
	return a.storage.ListDocuments()
}
