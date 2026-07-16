# Builder Image
FROM golang:1.23 as builder

WORKDIR /abs-metasearch
COPY . .
RUN go mod download
RUN go build -v -o bin/abs-metasearch

# Distribution Image
FROM alpine:latest

RUN apk add --no-cache libc6-compat

COPY --from=builder /abs-metasearch/bin/abs-metasearch /abs-metasearch

EXPOSE 5555

ENTRYPOINT ["/abs-metasearch"]
