package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"abs-metasearch/searchllm"

	"github.com/stretchr/testify/require"
)

func startMockServers(t *testing.T) (searXNG, llm *httptest.Server, cleanup func()) {
	t.Helper()

	mockSearXNG := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{
			"results": []map[string]string{
				{
					"title":   "The Hobbit - Wikipedia",
					"url":     "https://en.wikipedia.org/wiki/The_Hobbit",
					"content": "The Hobbit is a fantasy novel by J.R.R. Tolkien, published in 1937.",
					"engine":  "wikipedia",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		content := `{
  "books": [
    {
      "title": "The Hobbit",
      "author": "J.R.R. Tolkien",
      "publishedYear": "1937",
      "description": "A fantasy novel.",
      "genres": ["Fantasy", "Adventure"],
      "language": "English"
    }
  ]
}`
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"role":    "assistant",
						"content": content,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))

	cleanup = func() {
		mockSearXNG.Close()
		mockLLM.Close()
	}

	return mockSearXNG, mockLLM, cleanup
}

func TestSearchMetadata_Handler(t *testing.T) {
	mockSearXNG, mockLLM, cleanup := startMockServers(t)
	defer cleanup()

	client, err := searchllm.NewClient(mockSearXNG.URL, mockLLM.URL, "test-key", "gpt-4o", mockSearXNG.Client())
	require.NoError(t, err)

	origClient := searchllm.DefaultClient
	searchllm.DefaultClient = client
	defer func() { searchllm.DefaultClient = origClient }()

	router, err := NewRouter()
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/metadata/search?query=The+Hobbit&author=J.R.R.+Tolkien")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result struct {
		Matches []BookMetadata `json:"matches"`
	}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)
	require.Len(t, result.Matches, 1)
	require.Equal(t, "The Hobbit", result.Matches[0].Title)
	require.Equal(t, "J.R.R. Tolkien", *result.Matches[0].Author)
	require.Equal(t, "1937", *result.Matches[0].PublishedYear)
	require.Equal(t, []string{"Fantasy", "Adventure"}, *result.Matches[0].Genres)
}

func TestSearchMetadata_Handler_NoClient(t *testing.T) {
	origClient := searchllm.DefaultClient
	searchllm.DefaultClient = nil
	defer func() { searchllm.DefaultClient = origClient }()

	router, err := NewRouter()
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/metadata/search?query=The+Hobbit")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestSearchMetadata_Handler_NoQuery(t *testing.T) {
	router, err := NewRouter()
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/metadata/search")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSearchMetadata_Handler_NoAuthor(t *testing.T) {
	mockSearXNG, mockLLM, cleanup := startMockServers(t)
	defer cleanup()

	client, err := searchllm.NewClient(mockSearXNG.URL, mockLLM.URL, "test-key", "gpt-4o", mockSearXNG.Client())
	require.NoError(t, err)

	origClient := searchllm.DefaultClient
	searchllm.DefaultClient = client
	defer func() { searchllm.DefaultClient = origClient }()

	router, err := NewRouter()
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/metadata/search?query=The+Hobbit")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestSearchMetadata_Handler_EmptyResults(t *testing.T) {
	mockSearXNG := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]any{"results": []any{}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockSearXNG.Close()

	client, err := searchllm.NewClient(mockSearXNG.URL, "http://unused/v1", "test-key", "gpt-4o", mockSearXNG.Client())
	require.NoError(t, err)

	origClient := searchllm.DefaultClient
	searchllm.DefaultClient = client
	defer func() { searchllm.DefaultClient = origClient }()

	router, err := NewRouter()
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/metadata/search?query=NoSuchBookXYZ123")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result struct {
		Matches []BookMetadata `json:"matches"`
	}
	err = json.Unmarshal(body, &result)
	require.NoError(t, err)
	require.Empty(t, result.Matches)
}

func TestSearchMetadata_Handler_SearchError(t *testing.T) {
	mockSearXNG := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer mockSearXNG.Close()

	client, err := searchllm.NewClient(mockSearXNG.URL, "http://unused/v1", "test-key", "gpt-4o", mockSearXNG.Client())
	require.NoError(t, err)

	origClient := searchllm.DefaultClient
	searchllm.DefaultClient = client
	defer func() { searchllm.DefaultClient = origClient }()

	router, err := NewRouter()
	require.NoError(t, err)

	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/metadata/search?query=The+Hobbit")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestSearchllmBookToBookMetadata_Complete(t *testing.T) {
	b := searchllm.Book{
		Title:         "Test Book",
		Subtitle:      "A Subtitle",
		Author:        "Test Author",
		Narrator:      "Test Narrator",
		Publisher:     "Test Publisher",
		PublishedYear: "2023",
		Description:   "A test description.",
		Cover:         "https://example.com/cover.jpg",
		ISBN:          "9781234567890",
		ASIN:          "B0TEST1234",
		Genres:        []string{"Fiction", "Mystery"},
		Tags:          []string{"bestseller", "award-winning"},
		SeriesName:    "Test Series",
		Sequence:      "2",
		Language:      "English",
		Duration:      3600,
	}

	metadata := searchllmBookToBookMetadata(b)

	require.Equal(t, "Test Book", metadata.Title)
	require.Equal(t, "A Subtitle", *metadata.Subtitle)
	require.Equal(t, "Test Author", *metadata.Author)
	require.Equal(t, "Test Narrator", *metadata.Narrator)
	require.Equal(t, "Test Publisher", *metadata.Publisher)
	require.Equal(t, "2023", *metadata.PublishedYear)
	require.Equal(t, "A test description.", *metadata.Description)
	require.Equal(t, "https://example.com/cover.jpg", *metadata.Cover)
	require.Equal(t, "9781234567890", *metadata.Isbn)
	require.Equal(t, "B0TEST1234", *metadata.Asin)
	require.Equal(t, []string{"Fiction", "Mystery"}, *metadata.Genres)
	require.Equal(t, []string{"bestseller", "award-winning"}, *metadata.Tags)
	require.Len(t, *metadata.Series, 1)
	require.Equal(t, "Test Series", (*metadata.Series)[0].Series)
	require.Equal(t, "2", *(*metadata.Series)[0].Sequence)
	require.Equal(t, "English", *metadata.Language)
	require.Equal(t, 3600, *metadata.Duration)
}

func TestSearchllmBookToBookMetadata_Minimal(t *testing.T) {
	b := searchllm.Book{
		Title: "Just a Title",
	}

	metadata := searchllmBookToBookMetadata(b)

	require.Equal(t, "Just a Title", metadata.Title)
	require.Nil(t, metadata.Subtitle)
	require.Nil(t, metadata.Author)
	require.Nil(t, metadata.PublishedYear)
	require.Nil(t, metadata.Description)
	require.Nil(t, metadata.Cover)
	require.Nil(t, metadata.Genres)
	require.Nil(t, metadata.Series)
}
