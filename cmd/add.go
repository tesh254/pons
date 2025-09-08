package cmd

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tesh254/pons/internal/api"
	"github.com/tesh254/pons/internal/llm"
	"github.com/tesh254/pons/internal/scraper"
	"github.com/tesh254/pons/internal/storage"
)

var addCmd = &cobra.Command{
	Use:   "add [url_or_file_path]",
	Short: "Scrapes a URL or reads a file, generates embeddings, and stores the content",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		input := args[0]
		context, _ := cmd.Flags().GetString("context")
		verbose, _ := cmd.Flags().GetBool("verbose")

		dbPath := viper.GetString("db")
		workerURL := viper.GetString("worker-url")

		fmt.Println(input, dbPath, workerURL, context)

		// Initialize storage
	
st, err := storage.NewStorage(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		defer st.Close()

		// Initialize LLM
		if workerURL == "" {
			log.Fatalf("worker-url is required for add command")
		}
		emb := llm.NewEmbeddings(workerURL)

		// Initialize API
		ponsAPI := api.NewAPI(st, emb)

		var contentToStore string
		var docURL string
		var docTitle string
		var docDescription string
		var sourceType string

		if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
			// It's a URL, proceed with scraping
			url := input
			sourceType = "web_scrape"
			config := scraper.DefaultConfig()
			config.Verbose = verbose // Set verbosity for scraper
			s := scraper.New(url, config)
			if err := s.GetContent(); err != nil {
				log.Fatalf("Failed to get content for metadata: %v", err)
			}
			if err := s.GetMetadata(); err != nil {
				log.Fatalf("Failed to get metadata: %v", err)
			}
			if err := s.GetAllPaths(); err != nil {
				log.Fatalf("Failed to get all paths: %v", err)
			}

			// Process and store each page
			if verbose {
				fmt.Println("Processing and storing documents...")
			}
			parser := &scraper.Parser{}
			for subpath, content := range s.SubPathsHTMLContent {
				if verbose {
					fmt.Printf("  - Processing %s\n", subpath)
				}

				// Convert HTML to Markdown
				markdownContent, err := parser.ToMarkdown(content)
				if err != nil {
					log.Printf("Failed to convert HTML to markdown for %s: %v", subpath, err)
					continue
				}

				// Generate embeddings
				if verbose {
					fmt.Printf("    - Generating embeddings for %s\n", subpath)
				}
				embeddings, err := emb.GenerateEmbeddings(markdownContent)
				if err != nil {
					log.Printf("Failed to generate embeddings for %s: %v", subpath, err)
					continue
				}

				// Calculate checksum
				checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(markdownContent)))

				// Store document
			if verbose {
				fmt.Printf("    - Storing document in bbolt: %s\n", subpath)
			}

				if err := ponsAPI.UpsertDocument(url, subpath, s.Metadata.Title, s.Metadata.Description, markdownContent, checksum, context, sourceType, embeddings); err != nil {
					log.Printf("Failed to store document for %s: %v", subpath, err)
					continue
				}

				if verbose {
					fmt.Printf("    - Successfully added %s\n", subpath)
				}
			}
		} else {
			// It's a file path, read content directly
			filePath := input
			sourceType = "file_read"
			fileContent, err := os.ReadFile(filePath)
			if err != nil {
				log.Fatalf("Failed to read file %s: %v", filePath, err)
			}
			contentToStore = string(fileContent)
			docURL = "file://" + filePath // Use a file URL scheme
			docTitle = filepath.Base(filePath) // Use filename as title
			docDescription = ""

			// Generate embeddings
			if verbose {
				fmt.Printf("  - Generating embeddings for file %s\n", filePath)
			}
			embeddings, err := emb.GenerateEmbeddings(contentToStore)
			if err != nil {
				log.Fatalf("Failed to generate embeddings for file %s: %v", filePath, err)
			}

			// Calculate checksum
			checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(contentToStore)))

			// Store document
			if verbose {
				fmt.Printf("  - Storing document for file %s\n", filePath)
			}

			if err := ponsAPI.UpsertDocument(docURL, "", docTitle, docDescription, contentToStore, checksum, context, sourceType, embeddings); err != nil {
				log.Fatalf("Failed to store document for file %s: %v", filePath, err)
			}

			if verbose {
				fmt.Printf("  - Successfully added file %s\n", filePath)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	addCmd.Flags().StringP("context", "c", "default", "Context for the scraped documents")
}
