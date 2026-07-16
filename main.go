package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"abs-metasearch/searchllm"
	"abs-metasearch/server"
)

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config .oapigen.yaml schema/openapi.yaml

func main() {
	if err := loadEnvFile(".env"); err != nil {
		log.Printf("No .env file found, using system environment variables")
	}

	searchllm.InitDefaultClient()

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "5555"
	}
	serverAddress := "0.0.0.0:" + port

	router, err := server.NewRouter()
	if err != nil {
		log.Fatalf("Failed to create router: %s", err)
	}

	log.Printf("Server listening on %s\n", serverAddress)
	err = http.ListenAndServe(serverAddress, router)
	if err != nil {
		log.Fatalf("Server exited with error: %s", err)
	}
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if (strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) ||
			(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			value = value[1 : len(value)-1]
		}
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env file: %w", err)
	}
	return nil
}
