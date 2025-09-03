// Package scraper provides functionality to scrape web content and extract metadata.
//
// This package offers a configurable web scraper that can extract content, metadata,
// and crawl websites while respecting rate limits and timeouts. It provides
// functionality to parse HTML, extract metadata like title and description,
// and collect all paths from a website.
package scraper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Config holds configuration options for the scraper.
//
// This struct allows customization of the scraper's behavior including
// request parameters, timeouts, crawling depth, and rate limiting settings.
type Config struct {
	// UserAgent is the User-Agent header value sent with HTTP requests
	UserAgent string
	// Timeout specifies the maximum duration to wait for an HTTP request to complete
	Timeout time.Duration
	// MaxDepth defines how deep the crawler will follow links from the starting URL
	// A value of 0 means only the starting page, 1 means the starting page and all directly linked pages, etc.
	MaxDepth int
	// RequestDelay specifies the minimum time between requests to the same host
	// This helps prevent overwhelming servers with too many rapid requests
	RequestDelay time.Duration
	// MaxConcurrent limits the total number of concurrent HTTP requests
	// This applies across all hosts being scraped
	MaxConcurrent int
}

// DefaultConfig returns a default configuration with reasonable values.
//
// The default configuration includes a standard user agent, reasonable timeout,
// moderate crawl depth, and conservative rate limiting to be respectful to websites.
//
// Returns:
//   - A Config struct with default values
func DefaultConfig() *Config {
	return &Config{
		UserAgent:     "Mozilla/5.0 (compatible; PonsScraper/1.0)",
		Timeout:       10 * time.Second,
		MaxDepth:      3,
		RequestDelay:  1 * time.Second,
		MaxConcurrent: 2,
	}
}

// Metadata holds metadata information extracted from a webpage.
//
// This struct stores the extracted title and description from a web page's HTML content.
type Metadata struct {
	// Title is the content of the <title> tag
	Title string
	// Description is the content of the meta description tag
	Description string
}

// Scraper is responsible for scraping web content.
//
// It handles fetching web pages, extracting content and metadata,
// and crawling websites while respecting rate limits and timeouts.
type Scraper struct {
	// URL is the base URL to scrape
	URL string
	// Metadata contains extracted metadata from the scraped content
	Metadata Metadata
	// Content holds the parsed HTML content
	Content *html.Node
	// SubPaths contains all discovered paths during crawling
	SubPaths []string
	// Config contains all the configuration options for this scraper
	Config *Config
	// client is the HTTP client used for making requests
	client *http.Client
	// Rate limiting
	// lastRequestTime tracks the last request time per host for rate limiting
	lastRequestTime map[string]time.Time
	// requestSem is a semaphore channel to limit concurrent requests
	requestSem chan struct{}
	// mutex protects access to the lastRequestTime map
	mutex sync.Mutex
	// SubPathsHTMLContent stores the HTML content of each subpath
	SubPathsHTMLContent map[string]string
}

// New creates a new scraper with the given URL and configuration.
//
// This is the recommended way to create a new Scraper instance.
// It initializes all necessary internal structures and the HTTP client.
// If config is nil, default configuration will be used.
//
// Parameters:
//   - url: The base URL to scrape
//   - config: The configuration to use for this scraper (or nil for defaults)
//
// Returns:
//   - A new Scraper instance ready to use
func New(url string, config *Config) *Scraper {
	if config == nil {
		config = DefaultConfig()
	}

	client := &http.Client{
		Timeout: config.Timeout,
	}

	return &Scraper{
		URL:                 url,
		Config:              config,
		client:              client,
		lastRequestTime:     make(map[string]time.Time),
		requestSem:          make(chan struct{}, config.MaxConcurrent),
		SubPathsHTMLContent: make(map[string]string),
	}
}

// waitForRateLimit waits for rate limiting based on the host
func (s *Scraper) waitForRateLimit(host string) {
	// Acquire semaphore slot (limits concurrent requests)
	s.requestSem <- struct{}{}

	// Check and enforce per-host rate limiting
	s.mutex.Lock()
	lastReq, exists := s.lastRequestTime[host]
	now := time.Now()

	if exists {
		// Calculate time since last request
		elapsed := now.Sub(lastReq)

		// If not enough time has passed, sleep for the remaining duration
		if elapsed < s.Config.RequestDelay {
			s.mutex.Unlock()
			time.Sleep(s.Config.RequestDelay - elapsed)
			s.mutex.Lock()
		}
	}

	// Update last request time
	s.lastRequestTime[host] = time.Now()
	s.mutex.Unlock()
}

