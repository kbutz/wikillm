package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/tmc/langchaingo/schema"
)

// WikipediaPage represents a page in the Wikipedia dump
type WikipediaPage struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Content string `xml:"revision>text"`
}

// CleanWikiMarkup removes wiki markup from the content
func CleanWikiMarkup(content string) string {
	// Basic cleanup - a real implementation would be more sophisticated
	content = strings.ReplaceAll(content, "[[", "")
	content = strings.ReplaceAll(content, "]]", "")
	content = strings.ReplaceAll(content, "{{", "")
	content = strings.ReplaceAll(content, "}}", "")
	content = strings.ReplaceAll(content, "'''", "")
	content = strings.ReplaceAll(content, "''", "")
	content = strings.ReplaceAll(content, "<ref>", "")
	content = strings.ReplaceAll(content, "</ref>", "")

	// Remove common Wikipedia templates and formatting
	lines := strings.Split(content, "\n")
	var cleanLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#REDIRECT") ||
			strings.HasPrefix(line, "{{") || strings.HasPrefix(line, "|}") ||
			strings.HasPrefix(line, "|") {
			continue
		}
		cleanLines = append(cleanLines, line)
	}

	return strings.Join(cleanLines, " ")
}

// IndexWikipediaDump indexes a Wikipedia XML dump file using the new API
func (r *RAGPipeline) IndexWikipediaDump(dumpPath string) error {
	ctx := context.Background()

	file, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("failed to open dump file: %w", err)
	}
	defer file.Close()

	batchSize := 50
	var documents []schema.Document
	totalIndexed := 0

	decoder := xml.NewDecoder(file)
	var inPage bool
	var currentPage WikipediaPage

	log.Println("Starting Wikipedia indexing...")

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error decoding XML: %w", err)
		}

		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == "page" {
				inPage = true
				currentPage = WikipediaPage{}
			} else if inPage {
				switch se.Name.Local {
				case "title":
					var title string
					if err := decoder.DecodeElement(&title, &se); err != nil {
						log.Printf("Error decoding title: %v", err)
						continue
					}
					currentPage.Title = title
				case "id":
					if currentPage.ID == "" { // Only take the first ID (page ID, not revision ID)
						var id string
						if err := decoder.DecodeElement(&id, &se); err != nil {
							log.Printf("Error decoding ID: %v", err)
							continue
						}
						currentPage.ID = id
					}
				case "text":
					var content string
					if err := decoder.DecodeElement(&content, &se); err != nil {
						log.Printf("Error decoding content: %v", err)
						continue
					}
					currentPage.Content = content
				}
			}
		case xml.EndElement:
			if se.Name.Local == "page" && inPage {
				if currentPage.Title != "" && currentPage.ID != "" && currentPage.Content != "" {
					// Clean up the content
					cleanContent := CleanWikiMarkup(currentPage.Content)

					// Skip empty or very short content
					if len(cleanContent) < 100 {
						inPage = false
						continue
					}

					// Create document using the new schema
					doc := schema.Document{
						PageContent: cleanContent,
						Metadata: map[string]any{
							"id":     currentPage.ID,
							"title":  currentPage.Title,
							"source": "wikipedia",
						},
					}

					documents = append(documents, doc)
					totalIndexed++

					// Process batch when full
					if len(documents) >= batchSize {
						if err := r.ProcessBatch(ctx, documents); err != nil {
							return fmt.Errorf("error processing batch: %w", err)
						} else {
							log.Printf("Indexed %d pages", totalIndexed)
						}
						documents = documents[:0] // Reset slice
					}
				}
				inPage = false
			}
		}
	}

	// Process remaining documents
	if len(documents) > 0 {
		if err := r.ProcessBatch(ctx, documents); err != nil {
			return fmt.Errorf("error processing final batch: %w", err)
		}
	}

	log.Printf("Indexing complete. Total pages indexed: %d", totalIndexed)
	return nil
}