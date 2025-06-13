package input

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
)

var youtubeRegex = regexp.MustCompile(`^(https?://)?(www\.)?(youtube\.com|youtu\.be)/.+$`)

func ValidateURL(url string) (string, error) {
	if youtubeRegex.MatchString(url) {
		return url, nil
	}

	return "", errors.New("Invalid Youtube URL :(")
}

func GetSelectedVideoIndex(video *youtube.Video) (int, error) {
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

func ReadInputWithTimeout(prompt string, timeoutSec int) (string, error) {
	fmt.Print(prompt)
	inputChan := make(chan string)
	errorChan := make(chan error)

	go func() {
		reader := bufio.NewReader(os.Stdin)
		text, err := reader.ReadString('\n')
		if err != nil {
			errorChan <- err
			return
		}
		inputChan <- strings.TrimSpace(text)
	}()

	select {
	case input := <-inputChan:
		return input, nil
	case err := <-errorChan:
		return "", err
	case <-time.After(time.Duration(timeoutSec) * time.Second):
		return "", fmt.Errorf("timeout")
	}
}
