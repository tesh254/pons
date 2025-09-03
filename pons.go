package main

import (
	"fmt"
	"log"

	"github.com/tesh254/pons/internal/scraper"
)

func main() {
	url := "https://migraine.dev/docs"

	// 1. Create a new scraper
	s := scraper.New(url, scraper.DefaultConfig())

	// 2. Get content for the main page to extract metadata
	if err := s.GetContent(); err != nil {
		log.Fatalf("Failed to get content for metadata: %v", err)
	}

	// 3. Get metadata for the main page
	if err := s.GetMetadata(); err != nil {
		log.Fatalf("Failed to get metadata: %v", err)
	}
	fmt.Printf("Main Page Title: %s", s.Metadata.Title)
	fmt.Printf("Main Page Description: %s", s.Metadata.Description)

	// 4. Crawl the website to get all subpaths and their content
	// Note: GetContent was already called, but GetAllPaths will re-fetch
	// the base URL as part of the crawl.
	if err := s.GetAllPaths(); err != nil {
		log.Fatalf("Failed to get all paths: %v", err)
	}

	fmt.Println("--- Crawl Results ---")
	fmt.Printf("Found %d subpaths.", len(s.SubPaths))

	// 5. Access the results
	htmlContents := s.GetSubPathHTMLContent()
	for path, html := range htmlContents {
		fmt.Printf("Path: %s", path)
		fmt.Printf("HTML content length: %d", len(html))
	}
}
