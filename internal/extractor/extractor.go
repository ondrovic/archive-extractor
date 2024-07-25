package extractor

import (
	"archive-extractor/internal/archiver"
	"archive-extractor/internal/progress"
	"archive-extractor/internal/scanner"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	videoExtensions = []string{
		".3g2", ".3gp", ".aaf", ".asf", ".avchd", ".avi", ".drc", ".flv", ".m2v", ".m3u8",
		".m4p", ".m4v", ".mkv", ".mng", ".mov", ".mp2", ".mp4", ".mpe", ".mpeg", ".mpg",
		".mpv", ".mxf", ".nsv", ".ogg", ".ogv", ".qt", ".rm", ".rmvb", ".roq", ".svi",
		".ts", ".vob", ".webm", ".wmv", ".yuv",
	}
	imageExtensions = []string{
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".webp",
	}
)

func ProcessArchives(rootDir, outputDir, imageOutputDir, videoOutputDir string) error {
	files, err := scanner.ScanDirectory(rootDir)
	if err != nil {
		return err
	}

	// Create progress bars
	progressBars := progress.CreateDynamicProgressBars([]string{
		"Overall Progress",
		"Extraction Progress",
		"File Processing",
	}, []int{len(files), 100, 100})

	defer progressBars.Stop()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 4)

	for _, file := range files {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(file string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			progressBars.UpdateText("Extraction Progress", filepath.Base(file))
			err := extractArchive(file, outputDir, imageOutputDir, videoOutputDir, progressBars)
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
			progressBars.Increment("Overall Progress") // Increment Overall Progress
			if err == nil {
				deleteArchive(file)
			}
		}(file)
	}

	wg.Wait()

	progressBars.SetProgress("Overall Progress", 100) // Set Overall Progress to 100%
	
	return nil
}

func extractArchive(file, outputDir, imageOutputDir, videoOutputDir string, progressBars *progress.ProgressBars) error {
	archiveDir := filepath.Dir(file)
	tempDir, err := os.MkdirTemp(archiveDir, "tmp-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	progressBars.UpdateText("Extraction Progress", fmt.Sprintf("Extracting %s", filepath.Base(file)))
	progressBars.SetProgress("Extraction Progress", 0)

	// Extract the archive
	if err := archiver.ExtractArchive(file, tempDir, func(current, total int64) {
		progress := int(float64(current) / float64(total) * 100)
		progressBars.SetProgress("Extraction Progress", progress)
	}); err != nil {
		return err
	}

	// Find the root of the extracted contents
	extractedRoot := tempDir
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}
	if len(entries) == 1 && entries[0].IsDir() {
		extractedRoot = filepath.Join(tempDir, entries[0].Name())
	}

	if outputDir == "" && imageOutputDir == "" && videoOutputDir == "" {
		outputDir = archiveDir
		imageOutputDir = filepath.Join(archiveDir, "images")
		videoOutputDir = filepath.Join(archiveDir, "videos")
	}

	// Process the extracted files
	return filepath.Walk(extractedRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(extractedRoot, path)
		if err != nil {
			return err
		}

		var destPath string
		if isImageFile(path) && imageOutputDir != "" {
			destPath = filepath.Join(imageOutputDir, relPath)
		} else if isVideoFile(path) && videoOutputDir != "" {
			destPath = filepath.Join(videoOutputDir, relPath)
		} else if outputDir != "" {
			destPath = filepath.Join(outputDir, relPath)
		} else {
			destPath = filepath.Join(archiveDir, relPath)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		if err := copyFile(path, destPath); err != nil {
			return err
		}

		progressBars.UpdateText("File Processing", fmt.Sprintf("Processing %s", relPath))
		return processExtractedFiles(extractedRoot, outputDir, imageOutputDir, videoOutputDir, progressBars, filepath.Base(extractedRoot))
	})
}

