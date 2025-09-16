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

var contextsCmd = &cobra.Command{
	Use:   "contexts",
	Short: "Lists all unique contexts in the knowledge base",
	Run: func(cmd *cobra.Command, args []string) {
		dbPath := viper.GetString("db")
		workerURL := viper.GetString("worker-url") // workerURL is needed for API initialization

		// Initialize storage
		st, err := storage.NewStorage(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		defer st.Close()

		// Initialize LLM (even if not directly used by GetContexts, API requires it)
		emb := llm.NewEmbeddings(workerURL)

		// Initialize API
		ponsAPI := api.NewAPI(st, emb)

		contexts, err := ponsAPI.GetContexts()
		if err != nil {
			log.Fatalf("Failed to retrieve contexts: %v", err)
		}

		if len(contexts) == 0 {
			fmt.Println("No contexts found in the knowledge base.")
			return
		}

		fmt.Println("Available Contexts:")
		for _, context := range contexts {
			fmt.Printf("- %s\n", context)
		}
	},
}

func init() {
	rootCmd.AddCommand(contextsCmd)
}
