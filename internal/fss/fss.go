package fss

import (
	"fmt"
	"io/ioutil"
	"log"
	"mark"
	"os"
	"path/filepath"
	"time"
)

func SaveNote(meta mark.Header, content []byte) (string, error) {
	filename := GetFullPath(meta)
	_ = os.MkdirAll(GetPath(meta), 0755)
	err := ioutil.WriteFile(filename, content, 0644)
	return filename, err
}

func GetFilename(meta mark.Header) string {
	return fmt.Sprintf("%s.md", meta.CreatedAt.In(time.UTC).Format("2006-01-02_15:04:05Z0700_Monday"))
}
func GetLibPath() string {
	return filepath.Join(GetStoragePath(), "lib")
}
func GetPath(meta mark.Header) string {
	return filepath.Join(GetLibPath(), meta.CreatedAt.Format("2006"), meta.CreatedAt.Format("01"))
}
func GetFullPath(meta mark.Header) string {
	return filepath.Join(GetPath(meta), GetFilename(meta))
}

func GetStoragePath() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dirname, ".mark")
	_ = os.MkdirAll(path, 0755)
	return path
}

func GetFilenameToPath(filename string) (string, error) {
	timestamp, err := time.Parse("2006-01-02_15:04:05Z0700_Monday.md", filename)
	if err != nil {
		return "", err
	}
	return GetFullPath(mark.Header{CreatedAt: timestamp}), nil
}
