package video

import (
	"GoTest/animation"
	"GoTest/input"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kkdai/youtube/v2"
	"github.com/sqweek/dialog"
)

func DownloadSelectedFormat(rawUrl string) error {
	parsed, err := url.Parse(rawUrl)
	if err != nil {
		return fmt.Errorf("invalid url: %v", err)
	}

	var videoID string

	// Handle different YouTube URL formats
	if strings.Contains(parsed.Host, "youtu.be") {
		// For youtu.be/VIDEO_ID format
		videoID = strings.TrimPrefix(parsed.Path, "/")
	} else if strings.Contains(parsed.Host, "youtube.com") {
		// For youtube.com/watch?v=VIDEO_ID format
		videoID = parsed.Query().Get("v")
	} else {
		return fmt.Errorf("unsupported URL format")
	}

	if videoID == "" {
		return fmt.Errorf("missing video id")
	}

	cleanUrl := "https://www.youtube.com/watch?v=" + videoID
	fmt.Println("Corrected URL:", cleanUrl)

	headerTransport := roundTripperWithHeaders{
		rt: http.DefaultTransport,
		headers: map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/113.0.0.0 Safari/537.36",
		},
	}
	client := youtube.Client{
		HTTPClient: &http.Client{Transport: headerTransport},
	}

	video, err := client.GetVideo(cleanUrl)
	if err != nil {
		return fmt.Errorf("failed to get video: %v", err)
	}

	index, err := input.GetSelectedVideoIndex(video)
	if err != nil {
		return err
	}
	videoFormat := video.Formats[index]
	if videoFormat.URL == "" {
		return fmt.Errorf("selected video format has no URL (ciphered)")
	}

	var audioFormat *youtube.Format
	for _, f := range video.Formats {
		if f.AudioChannels > 0 && f.QualityLabel == "" && f.URL != "" {
			audioFormat = &f
			break
		}
	}
	if audioFormat == nil {
		return fmt.Errorf("no audio-only format found")
	}

	// Create temporary directory for downloads
	tempDir, err := createTempDir()
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir) // Clean up temp directory

	// Use temporary files with unique names
	timestamp := time.Now().UnixNano()
	videoPath := filepath.Join(tempDir, fmt.Sprintf("video_%d.mp4", timestamp))
	audioPath := filepath.Join(tempDir, fmt.Sprintf("audio_%d.m4a", timestamp))

	state := &animation.DownloadState{}

	fmt.Println("Downloading video...")
	err = downloadVideo(&client, video, &videoFormat, videoPath, state)
	if err != nil {
		return err
	}

	fmt.Println("Downloading audio...")
	err = downloadAudio(&client, video, audioFormat, audioPath)
	if err != nil {
		return err
	}

	// Create temporary merged file
	tempOutputPath := filepath.Join(tempDir, fmt.Sprintf("merged_%d.mp4", timestamp))

	fmt.Println("Merging video and audio...")
	// Use ffmpeg from PATH or current directory
	ffmpegCmd := "ffmpeg"
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		// If ffmpeg not in PATH, try current directory
		ffmpegCmd = "./ffmpeg.exe"
	}

	cmd := exec.Command(ffmpegCmd, "-y", "-i", videoPath, "-i", audioPath, "-c", "copy", tempOutputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg merge failed: %v", err)
	}

	// Show file dialog to choose save location
	fmt.Println("Choose where to save the video...")
	outputPath, err := showSaveDialog(video)
	if err != nil {
		return fmt.Errorf("failed to get save location: %v", err)
	}

	// Move the merged file to the chosen location
	err = moveFile(tempOutputPath, outputPath)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	fmt.Println("Download and merge complete:", outputPath)
	return nil
}

func showSaveDialog(video *youtube.Video) (string, error) {
	safeTitle := sanitizeFileName(video.Title)
	defaultName := strings.ReplaceAll(safeTitle, " ", "_") + ".mp4"

	// Get user's home directory for default location
	home, err := os.UserHomeDir()
	if err != nil {
		home = "." // Use current directory as fallback
	}
	defaultDir := filepath.Join(home, "Desktop")

	// Show save dialog
	filename, err := dialog.File().
		Title("Save Video As").
		Filter("MP4 Video Files", "mp4").
		SetStartDir(defaultDir).
		SetStartFile(defaultName).
		Save()

	if err != nil {
		return "", err
	}

	// Ensure the file has .mp4 extension
	if !strings.HasSuffix(strings.ToLower(filename), ".mp4") {
		filename += ".mp4"
	}

	return filename, nil
}

func moveFile(src, dst string) error {
	// Try to rename first (fastest if on same drive)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// If rename fails, copy and delete
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Remove source file after successful copy
	return os.Remove(src)
}

func createTempDir() (string, error) {
	// Try to create temp dir in system temp first
	tempDir, err := os.MkdirTemp("", "youtube_dl_*")
	if err != nil {
		// If system temp fails, try current directory
		tempDir, err = os.MkdirTemp(".", "temp_*")
		if err != nil {
			return "", err
		}
	}
	return tempDir, nil
}

func downloadVideo(client *youtube.Client, video *youtube.Video, format *youtube.Format, path string, state *animation.DownloadState) error {
	stream, size, err := client.GetStream(video, format)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create video file: %v", err)
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

	if err != nil {
		return fmt.Errorf("failed to download video: %v", err)
	}

	return nil
}

func downloadAudio(client *youtube.Client, video *youtube.Video, format *youtube.Format, path string) error {
	stream, _, err := client.GetStream(video, format)
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create audio file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		return fmt.Errorf("failed to download audio: %v", err)
	}

	return nil
}

func sanitizeFileName(name string) string {
	invalid := []string{`<`, `>`, `:`, `"`, `/`, `\`, `|`, `?`, `*`}
	for _, c := range invalid {
		name = strings.ReplaceAll(name, c, "_")
	}
	return name
}

type roundTripperWithHeaders struct {
	rt      http.RoundTripper
	headers map[string]string
}

func (r roundTripperWithHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}
	return r.rt.RoundTrip(req)
}
