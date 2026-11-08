package main

import (
	"encoding/json"
	"fmt"
	"go-llm-integration/internal/llm"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	// переменные окружения
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENROUTER_API_KEY is empty")
	}

	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		log.Println("OPENROUTER_MODEL is empty, using openrouter/free")
		model = "openrouter/free"
	}

	maxTokens := 2000
	raw := os.Getenv("LLM_MAX_TOKENS")

	if raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			log.Fatal("LLM_MAX_TOKENS must be an integer")
		}
		if value <= 0 {
			log.Fatal("LLM_MAX_TOKENS must be greater than 0")
		}

		maxTokens = value
	}
	log.Printf("LLM config: model=%s max_tokens=%d", model, maxTokens)
	//конец переменных окружения

	//Create NewClient
	client := llm.NewClient(apiKey)

	//Metrics
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatal("create metrics exporter: ", err)
	}

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(exporter),
	)
	meter := meterProvider.Meter("go-llm-integration")

	//Counter for requests
	requestCounter, err := meter.Int64Counter("llm_requests")
	if err != nil {
		log.Fatal("create requests counter: ", err)
	}

	//Counter for errors
	errorsCounter, err := meter.Int64Counter("llm_errors")
	if err != nil {
		log.Fatal("create errors counter: ", err)
	}

	//
	inputTokensCounter, err := meter.Int64Counter("llm_input_tokens")
	if err != nil {
		log.Fatal("create input token counter: ", err)
	}

	outputTokensCounter, err := meter.Int64Counter("llm_output_tokens")
	if err != nil {
		log.Fatal("create output token counte: ", err)
	}
	//

	//Histogram
	durationHistogram, err := meter.Float64Histogram(
		"llm_request_duration",
		metric.WithUnit("s"),
		metric.WithDescription("Duration of LLM requests"),
	)
	if err != nil {
		log.Fatal("create duration histogram: ", err)
	}

	//Handlers
	http.Handle("/", http.FileServer(http.Dir("web")))
	http.Handle("/metrics", promhttp.Handler())

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

		chatReq := llm.ChatRequest{
			Model:     model,
			Messages:  input.Messages,
			MaxTokens: maxTokens,
		}

		timeStart := time.Now()
		resp, err := client.ChatCompletion(r.Context(), chatReq)
		duration := time.Since(timeStart)

		requestCounter.Add(r.Context(), 1)
		durationHistogram.Record(r.Context(), duration.Seconds())
		if err != nil {
			errorsCounter.Add(r.Context(), 1)
			log.Println("Ошибка OpenRouter:", err)
			http.Error(w, "Не удалось получить ответ модели", http.StatusBadGateway)
			return
		}

		if len(resp.Choices) == 0 {
			errorsCounter.Add(r.Context(), 1)
			http.Error(w, "Модель вернула пустой ответ", http.StatusBadGateway)
			return
		}

		inputTokensCounter.Add(
			r.Context(),
			int64(resp.Usage.PromptTokens),
		)

		outputTokensCounter.Add(
			r.Context(),
			int64(resp.Usage.CompletionTokens),
		)

		log.Println("Причина завершения:", resp.Choices[0].FinishReason)
		log.Printf(
			"Вход: %d, выход: %d, всего: %d",
			resp.Usage.PromptTokens,
			resp.Usage.CompletionTokens,
			resp.Usage.TotalTokens,
		)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, resp.Choices[0].Message.Content)
	})
	log.Println("Открой http://localhost:8080")
	log.Fatal(http.ListenAndServe("127.0.0.1:8080", nil))
}
