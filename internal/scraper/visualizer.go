package scraper

import (
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

func (s *Scraper) displayInitBanner() {
	if s.Verbose {
		green := color.New(color.FgGreen).SprintFunc()
		banner := "==============================================================================\n"
		banner += green("       🌐 Web Scraper Initialized 🌐\n")
		banner += "==============================================================================\n"
		banner += fmt.Sprintf("Target URL: %s\n", s.URL)
		banner += "Configuration:\n"
		banner += fmt.Sprintf("  - Max Depth: %d\n", s.Config.MaxDepth)
		banner += fmt.Sprintf("  - Request Delay: %s\n", s.Config.RequestDelay)
		banner += fmt.Sprintf("  - Max Concurrent: %d\n", s.Config.MaxConcurrent)
		banner += "=============================================================================="
		fmt.Println(banner)
	}
}

func (s *Scraper) displayMetadata() {
	if s.Verbose {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleLight) // Consistent with displaySubpathResults
		t.AppendHeader(table.Row{"Field", "Value"})
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, Align: text.AlignLeft, WidthMax: 20}, // Limit label width
			{Number: 2, Align: text.AlignLeft, WidthMax: 80}, // Limit content width
		})

		// Add Title and Description rows
		t.AppendRow(table.Row{"Title", s.Metadata.Title})
		t.AppendRow(table.Row{"Description", s.Metadata.Description})
		t.AppendSeparator() // Adds row lines between entries
		t.Render()
	}
}

func (s *Scraper) displayCrawlStartBanner() {
	if s.Verbose {
		green := color.New(color.FgGreen).SprintFunc()
		banner := "==============================================================================\n"
		banner += "        " + green("🚀 Starting Website Crawl 🚀") + "\n"
		banner += "==============================================================================\n"
		banner += fmt.Sprintf("Base URL: %s\n", s.URL)
		banner += fmt.Sprintf("Max Depth: %d\n", s.Config.MaxDepth)
		banner += "=============================================================================="
		fmt.Println(banner)
	}
}

func (s *Scraper) displayCrawlEndBanner() {
	if s.Verbose {
		green := color.New(color.FgGreen).SprintFunc()
		banner := "==============================================================================\n"
		banner += "         " + green("✅ Crawl Complete ✅") + "\n"
		banner += "==============================================================================\n"
		banner += fmt.Sprintf("Found %d subpaths.\n", len(s.SubPaths))
		banner += "=============================================================================="
		fmt.Println(banner)
	}
}

func (s *Scraper) displaySubpathResults() {
	if s.Verbose {
		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.SetStyle(table.StyleLight) // Use light style for minimal borders
		t.AppendHeader(table.Row{"Path", "HTML Length"})
		t.SetColumnConfigs([]table.ColumnConfig{
			{Number: 1, Align: text.AlignLeft}, // Changed to text.AlignLeft
			{Number: 2, Align: text.AlignLeft}, // Changed to text.AlignLeft
		})

		for _, path := range s.SubPaths {
			length := len(s.SubPathsHTMLContent[path])
			t.AppendRow(table.Row{path, strconv.Itoa(length)})
		}
		t.AppendSeparator() // Adds row lines between entries
		t.Render()
	}
}

func (s *Scraper) displayError(err error) {
	if s.Verbose {
		red := color.New(color.FgRed).SprintFunc()
		box := "┌────── " + red("⚠ Error") + " ──────┐\n"
		box += fmt.Sprintf("│ %-20s │\n", err.Error())
		box += "└─────────────────────┘"
		fmt.Println(box)
	}
}

func (s *Scraper) startSpinner(message string) chan struct{} {
	done := make(chan struct{})
	if s.Verbose {
		os.Stdout.Sync()
		go func() {
			spinner := `|/-\`
			i := 0
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			var mu sync.Mutex
			for {
				select {
				case <-ticker.C:
					mu.Lock()
					fmt.Fprintf(os.Stdout, "\r%s... [%s]", color.YellowString("%s", message), string(spinner[i]))
					os.Stdout.Sync()
					mu.Unlock()
					i = (i + 1) % len(spinner)
				case <-done:
					mu.Lock()
					fmt.Fprintf(os.Stdout, "\r%s... [%s]\n", color.GreenString("%s", message), "✔")
					os.Stdout.Sync()
					mu.Unlock()
					return
				}
			}
		}()
	}
	return done
}
