package main

import (
	"GoTest/input"
	"GoTest/video"
	"fmt"
	"strings"
)

func main() {
mainLoop:
	for {
		url, err := input.ReadInputWithTimeout("Enter YouTube URL (or type 'q' to quit): ", 600)
		if err != nil {
			fmt.Println("\nNo activity for 10 minutes. Exiting.")
			return
		}
		if strings.ToLower(url) == "q" {
			return
		}

		url, err = input.ValidateURL(strings.TrimSpace(url))
		fmt.Println("Validated URL:", url)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		}

	downloadLoop:
		for {
			err := video.DownloadSelectedFormat(url)
			if err != nil {
				fmt.Println("Download failed:", err)
			}

			choice, err := input.ReadInputWithTimeout("\nWhat next? [1] Same link [2] New link [3] Quit: ", 600)
			if err != nil {
				fmt.Println("\nNo activity for 10 minutes. Exiting.")
				return
			}

			switch choice {
			case "1":
				continue downloadLoop
			case "2":
				continue mainLoop
			case "3":
				return
			default:
				fmt.Println("Invalid choice. Exiting.")
				return
			}
		}
	}
}
