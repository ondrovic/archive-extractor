package main

import (
    "fmt"
    "os"

	"archive-extractor/internal/utils"
    "archive-extractor/cmd"
)

func main() {
	utils.ClearConsole()
    if err := cmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}