// GetContent fetches the content of the URL and parses it.
//
// This method retrieves the HTML content from the scraper's URL and parses it.
// The method applies rate limiting and respects the configured timeout.
//
// Returns:
//   - An error if the content cannot be fetched or parsed, nil otherwise
func (s *Scraper) GetContent() error {
	doc, _, err := s.fetchURL(s.URL)
	if err != nil {
		return fmt.Errorf("failed to fetch content: %w", err)
	}

	s.Content = doc
	return nil
}

// GetSubPathHTMLContent returns the HTML content of a subpath.
func (s *Scraper) GetSubPathHTMLContent() map[string]string {
	return s.SubPathsHTMLContent
}

// extractTitle extracts the title from an HTML node
func extractTitle(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
		return n.FirstChild.Data
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title := extractTitle(c); title != "" {
			return title
		}
	}

	return ""
}

// extractDescription extracts the meta description from an HTML node
func extractDescription(n *html.Node) string {
	if n.Type == html.ElementNode && n.Data == "meta" {
		var isDesc, hasContent bool
		var content string

		for _, a := range n.Attr {
			if a.Key == "name" && a.Val == "description" {
				isDesc = true
			}
			if a.Key == "content" {
				content = a.Val
				hasContent = true
			}
		}

		if isDesc && hasContent {
			return content
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if desc := extractDescription(c); desc != "" {
			return desc
		}
	}

	return ""
}

// GetMetadata extracts metadata (title, description) from the HTML content.
//
// This method extracts the title and description from the HTML content.
// It requires that GetContent has been called first to populate the Content field.
//
// Returns:
//   - An error if the content hasn't been fetched yet, nil otherwise
func (s *Scraper) GetMetadata() error {
	if s.Content == nil {
		return errors.New("content not found, call GetContent first")
	}

	// Extract metadata using helper functions
	title := extractTitle(s.Content)
	description := extractDescription(s.Content)

	// Store extracted metadata
	s.Metadata.Title = title
	s.Metadata.Description = description

	return nil
}

// GetAllPaths crawls the website and collects all paths.
//
// This method performs a depth-first crawl of the website starting from the base URL.
// It respects the MaxDepth configuration and only follows links within the same host.
// The method stores all discovered paths in the SubPaths field.
//
// Returns:
//   - An error if the crawling fails, nil otherwise
func (s *Scraper) GetAllPaths() error {
	parsedBase, err := url.Parse(s.URL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	// Initialize maps to track visited URLs and found paths
	visited := make(map[string]bool)
	paths := make(map[string]bool)

	// Start crawling from the base URL
	err = s.Crawl(parsedBase, parsedBase, paths, visited, 0)
	if err != nil {
		return fmt.Errorf("crawling failed: %w", err)
	}

	// Convert paths map to slice for easier access
	pathSlice := make([]string, 0, len(paths))
	for path := range paths {
		pathSlice = append(pathSlice, path)
	}

	s.SubPaths = pathSlice

	return nil
}

// fetchURL fetches the content of a URL and returns the HTML document and its string representation
func (s *Scraper) fetchURL(urlStr string) (*html.Node, string, error) {
	// Parse URL to get host for rate limiting
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, "", fmt.Errorf("invalid URL: %w", err)
	}

	// Apply rate limiting based on host
	s.waitForRateLimit(parsedURL.Host)
	defer func() { <-s.requestSem }() // Release semaphore when done

	// Create a request with context and user agent
	ctx, cancel := context.WithTimeout(context.Background(), s.Config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", s.Config.UserAgent)

	// Make HTTP GET request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	// Check response status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Check if response is HTML
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return nil, "", fmt.Errorf("not HTML content: %s", contentType)
	}

	// Read body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read body: %w", err)
	}

	// Parse HTML
	doc, err := html.Parse(bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, string(bodyBytes), nil
}

// extractLinks extracts all links from an HTML document
func extractLinks(doc *html.Node, baseURL *url.URL, visited map[string]bool) []*url.URL {
	var links []*url.URL

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					link := attr.Val

					// Parse the link
					parsedLink, err := url.Parse(link)
					if err != nil {
						continue // Skip invalid URLs
					}

					// Resolve relative URLs
					parsedLink = baseURL.ResolveReference(parsedLink)

					// Only include links within the same host and not yet visited
					if parsedLink.Host == baseURL.Host && !visited[parsedLink.String()] {
						links = append(links, parsedLink)
					}
				}
			}
		}

		// Traverse child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)
	return links
}

