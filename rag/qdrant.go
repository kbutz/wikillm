package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

// QdrantCollectionInfo represents Qdrant collection information
type QdrantCollectionInfo struct {
	Result struct {
		Config struct {
			Params struct {
				Vectors struct {
					Size     int    `json:"size"`
					Distance string `json:"distance"`
				} `json:""`
			} `json:"params"`
		} `json:"config"`
	} `json:"result"`
}

// GetQdrantCollectionInfo gets information about an existing collection
func GetQdrantCollectionInfo(qdrantURL *url.URL, collectionName string) (*QdrantCollectionInfo, error) {
	checkURL := fmt.Sprintf("%s/collections/%s", qdrantURL.String(), collectionName)
	resp, err := http.Get(checkURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("collection does not exist")
	}

	var info QdrantCollectionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode collection info: %w", err)
	}

	return &info, nil
}

// DeleteQdrantCollection deletes a collection
func DeleteQdrantCollection(qdrantURL *url.URL, collectionName string) error {
	deleteURL := fmt.Sprintf("%s/collections/%s", qdrantURL.String(), collectionName)

	req, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to delete collection, status: %d", resp.StatusCode)
	}

	return nil
}

// CreateQdrantCollection creates a collection in Qdrant if it doesn't exist
func CreateQdrantCollection(qdrantURL *url.URL, collectionName string, vectorSize int, forceRecreate bool) error {
	// Check if collection exists and get its info
	info, err := GetQdrantCollectionInfo(qdrantURL, collectionName)
	if err == nil {
		// Collection exists, check dimensions
		existingSize := info.Result.Config.Params.Vectors.Size

		if existingSize == vectorSize {
			log.Printf("Collection %s exists with correct dimensions (%d)", collectionName, vectorSize)
			return nil
		}

		log.Printf("⚠️  Collection %s has dimension mismatch: expected %d, got %d",
			collectionName, vectorSize, existingSize)

		if !forceRecreate {
			return fmt.Errorf("dimension mismatch: collection has %d dimensions but embedding model produces %d. Use --force-recreate to fix", existingSize, vectorSize)
		}

		log.Printf("🗑️  Deleting existing collection %s...", collectionName)
		if err := DeleteQdrantCollection(qdrantURL, collectionName); err != nil {
			return fmt.Errorf("failed to delete existing collection: %w", err)
		}
	}

	// Create new collection
	createURL := fmt.Sprintf("%s/collections/%s", qdrantURL.String(), collectionName)

	// Prepare request body
	requestBody := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     vectorSize,
			"distance": "Cosine",
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create collection
	req, err := http.NewRequest("PUT", createURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to create collection, status: %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Printf("✅ Created collection %s with %d dimensions", collectionName, vectorSize)
	return nil
}
