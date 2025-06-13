package download

import (
	"os"
	"os/exec"
	"path/filepath"
)

func DownloadYoutubeAsMp3(videoURL string) error {

	outputDir := EnsureOutputFolder()

	cmd := exec.Command("./yt-dlp.exe", "-x", "--audio-format", "mp3", "-o", filepath.Join(outputDir, "%(title)s.%(ext)s"), videoURL)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()

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
