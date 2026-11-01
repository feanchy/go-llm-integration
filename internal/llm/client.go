package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"
)

const baseURL = "https://openrouter.ai/api/v1"

type Client struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
}

func NewClient(apiKey string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},

		apiKey:  apiKey,
		baseURL: baseURL,
	}
}

func (c *Client) ChatCompletion(ctx context.Context, request ChatRequest) (*ChatResponse, error) {
	req, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	reqq := bytes.NewReader(req)

	requestt, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://openrouter.ai/api/v1/chat/completions",
		reqq,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(requestt)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

/* type Message struct {
	Role    string `json:"string"`
	Content string `json:"content"`

type ChatRequest struct {
	Authorization string `json:"string"`
	Message       []Message
	MaxTokens     int64 `json:"int"`
}
*/