// Crawl recursively crawls a website starting from the given URL.
//
// This method implements a depth-first crawl of the website, following links
// within the same host up to the configured maximum depth. It respects rate
// limiting settings and tracks visited URLs to avoid cycles.
//
// Parameters:
//   - baseURL: The original base URL of the website
//   - currentURL: The current URL being crawled
//   - paths: A map to collect all unique paths found
//   - visited: A map of already visited URLs to avoid duplicates
//   - depth: The current crawl depth (0 for the starting URL)
//
// Returns:
//   - An error if the crawling fails catastrophically (individual page errors are logged but don't stop the crawl)
func (s *Scraper) Crawl(baseURL, currentURL *url.URL, paths, visited map[string]bool, depth int) error {
	// Check if we've reached the maximum crawl depth
	if depth > s.Config.MaxDepth {
		return nil
	}

	// Skip if already visited
	urlStr := currentURL.String()
	if visited[urlStr] {
		return nil
	}
	visited[urlStr] = true

	// Fetch and parse the URL
	doc, htmlContent, err := s.fetchURL(urlStr)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", urlStr, err)
	}

	// Extract path from current URL
	path := currentURL.Path
	if path == "" {
		path = "/"
	}
	paths[path] = true
	s.SubPathsHTMLContent[path] = htmlContent

	// Extract and process links
	links := extractLinks(doc, baseURL, visited)
	for _, link := range links {
		if err := s.Crawl(baseURL, link, paths, visited, depth+1); err != nil {
			// Log error but continue crawling other links
			fmt.Printf("Error crawling %s: %v\n", link.String(), err)
		}
	}

	return nil
}

// ScrapeContent fetches the URL and scrapes the main content.
//
// This method fetches the content of the scraper's URL and extracts the main text content from the page.
// It utilizes the existing scraper infrastructure for rate limiting and error handling.
//
// Returns:
//   - The extracted text content or an error if scraping fails
func (s *Scraper) ScrapeContent() (string, error) {
	// Fetch the content
	err := s.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to fetch content: %w", err)
	}

	// Find the main content node
	mainNode := findMainContentNode(s.Content)
	if mainNode == nil {
		return "", errors.New("could not find main content")
	}

	// Extract text from the main content node
	return extractText(mainNode), nil
}

// findMainContentNode finds the main content node in an HTML document.
//
// It looks for common content containers like <main>, <article>, or elements with
// id="content" or id="main". If none are found, it falls back to the <body> element.
//
// Parameters:
//   - doc: The HTML document to search in
//
// Returns:
//   - The main content node or nil if not found
func findMainContentNode(doc *html.Node) *html.Node {
	// Find the main content node
	var mainNode *html.Node
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "main" || n.Data == "article" {
				mainNode = n
				return
			}
			for _, a := range n.Attr {
				if a.Key == "id" && (a.Val == "content" || a.Val == "main") {
					mainNode = n
					return
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
			if mainNode != nil {
				return
			}
		}
	}
	f(doc)

	// If no main content node is found, use the body
	if mainNode == nil {
		var bodyNode *html.Node
		var findBody func(*html.Node)
		findBody = func(n *html.Node) {
			if n.Type == html.ElementNode && n.Data == "body" {
				bodyNode = n
				return
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				findBody(c)
				if bodyNode != nil {
					return
				}
			}
		}
		findBody(doc)
		mainNode = bodyNode
	}

	return mainNode
}

// extractText recursively extracts all text from an HTML node.
//
// This function traverses the HTML node tree and extracts all text content,
// skipping script and style elements. It concatenates text nodes with spaces.
//
// Parameters:
//   - n: The HTML node to extract text from
//
// Returns:
//   - The extracted text content
func extractText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
		return ""
	}

	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += extractText(c) + " "
	}
	return text
}
