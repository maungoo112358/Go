package main

import (
	"GoTest/download"
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {

	timeout := 10 * time.Minute
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\nEnter Youtube URL=> ")

		inputCh := make(chan string)
		go func() {
			text, _ := reader.ReadString('\n')
			inputCh <- strings.TrimSpace(text)
		}()

		select {
		case videoURL := <-inputCh:
			if !isYoutubeURL(videoURL) {
				fmt.Println("Invalid Youtube URL")
				continue
			}

			if err := download.DownloadYoutubeAsMp3(videoURL); err != nil {
				fmt.Println("Download failed :( ", err)
			} else {
				fmt.Println("Download Complete.")
			}
			fmt.Println("Download Another? (y/n)=> ")
			answer, _ := reader.ReadString('\n')
			if strings.TrimSpace(strings.ToLower(answer)) != "y" {
				fmt.Println("Existing...")
				return
			}

		case <-time.After(timeout):
			fmt.Println("\nNo input for 10minues. Existing")
			return
		}
	}
}

func isYoutubeURL(link string) bool {
	return strings.Contains(link, "youtube.com/watch") || strings.Contains(link, "youtu.be/")
}
