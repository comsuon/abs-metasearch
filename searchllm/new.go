package searchllm

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"abs-metasearch/utils"
)

const (
	envSearXNGURL     = "SEARXNG_URL"
	envLLMEndpoint    = "LLM_ENDPOINT"
	envLLMAPIKey      = "LLM_API_KEY"
	envLLMModel       = "LLM_MODEL"
	envLLMTimeout     = "LLM_TIMEOUT"
	envSearXNGTimeout = "SEARXNG_TIMEOUT"

	defaultTimeout = 60 * time.Second
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

	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}
	httpClient.Timeout = parseTimeout(envSearXNGTimeout, httpClient.Timeout)
	llmTimeout := parseTimeout(envLLMTimeout, httpClient.Timeout)

	return NewClient(searxngURL, llmEndpoint, llmAPIKey, llmModel, httpClient, llmTimeout)
}

func parseTimeout(envVar string, fallback time.Duration) time.Duration {
	val := os.Getenv(envVar)
	if val == "" {
		return fallback
	}
	secs, err := strconv.Atoi(val)
	if err != nil || secs <= 0 {
		return fallback
	}
	return time.Duration(secs) * time.Second
}

func NewClient(
	searxngURL, llmEndpoint, llmAPIKey, llmModel string,
	httpClient *http.Client,
	llmTimeout time.Duration,
) (*Client, error) {
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

	llmHTTPClient := &http.Client{
		Timeout: llmTimeout,
	}

	llmClient := &LLMHTTPClient{
		client:   llmHTTPClient,
		endpoint: parsedLLMEndpoint,
		apiKey:   llmAPIKey,
		model:    llmModel,
	}

	return &Client{
		searchClient: searchClient,
		llmClient:    llmClient,
	}, nil
}
