package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
)

type Embeddings struct {
	client *http.Client
	url    string
}

// NewEmbeddings creates a new Embeddings instance with the Cloudflare Worker URL.
func NewEmbeddings(workerURL string) *Embeddings {
	return &Embeddings{
		client: &http.Client{},
		url:    workerURL,
	}
}

// embeddingResponse matches the Cloudflare Worker’s JSON response structure.
type embeddingResponse struct {
	Response [][]float32 `json:"response"`
	Shape    []int       `json:"shape"`
	Pooling  string      `json:"pooling"`
}

// GenerateEmbeddings sends text to the Cloudflare Worker and returns embeddings.
func (e *Embeddings) GenerateEmbeddings(content string) ([]float32, error) {
	// Prepare JSON payload
	payload := map[string]string{"text": content}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	// Make HTTP POST request to Cloudflare Worker
	req, err := http.NewRequest("POST", e.url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var result embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if len(result.Response) == 0 || len(result.Response[0]) == 0 {
		return nil, fmt.Errorf("empty embedding returned")
	}

	return result.Response[0], nil
}

// CosineSimilarity computes the cosine similarity between two vectors.
func (e *Embeddings) cosineSimilarity(a, b []float32) (float64, error) {
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

// GetSimilarity computes similarity between a query and precomputed embeddings.
func (e *Embeddings) GetSimilarity(queryEmbedding []float32, contentEmbedding []float32) (float64, error) {
	// Compute cosine similarity
	return e.cosineSimilarity(queryEmbedding, contentEmbedding)
}

// Marshal serializes embeddings to JSON string.
func (e *Embeddings) Marshal(embeddings []float32) (string, error) {
	b, err := json.MarshalIndent(embeddings, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal embeddings: %v", err)
	}
	return string(b), nil
}

// Unmarshal deserializes JSON string to embeddings.
func (e *Embeddings) Unmarshal(data string) ([]float32, error) {
	var embeddings []float32
	if err := json.Unmarshal([]byte(data), &embeddings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal embeddings: %v", err)
	}
	return embeddings, nil
}
