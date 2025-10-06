package internal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/parikshith521/neo-sync/models"
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

func BuildIndex(targetDir string) (models.FileState, models.DirState, error) {
	dirState := make(models.DirState)
	fileState := make(models.FileState)
	err := filepath.WalkDir(targetDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirState[path] = true
			log.Println("Added directory:", path)
		} else {
			info, err := d.Info()
			if err != nil {
				return err
			}
			hash, err := computeSHA256(path)
			if err != nil {
				log.Printf("Error computing hash for newly found file %s: %v", path, err)
				return err
			}
			fileState[path] = &models.FileInfo{
				Name:    path,
				ModTime: info.ModTime(),
				Size:    info.Size(),
				Hash:    hash,
			}
			log.Println("Watching file: ", path)
		}
		return nil
	})
	return fileState, dirState, err
}

func StartServer(rootDir, port string) {
	mux := http.NewServeMux()

	// Serves state
	mux.HandleFunc("/index", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request for /index from %s", r.RemoteAddr)
		fileIndex, dirSet, err := BuildIndex(rootDir)
		if err != nil {
			http.Error(w, "Failed to build index", http.StatusInternalServerError)
			log.Printf("Error building index: %v", err)
			return
		}

		responseData := models.ResponseData{
			Files: fileIndex,
			Dirs:  dirSet,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(responseData); err != nil {
			log.Printf("Error encoding index response: %v", err)
		}
	})

	// Serves file content
	fileServer := http.FileServer(http.Dir(rootDir))
	mux.Handle("/files/", http.StripPrefix("/files/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Received request for file %s from %s", r.URL.Path, r.RemoteAddr)
		fileServer.ServeHTTP(w, r)
	})))

	log.Printf("Starting server on port %s to serve files from '%s'", port, rootDir)
	if err := http.ListenAndServe(port, mux); err != nil && err != http.ErrServerClosed {
		log.Fatalf("FATAL: Could not start server: %v", err)
	}
}
