package searchllm

import "strings"

type Book struct {
	Title         string   `json:"title"`
	Subtitle      string   `json:"subtitle,omitempty"`
	Author        string   `json:"author,omitempty"`
	Narrator      string   `json:"narrator,omitempty"`
	Publisher     string   `json:"publisher,omitempty"`
	PublishedYear string   `json:"publishedYear,omitempty"`
	Description   string   `json:"description,omitempty"`
	Cover         string   `json:"cover,omitempty"`
	ISBN          string   `json:"isbn,omitempty"`
	ASIN          string   `json:"asin,omitempty"`
	Genres        []string `json:"genres,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	SeriesName    string   `json:"series,omitempty"`
	Sequence      string   `json:"sequence,omitempty"`
	Language      string   `json:"language,omitempty"`
	Duration      int      `json:"duration,omitempty"`
}

func (b Book) IsValid() bool {
	return strings.TrimSpace(b.Title) != ""
}

type LLMResponse struct {
	Books []Book `json:"books"`
}
