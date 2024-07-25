package progress

import (
	"io"

	"github.com/pterm/pterm"
)

type ProgressReader struct {
	Reader   io.Reader
	Callback func(int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Callback(int64(n))
	return n, err
}

type ProgressBars struct {
	bars       []*pterm.ProgressbarPrinter
	progress   []int
	total      []int
	multi      *pterm.MultiPrinter
}

type ProgressCallback func(current, total int64)

func CreateDynamicProgressBars(names []string, totals []int) *ProgressBars {
	multi := pterm.DefaultMultiPrinter
	multi.Start() // Start the multi printer
	
	bars := make([]*pterm.ProgressbarPrinter, len(names))
	progress := make([]int, len(names))

	for i, name := range names {
		bar, _ := pterm.DefaultProgressbar.WithTotal(totals[i]).WithWriter(multi.NewWriter()).WithMaxWidth(100).Start(name)
		bars[i] = bar
	}

	return &ProgressBars{
		bars:     bars,
		progress: progress,
		total:    totals,
		multi:    &multi,
	}
}

func (pb *ProgressBars) SetProgress(index, progress int) {
	if index >= 0 && index < len(pb.bars) {
		pb.progress[index] = progress
		pb.bars[index].Current = progress
	}
}

func (pb *ProgressBars) Increment(index int) {
	if index >= 0 && index < len(pb.bars) {
		pb.progress[index]++
		pb.bars[index].Increment()
	}
}

func (pb *ProgressBars) UpdateText(index int, text string) {
	if index >= 0 && index < len(pb.bars) {
		pb.bars[index].UpdateTitle(text)
	}
}

func (pb *ProgressBars) Start() {
	pb.multi.Start()
}

func (pb *ProgressBars) Stop() {
	pb.multi.Stop()
}