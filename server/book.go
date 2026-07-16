package server

import (
	"context"
	"fmt"

	"abs-metasearch/searchllm"

	"github.com/samber/lo"
)

func searchMetadataBooks(ctx context.Context, title string, author *string) ([]BookMetadata, error) {
	if searchllm.DefaultClient == nil {
		return nil, fmt.Errorf(
			"searchllm client not configured: set SEARXNG_URL, LLM_ENDPOINT, LLM_API_KEY, and LLM_MODEL env vars",
		)
	}

	llmBooks, err := searchllm.DefaultClient.ExtractMetadata(ctx, title, author)
	if err != nil {
		return nil, err
	}

	if len(llmBooks) > 10 {
		llmBooks = llmBooks[:10]
	}

	books := make([]BookMetadata, 0, len(llmBooks))
	for _, llmBook := range llmBooks {
		book := searchllmBookToBookMetadata(llmBook)
		books = append(books, book)
	}

	return books, nil
}

func searchllmBookToBookMetadata(llmBook searchllm.Book) BookMetadata {
	return BookMetadata{
		Title:         llmBook.Title,
		Subtitle:      strPtr(llmBook.Subtitle),
		Author:        strPtr(llmBook.Author),
		Narrator:      strPtr(llmBook.Narrator),
		Publisher:     strPtr(llmBook.Publisher),
		PublishedYear: strPtr(llmBook.PublishedYear),
		Description:   strPtr(llmBook.Description),
		Cover:         strPtr(llmBook.Cover),
		Isbn:          strPtr(llmBook.ISBN),
		Asin:          strPtr(llmBook.ASIN),
		Genres:        slicePtr(llmBook.Genres),
		Tags:          slicePtr(llmBook.Tags),
		Series:        seriesPtr(llmBook.SeriesName, llmBook.Sequence),
		Language:      strPtr(llmBook.Language),
		Duration:      intPtr(llmBook.Duration),
	}
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func slicePtr(s []string) *[]string {
	if len(s) > 0 {
		return &s
	}
	return nil
}

func intPtr(n int) *int {
	if n > 0 {
		return &n
	}
	return nil
}

func seriesPtr(name, sequence string) *[]SeriesMetadata {
	if name == "" {
		return nil
	}
	s := []SeriesMetadata{{
		Series:   name,
		Sequence: lo.Ternary(sequence != "", &sequence, nil),
	}}
	return &s
}
