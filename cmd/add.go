package cmd

import (
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tesh254/pons/internal/api"
	"github.com/tesh254/pons/internal/llm"
	"github.com/tesh254/pons/internal/scraper"
	"github.com/tesh254/pons/internal/storage"
)

var addCmd = &cobra.Command{
	Use:   "add [url]",
	Short: "Scrapes a URL, generates embeddings, and stores the content",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		dbPath := viper.GetString("db")
		workerURL := viper.GetString("worker-url")

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

		// Scrape the URL
		config := scraper.DefaultConfig()
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
		for subpath, content := range s.SubPathsHTMLContent {
			var parser scraper.Parser
			// Convert HTML to Markdown
			markdownContent, err := parser.ToMarkdown(content)
			if err != nil {
				log.Printf("Failed to convert HTML to markdown for %s: %v", subpath, err)
				continue
			}

			// Generate embeddings
			embeddings, err := emb.GenerateEmbeddings(markdownContent)
			if err != nil {
				log.Printf("Failed to generate embeddings for %s: %v", subpath, err)
				continue
			}

			// Calculate checksum
			checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(markdownContent)))

			// Store document
			if err := ponsAPI.StoreDocument(subpath, markdownContent, checksum, embeddings); err != nil {
				log.Printf("Failed to store document for %s: %v", subpath, err)
				continue
			}

			fmt.Printf("Successfully added %s\n", subpath)
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
