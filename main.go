package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

func main() {

	cmd := runCMD()
	defer cmd.Process.Kill()

	if err := waitForChromeReady("9222", 10*time.Second); err != nil {
		log.Fatal(err)
	}

	baseURL := "https://www.pinterest.com/"
	totalScroll := 100

	images, err := collectImageLinks(baseURL, totalScroll)
	if err != nil {
		panic(err)
	}

	// for _, image := range images {
	// 	fmt.Println("Image URL=> ", image)
	// }

	os.MkdirAll("./images", os.ModePerm)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // max 5 concurrent downloads

	for _, src := range images {
		wg.Add(1)
		sem <- struct{}{}

		go func(url string) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := downloadImage(url, "./images"); err != nil {
				fmt.Println("Failed :", url)
			} else {
				fmt.Println("Downloaded:", url)
			}
		}(src)
	}

	wg.Wait()

}

func extractHighestResImages(html string) []string {

	var images []string

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(html))

	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if srcset, exists := s.Attr("srcset"); exists {
			parts := strings.Split(srcset, ",")
			lasts := strings.TrimSpace(parts[len(parts)-1])
			url := strings.Fields(lasts)[0]
			images = append(images, url)
		} else if src, exists := s.Attr("src"); exists {
			images = append(images, src)
		}
	})
	return images
}

func collectImageLinks(url string, totalScroll int) ([]string, error) {
	allocatorCtx, _ := chromedp.NewRemoteAllocator(context.Background(), "http://localhost:9222")
	ctx, cancelCtx := chromedp.NewContext(allocatorCtx)
	defer cancelCtx()

	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		return nil, err
	}

	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		return nil, err
	}

	imageSet := make(map[string]struct{})
	spinner := []rune{'|', '/', '-', '\\'}

	for i := 0; i < totalScroll; i++ {
		fmt.Printf("\rScrolling %d/%d %c", i+1, totalScroll, spinner[i%len(spinner)])

		err := chromedp.Run(ctx,
			chromedp.Evaluate(`window.scrollBy(0, 500)`, nil),
			chromedp.Sleep(1*time.Second),
		)
		if err != nil {
			return nil, fmt.Errorf("scroll failed at %d: %w", i, err)
		}

		var html string
		if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html)); err != nil {
			return nil, fmt.Errorf("failed to extract HTML at scroll %d: %w", i, err)
		}

		imgs := extractHighestResImages(html)
		for _, img := range imgs {
			imageSet[img] = struct{}{}
		}
	}

	fmt.Printf("\nCollected %d unique image URLs.\n", len(imageSet))

	var images []string
	for img := range imageSet {
		images = append(images, img)
	}

	return images, nil
}

func downloadImage(imageURL, outputDir string) error {
	u, err := url.Parse(imageURL)
	if err != nil {
		return err
	}

	fileName := filepath.Base(u.Path)

	if !strings.Contains(fileName, ".") {
		q := u.Query()
		ext := q.Get("fm")
		if ext == "" {
			ext = "jpg"
		}

		fileName += "." + ext
	}

	resp, err := http.Get(imageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	outPath := filepath.Join(outputDir, fileName)
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	return err
}

func loadCookiesFromJSON(path string) ([]*network.CookieParam, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw []map[string]interface{}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	var cookies []*network.CookieParam
	for _, c := range raw {
		name := c["name"].(string)
		value := c["value"].(string)
		domain := c["domain"].(string)
		path := c["path"].(string)

		cookies = append(cookies, &network.CookieParam{
			Name:   name,
			Value:  value,
			Domain: domain,
			Path:   path,
		})
	}
	return cookies, nil
}

func runCMD() *exec.Cmd {
	cmd := exec.Command(`C:\Program Files\Google\Chrome\Application\chrome.exe`, "--remote-debugging-port=9222", "--user-data-dir=C:\\chrome-pinterest-profile")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start Chrome: %v", err)
	}
	return cmd
}

func waitForChromeReady(port string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:" + port + "/json/version")
		if err == nil {
			resp.Body.Close()
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("Chrome did not open in time")
}
