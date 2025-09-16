package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tesh254/pons/internal/api"
	"github.com/tesh254/pons/internal/llm"
	"github.com/tesh254/pons/internal/storage"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [url]",
	Short: "Deletes a document from the database",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		home, _ := os.UserHomeDir()
		dbPath := filepath.Join(home, ".pons_data", "pons.db")
		workerURL := "https://vectors.madebyknnls.com"

		st, err := storage.NewStorage(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		defer st.Close()

		emb := llm.NewEmbeddings(workerURL)
		ponsAPI := api.NewAPI(st, emb)

		context, _ := cmd.Flags().GetString("context") // Retrieve context flag
		if err := ponsAPI.DeleteDocument(url, context); err != nil {
			log.Fatalf("Failed to delete document: %v", err)
		}

		fmt.Printf("Document with URL '%s' deleted successfully.\n", url)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringP("context", "c", "", "Context of the document to delete")
}
