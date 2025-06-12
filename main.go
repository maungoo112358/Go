package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

func main() {

	baseURL := "https://unsplash.com/"
	resp, err := http.Get(baseURL)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()
	doc, err := html.Parse(resp.Body)

	if err != nil {
		log.Fatal(err)
	}

	parsedURL, _ := url.Parse(baseURL)
	var images []string
	extractImgSrcs(doc, parsedURL, &images)

	// for _, image := range images {
	// 	fmt.Println("Image URL=> ", image)
	// }

	os.MkdirAll("./images", os.ModePerm)

	for _, src := range images {
		err := downloadImage(src, "./images")
		if err != nil {
			fmt.Println("Failed :(")
		} else {
			fmt.Println("Downlaoded:=> ", src)
		}

	}

}

func extractImgSrcs(n *html.Node, base *url.URL, list *[]string) {

	if n.Type == html.ElementNode && n.Data == "img" {
		var imageUrl string

		for _, attr := range n.Attr {
			if attr.Key == "src" {
				imageUrl = attr.Val
			}

			if attr.Key == "srcset" && imageUrl == "" {
				parts := strings.Split(attr.Val, ",")
				last := strings.TrimSpace(parts[len(parts)-1])
				urlPart := strings.Fields(last)[0]
				imageUrl = urlPart
			}
		}

		if imageUrl != "" {
			resolved, err := base.Parse(imageUrl)

			if err == nil {
				*list = append(*list, resolved.String())
			}
		}

	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractImgSrcs(c, base, list)
	}

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
