package application

import (
	"os"
	"time"

	"github.com/rs/zerolog/log"
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
		log.Trace().Err(err).Str("filePath", filePath).Msg("Error opening html file to get title.")
		return
	}
	defer file.Close()

	doc, err := html.Parse(file)
	if err != nil {
		log.Trace().Err(err).Str("filePath", filePath).Msg("Error parsing HTML to get title.")
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
		log.Trace().Err(err).Str("filePath", filePath).Msg("Failed to stat file.")
		return
	}
	return fileInfo.ModTime()
}
