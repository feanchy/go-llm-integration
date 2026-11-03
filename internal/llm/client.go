package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	baseURL      = "https://openrouter.ai/api/v1"
	chatEndpoint = "/chat/completions"
)

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
	body, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+chatEndpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send chat request: %w", err)
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		errorBody, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return nil, fmt.Errorf("read error response: %w", err)
		}

		return nil, fmt.Errorf(
			"openrouter API error: %s, %s",
			httpResp.Status,
			string(errorBody),
		)
	}

	var chatResp ChatResponse

	if err := json.NewDecoder(httpResp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &chatResp, nil

}

func (c *Client) ChatCompletionStream(
	ctx context.Context,
	request ChatRequest,
	onChunk func(string),
) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("marshal chat request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+chatEndpoint,
		bytes.NewReader(body),
	)
	if err != nil {
		return fmt.Errorf("create HTTP request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send chat request: %w", err)
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		errorBody, err := io.ReadAll(httpResp.Body)
		if err != nil {
			return fmt.Errorf("read error response: %w", err)
		}

		return fmt.Errorf(
			"openrouter API error: %s, %s",
			httpResp.Status,
			string(errorBody),
		)
	}

	scanner := bufio.NewScanner(httpResp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read stream: %w", err)
	}

	return nil
}
