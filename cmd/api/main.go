package main

import (
	"encoding/json"
	"fmt"
	"go-llm-integration/internal/llm"
	"log"
	"net/http"
	"os"
)

func main() {
	apiKey := os.Getenv("OPENROUTER_API_KEY")

	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is empty")
	}

	client := llm.NewClient(apiKey)

	http.Handle("/", http.FileServer(http.Dir("web")))

	http.HandleFunc("/api/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "Используй POST", http.StatusMethodNotAllowed)
			return
		}
		var input struct {
			Messages []llm.Message `json:"messages"`
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		defer r.Body.Close()

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Некорректный JSON или слишком большой запрос", http.StatusBadRequest)
			return
		}

		if len(input.Messages) == 0 {
			http.Error(w, "Список сообщений пуст", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "Получено сообщений: %d", len(input.Messages))

		chatReq := llm.ChatRequest{
			Model:     "openrouter/free",
			Messages:  input.Messages,
			MaxTokens: 500,
		}
		resp, err := client.ChatCompletion(r.Context(), chatReq)
		fmt.Println(resp, err)
	})
	log.Println("Открой http://localhost:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
