package video

import (
	"GoTest/animation"
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/kkdai/youtube/v2"
)

func DownloadSelectedFormat(url string) error {
	client := youtube.Client{}

	video, err := client.GetVideo(url)
	if err != nil {
		return err
	}

	fmt.Println("Available Formats:")
	for i, f := range video.Formats {
		if f.QualityLabel != "" {
			fmt.Printf("[%d] %s (%s)\n", i, f.QualityLabel, f.MimeType)
		}
	}

	fmt.Println("Enter a format number to download:")
	reader := bufio.NewReader(os.Stdin)
	raw, _ := reader.ReadString('\n')
	indexStr := strings.TrimSpace(raw)
	index, err := strconv.Atoi(indexStr)
	if err != nil || index < 0 || index >= len(video.Formats) {
		return fmt.Errorf("invalid format selection")
	}

	videoFormat := video.Formats[index]

	var audioFormat *youtube.Format
	for _, f := range video.Formats {
		if f.AudioChannels > 0 && f.QualityLabel == "" {
			audioFormat = &f
			break
		}
	}
	if audioFormat == nil {
		return fmt.Errorf("no audio-only format found")
	}

	videoStream, size, err := client.GetStream(video, &videoFormat)
	if err != nil {
		return err
	}

	videoFile, err := os.Create("video.mp4")
	if err != nil {
		return err
	}
	defer videoFile.Close()

	state := &animation.DownloadState{}
	pr := &animation.ProgressReader{Reader: videoStream, Total: size, State: state}

	var wg sync.WaitGroup
	wg.Add(1)
	go animation.SpinnerAnimation(state, &wg)

	_, err = videoFile.ReadFrom(pr)

	state.Mu.Lock()
	state.Done = true
	state.Mu.Unlock()
	wg.Wait()

	audioStream, _, err := client.GetStream(video, audioFormat)
	if err != nil {
		return err
	}

	audioFile, err := os.Create("audio.m4a")
	if err != nil {
		return err
	}
	defer audioFile.Close()

	_, err = io.Copy(audioFile, audioStream)
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %v", err)
	}
	downloadDir := filepath.Join(home, "Desktop", "Video-Download")
	os.MkdirAll(downloadDir, os.ModePerm)

	// Merge using ffmpeg
	safeTitle := sanitizeFileName(video.Title)
	outputName := strings.ReplaceAll(safeTitle, " ", "_") + ".mp4"
	outputPath := filepath.Join(downloadDir, outputName)
	cmd := exec.Command("ffmpeg", "-y", "-i", "video.mp4", "-i", "audio.m4a", "-c", "copy", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg merge failed: %v", err)
	}

	videoFile.Close()
	audioFile.Close()

	os.Remove("video.mp4")
	os.Remove("audio.m4a")

	fmt.Println("Download and merge complete:", outputPath)

	return nil

}

func sanitizeFileName(name string) string {
	invalid := []string{`<`, `>`, `:`, `"`, `/`, `\`, `|`, `?`, `*`}
	for _, c := range invalid {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}
