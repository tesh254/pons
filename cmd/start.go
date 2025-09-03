package cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tesh254/pons/internal/api"
	"github.com/tesh254/pons/internal/core"
	"github.com/tesh254/pons/internal/llm"
	"github.com/tesh254/pons/internal/storage"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the MCP server",
	Run: func(cmd *cobra.Command, args []string) {
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
			log.Fatalf("worker-url is required for start command")
		}
		emb := llm.NewEmbeddings(workerURL)

		// Initialize API
		ponsAPI := api.NewAPI(st, emb)

		// Start MCP server
		log.Println("Starting MCP server...")
		var mcpServer core.Core
		if err := mcpServer.StartServer(ponsAPI); err != nil {
			log.Fatalf("Failed to start MCP server: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
