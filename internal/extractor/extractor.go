package extractor

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"archive-extractor/internal/archiver"
	"archive-extractor/internal/progress"
	"archive-extractor/internal/scanner"
)

func ProcessArchives(rootDir, outputDir, imageOutputDir, videoOutputDir string) error {
	files, err := scanner.ScanDirectory(rootDir)
	if err != nil {
		return fmt.Errorf("error scanning directory: %v", err)
	}

	progress.InitProgress(len(files))

	for i, file := range files {
		progress.UpdateProgress(i+1, fmt.Sprintf("Processing %s", filepath.Base(file)))

		if err := extractArchive(file, outputDir, imageOutputDir, videoOutputDir); err != nil {
			progress.PrintInfo(fmt.Sprintf("Failed to extract %s: %v", filepath.Base(file), err))
			continue
		}
	}

	progress.FinishProgress()

	for _, file := range files {
		if err := deleteArchive(file); err != nil {
			progress.PrintSuccess(fmt.Sprintf("Failed to delete %s: %v", filepath.Base(file), err))
		}
	}

	progress.PrintSuccess("All archives processed successfully!")
	return nil
}

func extractArchive(file, outputDir, imageOutputDir, videoOutputDir string) error {
	tempDir, err := os.MkdirTemp("", "archive-extract-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := archiver.ExtractArchive(file, tempDir); err != nil {
		return err
	}

	// If no output directory is specified, use the parent directory of the archive
	if outputDir == "" && imageOutputDir == "" && videoOutputDir == "" {
		outputDir = filepath.Dir(file)
	}

	return processExtractedFiles(tempDir, outputDir, imageOutputDir, videoOutputDir)
}

func processExtractedFiles(tempDir, outputDir, imageOutputDir, videoOutputDir string) error {
	return filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}

		var destPath string
		if outputDir != "" {
			destPath = filepath.Join(outputDir, relPath)
		} else if imageOutputDir != "" && isImageFile(path) {
			destPath = filepath.Join(imageOutputDir, filepath.Base(path))
		} else if videoOutputDir != "" && isVideoFile(path) {
			destPath = filepath.Join(videoOutputDir, filepath.Base(path))
		} else {
			destPath = filepath.Join(outputDir, relPath)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Use copyFile instead of os.Rename
		if err := copyFile(path, destPath); err != nil {
			return err
		}

		// Remove the source file after successful copy
		return os.Remove(path)
	})
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp"}
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	videoExts := []string{".mp4", ".avi", ".mov", ".mkv", ".wmv", ".flv", ".webm", ".ts"}
	for _, vidExt := range videoExts {
		if ext == vidExt {
			return true
		}
	}
	return false
}

func deleteArchive(file string) error {
	progress.PrintInfo(fmt.Sprintf("Deleting archive: %s", file))
	return os.Remove(file)
}
