package application

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/net/html"
)

func findFirstH1(node *html.Node) *html.Node {
	if node.Type == html.ElementNode && node.Data == "h1" {
		return node
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if h1Node := findFirstH1(child); h1Node != nil {
			return h1Node
		}
	}

	return nil
}

func GetTitleForHtmlFile(filePath string) (val *string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening the file:", err)
		return
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		fmt.Println("Error parsing the HTML:", err)
		return
	}

	h1Element := findFirstH1(doc)
	if h1Element != nil {
		return &h1Element.FirstChild.Data
	} else {
		return nil
	}
}

func GetCreationTimeForFile(filePath string) (creationTime time.Time) {
	fileInfo, err := os.Stat(filePath)

	if err != nil {
		return
	}
	return fileInfo.ModTime()
}
