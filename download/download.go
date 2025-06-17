package download

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
)

func DownloadYoutubeAsMp3(videoURL string) error {
	cleanURL, err := CleanYoutubeURL(videoURL)
	if err != nil {
		return err
	}

	outputDir := EnsureOutputFolder()

	cmd := exec.Command("./yt-dlp.exe", "-x", "--audio-format", "mp3", "-o", filepath.Join(outputDir, "%(title)s.%(ext)s"), cleanURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func CleanYoutubeURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	videoID := q.Get("v")
	if videoID == "" {
		return "", fmt.Errorf("missing v parameter in URL")
	}
	return "https://www.youtube.com/watch?v=" + videoID, nil
}

func getDesktopPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Desktop", "YoutubeMp3")
}

func EnsureOutputFolder() string {
	outputDir := getDesktopPath()
	os.MkdirAll(outputDir, os.ModePerm)
	return outputDir
}
