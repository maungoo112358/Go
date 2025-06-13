package main

import (
	"GoTest/input"
	"GoTest/video"
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {

	//https://www.youtube.com/watch?v=G9RpHfPyEx8
	fmt.Println("Enter Youtube URL=> ")
	reader := bufio.NewReader(os.Stdin)
	raw, _ := reader.ReadString('\n')
	url := strings.TrimSpace(raw)

	url, err := input.ValidateURL(url)
	if err != nil {
		fmt.Println("Error: ", err)
		return
	}

	fmt.Println("Valid Youtube URL => ", url)

	if err := video.DownloadSelectedFormat(url); err != nil {
		fmt.Println("Failed to fetch formats: ", err)
	}
}
