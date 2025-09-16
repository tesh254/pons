package cmd

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tesh254/pons/internal/storage"
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Deletes all documents from the database",
	Run: func(cmd *cobra.Command, args []string) {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("\033[31mWARNING: This will delete all data from the database and is not recoverable.\033[0m")
		fmt.Print("Are you sure you want to continue? (yes/no): ")

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read response: %v", err)
		}

		if strings.TrimSpace(strings.ToLower(response)) != "yes" {
			fmt.Println("Clean operation cancelled.")
			return
		}

		home, _ := os.UserHomeDir()
		dbPath := filepath.Join(home, ".pons_data", "pons.db")

		st, err := storage.NewStorage(dbPath)
		if err != nil {
			log.Fatalf("Failed to initialize storage: %v", err)
		}
		defer st.Close()

		if err := st.Clean(); err != nil {
			log.Fatalf("Failed to clean database: %v", err)
		}

		fmt.Println("Database cleaned successfully.")
	},
}

func init() {
	rootCmd.AddCommand(cleanCmd)
}
