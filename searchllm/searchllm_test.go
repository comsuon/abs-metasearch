package searchllm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSearXNGResults(t *testing.T) {
	response := searxngResponse{
		Results: []SearchResult{
			{
				Title:   "The Hobbit - Wikipedia",
				URL:     "https://en.wikipedia.org/wiki/The_Hobbit",
				Content: "The Hobbit, or There and Back Again is a children's fantasy novel...",
				Engine:  "wikipedia",
			},
			{
				Title:   "The Hobbit - Goodreads",
				URL:     "https://www.goodreads.com/book/show/5907.The_Hobbit",
				Content: "The Hobbit book. Read 67890 reviews...",
				Engine:  "goodreads",
			},
		},
	}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var parsed searxngResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	require.Len(t, parsed.Results, 2)
	require.Equal(t, "The Hobbit - Wikipedia", parsed.Results[0].Title)
	require.Equal(t, "wikipedia", parsed.Results[0].Engine)
}

func TestParseSearXNGResults_Empty(t *testing.T) {
	response := searxngResponse{Results: []SearchResult{}}

	data, err := json.Marshal(response)
	require.NoError(t, err)

	var parsed searxngResponse
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)
	require.Empty(t, parsed.Results)
}

func TestSearXNGClient_Search(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/search", r.URL.Path)
		assert.Equal(t, "json", r.URL.Query().Get("format"))
		assert.Equal(t, "The Hobbit J.R.R. Tolkien book", r.URL.Query().Get("q"))

		resp := searxngResponse{
			Results: []SearchResult{
				{
					Title:   "The Hobbit - Wikipedia",
					URL:     "https://en.wikipedia.org/wiki/The_Hobbit",
					Content: "A children's fantasy novel by J.R.R. Tolkien.",
					Engine:  "wikipedia",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "http://localhost:8080/v1", "test-key", "gpt-4o", server.Client())
	require.NoError(t, err)

	results, err := client.searchClient.Search(context.Background(), "The Hobbit J.R.R. Tolkien book")
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, "The Hobbit - Wikipedia", results[0].Title)
}

func TestSearXNGClient_Search_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": "something went wrong"}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL, "http://localhost:8080/v1", "test-key", "gpt-4o", server.Client())
	require.NoError(t, err)

	_, err = client.searchClient.Search(context.Background(), "test query")
	require.Error(t, err)
}
