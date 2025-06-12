package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

func main() {

	baseURL := "https://unsplash.com/"
	totalScroll := 10000

	images, err := collectImageLinks(baseURL, totalScroll)
	if err != nil {
		panic(err)
	}

	// for _, image := range images {
	// 	fmt.Println("Image URL=> ", image)
	// }

	os.MkdirAll("./images", os.ModePerm)

	for _, src := range images {
		err := downloadImage(src, "./images")
		if err != nil {
			fmt.Println("Failed :(")
		} else {
			fmt.Println("Downloaded:=> ", src)
		}

	}

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
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	if err := chromedp.Run(ctx, chromedp.Navigate(url)); err != nil {
		return nil, err
	}

	var html string
	spinner := []rune{'|', '/', '-', '\\'}

	for i := 0; i < totalScroll; i++ {
		fmt.Printf("\r Scrolling %d/%d %c", i+1, totalScroll, spinner[i%len(spinner)])

		err := chromedp.Run(ctx,
			chromedp.Evaluate(`window.scrollBy(0,1200)`, nil),
			chromedp.Sleep(100*time.Microsecond),
		)

		if err != nil {
			return nil, fmt.Errorf("scroll failed on iteration %d: %w", i, err)
		}
	}
	fmt.Println("\nExtracting HTML...")

	err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html))
	if err != nil {
		return nil, fmt.Errorf("failed to get HTML: %w", err)
	}

	images := extractHighestResImages(html)
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
