package searchllm

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBook_IsValid(t *testing.T) {
	require.True(t, Book{Title: "The Hobbit"}.IsValid())
	require.False(t, Book{Title: ""}.IsValid())
	require.False(t, Book{Title: "   "}.IsValid())
}

func TestParseLLMResponse_Complete(t *testing.T) {
	response := []byte(`{
  "books": [
    {
      "title": "The Hobbit",
      "subtitle": "There and Back Again",
      "author": "J.R.R. Tolkien",
      "publishedYear": "1937",
      "description": "Bilbo Baggins is a hobbit who enjoys a comfortable life.",
      "cover": "https://example.com/cover.jpg",
      "isbn": "9780547928227",
      "genres": ["Fantasy", "Adventure", "Classic"],
      "series": "Middle-earth",
      "sequence": "1",
      "language": "English",
      "publisher": "Allen & Unwin"
    }
  ]
}`)

	books, err := parseLLMResponse(response)
	require.NoError(t, err)
	require.Len(t, books, 1)

	b := books[0]
	require.Equal(t, "The Hobbit", b.Title)
	require.Equal(t, "There and Back Again", b.Subtitle)
	require.Equal(t, "J.R.R. Tolkien", b.Author)
	require.Equal(t, "1937", b.PublishedYear)
	require.Equal(t, "Bilbo Baggins is a hobbit who enjoys a comfortable life.", b.Description)
	require.Equal(t, "https://example.com/cover.jpg", b.Cover)
	require.Equal(t, "9780547928227", b.ISBN)
	require.Equal(t, []string{"Fantasy", "Adventure", "Classic"}, b.Genres)
	require.Equal(t, "Middle-earth", b.SeriesName)
	require.Equal(t, "1", b.Sequence)
	require.Equal(t, "English", b.Language)
	require.Equal(t, "Allen & Unwin", b.Publisher)
}

func TestParseLLMResponse_Minimal(t *testing.T) {
	response := []byte(`{
  "books": [
    {
      "title": "Unknown Book"
    }
  ]
}`)

	books, err := parseLLMResponse(response)
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "Unknown Book", books[0].Title)
}

func TestParseLLMResponse_MalformedJSON(t *testing.T) {
	response := []byte(`{ this is not json }`)

	_, err := parseLLMResponse(response)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse")
}

func TestParseLLMResponse_EmptyArray(t *testing.T) {
	response := []byte(`{ "books": [] }`)

	books, err := parseLLMResponse(response)
	require.NoError(t, err)
	require.Len(t, books, 0)
}

func TestParseLLMResponse_FiltersInvalidBooks(t *testing.T) {
	response := []byte(`{
  "books": [
    { "title": "" },
    { "title": "Valid Book" },
    { "title": "   " }
  ]
}`)

	books, err := parseLLMResponse(response)
	require.NoError(t, err)
	require.Len(t, books, 1)
	require.Equal(t, "Valid Book", books[0].Title)
}

func TestParseLLMResponse_MultipleBooks(t *testing.T) {
	response := []byte(`{
  "books": [
    { "title": "Book One", "author": "Author A" },
    { "title": "Book Two", "author": "Author B" }
  ]
}`)

	books, err := parseLLMResponse(response)
	require.NoError(t, err)
	require.Len(t, books, 2)
	require.Equal(t, "Book One", books[0].Title)
	require.Equal(t, "Book Two", books[1].Title)
}
