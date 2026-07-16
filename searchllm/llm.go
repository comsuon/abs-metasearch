package searchllm

import (
	"abs-metasearch/utils"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionChoice struct {
	Message chatMessage `json:"message"`
}

type chatCompletionResponse struct {
	Choices []chatCompletionChoice `json:"choices"`
}

type LLMHTTPClient struct {
	client   *http.Client
	endpoint *url.URL
	apiKey   string
	model    string
}

func (c *LLMHTTPClient) ChatCompletion(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := struct {
		Model       string        `json:"model"`
		Messages    []chatMessage `json:"messages"`
		MaxTokens   int           `json:"max_tokens,omitempty"`
		Temperature float64       `json:"temperature,omitempty"`
		Stream      bool          `json:"stream"`
	}{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   4096,
		Temperature: 0.3,
		Stream:      false,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	requestURL := c.endpoint.JoinPath("chat", "completions")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL.String(), bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make LLM request: %w", err)
	}
	defer resp.Body.Close()

	if httpErr := utils.HTTPResponseError(resp); httpErr != nil {
		return "", httpErr
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read LLM response: %w", err)
	}

	var completion chatCompletionResponse
	if err := json.Unmarshal(respBody, &completion); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("LLM returned no choices")
	}

	content := completion.Choices[0].Message.Content
	return content, nil
}

func (c *Client) ExtractMetadata(ctx context.Context, title string, author *string) ([]Book, error) {
	searchQuery := fmt.Sprintf("%s book", title)
	if author != nil && *author != "" {
		searchQuery = fmt.Sprintf("%s %s book", title, *author)
	}

	results, err := c.searchClient.Search(ctx, searchQuery)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	searchContext := searchResultsToContext(results)
	systemPrompt := buildSystemPrompt()
	userPrompt := buildUserPrompt(title, author, searchContext)

	llmContent, err := c.llmClient.ChatCompletion(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM extraction failed: %w", err)
	}

	llmContent = extractJSON(llmContent)

	books, err := parseLLMResponse([]byte(llmContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse extracted metadata: %w", err)
	}

	return books, nil
}

func extractJSON(content string) string {
	content = strings.TrimSpace(content)
	content = stripMarkdownCodeBlock(content)
	return extractFirstJSONObject(content)
}

func stripMarkdownCodeBlock(content string) string {
	if start := strings.Index(content, "```json"); start != -1 {
		content = content[start+7:]
		if end := strings.Index(content, "```"); end != -1 {
			content = content[:end]
		}
		return content
	}
	if start := strings.Index(content, "```"); start != -1 {
		content = content[start+3:]
		if end := strings.Index(content, "```"); end != -1 {
			content = content[:end]
		}
	}
	return content
}

func extractFirstJSONObject(content string) string {
	start := strings.Index(content, "{")
	if start == -1 {
		return content
	}

	depth := 0
	end := -1
	for i := start; i < len(content); i++ {
		switch content[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				end = i + 1
				i = len(content)
			}
		}
	}

	if end > start && end <= len(content) {
		return content[start:end]
	}
	return content
}

func searchResultsToContext(results []SearchResult) string {
	var sb strings.Builder
	for i, r := range results {
		if i >= 10 {
			break
		}
		_, _ = sb.WriteString(
			fmt.Sprintf("Result %d:\nTitle: %s\nURL: %s\nSnippet: %s\n\n", i+1, r.Title, r.URL, r.Content),
		)
	}
	return sb.String()
}

func buildSystemPrompt() string {
	return "" +
		"You are a book metadata extraction assistant. " +
		"Given search results about a book, extract structured metadata in JSON format.\n\n" +
		"Output a JSON object with an array of \"books\". " +
		"Each book should have these fields (only include fields you can confidently extract):\n\n" +
		"- title (required): The book title without subtitle or series info\n" +
		"- subtitle: The subtitle if present\n" +
		"- author: The author name(s)\n" +
		"- narrator: The narrator name(s) if it's an audiobook\n" +
		"- publisher: The publisher name\n" +
		"- publishedYear: The year of publication as a string (e.g. \"2023\")\n" +
		"- description: A brief description/summary of the book\n" +
		"- cover: URL to the cover image\n" +
		"- isbn: ISBN-13 or ISBN-10\n" +
		"- asin: Amazon ASIN\n" +
		"- genres: Array of genre strings (e.g. [\"Fantasy\", \"Fiction\"])\n" +
		"- tags: Array of tag strings\n" +
		"- series: The series name\n" +
		"- sequence: The book's position in the series as a string (e.g. \"2\")\n" +
		"- language: The language the book is written in\n" +
		"- duration: Duration in seconds (for audiobooks)\n\n" +
		"If you find multiple matching books, include them all in the \"books\" array, ordered by relevance.\n" +
		"If you cannot find any matching book, return an empty \"books\" array.\n\n" +
		"Example response:\n" +
		"{\n" +
		"  \"books\": [\n" +
		"    {\n" +
		"      \"title\": \"The Hobbit\",\n" +
		"      \"subtitle\": \"There and Back Again\",\n" +
		"      \"author\": \"J.R.R. Tolkien\",\n" +
		"      \"narrator\": \"Andy Serkis\",\n" +
		"      \"publisher\": \"HarperCollins\",\n" +
		"      \"publishedYear\": \"1937\",\n" +
		"      \"description\": \"Bilbo Baggins is a hobbit who enjoys a comfortable...\",\n" +
		"      \"cover\": \"https://example.com/cover.jpg\",\n" +
		"      \"isbn\": \"9780547928227\",\n" +
		"      \"genres\": [\"Fantasy\", \"Adventure\", \"Classic\"],\n" +
		"      \"tags\": [\"middle-earth\", \"quest\"],\n" +
		"      \"series\": \"Middle-earth\",\n" +
		"      \"sequence\": \"1\",\n" +
		"      \"language\": \"English\",\n" +
		"      \"duration\": 0\n" +
		"    }\n" +
		"  ]\n" +
		"}"
}

func buildUserPrompt(title string, author *string, resultsContext string) string {
	var sb strings.Builder
	_, _ = sb.WriteString(fmt.Sprintf("Find metadata for the book: %s\n", title))
	if author != nil && *author != "" {
		_, _ = sb.WriteString(fmt.Sprintf("Author: %s\n", *author))
	}
	_, _ = sb.WriteString("\nSearch results:\n")
	_, _ = sb.WriteString(resultsContext)
	_, _ = sb.WriteString("\nExtract the book metadata as JSON.")
	return sb.String()
}

func parseLLMResponse(responseBody []byte) ([]Book, error) {
	var llmResp LLMResponse
	if err := json.Unmarshal(responseBody, &llmResp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	books := make([]Book, 0, len(llmResp.Books))
	for _, b := range llmResp.Books {
		if b.IsValid() {
			books = append(books, b)
		}
	}

	return books, nil
}
