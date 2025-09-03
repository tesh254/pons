# Scraper Usage Guide

This guide provides examples on how to use the `scraper` package to scrape web content.

## Creating a Scraper

To start, create a new `Scraper` instance using the `New` function. You can provide a URL and an optional configuration. If no configuration is provided, default settings will be used.

```go
package main

import (
	"fmt"
	"log"

	"your_project_path/internal/scraper"
)

func main() {
	url := "https://example.com"
	s := scraper.New(url, nil) // Using default config

	fmt.Printf("Scraper created for URL: %s", s.URL)
}
```

## Fetching Content and Metadata

You can fetch the content of the main URL and then extract metadata like title and description.

```go
package main

import (
	"fmt"
	"log"

	"your_project_path/internal/scraper"
)

func main() {
	url := "https://example.com"
	s := scraper.New(url, nil)

	// Get content of the main page
	if err := s.GetContent(); err != nil {
		log.Fatalf("Failed to get content: %v", err)
	}

	// Get metadata
	if err := s.GetMetadata(); err != nil {
		log.Fatalf("Failed to get metadata: %v", err)
	}

	fmt.Printf("Title: %s", s.Metadata.Title)
	fmt.Printf("Description: %s", s.Metadata.Description)
}
```

## Scraping Main Content Text

To get the main text content from the page, you can use `ScrapeContent`.

```go
package main

import (
	"fmt"
	"log"

	"your_project_path/internal/scraper"
)

func main() {
	url := "https://example.com"
	s := scraper.New(url, nil)

	textContent, err := s.ScrapeContent()
	if err != nil {
		log.Fatalf("Failed to scrape content: %v", err)
	}

	fmt.Println("Scraped Text Content:")
	fmt.Println(textContent)
}
```

## Crawling a Website

To crawl a website and get all its subpaths and their HTML content, you can use the `GetAllPaths` method. This will populate the `SubPaths` and `SubPathsHTMLContent` fields in your scraper instance.

```go
package main

import (
	"fmt"
	"log"
	"time"

	"your_project_path/internal/scraper"
)

func main() {
	url := "https://example.com"
	// Let's use a custom config for crawling
	config := scraper.DefaultConfig()
	config.MaxDepth = 1 // Only crawl one level deep
	
	s := scraper.New(url, config)

	// Crawl the website
	if err := s.GetAllPaths(); err != nil {
		log.Fatalf("Failed to get all paths: %v", err)
	}

	fmt.Println("Discovered Subpaths:")
	for _, path := range s.SubPaths {
		fmt.Println(path)
	}

	fmt.Println("
HTML Content of Subpaths:")
	subpathContents := s.GetSubPathHTMLContent()
	for path, content := range subpathContents {
		fmt.Printf("--- Content for %s ---", path)
		// Printing only first 100 chars for brevity
		if len(content) > 100 {
			fmt.Println(content[:100] + "...")
		} else {
			fmt.Println(content)
		}
		fmt.Println("--------------------------")
	}
}
```

## Chaining Methods for Full Scrape

Here is an example of how you can chain methods to perform a full scrape of a website, getting all paths, their content, and metadata for the main page.

```go
package main

import (
	"fmt"
	"log"
	"time"

	"your_project_path/internal/scraper"
)

func main() {
	url := "https://example.com"
	
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
```
