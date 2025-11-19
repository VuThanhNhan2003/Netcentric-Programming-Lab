package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/html"
	// "io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ============================================================================
// Data structures
// ============================================================================

type Book struct {
	Title  string  `json:"title"`
	Price  string  `json:"price"`
	Rating string  `json:"rating"`
	URL    string  `json:"url"`
}

type ScraperStats struct {
	PagesScraped int
	BooksFound   int
	Errors       int
	StartTime    time.Time
	EndTime      time.Time
}

// ============================================================================
// Utility functions
// ============================================================================

// fetchPage downloads and parses HTML from a URL
func fetchPage(pageURL string) (*html.Node, error) {
	resp, err := http.Get(pageURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	return doc, nil
}

// extractBooks extracts book info from one page
func extractBooks(doc *html.Node, baseURL string) []Book {
	var books []Book
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "article" {
			for _, a := range n.Attr {
				if a.Key == "class" && strings.Contains(a.Val, "product_pod") {
					var b Book

					// Find title and relative link
					findLink := func(n *html.Node) {
						if n.Type == html.ElementNode && n.Data == "a" {
							for _, a := range n.Attr {
								if a.Key == "title" {
									b.Title = a.Val
								}
								if a.Key == "href" {
									link, _ := url.JoinPath(baseURL, a.Val)
									b.URL = link
								}
							}
						}
					}

					// Find price and rating
					findPrice := func(n *html.Node) {
						if n.Type == html.ElementNode && n.Data == "p" {
							for _, a := range n.Attr {
								if a.Key == "class" && strings.Contains(a.Val, "price_color") && n.FirstChild != nil {
									b.Price = n.FirstChild.Data
								}
								if a.Key == "class" && strings.Contains(a.Val, "star-rating") {
									b.Rating = strings.TrimPrefix(a.Val, "star-rating ")
								}
							}
						}
					}

					// Walk inside <article>
					var inner func(*html.Node)
					inner = func(c *html.Node) {
						findLink(c)
						findPrice(c)
						for child := c.FirstChild; child != nil; child = child.NextSibling {
							inner(child) // recursive
						}
					}
					inner(n)

					if b.Title != "" {
						books = append(books, b)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return books
}

// getNextPageURL finds the next page link and returns absolute URL
func getNextPageURL(doc *html.Node, baseURL string) (string, bool) {
	var nextURL string
	var found bool

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "li" {
			for _, a := range n.Attr {
				if a.Key == "class" && a.Val == "next" {
					// find <a> inside it
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if c.Type == html.ElementNode && c.Data == "a" {
							for _, ha := range c.Attr {
								if ha.Key == "href" {
									// Join with base catalog URL, not current page
									u, _ := url.Parse(ha.Val)
									base, _ := url.Parse(baseURL)
									nextURL = base.ResolveReference(u).String()
									found = true
									return
								}
							}
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil && !found; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)
	return nextURL, found
}


// ============================================================================
// Main scraper logic
// ============================================================================

func scrapePaginatedBooks(baseURL string, maxPages int) ([]Book, *ScraperStats, error) {
	stats := &ScraperStats{StartTime: time.Now()}
	var allBooks []Book
	currentURL := baseURL

	for page := 1; page <= maxPages; page++ {
		fmt.Printf("Scraping page %d/%d...\n", page, maxPages)

		doc, err := fetchPage(currentURL)
		if err != nil {
			fmt.Printf("  Error loading page: %v\n", err)
			stats.Errors++
			time.Sleep(2 * time.Second)
			continue
		}

		books := extractBooks(doc, baseURL)
		fmt.Printf("  Found %d books\n", len(books))
		allBooks = append(allBooks, books...)
		stats.PagesScraped++
		stats.BooksFound += len(books)

		nextURL, ok := getNextPageURL(doc, currentURL)
		if !ok {
			break
		}
		currentURL = nextURL

		// Rate limit
		time.Sleep(1 * time.Second)
	}

	stats.EndTime = time.Now()
	return allBooks, stats, nil
}

// ============================================================================
// Reporting
// ============================================================================

func printStats(stats *ScraperStats) {
	duration := stats.EndTime.Sub(stats.StartTime).Seconds()
	avgBooks := 0.0
	if stats.PagesScraped > 0 {
		avgBooks = float64(stats.BooksFound) / float64(stats.PagesScraped)
	}
	fmt.Println("\n=== Scraping Statistics ===")
	fmt.Printf("Pages scraped: %d\n", stats.PagesScraped)
	fmt.Printf("Total books found: %d\n", stats.BooksFound)
	fmt.Printf("Errors: %d\n", stats.Errors)
	fmt.Printf("Duration: %.2f seconds\n", duration)
	fmt.Printf("Average books per page: %.1f\n", avgBooks)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	baseURL := "http://books.toscrape.com/catalogue/"
	startPage := "page-1.html"
	maxPages := 5

	fmt.Printf("Starting paginated scraper...\n")
	fmt.Printf("Max pages: %d\n\n", maxPages)

	allBooks, stats, err := scrapePaginatedBooks(baseURL+startPage, maxPages)
	if err != nil {
		fmt.Printf("Scraping failed: %v\n", err)
		return
	}

	printStats(stats)

	data, _ := json.MarshalIndent(allBooks, "", "  ")
	filename := "paginated_books.json"
	_ = os.WriteFile(filename, data, 0644)

	fmt.Printf("\nSaved %d books to %s\n", len(allBooks), filename)
}
