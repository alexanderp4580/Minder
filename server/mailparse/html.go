package mailparse

import (
	"bytes"
	"golang.org/x/net/html"
	"net/url"
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
			case "br", "p", "div", "li", "tr", "td", "th", "h1", "h2", "h3", "section", "address":
				out.WriteString("\n")
			}
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
		}
		start := out.Len()
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			label := out.String()[start:]
			for _, attr := range n.Attr {
				if attr.Key != "href" {
					continue
				}
				target := strings.TrimSpace(attr.Val)
				parsed, err := url.Parse(target)
				if err != nil {
					break
				}
				allowed := (parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil) ||
					((parsed.Scheme == "tel" || parsed.Scheme == "mailto") && (parsed.Opaque != "" || parsed.Path != ""))
				if allowed && !strings.Contains(label, target) {
					out.WriteString(" (")
					out.WriteString(target)
					out.WriteString(") ")
				}
				break
			}
		}
	}
	walk(root)
	return strings.TrimSpace(out.String())
}
