package progress

import (
    "fmt"
    "github.com/pterm/pterm"
)

var (
    progressBar *pterm.ProgressbarPrinter
)

func InitProgress(total int) {
    progressBar, _ = pterm.DefaultProgressbar.
        WithTotal(total).
        WithTitle("Processing archives").
        WithShowCount(true).
        WithShowPercentage(true).
        Start()
}

func UpdateProgress(current int, message string) {
    progressBar.UpdateTitle(fmt.Sprintf("[%d/%d] %s", current, progressBar.Total, message))
    progressBar.Add(1)
}

func FinishProgress() {
    progressBar.Stop()
}

func PrintInfo(message string) {
    pterm.Info.Println(message)
}

func PrintSuccess(message string) {
    pterm.Success.Println(message)
}