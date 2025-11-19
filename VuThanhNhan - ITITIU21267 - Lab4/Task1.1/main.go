package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

type Book struct {
	Title        string `json:"title"`
	Price        string `json:"price"`
	Rating       string `json:"rating"`
	Availability string `json:"availability"`
	ImageURL     string `json:"image_url"`
}

// Scrape all book data from a page
func scrapeBooks(url string) ([]Book, error) {
	resp, err := http.Get(url) 
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err) 
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK { 
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body) // read entire body
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	doc, err := html.Parse(bytes.NewReader(body)) // parse HTML 
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var books []Book

	// Find all <article class="product_pod">
	var walk func(*html.Node) // recursive walk function
	walk = func(n *html.Node) { // traverse nodes
		if n.Type == html.ElementNode && n.Data == "article" {
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "product_pod") {
					book := extractBookData(n)
					books = append(books, book) // add book to list
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c) // recursive call
		}
	}
	walk(doc) // start walking from root

	return books, nil
}

// Extract book details from one <article>
func extractBookData(n *html.Node) Book {
	book := Book{}

	var walk func(*html.Node) 
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h3":
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					if c.Type == html.ElementNode && c.Data == "a" {
						for _, attr := range c.Attr {
							if attr.Key == "title" {
								book.Title = strings.TrimSpace(attr.Val)
							}
						}
					}
				}
			case "p":
				for _, attr := range n.Attr {
					// price
					if attr.Key == "class" && strings.Contains(attr.Val, "price_color") {
						if n.FirstChild != nil {
							book.Price = strings.TrimSpace(n.FirstChild.Data)
						}
					}
					// rating
					if attr.Key == "class" && strings.Contains(attr.Val, "star-rating") {
						classes := strings.Split(attr.Val, " ")
						if len(classes) > 1 {
							book.Rating = strings.TrimSpace(classes[1])
						}
					}
					// availability
					if attr.Key == "class" && strings.Contains(attr.Val, "availability") {
						book.Availability = strings.TrimSpace(extractText(n))
					}
				}
			case "img":
				for _, attr := range n.Attr {
					if attr.Key == "src" {
						book.ImageURL = "http://books.toscrape.com/" + strings.TrimPrefix(attr.Val, "../")
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)

	return book
}

// Extract text content (trimmed)
func extractText(n *html.Node) string {
	if n.Type == html.TextNode { 
		return strings.TrimSpace(n.Data)
	}
	var text string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += extractText(c)
	}
	return strings.TrimSpace(text)
}

// Save books to JSON file
func saveBooksToJSON(books []Book, filename string) error {
	data, err := json.MarshalIndent(books, "", "  ") // marshal to JSON with indentation
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return os.WriteFile(filename, data, 0644) // write JSON to file
}

// Helper: Convert "£51.77" to float64
func priceToFloat(price string) float64 {
	p := strings.TrimPrefix(price, "£")
	f, _ := strconv.ParseFloat(p, 64)
	return f
}

func main() {
	url := "http://books.toscrape.com/catalogue/page-1.html"
	fmt.Printf("Scraping books from: %s\n", url)

	books, err := scrapeBooks(url)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Found %d books\n\n", len(books))

	// Print first book as sample
	if len(books) > 0 {
		fmt.Println("Book 1:")
		fmt.Printf("  Title: %s\n", books[0].Title)
		fmt.Printf("  Price: %s\n", books[0].Price)
		fmt.Printf("  Rating: %s\n", books[0].Rating)
		fmt.Printf("  Availability: %s\n\n", books[0].Availability)
	}

	// Calculate average price
	var total float64
	for _, b := range books {
		total += priceToFloat(b.Price)
	}
	avg := total / float64(len(books))

	fmt.Printf("Summary:\n")
	fmt.Printf("  Total books: %d\n", len(books))
	fmt.Printf("  Average price: £%.2f\n\n", avg)

	// Save to JSON
	if err := saveBooksToJSON(books, "books.json"); err != nil {
		fmt.Printf("Failed to save JSON: %v\n", err)
		return
	}

	fmt.Printf("Saved %d books to books.json\n", len(books))
}
