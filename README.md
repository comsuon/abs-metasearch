# abs-metasearch

An AI-powered book metadata provider for AudiobookShelf that uses SearXNG + LLM to find and extract structured metadata for any book.

> Based on [abs-tract](https://github.com/ahobsonsayers/abs-tract) by [Arran Hobson Sayers](https://github.com/ahobsonsayers) — stripped down to the AI search provider with `.env` configuration support.

## How it works

1. **Search**: Queries a [SearXNG](https://github.com/searxng/searxng) instance for `{title} {author} book`
2. **Extract**: Feeds search results to an LLM (any OpenAI-compatible API) which extracts structured metadata

## Metadata Provided

- Title, Subtitle, Author, Narrator (for audiobooks)
- Publisher, Published Year, Description
- Cover URL, ISBN, ASIN
- Genres, Tags, Series Name & Position
- Language, Duration (for audiobooks)

## Configuration

Copy `.env.example` to `.env` and fill in your values:

| Variable | Required | Default | Description |
|---|---|---|---|
| `SEARXNG_URL` | No | `http://localhost:8080` | SearXNG instance URL |
| `LLM_ENDPOINT` | **Yes** | — | OpenAI-compatible API endpoint |
| `LLM_API_KEY` | No | — | API key (not needed for local LLMs) |
| `LLM_MODEL` | No | `gpt-4o` | Model name |
| `SERVER_PORT` | No | `5555` | Server port |

### LLM examples

```bash
# OpenAI
LLM_ENDPOINT=https://api.openai.com/v1
LLM_API_KEY=sk-abc123
LLM_MODEL=gpt-4o

# Ollama (local)
LLM_ENDPOINT=http://localhost:11434/v1
LLM_MODEL=llama3
```

## Quick start

```bash
# Clone and build
git clone https://github.com/comsuon/abs-metasearch.git
cd abs-metasearch

# Configure
cp .env.example .env
# Edit .env with your SearXNG URL and LLM credentials

# Run
go run .
```

## Docker

### Build locally

```bash
docker build -t abs-metasearch .
docker run -d --name abs-metasearch -p 5555:5555 --env-file .env abs-metasearch
```

### Docker Compose

```yaml
services:
  abs-metasearch:
    build: .
    container_name: abs-metasearch
    ports:
      - "5555:5555"
    env_file:
      - .env
    restart: unless-stopped
```

## Test

```bash
curl "http://localhost:5555/metadata/search?query=The+Hobbit&author=J.R.R.+Tolkien"
```

Expected response:

```json
{
  "matches": [
    {
      "title": "The Hobbit",
      "author": "J.R.R. Tolkien",
      "publishedYear": "1937",
      "description": "A fantasy novel...",
      "genres": ["Fantasy", "Adventure"],
      "language": "English"
    }
  ]
}
```

## AudiobookShelf setup

`Settings -> Item Metadata Utils -> Custom Metadata Providers -> Add`

- **Name**: `MetaSearch`
- **URL**: `http://<your_address>:5555/metadata`
- **Authorization Header Value**: leave empty
