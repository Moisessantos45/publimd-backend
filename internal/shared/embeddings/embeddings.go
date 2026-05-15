package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type PostEmbeddingRequest struct {
	ID       uint64   `json:"id"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
	Category string   `json:"category"`
}

type QueryEmbeddingRequest struct {
	Query string `json:"query"`
}

type PostEmbeddingResponse struct {
	ID         uint64    `json:"id"`
	TextUsed   string    `json:"text_used"`
	Dimensions int       `json:"dimensions"`
	Embedding  []float32 `json:"embedding"`
}

type QueryEmbeddingResponse struct {
	Dimensions int       `json:"dimensions"`
	Embedding  []float32 `json:"embedding"`
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient() *Client {
	baseUrl := os.Getenv("EMBEDDINGS_SERVICE_URL")
	if baseUrl == "" {
		baseUrl = "http://localhost:8000"
	}
	return &Client{
		BaseURL: baseUrl,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) GeneratePostEmbedding(ctx context.Context, post PostEmbeddingRequest) (*PostEmbeddingResponse, error) {
	body, err := json.Marshal(post)
	if err != nil {
		return nil, fmt.Errorf("marshal post request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/embeddings/post",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var raw map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&raw)
		return nil, fmt.Errorf("embedding api error: status=%d body=%v", resp.StatusCode, raw)
	}

	var result PostEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) GenerateQueryEmbedding(ctx context.Context, query string) (*QueryEmbeddingResponse, error) {
	payload := QueryEmbeddingRequest{
		Query: query,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal query request: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.BaseURL+"/embeddings/query",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call embedding api: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var raw map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&raw)
		return nil, fmt.Errorf("embedding api error: status=%d body=%v", resp.StatusCode, raw)
	}

	var result QueryEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func ToPGVector(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}

	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = fmt.Sprintf("%f", v)
	}

	return "[" + strings.Join(parts, ",") + "]"
}

func NewPostEmbeddingRequest(id uint64, title, content string, tags []string, category string) PostEmbeddingRequest {

	return PostEmbeddingRequest{
		ID:       id,
		Title:    title,
		Content:  content,
		Tags:     tags,
		Category: category,
	}
}
