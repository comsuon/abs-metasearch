package searchllm

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"abs-metasearch/utils"
)

const (
	envSearXNGURL  = "SEARXNG_URL"
	envLLMEndpoint = "LLM_ENDPOINT"
	envLLMAPIKey   = "LLM_API_KEY"
	envLLMModel    = "LLM_MODEL"
)

var DefaultClient *Client

type Client struct {
	searchClient *SearXNGClient
	llmClient    *LLMHTTPClient
}

func InitDefaultClient() {
	var err error
	DefaultClient, err = NewClientFromEnv()
	if err != nil {
		DefaultClient = nil
	}
}

func NewClientFromEnv() (*Client, error) {
	searxngURL := os.Getenv(envSearXNGURL)
	if searxngURL == "" {
		searxngURL = defaultSearXNGURL
	}

	llmEndpoint := os.Getenv(envLLMEndpoint)
	if llmEndpoint == "" {
		return nil, fmt.Errorf("environment variable %s is required", envLLMEndpoint)
	}

	llmAPIKey := os.Getenv(envLLMAPIKey)
	llmModel := os.Getenv(envLLMModel)
	if llmModel == "" {
		llmModel = "gpt-4o"
	}

	return NewClient(searxngURL, llmEndpoint, llmAPIKey, llmModel, http.DefaultClient)
}

func NewClient(searxngURL, llmEndpoint, llmAPIKey, llmModel string, httpClient *http.Client) (*Client, error) {
	parsedSearXNGURL, err := url.Parse(strings.TrimRight(searxngURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid SearXNG URL: %w", err)
	}

	searchClient := &SearXNGClient{
		client:     httpClient,
		searxngURL: utils.CloneURL(parsedSearXNGURL),
	}

	parsedLLMEndpoint, err := url.Parse(strings.TrimRight(llmEndpoint, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid LLM endpoint: %w", err)
	}

	llmClient := &LLMHTTPClient{
		client:   httpClient,
		endpoint: parsedLLMEndpoint,
		apiKey:   llmAPIKey,
		model:    llmModel,
	}

	return &Client{
		searchClient: searchClient,
		llmClient:    llmClient,
	}, nil
}
