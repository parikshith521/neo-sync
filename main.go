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
							log.Printf("Error adding new directory %s to watcher: %v", event.Name, err)
						} else {
							dirState[event.Name] = true
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
					if _, isDir := dirState[event.Name]; isDir {
						// it's a directory that was renamed
						// delete dir state, for files, if it has directory prefix delete them from state also
						delete(dirState, event.Name)
						for dirPath := range dirState {
							if strings.HasPrefix(dirPath, event.Name+string(os.PathSeparator)) {
								delete(dirState, dirPath)
							}
						}
						for filePath := range fileState {
							if strings.HasPrefix(filePath, event.Name+string(os.PathSeparator)) {
								delete(fileState, filePath)
							}
						}
					} else {
						// it's a file that was renamed, so just delete it's state
						delete(fileState, event.Name)
					}
					// add new states using dir walk
					err := filepath.WalkDir(filepath.Dir(event.Name), func(path string, d os.DirEntry, err error) error {
						_, isDirTracked := dirState[path]
						_, isFileTracked := fileState[path]

						if !isDirTracked && !isFileTracked {
							info, statErr := d.Info()
							if statErr != nil {
								return statErr
							}
							if info.IsDir() {
								dirState[path] = true
								err := watcher.Add(path)
								if err != nil {
									log.Printf("Error adding new directory %s to watcher: %v", event.Name, err)
								} else {
									dirState[path] = true
									log.Printf("Added new directory to watcher: %s", event.Name)
								}
							} else {
								hash, err := computeSHA256(path)
								if err != nil {
									log.Printf("Error computing hash for newly found file %s: %v", event.Name, err)
									return err
								}
								fileState[event.Name] = &FileInfo{
									Name:    event.Name,
									ModTime: info.ModTime(),
									Size:    info.Size(),
									Hash:    hash,
								}
								log.Println("STATUS UPDATE: New file found: ", fileState[event.Name])
							}
						}
						return nil
					})
					if err != nil {
						log.Printf("ERROR during re-scan of %s: %v", filepath.Dir(event.Name), err)
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
