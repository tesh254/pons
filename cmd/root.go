package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "pons",
	Short: "Pons is a tool for creating and querying a local knowledge base.",
	Long:  `Pons is a CLI tool that allows you to scrape websites, generate embeddings, and store them in a local vector database. You can then query the database using natural language.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pons/config.yaml)")

	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	rootCmd.PersistentFlags().String("db", filepath.Join(home, ".pons_data", "pons.db"), "Path to the database file")
	rootCmd.PersistentFlags().String("worker-url", "https://vectors.madebyknnls.com", "Cloudflare worker URL for embeddings")

	viper.BindPFlag("db", rootCmd.PersistentFlags().Lookup("db"))
	viper.BindPFlag("worker-url", rootCmd.PersistentFlags().Lookup("worker-url"))
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		configPath := filepath.Join(home, ".pons")
		viper.AddConfigPath(configPath)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// Create config file if it doesn't exist
		if err := os.MkdirAll(configPath, os.ModePerm); err != nil {
			fmt.Println("Error creating config directory:", err)
			os.Exit(1)
		}
		configFile := filepath.Join(configPath, "config.yaml")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			if err := viper.SafeWriteConfig(); err != nil {
				// if we get "no configuration file found" error, we can ignore it
				if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
					fmt.Println("Error writing config file:", err)
					os.Exit(1)
				}
			}
		}
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
