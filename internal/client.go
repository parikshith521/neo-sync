package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/parikshith521/neo-sync/models"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL:    baseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) FetchIndex() (*models.ResponseData, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/index")
	if err != nil {
		return nil, fmt.Errorf("failed to get index from peer: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("peer returned non-200 status for index: %s", resp.Status)
	}

	var responseData models.ResponseData
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, fmt.Errorf("failed to decode index response from peer: %w", err)
	}

	return &responseData, nil
}

func (c *Client) FetchFile(remotePath, destPath string) error {
	// URL-encode the path to handle special characters safely.
	encodedPath := url.PathEscape(remotePath)
	fileURL := c.BaseURL + "/files/" + encodedPath

	resp, err := c.HTTPClient.Get(fileURL)
	if err != nil {
		return fmt.Errorf("failed to request file %s from peer: %w", remotePath, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer returned non-200 status for file %s: %s", remotePath, resp.Status)
	}

	// Create the destination file.
	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("failed to open destination file %s: %w", destPath, err)
	}
	defer outFile.Close()

	// Stream the file content directly to the file, overwriting its contents.
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file content to %s: %w", destPath, err)
	}

	return nil
}

func SyncWithPeer(peerAddr string) {
	client := NewClient(peerAddr)
	remoteIndex, err := client.FetchIndex()
	if err != nil {
		log.Fatalf("FATAL: Could not fetch index: %v", err)
	}
	// compare and fetch and action...
}
