package searchllm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"abs-metasearch/utils"

	"github.com/samber/lo"
)

const defaultSearXNGURL = "http://localhost:8080"

var defaultSearXNGParsedURL *url.URL = lo.Must(url.Parse(defaultSearXNGURL))

var DefaultSearXNGURL = func() *url.URL { return utils.CloneURL(defaultSearXNGParsedURL) }

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

type searxngResponse struct {
	Results []SearchResult `json:"results"`
}

type SearXNGClient struct {
	client     *http.Client
	searxngURL *url.URL
}

func (c *SearXNGClient) URL() *url.URL { return utils.CloneURL(c.searxngURL) }

func (c *SearXNGClient) Search(ctx context.Context, query string) ([]SearchResult, error) {
	queryParams := url.Values{}
	queryParams.Add("format", "json")
	queryParams.Add("q", query)

	requestURL := c.URL()
	requestURL = requestURL.JoinPath("search")
	requestURL.RawQuery = queryParams.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make search request: %w", err)
	}
	defer resp.Body.Close()

	if httpErr := utils.HTTPResponseError(resp); httpErr != nil {
		return nil, httpErr
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var result searxngResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return result.Results, nil
}
