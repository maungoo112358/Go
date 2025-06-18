package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/webp"
)

type Frame struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type FrameInfo struct {
	Frame Frame `json:"frame"`
}

type SpriteSheet struct {
	Frames map[string]FrameInfo `json:"frames"`
}

func getDesktopPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	return filepath.Join(home, "Desktop")
}

func ensureFolderExists(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}
}

func findFilesByExtensions(dir string, exts []string) []string {
	matches := []string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return matches
	}
	for _, e := range entries {
		if !e.IsDir() {
			for _, ext := range exts {
				if strings.HasSuffix(strings.ToLower(e.Name()), ext) {
					matches = append(matches, filepath.Join(dir, e.Name()))
				}
			}
		}
	}
	return matches
}

func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".webp":
		return webp.Decode(f)
	case ".png":
		return png.Decode(f)
	case ".jpg", ".jpeg":
		return jpeg.Decode(f)
	default:
		return nil, fmt.Errorf("unsupported image format: %s", ext)
	}
}

func green(text string) string {
	return "\033[32m" + text + "\033[0m"
}

func red(text string) string {
	return "\033[31m" + text + "\033[0m"
}

func processAtlas(jsonPath, imagePath, outputPath string) {
	jsonData, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		panic(err)
	}

	var sheet SpriteSheet
	err = json.Unmarshal(jsonData, &sheet)
	if err != nil {
		panic(err)
	}

	atlasImg, err := decodeImage(imagePath)
	if err != nil {
		panic(err)
	}

	os.MkdirAll(outputPath, 0755)

	for name, info := range sheet.Frames {
		rect := image.Rect(info.Frame.X, info.Frame.Y, info.Frame.X+info.Frame.W, info.Frame.Y+info.Frame.H)
		crop := image.NewRGBA(image.Rect(0, 0, info.Frame.W, info.Frame.H))
		draw.Draw(crop, crop.Bounds(), atlasImg, rect.Min, draw.Src)

		outFile := filepath.Join(outputPath, name)
		os.MkdirAll(filepath.Dir(outFile), 0755)
		f, err := os.Create(outFile)
		if err != nil {
			panic(err)
		}
		err = png.Encode(f, crop)
		f.Close()
		if err != nil {
			panic(err)
		}
		fmt.Println(green("Saved: ") + outFile)
	}
}

func main() {
	basePath := filepath.Join(getDesktopPath(), "Atlas Splitter")
	scanner := bufio.NewScanner(os.Stdin)

	for {
		ensureFolderExists(basePath)
		fmt.Printf("\n%s\n%s\n",
			green("[ Step 1 ] Place exactly one .json and one image file (.webp/.png/.jpg) inside this folder:"),
			green(basePath))
		fmt.Print(green("After adding those files, type 'y' to continue or 'n' to quit: "))

		scanner.Scan()
		input := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if input != "y" {
			fmt.Println(green("Exiting."))
			return
		}

		ensureFolderExists(basePath)

		jsonFiles := findFilesByExtensions(basePath, []string{".json"})
		imageFiles := findFilesByExtensions(basePath, []string{".webp", ".png", ".jpg", ".jpeg"})

		if len(jsonFiles) == 0 {
			fmt.Println(red("Error: No .json file found."))
			continue
		} else if len(jsonFiles) > 1 {
			fmt.Println(red("Error: Multiple .json files found. Please leave only one."))
			continue
		}

		if len(imageFiles) == 0 {
			fmt.Println(red("Error: No image file (.webp/.png/.jpg) found."))
			continue
		} else if len(imageFiles) > 1 {
			fmt.Println(red("Error: Multiple image files found. Please leave only one."))
			continue
		}

		outputPath := filepath.Join(basePath, "output")
		processAtlas(jsonFiles[0], imageFiles[0], outputPath)
		fmt.Println("\n" + green("Done. You can replace the files and type 'y' to split again, or 'n' to quit."))
	}
}
