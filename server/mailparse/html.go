package mailparse

import (
	"bytes"
	"golang.org/x/net/html"
	"strings"
)

func htmlText(data []byte) string {
	root, e := html.Parse(bytes.NewReader(data))
	if e != nil {
		return ""
	}
	var out strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "head", "iframe", "object":
				return
			case "br", "p", "div", "li", "tr":
				out.WriteString("\n")
			}
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	return strings.TrimSpace(out.String())
}
