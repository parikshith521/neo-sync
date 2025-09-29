package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileInfo struct {
	Name    string    `json:"name"`
	ModTime time.Time `json:"modTime"`
	Size    int64     `json:"size"`
	Hash    string    `json:"hash"`
}

type FileState map[string]*FileInfo
type DirState map[string]bool

// Thread safety needs to be taken care of.
var (
	fileState = make(FileState)
	dirState  = make(DirState)
)

func computeSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

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
				// Listen to events.
				switch {
				// Listen to file/directory creation events.
				case event.Has(fsnotify.Create):
					info, err := os.Stat(event.Name)
					if err != nil {
						log.Printf("Error fetching details from OS for file/dir %s: %v", event.Name, err)
						continue
					}
					if info.IsDir() {
						err := watcher.Add(event.Name)
						if err != nil {
							dirState[event.Name] = true
							log.Printf("Error adding new directory %s to watcher: %v", event.Name, err)
						} else {
							log.Printf("Added new directory to watcher: %s", event.Name)
						}
					} else {
						hash, err := computeSHA256(event.Name)
						if err != nil {
							log.Printf("Error computing hash for newly created file %s: %v", event.Name, err)
							continue
						}
						fileState[event.Name] = &FileInfo{
							Name:    event.Name,
							ModTime: info.ModTime(),
							Size:    info.Size(),
							Hash:    hash,
						}
						log.Println("STATUS UPDATE: New file created: ", fileState[event.Name])
					}

				// Listen to file write events.
				case event.Has(fsnotify.Write):
					log.Printf("FILE WRITE EVENT: %s", event.Name)
					info, err := os.Stat(event.Name)
					if err != nil {
						log.Printf("Error fetching details from OS for file %s: %v", event.Name, err)
						continue
					}
					hash, err := computeSHA256(event.Name)
					if err != nil {
						log.Printf("Error computing hash for modified file %s: %v", event.Name, err)
						continue
					}
					fileState[event.Name] = &FileInfo{
						Name:    event.Name,
						ModTime: info.ModTime(),
						Size:    info.Size(),
						Hash:    hash,
					}
					log.Println("STATUS UPDATE: File modified: ", fileState[event.Name])

				// Listen to file/directory deletion events.
				case event.Has(fsnotify.Remove):
					_, isDir := dirState[event.Name]
					if isDir {
						delete(dirState, event.Name)
						// Remove all files within this directory from the state.
						for filepath := range fileState {
							if strings.HasPrefix(filepath, event.Name+string(os.PathSeparator)) {
								delete(fileState, filepath)
							}
						}
						log.Printf("DELETE EVENT: Directory deleted: %s", event.Name)
					} else {
						delete(fileState, event.Name)
						log.Printf("DELETE EVENT: File deleted: %s", event.Name)
					}

				// Listen to file/directory rename events.
				case event.Has(fsnotify.Rename):
					log.Printf("RENAME EVENT: %s", event.Name)
					_, isDir := dirState[event.Name]
					if isDir {
						delete(dirState, event.Name)
						// new inner directory paths should be added to watcher + state update, inner files => state update

						log.Printf("RENAME EVENT: Directory renamed, old name: %s", event.Name)
					} else {
						delete(fileState, event.Name)
						log.Printf("RENAME EVENT: File renamed, old name: %s", event.Name)
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
	// Should be passed as a CLA in the future.
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
