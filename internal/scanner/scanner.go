package scanner

import (
    "os"
    "path/filepath"

    "archive-extractor/internal/archiver"
)

func ScanDirectory(root string) ([]string, error) {
    var files []string
    err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if !info.IsDir() && archiver.IsArchive(path) {
            files = append(files, path)
        }
        return nil
    })
    return files, err
}