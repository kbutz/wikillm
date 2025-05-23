package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// WikipediaPage represents a page in the Wikipedia dump
type WikipediaPage struct {
	Title   string `xml:"title"`
	ID      string `xml:"id"`
	Content string `xml:"revision>text"`
}

// WikipediaIndex manages the indexing and searching of Wikipedia content
type WikipediaIndex struct {
	index bleve.Index
	path  string
}

// NewWikipediaIndex creates a new Wikipedia index
func NewWikipediaIndex(indexPath string) (*WikipediaIndex, error) {
	// Check if the index already exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		// Create the directory if it doesn't exist
		if err := os.MkdirAll(indexPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create index directory: %w", err)
		}

		// Create a new index
		indexMapping := buildIndexMapping()
		index, err := bleve.New(filepath.Join(indexPath, "wikipedia.bleve"), indexMapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %w", err)
		}

		return &WikipediaIndex{
			index: index,
			path:  indexPath,
		}, nil
	}

	// Open the existing index
	index, err := bleve.Open(filepath.Join(indexPath, "wikipedia.bleve"))
	if err != nil {
		return nil, fmt.Errorf("failed to open index: %w", err)
	}

	return &WikipediaIndex{
		index: index,
		path:  indexPath,
	}, nil
}

// buildIndexMapping creates the mapping for the Wikipedia index
func buildIndexMapping() mapping.IndexMapping {
	// Create a default mapping
	indexMapping := bleve.NewIndexMapping()

	// Create a document mapping for Wikipedia pages
	pageMapping := bleve.NewDocumentMapping()

	// Add field mappings
	titleFieldMapping := bleve.NewTextFieldMapping()
	titleFieldMapping.Store = true
	titleFieldMapping.Index = true
	titleFieldMapping.IncludeTermVectors = true
	titleFieldMapping.IncludeInAll = true
	pageMapping.AddFieldMappingsAt("title", titleFieldMapping)

	contentFieldMapping := bleve.NewTextFieldMapping()
	contentFieldMapping.Store = true
	contentFieldMapping.Index = true
	contentFieldMapping.IncludeTermVectors = true
	contentFieldMapping.IncludeInAll = true
	pageMapping.AddFieldMappingsAt("content", contentFieldMapping)

	// Add the document mapping to the index mapping
	indexMapping.AddDocumentMapping("page", pageMapping)

	return indexMapping
}

// IndexWikipediaDump indexes a Wikipedia XML dump file
func (wi *WikipediaIndex) IndexWikipediaDump(dumpPath string) error {
	// Open the dump file
	file, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("failed to open dump file: %w", err)
	}
	defer file.Close()

	// Create a batch indexer
	batch := wi.index.NewBatch()
	batchSize := 1000
	batchCount := 0
	totalIndexed := 0

	// Create an XML decoder
	decoder := xml.NewDecoder(file)
	var inPage bool
	var currentPage WikipediaPage

	// Process the XML dump
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
				if se.Name.Local == "title" {
					var title string
					if err := decoder.DecodeElement(&title, &se); err != nil {
						log.Printf("Error decoding title: %v", err)
						continue
					}
					currentPage.Title = title
				} else if se.Name.Local == "id" && currentPage.ID == "" {
					var id string
					if err := decoder.DecodeElement(&id, &se); err != nil {
						log.Printf("Error decoding ID: %v", err)
						continue
					}
					currentPage.ID = id
				} else if se.Name.Local == "text" {
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
				// Index the page
				if currentPage.Title != "" && currentPage.ID != "" {
					// Clean up the content (remove wiki markup)
					cleanContent := cleanWikiMarkup(currentPage.Content)

					// Create a document to index
					doc := map[string]interface{}{
						"title":   currentPage.Title,
						"id":      currentPage.ID,
						"content": cleanContent,
					}

					// Add the document to the batch
					if err := batch.Index(currentPage.ID, doc); err != nil {
						log.Printf("Error indexing document: %v", err)
						continue
					}

					batchCount++
					totalIndexed++

					// Commit the batch if it's full
					if batchCount >= batchSize {
						if err := wi.index.Batch(batch); err != nil {
							log.Printf("Error committing batch: %v", err)
						}
						batch = wi.index.NewBatch()
						batchCount = 0
						log.Printf("Indexed %d pages", totalIndexed)
					}
				}
				inPage = false
			}
		}
	}

	// Commit any remaining documents
	if batchCount > 0 {
		if err := wi.index.Batch(batch); err != nil {
			log.Printf("Error committing final batch: %v", err)
		}
	}

	log.Printf("Indexing complete. Total pages indexed: %d", totalIndexed)
	return nil
}

// cleanWikiMarkup removes wiki markup from the content
func cleanWikiMarkup(content string) string {
	// This is a very basic cleanup - a real implementation would be more sophisticated
	content = strings.ReplaceAll(content, "[[", "")
	content = strings.ReplaceAll(content, "]]", "")
	content = strings.ReplaceAll(content, "{{", "")
	content = strings.ReplaceAll(content, "}}", "")
	content = strings.ReplaceAll(content, "'''", "")
	content = strings.ReplaceAll(content, "''", "")

	// Remove references
	content = strings.ReplaceAll(content, "<ref>", "")
	content = strings.ReplaceAll(content, "</ref>", "")

	return content
}

// Search searches the Wikipedia index for the given query
func (wi *WikipediaIndex) Search(query string, limit int) ([]map[string]interface{}, error) {
	// Create a search request
	searchRequest := bleve.NewSearchRequest(bleve.NewQueryStringQuery(query))
	searchRequest.Fields = []string{"title", "content"}
	searchRequest.Size = limit
	searchRequest.Highlight = bleve.NewHighlight()

	// Execute the search
	searchResults, err := wi.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search error: %w", err)
	}

	// Process the results
	results := make([]map[string]interface{}, 0, len(searchResults.Hits))
	for _, hit := range searchResults.Hits {
		result := map[string]interface{}{
			"id":    hit.ID,
			"score": hit.Score,
		}

		// Add fields
		for field, value := range hit.Fields {
			result[field] = value
		}

		// Add highlights
		if len(hit.Fragments) > 0 {
			result["highlights"] = hit.Fragments
		}

		results = append(results, result)
	}

	return results, nil
}

// Close closes the index
func (wi *WikipediaIndex) Close() error {
	return wi.index.Close()
}