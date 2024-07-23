package main

import (
	"fmt"
	"os"

	"archive-extractor/cmd"
	"archive-extractor/internal/utils"
)

func main() {
	utils.ClearConsole()
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
