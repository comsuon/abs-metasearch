package searchllm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildExtractionPrompt(t *testing.T) {
	systemPrompt := buildSystemPrompt()
	require.Contains(t, systemPrompt, "title (required)")
	require.Contains(t, systemPrompt, "books")

	userPrompt := buildUserPrompt("The Hobbit", strPtr("J.R.R. Tolkien"), "Fake search context")
	require.Contains(t, userPrompt, "The Hobbit")
	require.Contains(t, userPrompt, "J.R.R. Tolkien")
	require.Contains(t, userPrompt, "Fake search context")
}

func TestBuildExtractionPrompt_NoAuthor(t *testing.T) {
	userPrompt := buildUserPrompt("The Hobbit", nil, "Fake search context")
	require.Contains(t, userPrompt, "The Hobbit")
	require.NotContains(t, userPrompt, "Author:")
	require.Contains(t, userPrompt, "Fake search context")
}

func TestBuildExtractionPrompt_EmptyAuthor(t *testing.T) {
	userPrompt := buildUserPrompt("The Hobbit", strPtr(""), "Fake search context")
	require.Contains(t, userPrompt, "The Hobbit")
	require.NotContains(t, userPrompt, "Author:")
}

func TestExtractJSON_PlainJSON(t *testing.T) {
	input := `{"books":[{"title":"Test"}]}`
	result := extractJSON(input)
	require.Equal(t, input, result)
}

func TestExtractJSON_InMarkdownCodeBlock(t *testing.T) {
	input := "```json\n{\"books\":[{\"title\":\"Test\"}]}\n```"
	result := extractJSON(input)
	require.JSONEq(t, `{"books":[{"title":"Test"}]}`, result)
}

func TestExtractJSON_SurroundedByText(t *testing.T) {
	input := "Here is the result: {\"books\":[{\"title\":\"Test\"}]} End of result."
	result := extractJSON(input)
	require.JSONEq(t, `{"books":[{"title":"Test"}]}`, result)
}

func TestSearchResultsToContext(t *testing.T) {
	results := []SearchResult{
		{Title: "Result 1", URL: "https://example.com/1", Content: "Content 1"},
		{Title: "Result 2", URL: "https://example.com/2", Content: "Content 2"},
	}

	searchContext := searchResultsToContext(results)
	require.Contains(t, searchContext, "Result 1")
	require.Contains(t, searchContext, "Result 2")
	require.Contains(t, searchContext, "https://example.com/1")
	require.Contains(t, searchContext, "Content 1")
}

func TestSearchResultsToContext_WithImage(t *testing.T) {
	results := []SearchResult{
		{Title: "Cover", URL: "https://example.com/book", ImgSrc: "https://example.com/cover.jpg"},
	}

	searchContext := searchResultsToContext(results)
	require.Contains(t, searchContext, "Image: https://example.com/cover.jpg")
	require.NotContains(t, searchContext, "Snippet:")
}

func TestSearchResultsToContext_LimitsTo10(t *testing.T) {
	results := make([]SearchResult, 15)
	for i := range results {
		results[i] = SearchResult{
			Title: "Result",
			URL:   "https://example.com",
		}
	}

	searchContext := searchResultsToContext(results)
	require.NotContains(t, searchContext, "Result 11")
}

func TestLLMHTTPClient_ChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/chat/completions", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		var reqBody struct {
			Model    string        `json:"model"`
			Messages []chatMessage `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&reqBody)

		assert.Equal(t, "gpt-4o", reqBody.Model)
		assert.Len(t, reqBody.Messages, 2)
		assert.Equal(t, "system", reqBody.Messages[0].Role)
		assert.Equal(t, "user", reqBody.Messages[1].Role)

		resp := chatCompletionResponse{
			Choices: []chatCompletionChoice{
				{Message: chatMessage{
					Role:    "assistant",
					Content: `{"books":[{"title":"Test Book","author":"Test Author"}]}`,
				}},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	llmClient, err := NewClient(
		"http://localhost:8080", server.URL, "test-key", "gpt-4o", server.Client(), 30*time.Second,
	)
	require.NoError(t, err)

	content, err := llmClient.llmClient.ChatCompletion(context.Background(), "system prompt", "user prompt")
	require.NoError(t, err)
	require.Contains(t, content, "Test Book")
}

func TestExtractMetadata_E2E(t *testing.T) {
	mockSearXNG := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := searxngResponse{
			Results: []SearchResult{
				{
					Title:   "The Hobbit - Wikipedia",
					URL:     "https://en.wikipedia.org/wiki/The_Hobbit",
					Content: "The Hobbit is a fantasy novel by J.R.R. Tolkien, published in 1937.",
					Engine:  "wikipedia",
				},
				{
					Title:  "The Hobbit Cover",
					URL:    "https://example.com/hobbit",
					ImgSrc: "https://example.com/hobbit-cover.jpg",
					Engine: "google images",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockSearXNG.Close()

	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := chatCompletionResponse{
			Choices: []chatCompletionChoice{
				{Message: chatMessage{
					Role: "assistant",
					Content: `{
  "books": [
    {
      "title": "The Hobbit",
      "author": "J.R.R. Tolkien",
      "publishedYear": "1937",
      "description": "A fantasy novel about Bilbo Baggins.",
      "cover": "https://example.com/hobbit-cover.jpg",
      "genres": ["Fantasy", "Adventure"],
      "language": "English"
    }
  ]
}`,
				}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockLLM.Close()

	client, err := NewClient(
		mockSearXNG.URL, mockLLM.URL, "test-key", "gpt-4o", mockSearXNG.Client(), 30*time.Second,
	)
	require.NoError(t, err)

	books, err := client.ExtractMetadata(context.Background(), "The Hobbit", strPtr("J.R.R. Tolkien"))
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "The Hobbit", books[0].Title)
	require.Equal(t, "J.R.R. Tolkien", books[0].Author)
	require.Equal(t, "https://example.com/hobbit-cover.jpg", books[0].Cover)
	require.Equal(t, "1937", books[0].PublishedYear)
	require.Equal(t, []string{"Fantasy", "Adventure"}, books[0].Genres)
}

func strPtr(s string) *string {
	return &s
}
