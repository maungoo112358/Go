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

	index, err := getSelectedVideoIndex(video)
	if err != nil {
		return err
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

	videoPath := "video.mp4"
	audioPath := "audio.m4a"
	state := &animation.DownloadState{}

	err = downloadVideo(&client, video, &videoFormat, videoPath, state)
	if err != nil {
		return err
	}

	err = downloadAudio(&client, video, audioFormat, audioPath)
	if err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home dir: %v", err)
	}

	outputPath := getOutputPath(video, home)

	cmd := exec.Command("ffmpeg", "-y", "-i", videoPath, "-i", audioPath, "-c", "copy", outputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg merge failed: %v", err)
	}

	os.Remove(videoPath)
	os.Remove(audioPath)

	fmt.Println("Download and merge complete:", outputPath)
	return nil
}

func getOutputPath(video *youtube.Video, home string) string {
	downloadDir := filepath.Join(home, "Desktop", "Video-Download")
	os.MkdirAll(downloadDir, os.ModePerm)

	safeTitle := sanitizeFileName(video.Title)
	outputName := strings.ReplaceAll(safeTitle, " ", "_") + ".mp4"
	outputPath := filepath.Join(downloadDir, outputName)
	return outputPath
}

func getSelectedVideoIndex(video *youtube.Video) (int, error) {
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
		return 0, fmt.Errorf("invalid format selection")
	}
	return index, nil
}

func downloadVideo(client *youtube.Client, video *youtube.Video, format *youtube.Format, path string, state *animation.DownloadState) error {
	stream, size, err := client.GetStream(video, format)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	pr := &animation.ProgressReader{Reader: stream, Total: size, State: state}

	var wg sync.WaitGroup
	wg.Add(1)
	go animation.SpinnerAnimation(state, &wg)

	_, err = file.ReadFrom(pr)

	state.Mu.Lock()
	state.Done = true
	state.Mu.Unlock()
	wg.Wait()

	return err
}

func downloadAudio(client *youtube.Client, video *youtube.Video, format *youtube.Format, path string) error {
	stream, _, err := client.GetStream(video, format)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	return err
}

func sanitizeFileName(name string) string {
	invalid := []string{`<`, `>`, `:`, `"`, `/`, `\`, `|`, `?`, `*`}
	for _, c := range invalid {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}
