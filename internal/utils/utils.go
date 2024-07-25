package utils

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// ClearConsole exports the functions for use in other packages
func ClearConsole() {
	var clearCmd *exec.Cmd

	switch runtime.GOOS {
	case "linux", "darwin":
		clearCmd = exec.Command("clear")
	case "windows":
		clearCmd = exec.Command("cmd", "/c", "cls")
	default:
		fmt.Println("Unsupported platform")
		return
	}

	clearCmd.Stdout = os.Stdout
	clearCmd.Run()
}

func SanitizeFileName(fileName string) string {
	// Remove leading/trailing whitespace
	fileName = strings.TrimSpace(fileName)

	// Find the last index of '/'
	if lastSlash := strings.LastIndex(fileName, "/"); lastSlash != -1 {
		// Keep the substring after the last '/'
		fileName = fileName[lastSlash+1:]
	}
	
	// Replace problematic characters
	fileName = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '/', '\\', '|', '?', '*':
			return '_'
		default:
			return r
		}
	}, fileName)

	// Ensure the file name isn't empty after sanitization
	if fileName == "" {
		return "_"
	}

	return fileName
}

func CleanFilePath(fileHeaderName string, path string) string {
	if !strings.HasSuffix(path, fileHeaderName) {
		return strings.Replace(path, fileHeaderName, "", -1)
	}

	return path
}