package internal

import (
	"fmt"
	"path/filepath"

	"github.com/parikshith521/neo-sync/models"
)

func getRootDir(dirState models.DirState) string {
	var rootDir string
	for path := range dirState {
		if rootDir == "" || len(path) < len(rootDir) {
			rootDir = path
		}
	}
	return rootDir
}

func convertRemoteToLocalPath(localRoot, remoteRoot, remotePath string) (string, error) {
	relativePath, err := filepath.Rel(remoteRoot, remotePath)
	if err != nil {
		return "", fmt.Errorf("the provided remote path is not inside the remote root: %w", err)
	}
	localPath := filepath.Join(localRoot, relativePath)
	return localPath, nil
}

func Compare(localDirState models.DirState, localFileState models.FileState, remoteDirState models.DirState, remoteFileState models.FileState) ([]string, error) {
	var actions []string

	localRoot := getRootDir(localDirState)
	remoteRoot := getRootDir(remoteDirState)
	// Directory state comparison
	for dirPath := range remoteDirState {
		localDirPath, err := convertRemoteToLocalPath(localRoot, remoteRoot, dirPath)
		if err != nil {
			return nil, err
		}
		if _, ok := localDirState[localDirPath]; !ok {
			command := "mkdir " + localDirPath
			actions = append(actions, command)
		}
	}
	for dirPath := range localDirState {
		remoteDirPath, err := convertRemoteToLocalPath(remoteRoot, localRoot, dirPath)
		if err != nil {
			return nil, err
		}
		if _, ok := remoteDirState[remoteDirPath]; !ok {
			command := "rm -r " + dirPath
			actions = append(actions, command)
		}
	}

	// File state comparison
	// Need to consider file hash mismatch
	for filePath := range remoteFileState {
		localFilePath, err := convertRemoteToLocalPath(localRoot, remoteRoot, filePath)
		if err != nil {
			return nil, err
		}
		if _, ok := localFileState[localFilePath]; !ok {
			command := "touch " + localFilePath
			actions = append(actions, command)
		}
	}
	for filePath := range localFileState {
		remoteFilePath, err := convertRemoteToLocalPath(remoteRoot, localRoot, filePath)
		if err != nil {
			return nil, err
		}
		if _, ok := remoteFileState[remoteFilePath]; !ok {
			command := "rm " + filePath
			actions = append(actions, command)
		}
	}

	return actions, nil
}
