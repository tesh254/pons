package cmd

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tesh254/pons/internal/api"
	"github.com/tesh254/pons/internal/llm"
	"github.com/tesh254/pons/internal/storage"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Searches the knowledge base for relevant documents",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		numResults, _ := cmd.Flags().GetInt("num-results")
		context, _ := cmd.Flags().GetString("context")
		verbose, _ := cmd.Flags().GetBool("verbose")

		dbPath := viper.GetString("db")
		workerURL := viper.GetString("worker-url")

		if verbose {
			fmt.Printf("Searching for: %s\n", query)
			fmt.Printf("Database path: %s\n", dbPath)
			fmt.Printf("Worker URL: %s\n", workerURL)
			fmt.Printf("Number of results: %d\n", numResults)
			fmt.Printf("Context: %s\n", context)
		}

		// Initialize storage
		st, err := storage.NewStorage(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		defer st.Close()

		// Initialize LLM
		if workerURL == "" {
			log.Fatalf("worker-url is required for search command")
		}
		emb := llm.NewEmbeddings(workerURL)

		// Initialize API
		ponsAPI := api.NewAPI(st, emb)

		// Generate embedding for the query
		if verbose {
			fmt.Println("Generating embeddings for query...")
		}
		queryEmbedding, err := emb.GenerateEmbeddings(query)
		if err != nil {
			log.Fatalf("Failed to generate embeddings for query: %v", err)
		}

		// Perform search
		if verbose {
			fmt.Println("Performing search...")
		}
		results, err := ponsAPI.Search(queryEmbedding, numResults, context)
		if err != nil {
			if err.Error() == "no documents in storage to search" {
				fmt.Println("No documents found in storage for the provided context.")
				return
			}
			log.Fatalf("Search failed: %v", err)
		}

		if len(results) == 0 {
			fmt.Println("No relevant documents found.")
			return
		}

		fmt.Println("\nSearch Results:")
		for i, result := range results {
			fmt.Printf("%d. URL: %s (Score: %.4f)\n", i+1, result.Doc.URL, result.Score)
			// Optionally print title/description/content snippet
			if verbose {
				fmt.Printf("   Title: %s\n", result.Doc.Title)
				fmt.Printf("   Description: %s\n", result.Doc.Description)
				// fmt.Printf("   Content Snippet: %s...\n", result.Doc.Content[:min(len(result.Doc.Content), 200)])
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().IntP("num-results", "n", 5, "Number of search results to return")
	searchCmd.Flags().StringP("context", "c", "", "Context to search within (e.g., 'shopify-admin')")
	searchCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
}

