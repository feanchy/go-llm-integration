package llm

type Message struct {
	Role    string `json:"string"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Authorization string `json:"string"`
	Message       []Message
	MaxTokens     int64 `json:"int"`
}

type ChatResponse struct {
}
