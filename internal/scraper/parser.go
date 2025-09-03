package scraper

import (
	htm "github.com/JohannesKaufmann/html-to-markdown/v2"
)

type Parser struct{}

// ToMarkdown converts HTML content to Markdown format.
func (p *Parser) ToMarkdown(htmlString string) (string, error) {
	markdown, err := htm.ConvertString(htmlString)
	if err != nil {
		return "", err
	}
	return markdown, nil
}
