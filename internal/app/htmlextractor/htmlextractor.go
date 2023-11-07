package htmlextractor

import (
	"bytes"
	"strings"

	"golang.org/x/net/html"
)

func ExtractTextFromHTML(htmlData []byte) (string, error) {
	node, err := html.Parse(bytes.NewReader(htmlData))
	if err != nil {
		return "", err
	}

	var textBuilder strings.Builder
	extractNodeText(node, &textBuilder)

	return textBuilder.String(), nil
}

func extractNodeText(n *html.Node, tb *strings.Builder) {
	if n.Type == html.TextNode {
		tb.WriteString(n.Data)
		tb.WriteString(" ")
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractNodeText(c, tb)
	}
}
