package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "book-catalog-grpc/proto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.Dial("localhost:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewBookCatalogClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Println("=== Test 1: Search by Title ===")
	doSearch(ctx, client, "go", "title")

	fmt.Println("\n=== Test 2: Search by Author ===")
	doSearch(ctx, client, "Robert", "author")

	fmt.Println("\n=== Test 3: Search by ISBN (exact) ===")
	doSearch(ctx, client, "9780134190440", "isbn")

	fmt.Println("\n=== Test 4: Search all fields ===")
	doSearch(ctx, client, "Learning", "all")

	fmt.Println("\n=== Test 5: Empty search (expect error) ===")
	doSearch(ctx, client, "", "title")

	fmt.Println("\n=== Test 6: Filter by price (20 - 45) ===")
	doFilter(ctx, client, 20.0, 45.0, 0, 0)

	fmt.Println("\n=== Test 7: Filter by year (>=2010) ===")
	doFilter(ctx, client, 0, 0, 2010, 0)

	fmt.Println("\n=== Test 8: Invalid filter (min > max) expect error ===")
	doFilter(ctx, client, 50.0, 20.0, 0, 0)

	fmt.Println("\n=== Test 9: Get Stats ===")
	doStats(ctx, client)
}

func doSearch(ctx context.Context, client pb.BookCatalogClient, q string, field string) {
	resp, err := client.SearchBooks(ctx, &pb.SearchBooksRequest{Query: q, Field: field})
	if err != nil {
		fmt.Printf("Search error: %v\n", err)
		return
	}
	fmt.Printf("Query=%q, Found=%d\n", resp.Query, resp.Count)
	for i, b := range resp.Books {
		fmt.Printf("%d. %s by %s (ISBN:%s) $%.2f, year=%d\n", i+1, b.Title, b.Author, b.Isbn, b.Price, b.PublishedYear)
	}
}

func doFilter(ctx context.Context, client pb.BookCatalogClient, minPrice, maxPrice float32, minYear, maxYear int32) {
	resp, err := client.FilterBooks(ctx, &pb.FilterBooksRequest{
		MinPrice: minPrice, MaxPrice: maxPrice, MinYear: minYear, MaxYear: maxYear,
	})
	if err != nil {
		fmt.Printf("Filter error: %v\n", err)
		return
	}
	fmt.Printf("Filter found %d books\n", resp.Count)
	for i, b := range resp.Books {
		fmt.Printf("%d. %s by %s - $%.2f (year %d)\n", i+1, b.Title, b.Author, b.Price, b.PublishedYear)
	}
}

func doStats(ctx context.Context, client pb.BookCatalogClient) {
	resp, err := client.GetStats(ctx, &pb.GetStatsRequest{})
	if err != nil {
		fmt.Printf("Stats error: %v\n", err)
		return
	}
	fmt.Printf("Total books: %d\nAverage price: %.2f\nTotal stock: %d\nYear range: %d - %d\n",
		resp.TotalBooks, resp.AveragePrice, resp.TotalStock, resp.EarliestYear, resp.LatestYear)
}
