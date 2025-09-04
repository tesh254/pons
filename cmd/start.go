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
		httpAddress := viper.GetString("http-address")
		// Initialize storage
		st, err := storage.NewStorage(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		defer st.Close()

		// Initialize LLM
		emb := llm.NewEmbeddings(workerURL)

		// Initialize API
		ponsAPI := api.NewAPI(st, emb)

		// Start MCP server
		log.Println("Starting MCP server...")
		mcpServer := &core.Core{}
		if err := mcpServer.StartServer(ponsAPI, httpAddress); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
	startCmd.Flags().String("http-address", "localhost:9014", "HTTP address to listen on")
	startCmd.Flags().String("transport", "stdio", "Transport type (stdio or http)")
	viper.BindPFlag("http-address", startCmd.Flags().Lookup("http-address"))
	viper.BindPFlag("transport", startCmd.Flags().Lookup("transport"))
}
