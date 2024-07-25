package progress

import (
	"io"
	"sync"

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

type ProgressCallback func(current, total int64)

type ProgressBars struct {
	bars     map[string]*pterm.ProgressbarPrinter
	progress map[string]int
	total    map[string]int // Use map for total progress
	mu       sync.Mutex      // Mutex for synchronization
	multi    *pterm.MultiPrinter
}

func (pb *ProgressBars) SetProgress(name string, progress int) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if bar, exists := pb.bars[name]; exists {
		pb.progress[name] = progress
		bar.Current = progress
	}
}

func (pb *ProgressBars) Increment(name string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if bar, exists := pb.bars[name]; exists {
		pb.progress[name]++
		bar.Increment()
	}
}

func (pb *ProgressBars) UpdateText(name string, text string) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	if bar, exists := pb.bars[name]; exists {
		bar.UpdateTitle(text)
	}
}

func CreateDynamicProgressBars(names []string, totals []int) *ProgressBars {
	if len(names) != len(totals) {
		pterm.Error.Println("Names and totals length mismatch")
		return nil
	}

	multi := pterm.DefaultMultiPrinter
	multi.Start() // Start the multi printer

	// Initialize maps for bars and progress
	bars := make(map[string]*pterm.ProgressbarPrinter)
	progress := make(map[string]int)
	total := make(map[string]int) // Changed to map for total progress

	// Create progress bars and store them in the map
	for i, name := range names {
		bar, err := pterm.DefaultProgressbar.WithTotal(totals[i]).WithWriter(multi.NewWriter()).WithMaxWidth(100).Start(name)
		if err != nil {
			pterm.Error.Printf("Failed to create progress bar for %s: %v", name, err)
			continue
		}
		bars[name] = bar
		progress[name] = 0 // Initialize progress for each bar
		total[name] = totals[i] // Store total for each bar
	}

	return &ProgressBars{
		bars:     bars,
		progress: progress,
		total:    total,
		multi:    &multi,
	}
}

func (pb *ProgressBars) Start() {
	pb.multi.Start()
}

func (pb *ProgressBars) Stop() {
	pb.multi.Stop()
}