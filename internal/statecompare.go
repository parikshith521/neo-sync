package internal

import "github.com/parikshith521/neo-sync/models"

func Compare(localDirState models.DirState, localFileState models.FileState, remoteDirState models.DirState, remoteFileState models.FileState) []string {
	var actions []string
	// Directory state comparison
	for dirPath := range remoteDirState {
		if _, ok := localDirState[dirPath]; !ok {
			command := "mkdir " + dirPath
			actions = append(actions, command)
		}
	}
	for dirPath := range localDirState {
		if _, ok := remoteDirState[dirPath]; !ok {
			command := "rm -r " + dirPath
			actions = append(actions, command)
		}
	}

	// File state comparison
	for filePath := range remoteFileState {
		if _, ok := localFileState[filePath]; !ok {
			command := "touch " + filePath
			actions = append(actions, command)
		}
	}
	for filePath := range localFileState {
		if _, ok := remoteFileState[filePath]; !ok {
			command := "rm " + filePath
			actions = append(actions, command)
		}
	}

	return actions
}
