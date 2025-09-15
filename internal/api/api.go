package api

import (
	"fmt"
	"log"
	"math"
	"sort"

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
func (a *API) UpsertDocument(baseURL, url, title, description, content, checksum, context, sourceType string, embeddings []float32) error {
	doc := &storage.Document{
		URL:         baseURL + url,
		Title:       title,
		Description: description,
		Content:     content,
		Checksum:    checksum,
		Embeddings:  embeddings,
		Context:     context,
		SourceType:  sourceType,
	}
	return a.storage.UpsertDocument(doc)
}

// GetDocument retrieves a document by URL.
func (a *API) GetDocument(url, context string) (*storage.Document, error) {
	return a.storage.GetDocument(url, context)
}

// DeleteDocument deletes a document by URL.
func (a *API) DeleteDocument(url string) error {
	return a.storage.DeleteDocumentsByPrefix(url)
}

type SearchResult struct {
	Doc   *storage.Document
	Score float64
}

// Search finds the most similar documents to a query embedding, up to numResults, optionally filtered by context.
func (a *API) Search(queryEmbedding []float32, numResults int, context string) ([]SearchResult, error) {
	docs, err := a.storage.ListDocuments(context, numResults)
	if err != nil {
		return nil, fmt.Errorf("failed to list documents for search: %v", err)
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no documents in storage to search")
	}

	var results []SearchResult

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
		// log.Printf("Document %s similarity: %f", doc.URL, similarity) // Commented out for less verbose logging

		results = append(results, SearchResult{Doc: doc, Score: similarity})
	}

	// Sort results by similarity in descending order
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Return top N results
	if len(results) > numResults {
		return results[:numResults], nil
	}

	return results, nil
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

// ListDocuments lists all documents, optionally filtered by context.
func (a *API) ListDocuments(context string, limit int) ([]*storage.Document, error) {
	if limit <= 0 {
		limit = 10 // Default limit
	}
	return a.storage.ListDocuments(context, limit)
}