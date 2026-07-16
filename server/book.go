package server

import (
	"context"
	"fmt"

	"abs-metasearch/searchllm"
	"github.com/samber/lo"
)

func searchMetadataBooks(ctx context.Context, title string, author *string) ([]BookMetadata, error) {
	if searchllm.DefaultClient == nil {
		return nil, fmt.Errorf("searchllm client not configured: set SEARXNG_URL, LLM_ENDPOINT, LLM_API_KEY, and LLM_MODEL environment variables")
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
	var subtitle *string
	if llmBook.Subtitle != "" {
		subtitle = &llmBook.Subtitle
	}

	var author *string
	if llmBook.Author != "" {
		author = &llmBook.Author
	}

	var publishedYear *string
	if llmBook.PublishedYear != "" {
		publishedYear = &llmBook.PublishedYear
	}

	var cover *string
	if llmBook.Cover != "" {
		cover = &llmBook.Cover
	}

	var isbn *string
	if llmBook.ISBN != "" {
		isbn = &llmBook.ISBN
	}

	var asin *string
	if llmBook.ASIN != "" {
		asin = &llmBook.ASIN
	}

	var description *string
	if llmBook.Description != "" {
		description = &llmBook.Description
	}

	var publisher *string
	if llmBook.Publisher != "" {
		publisher = &llmBook.Publisher
	}

	var language *string
	if llmBook.Language != "" {
		language = &llmBook.Language
	}

	var narrator *string
	if llmBook.Narrator != "" {
		narrator = &llmBook.Narrator
	}

	var genres *[]string
	if len(llmBook.Genres) > 0 {
		genres = &llmBook.Genres
	}

	var tags *[]string
	if len(llmBook.Tags) > 0 {
		tags = &llmBook.Tags
	}

	var series *[]SeriesMetadata
	if llmBook.SeriesName != "" {
		seriesVal := []SeriesMetadata{{
			Series:   llmBook.SeriesName,
			Sequence: lo.Ternary(llmBook.Sequence != "", &llmBook.Sequence, nil),
		}}
		series = &seriesVal
	}

	var duration *int
	if llmBook.Duration > 0 {
		duration = &llmBook.Duration
	}

	return BookMetadata{
		Title:         llmBook.Title,
		Subtitle:      subtitle,
		Author:        author,
		Narrator:      narrator,
		Publisher:     publisher,
		PublishedYear: publishedYear,
		Description:   description,
		Cover:         cover,
		Isbn:          isbn,
		Asin:          asin,
		Genres:        genres,
		Tags:          tags,
		Series:        series,
		Language:      language,
		Duration:      duration,
	}
}
