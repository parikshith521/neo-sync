package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileInfo struct {
	Name    string    `json:"name"`
	ModTime time.Time `json:"modTime"`
	Size    int64     `json:"size"`
	Hash    string    `json:"hash"`
}

type FileState map[string]FileInfo
type DirState []string

var fileState FileState
var dirState DirState

func main() {
	// Create new watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	// Start listening for events.
	go func() {
		// Infinite loop to listen.
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Printf("EVENT: %s | OPERATION: %s", event.Name, event.Op)
				// Listen to directory creation events.
				if event.Has(fsnotify.Create) {
					info, err := os.Stat(event.Name)
					if err == nil && info.IsDir() {
						err := watcher.Add(event.Name)
						if err != nil {
							log.Printf("Error adding new directory %s to watcher: %v", event.Name, err)
						} else {
							log.Printf("Added new directory to watcher: %s", event.Name)
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("Wacher Error: ", err)
			}
		}
	}()

	// Add tmp directory to be watched (recursive).
	// Should be passed a CLA in the future.
	targetDir := "/home/pari/Desktop/Dev/tmp"
	err = filepath.WalkDir(targetDir, func(path string, d os.DirEntry, err error) error {
		if d.IsDir() {
			err := watcher.Add(path)
			if err != nil {
				return err
			}
			log.Println("Watching directory: ", path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Error walking target directory: %v", err)
	}

	// Block main gorountine forever.
	<-make(chan struct{})
}
