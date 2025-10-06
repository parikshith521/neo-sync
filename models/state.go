package models

import "time"

type FileInfo struct {
	Name    string    `json:"name"`
	ModTime time.Time `json:"modTime"`
	Size    int64     `json:"size"`
	Hash    string    `json:"hash"`
}

type FileState map[string]*FileInfo
type DirState map[string]bool

type ResponseData struct {
	Files FileState `json:"files"`
	Dirs  DirState  `json:"dirs"`
}
