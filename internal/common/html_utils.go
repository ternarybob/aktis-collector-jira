package common

import (
	"strings"

	"golang.org/x/net/html"
)

// ExtractText gets all text content from an HTML node and its children
func ExtractText(node *html.Node) string {
	var text strings.Builder

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(node)
	return strings.TrimSpace(text.String())
}

// FindNodesByTag finds all nodes with a specific tag name
func FindNodesByTag(root *html.Node, tagName string) []*html.Node {
	var nodes []*html.Node

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == tagName {
			nodes = append(nodes, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(root)
	return nodes
}

// FindNodesByAttribute finds all nodes with a specific attribute
func FindNodesByAttribute(root *html.Node, attrKey, attrValue string) []*html.Node {
	var nodes []*html.Node

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, attr := range n.Attr {
				if attr.Key == attrKey && attr.Val == attrValue {
					nodes = append(nodes, n)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(root)
	return nodes
}

// GetAttribute gets the value of an attribute from a node
func GetAttribute(node *html.Node, attrKey string) string {
	if node.Type != html.ElementNode {
		return ""
	}
	for _, attr := range node.Attr {
		if attr.Key == attrKey {
			return attr.Val
		}
	}
	return ""
}

// HasAttribute checks if a node has a specific attribute
func HasAttribute(node *html.Node, attrKey string) bool {
	if node.Type != html.ElementNode {
		return false
	}
	for _, attr := range node.Attr {
		if attr.Key == attrKey {
			return true
		}
	}
	return false
}

// FindLinks extracts all href URLs from a node
func FindLinks(node *html.Node) []string {
	var links []string

	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(node)
	return links
}

// RenderHTML converts HTML node to string
func RenderHTML(n *html.Node, b *strings.Builder) {
	if n.Type == html.TextNode {
		b.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		RenderHTML(c, b)
	}
}
