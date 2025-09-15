package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/tesh254/pons/internal/storage"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "Lists all documents in the database",
	Run: func(cmd *cobra.Command, args []string) {
		home, _ := os.UserHomeDir()
		dbPath := filepath.Join(home, ".pons_data", "pons.db")

		st, err := storage.NewStorage(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		defer st.Close()

		docs, err := st.ListDocuments("", 1000)
		if err != nil {
			log.Fatalf("Failed to list documents: %v", err)
		}

		if len(docs) == 0 {
			fmt.Println("No documents found.")
			return
		}

		for _, doc := range docs {
			fmt.Printf("URL: %s\nSource Type: %s\nChecksum: %s\nContent Length: %d\nEmbeddings Length: %d\n\n", doc.URL, doc.SourceType, doc.Checksum, len(doc.Content), len(doc.Embeddings))
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
