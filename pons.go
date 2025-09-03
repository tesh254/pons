package main

import (
	"flag"
	"log"

	"github.com/tesh254/pons/internal/scraper"
)

func main() {
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	url := "https://migraine.dev/docs"

	// 1. Create a new scraper
	config := scraper.DefaultConfig()
	config.Verbose = *verbose
	s := scraper.New(url, config)

	// 2. Get content for the main page to extract metadata
	if err := s.GetContent(); err != nil {
		log.Fatalf("Failed to get content for metadata: %v", err)
	}

	// 3. Get metadata for the main page
	if err := s.GetMetadata(); err != nil {
		log.Fatalf("Failed to get metadata: %v", err)
	}

	// 4. Crawl the website to get all subpaths and their content
	if err := s.GetAllPaths(); err != nil {
		log.Fatalf("Failed to get all paths: %v", err)
	}
}
