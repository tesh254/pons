package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver/v4"
	"github.com/google/go-github/v30/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/tesh254/pons/internal/constants"
	"github.com/tesh254/pons/internal/version"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:     "pons",
	Aliases: []string{"pn"},
	Short:   "Pons is a tool for creating and querying a local knowledge base.",
	Long:    `Pons is a CLI tool that allows you to scrape websites, generate embeddings, and store them in a local vector database. You can then query the database using natural language.`,
	Version: constants.VERSION(),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle version flag specially to show detailed info
		if versionFlag, _ := cmd.Flags().GetBool("version"); versionFlag {
			fmt.Println(constants.DETAILED_VERSION())
			return nil
		}

		if cmd.Flags().NFlag() == 0 && len(args) == 0 {
			fmt.Print(constants.ASCII)
			fmt.Println(constants.CurrentOSWithVersion())
			fmt.Printf("\n%s\n", constants.GetReleaseInfo())
		}
		return nil
	},
}

// Version command with multiple output formats
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long: `Show version information for migraine.

This command displays version information extracted automatically from 
the Go build system, including Git commit, build date, and more.`,
	Run: func(cmd *cobra.Command, args []string) {
		jsonFlag, _ := cmd.Flags().GetBool("json")
		shortFlag, _ := cmd.Flags().GetBool("short")
		commitFlag, _ := cmd.Flags().GetBool("commit")

		switch {
		case jsonFlag:
			fmt.Println(version.GetJSONVersion())
		case shortFlag:
			fmt.Println(version.GetShortVersion())
		case commitFlag:
			fmt.Println(version.GetVersionWithCommit())
		default:
			fmt.Println(version.GetDetailedVersion())

			// Add extra info for development builds
			if version.IsDevelopment() {
				fmt.Printf("\n%sNote:%s This is a development build.\n",
					"\033[33m", "\033[0m")
			}
		}
	},
}

// Build info command for detailed build information
var buildInfoCmd = &cobra.Command{
	Use:   "buildinfo",
	Short: "Show detailed build information",
	Long:  `Show comprehensive build information including module details, VCS info, and build settings.`,
	Run: func(cmd *cobra.Command, args []string) {
		info := version.GetBuildInfo()

		fmt.Printf("Build Information:\n")
		fmt.Printf("==================\n")
		fmt.Printf("Version:      %s\n", info.Version)
		fmt.Printf("Git Commit:   %s\n", info.GitCommit)
		if info.GitTag != "unknown" {
			fmt.Printf("Git Tag:      %s\n", info.GitTag)
		}
		fmt.Printf("Build Date:   %s\n", info.BuildDate)
		fmt.Printf("Go Version:   %s\n", info.GoVersion)
		fmt.Printf("Platform:     %s\n", info.Platform)
		fmt.Printf("Compiler:     %s\n", info.Compiler)
		fmt.Printf("Modified:     %t\n", info.IsModified)
		if info.ModulePath != "" {
			fmt.Printf("Module Path:  %s\n", info.ModulePath)
		}
		if info.ModuleSum != "" {
			fmt.Printf("Module Sum:   %s\n", info.ModuleSum)
		}

		// Show build type
		fmt.Printf("\nBuild Type:   ")
		if version.IsRelease() {
			fmt.Printf("%sRelease%s\n", "\033[32m", "\033[0m")
		} else {
			fmt.Printf("%sDevelopment%s\n", "\033[33m", "\033[0m")
		}
	},
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

	// Version command flags
	versionCmd.Flags().Bool("json", false, "Output version information in JSON format")
	versionCmd.Flags().BoolP("short", "s", false, "Output short version only")
	versionCmd.Flags().BoolP("commit", "c", false, "Output version with commit hash")
	rootCmd.AddCommand(buildInfoCmd)
	rootCmd.AddCommand(versionCmd)

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

	checkVersion()
}

func checkVersion() {
	client := github.NewClient(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	release, _, err := client.Repositories.GetLatestRelease(ctx, "tesh254", "pons")
	if err != nil {
		return
	}

	latestVersion, err := semver.Parse(release.GetTagName()[1:])
	if err != nil {
		return
	}

	currentVersionStr := version.GetVersion()
	currentVersion, err := semver.Parse(currentVersionStr[1:])
	if err != nil {
		return
	}

	if latestVersion.LE(currentVersion) {
		return
	}

	fmt.Printf("\nA new version of pons is available: %s\n", latestVersion)

	exe, err := os.Executable()
	if err != nil {
		return
	}

	var updateInstruction string
	if strings.Contains(exe, "brew") {
		updateInstruction = "To update, run: brew upgrade pons"
	} else {
		updateInstruction = "To update, run: curl -sSL https://raw.githubusercontent.com/tesh254/pons/main/install.sh | sh"
	}

	border := strings.Repeat("─", len(updateInstruction)+4)
	fmt.Println("┌" + border + "┐")
	fmt.Println("│  " + updateInstruction + "  │")
	fmt.Println("└" + border + "┘")
}