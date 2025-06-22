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
	"strconv"
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

func parseAtlas(path string) map[string]FrameInfo {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(data), "\n")
	frames := make(map[string]FrameInfo)
	var name string
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "size:") || strings.HasPrefix(line, "format:") ||
			strings.HasPrefix(line, "filter:") || strings.HasPrefix(line, "repeat:") {
			continue
		}
		if !strings.Contains(line, ":") {
			name = line
			continue
		}
		if strings.HasPrefix(line, "xy:") && i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "size:") {
			xy := strings.Split(strings.TrimSpace(strings.TrimPrefix(line, "xy:")), ",")
			size := strings.Split(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[i+1]), "size:")), ",")
			x, _ := strconv.Atoi(strings.TrimSpace(xy[0]))
			y, _ := strconv.Atoi(strings.TrimSpace(xy[1]))
			w, _ := strconv.Atoi(strings.TrimSpace(size[0]))
			h, _ := strconv.Atoi(strings.TrimSpace(size[1]))
			frames[name+".png"] = FrameInfo{Frame: Frame{X: x, Y: y, W: w, H: h}}
		}
	}
	return frames
}

func processAtlas(metaPath, imagePath, outputPath string) {
	atlasImg, err := decodeImage(imagePath)
	if err != nil {
		panic(err)
	}

	frames := make(map[string]FrameInfo)
	if strings.HasSuffix(metaPath, ".atlas") {
		frames = parseAtlas(metaPath)
	} else {
		jsonData, err := ioutil.ReadFile(metaPath)
		if err != nil {
			panic(err)
		}
		var sheet SpriteSheet
		err = json.Unmarshal(jsonData, &sheet)
		if err != nil {
			panic(err)
		}
		frames = sheet.Frames
	}

	os.MkdirAll(outputPath, 0755)

	for name, info := range frames {
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
			green("[ Step 1 ] Place exactly one .json/.atlas and one image file (.webp/.png/.jpg) inside this folder:"),
			green(basePath))
		fmt.Print(green("After adding those files, type 'y' to continue or 'n' to quit: "))

		scanner.Scan()
		input := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if input != "y" {
			fmt.Println(green("Exiting."))
			return
		}

		ensureFolderExists(basePath)

		metaFiles := findFilesByExtensions(basePath, []string{".json", ".atlas"})
		imageFiles := findFilesByExtensions(basePath, []string{".webp", ".png", ".jpg", ".jpeg"})

		if len(metaFiles) == 0 {
			fmt.Println(red("Error: No .json or .atlas file found."))
			continue
		} else if len(metaFiles) > 1 {
			fmt.Println(red("Error: Multiple .json/.atlas files found. Please leave only one."))
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
		processAtlas(metaFiles[0], imageFiles[0], outputPath)
		fmt.Println("\n" + green("Done. You can replace the files and type 'y' to split again, or 'n' to quit."))
	}
}
