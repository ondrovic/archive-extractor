package cmd

import (
	"archive-extractor/internal/extractor"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-archive-extractor [root directory]",
	Short: "Extract archives from a directory",
	Args:  cobra.ExactArgs(1),
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringP("output", "o", "", "Output location for all files")
	rootCmd.Flags().StringP("image-output", "i", "", "Output directory for images")
	rootCmd.Flags().StringP("video-output", "v", "", "Output directory for videos")
}

func Execute() error {
	pterm.DefaultHeader.WithFullWidth().WithBackgroundStyle(pterm.NewStyle(pterm.BgDarkGray)).WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).Println("Archive Extractor")
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	rootDir := args[0]
	outputDir, _ := cmd.Flags().GetString("output")
	imageOutputDir, _ := cmd.Flags().GetString("image-output")
	videoOutputDir, _ := cmd.Flags().GetString("video-output")

	return extractor.ProcessArchives(rootDir, outputDir, imageOutputDir, videoOutputDir)
}
