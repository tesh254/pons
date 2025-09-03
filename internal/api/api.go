package api

import (
	"fmt"
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

// StoreDocument stores a new document.
func (a *API) StoreDocument(url, content, checksum string, embeddings []float32) error {
	doc := &storage.Document{
		URL:        url,
		Content:    content,
		Checksum:   checksum,
		Embeddings: embeddings,
	}
	return a.storage.StoreDocument(doc)
}

// GetDocument retrieves a document by URL.
func (a *API) GetDocument(url string) (*storage.Document, error) {
	return a.storage.GetDocument(url)
}

// DeleteDocument deletes a document by URL.
func (a *API) DeleteDocument(url string) error {
	return a.storage.DeleteDocument(url)
}

// UpdateDocument updates an existing document.
func (a *API) UpdateDocument(url, content, checksum string, embeddings []float32) error {
	// For bbolt, "update" is the same as "store" since Put replaces the value.
	return a.StoreDocument(url, content, checksum, embeddings)
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
			continue // Skip documents without embeddings
		}
		similarity, err := cosineSimilarity(queryEmbedding, doc.Embeddings)
		if err != nil {
			// Log or handle the error for this specific document comparison
			continue
		}

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
