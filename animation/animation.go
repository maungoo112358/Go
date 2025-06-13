package animation

import (
	"fmt"
	"io"
	"sync"
	"time"
)

type DownloadState struct {
	Percent float64
	Done    bool
	Mu      sync.Mutex
}

type ProgressReader struct {
	Reader    io.Reader
	Total     int64
	BytesRead int64
	State     *DownloadState
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.BytesRead += int64(n)

	if pr.Total > 0 {
		pr.State.Mu.Lock()
		pr.State.Percent = (float64(pr.BytesRead) / float64(pr.Total)) * 100
		pr.State.Mu.Unlock()
	}

	return n, err
}

func SpinnerAnimation(state *DownloadState, wg *sync.WaitGroup) {
	defer wg.Done()

	dotsList := []string{".", "..", "..."}
	frames := []string{"|", "/", "-", "\\"}
	i := 0

	for {
		time.Sleep(150 * time.Millisecond)

		state.Mu.Lock()
		done := state.Done
		pct := state.Percent
		state.Mu.Unlock()

		if done {
			return
		}

		dots := fmt.Sprintf("%-3s", dotsList[i%len(dotsList)])
		frame := fmt.Sprintf("%-2s", frames[i%len(frames)])
		fmt.Printf("\rDownloading%s %s %.0f%%", dots, frame, pct)

		i++
	}
}
