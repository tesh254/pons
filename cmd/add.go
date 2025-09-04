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
		verbose, _ := cmd.Flags().GetBool("verbose")

		fmt.Println(url, dbPath, workerURL)

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

			// if verbose {
			// 	fmt.Printf("    - Markdown content for %s:\n%s\n", subpath, markdownContent)
			// }

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

			if err := ponsAPI.UpsertDocument(url, subpath, markdownContent, checksum, embeddings); err != nil {
				log.Printf("Failed to store document for %s: %v", subpath, err)
				continue
			}

			if verbose {
				fmt.Printf("    - Successfully added %s\n", subpath)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
	addCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
}
