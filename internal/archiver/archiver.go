package archiver

import (
	"archive-extractor/internal/utils"
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bodgit/sevenzip"
	"github.com/nwaples/rardecode"
)

var (
	supportedArchives = []string{
		".7z", ".zip", ".rar", ".gz", ".tar", ".bz2", ".xz",
	}
	filesToSkip = []string{
		"osx", "OSX", ".DS_STORE",
	}
)

type progressReader struct {
	reader   io.Reader
	callback func(int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	pr.callback(int64(n))
	return n, err
}

type ArchiveFile interface {
	Name() string
	HeaderName() string
}

type zipFile struct {
	*zip.File
}

func (f zipFile) Name() string {
	return f.File.Name
}

func (f zipFile) HeaderName() string {
	return f.File.FileHeader.Name
}

type rarFile struct {
	*rardecode.FileHeader
}

func (f rarFile) Name() string {
	return f.FileHeader.Name
}

func (f rarFile) HeaderName() string {
	return f.FileHeader.Name // RAR doesn't have a separate header name
}

type sevenZipFile struct {
	*sevenzip.File
}

func (f sevenZipFile) Name() string {
	return f.File.Name
}

func (f sevenZipFile) HeaderName() string {
	return f.File.Name // 7zip doesn't have a separate header name
}

func IsArchive(file string) bool {
	ext := strings.ToLower(filepath.Ext(file))
	for _, e := range supportedArchives {
		if ext == e {
			return true
		}
	}
	return false
}

type ProgressCallback func(current, total int64)

func ExtractArchive(src, dest string, progressCallback ProgressCallback) (error) {
	ext := strings.ToLower(filepath.Ext(src))
	switch ext {
	case ".zip":
		return extractZip(src, dest, progressCallback)
	case ".rar":
		return extractRar(src, dest, progressCallback)
	default:
		return extractSevenZip(src, dest, progressCallback)
	}
}

func shouldSkip(f ArchiveFile) bool {
	name := f.Name()
	baseName := filepath.Base(name)
	headerName := f.HeaderName()
	baseHeaderName := filepath.Base(headerName)

	for _, pattern := range filesToSkip {
		if strings.Contains(name, pattern) || baseName == pattern ||
			strings.Contains(headerName, pattern) || baseHeaderName == pattern {
			return true
		}
	}
	return false
}

func extractZip(src, dest string, progressCallback ProgressCallback) (error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip: %v", err)
	}
	defer r.Close()

	var totalSize int64
	for _, f := range r.File {
		totalSize += int64(f.UncompressedSize64)
	}

	var extractedSize int64
	var mu sync.Mutex
	var firstFileHeader *zip.FileHeader

	for _, f := range r.File {
		// skip files containing filesToSkip
		if shouldSkip(zipFile{f}) {
			continue
		}

		// Store the first file header
		if firstFileHeader == nil {
			firstFileHeader = &f.FileHeader
		}

		// Sanitize the file path
		cleanFileHeader := utils.SanitizeFileName(firstFileHeader.Name)
		cleanName := utils.SanitizeFileName(f.Name)
		path := filepath.Join(dest, filepath.FromSlash(cleanName))

		path = utils.CleanFilePath(cleanFileHeader, path)

		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", cleanName)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", path, err)
			}
			continue
		}

		// Create the directory for the file
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %v", path, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in zip: %v", f.Name, err)
		}
		
		destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file %s: %v", path, err)
		}

		_, err = io.Copy(destFile, &progressReader{
			reader: rc,
			callback: func(size int64) {
				mu.Lock()
				extractedSize += size
				progressCallback(extractedSize, totalSize)
				mu.Unlock()
			},
		})

		rc.Close()
		destFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file %s: %v", f.Name, err)
		}
	}

	return nil
}

func extractRar(src, dest string, progressCallback ProgressCallback) error {
	r, err := rardecode.OpenReader(src, "")
	if err != nil {
		return fmt.Errorf("failed to open rar: %v", err)
	}
	defer r.Close()

	var totalSize int64
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read rar header: %v", err)
		}
		totalSize += header.UnPackedSize
	}

	// Reset the reader to start from the beginning
	r, err = rardecode.OpenReader(src, "")
	if err != nil {
		return fmt.Errorf("failed to reopen rar: %v", err)
	}

	var extractedSize int64
	var mu sync.Mutex

	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read rar header: %v", err)
		}

		if shouldSkip(rarFile{header}) {
			continue
		}

		// Sanitize the file path

		cleanName := utils.SanitizeFileName(header.Name)
		path := filepath.Join(dest, filepath.FromSlash(cleanName))
		
		// Ensure the file path is within the destination directory
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", cleanName)
		}

		if header.IsDir {
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", path, err)
			}
			continue
		}

		// Create the directory for the file
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %v", path, err)
		}

		destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %v", path, err)
		}

		_, err = io.Copy(destFile, &progressReader{
			reader: r,
			callback: func(size int64) {
				mu.Lock()
				extractedSize += size
				progressCallback(extractedSize, totalSize)
				mu.Unlock()
			},
		})

		destFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file %s: %v", header.Name, err)
		}
	}

	return nil
}

func extractSevenZip(src, dest string, progressCallback ProgressCallback) error {
	r, err := sevenzip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open 7z: %v", err)
	}
	defer r.Close()

	var totalSize int64
	for _, f := range r.File {
		totalSize += int64(f.UncompressedSize)
	}

	var extractedSize int64
	var mu sync.Mutex

	for _, f := range r.File {
		if shouldSkip(sevenZipFile{f}) {
			continue
		}
		// Sanitize the file path
		cleanName := utils.SanitizeFileName(f.Name)
		path := filepath.Join(dest, filepath.FromSlash(cleanName))

		// Ensure the file path is within the destination directory
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid file path: %s", cleanName)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %v", path, err)
			}
			continue
		}

		// Create the directory for the file
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %v", path, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file %s in 7z: %v", f.Name, err)
		}

		destFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create file %s: %v", path, err)
		}

		_, err = io.Copy(destFile, &progressReader{
			reader: rc,
			callback: func(size int64) {
				mu.Lock()
				extractedSize += size
				progressCallback(extractedSize, totalSize)
				mu.Unlock()
			},
		})

		rc.Close()
		destFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file %s: %v", f.Name, err)
		}
	}

	return nil
}