func processExtractedFiles(extractedRoot, outputDir, imageOutputDir, videoOutputDir string, progressBars *progress.ProgressBars, topLevelDir string) error {
	var totalFiles, processedFiles int
	err := filepath.Walk(extractedRoot, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			totalFiles++
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Reset file processing progress to 0 before starting
	progressBars.SetProgress("File Processing", 0)

	return filepath.Walk(extractedRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		progressBars.UpdateText("File Processing", fmt.Sprintf("Processing %s", filepath.Base(path)))

		relPath, err := filepath.Rel(extractedRoot, path)
		if err != nil {
			return err
		}

		// Remove the top-level directory from the relative path
		relPath = strings.TrimPrefix(relPath, topLevelDir)
		relPath = strings.TrimPrefix(relPath, string(os.PathSeparator))

		var destPath string
		if isImageFile(path) && imageOutputDir != "" {
			destPath = filepath.Join(imageOutputDir, relPath)
		} else if isVideoFile(path) && videoOutputDir != "" {
			destPath = filepath.Join(videoOutputDir, relPath)
		} else if outputDir != "" {
			destPath = filepath.Join(outputDir, relPath)
		} else {
			destPath = filepath.Join(filepath.Dir(extractedRoot), relPath)
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		if err := copyFile(path, destPath); err != nil {
			return err
		}

		processedFiles++
		progress := int(float64(processedFiles) / float64(totalFiles) * 100)
		progressBars.SetProgress("File Processing", progress)

		return nil
	})
}

// func ProcessArchives(rootDir, outputDir, imageOutputDir, videoOutputDir string) error {
// 	files, err := scanner.ScanDirectory(rootDir)
// 	if err != nil {
// 		return err
// 	}

// 	// Create progress bars
// 	progressBars := progress.CreateDynamicProgressBars([]string{
// 		"Overall Progress",
// 		"Extraction Progress",
// 		"File Processing",
// 	}, []int{len(files), 100, 100})
	
// 	defer progressBars.Stop()

// 	var wg sync.WaitGroup
// 	semaphore := make(chan struct{}, 4)

// 	for i, file := range files {
// 		wg.Add(1)
// 		semaphore <- struct{}{}
// 		go func(file string, index int) {
// 			defer wg.Done()
// 			defer func() { <-semaphore }()

// 			progressBars.UpdateText(1, filepath.Base(file))
// 			err := extractArchive(file, outputDir, imageOutputDir, videoOutputDir, progressBars)
// 			if err != nil {
// 				fmt.Printf("Error: %s\n", err.Error())
// 			}
// 			progressBars.Increment(0) // Increment Overall Progress
// 			if err == nil {
// 				deleteArchive(file)
// 			}
// 		}(file, i)
// 	}

// 	wg.Wait()

// 	progressBars.SetProgress(0, 100) // Set Overall Progress to 100%
	
// 	return nil
// }
// func extractArchive(file, outputDir, imageOutputDir, videoOutputDir string, progressBars *progress.ProgressBars) error {
// 	archiveDir := filepath.Dir(file)
// 	tempDir, err := os.MkdirTemp(archiveDir, "tmp-")
// 	if err != nil {
// 		return err
// 	}
// 	defer os.RemoveAll(tempDir)

// 	progressBars.UpdateText(1, fmt.Sprintf("Extracting %s", filepath.Base(file)))
// 	progressBars.SetProgress(1, 0)

// 	// Extract the archive
// 	if err := archiver.ExtractArchive(file, tempDir, func(current, total int64) {
// 		progress := int(float64(current) / float64(total) * 100)
// 		progressBars.SetProgress(1, progress)
// 	}); err != nil {
// 		return err
// 	}

// 	// Find the root of the extracted contents
// 	extractedRoot := tempDir
// 	entries, err := os.ReadDir(tempDir)
// 	if err != nil {
// 		return err
// 	}
// 	if len(entries) == 1 && entries[0].IsDir() {
// 		extractedRoot = filepath.Join(tempDir, entries[0].Name())
// 	}

// 	if outputDir == "" && imageOutputDir == "" && videoOutputDir == "" {
// 		outputDir = archiveDir
// 		imageOutputDir = filepath.Join(archiveDir, "images")
// 		videoOutputDir = filepath.Join(archiveDir, "videos")
// 	}

// 	// Process the extracted files
// 	return filepath.Walk(extractedRoot, func(path string, info os.FileInfo, err error) error {
// 		if err != nil || info.IsDir() {
// 			return nil
// 		}

// 		relPath, err := filepath.Rel(extractedRoot, path)
// 		if err != nil {
// 			return err
// 		}

// 		var destPath string
// 		if isImageFile(path) && imageOutputDir != "" {
// 			destPath = filepath.Join(imageOutputDir, relPath)
// 		} else if isVideoFile(path) && videoOutputDir != "" {
// 			destPath = filepath.Join(videoOutputDir, relPath)
// 		} else if outputDir != "" {
// 			destPath = filepath.Join(outputDir, relPath)
// 		} else {
// 			destPath = filepath.Join(archiveDir, relPath)
// 		}

// 		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
// 			return err
// 		}

// 		if err := copyFile(path, destPath); err != nil {
// 			return err
// 		}

// 		progressBars.UpdateText(2, fmt.Sprintf("Extracting %s", relPath))
// 		return processExtractedFiles(extractedRoot, outputDir, imageOutputDir, videoOutputDir, progressBars, filepath.Base(extractedRoot))
// 	})
// }

// func processExtractedFiles(extractedRoot, outputDir, imageOutputDir, videoOutputDir string, progressBars *progress.ProgressBars, topLevelDir string) error {
// 	var totalFiles, processedFiles int
// 	err := filepath.Walk(extractedRoot, func(path string, info os.FileInfo, err error) error {
// 		if !info.IsDir() {
// 			totalFiles++
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	// Reset file processing progress to 0 before starting
// 	progressBars.SetProgress(2, 0)

// 	return filepath.Walk(extractedRoot, func(path string, info os.FileInfo, err error) error {
// 		if err != nil || info.IsDir() {
// 			return nil
// 		}

// 		progressBars.UpdateText(2, fmt.Sprintf("Extracting %s", filepath.Base(path)))

// 		relPath, err := filepath.Rel(extractedRoot, path)
// 		if err != nil {
// 			return err
// 		}

// 		// Remove the top-level directory from the relative path
// 		relPath = strings.TrimPrefix(relPath, topLevelDir)
// 		relPath = strings.TrimPrefix(relPath, string(os.PathSeparator))

// 		var destPath string
// 		if isImageFile(path) && imageOutputDir != "" {
// 			destPath = filepath.Join(imageOutputDir, relPath)
// 		} else if isVideoFile(path) && videoOutputDir != "" {
// 			destPath = filepath.Join(videoOutputDir, relPath)
// 		} else if outputDir != "" {
// 			destPath = filepath.Join(outputDir, relPath)
// 		} else {
// 			destPath = filepath.Join(filepath.Dir(extractedRoot), relPath)
// 		}

// 		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
// 			return err
// 		}

// 		if err := copyFile(path, destPath); err != nil {
// 			return err
// 		}

// 		processedFiles++
// 		progress := int(float64(processedFiles) / float64(totalFiles) * 100)
// 		progressBars.SetProgress(2, progress)

// 		progressBars.SetProgress(1, 100)
// 		return nil
// 	})
// }

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
    return err
}

func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, imgExt := range imageExtensions {
		if ext == imgExt {
			return true
		}
	}
	return false
}

func isVideoFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, vidExt := range videoExtensions {
		if ext == vidExt {
			return true
		}
	}
	return false
}

func deleteArchive(file string) error {
	return os.Remove(file)
}
