package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "book-catalog-grpc/proto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

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

	// === Test 1 ===
	fmt.Println("=== Test 1: Search by Title ===")
	fmt.Println(`Searching for "go"...`)
	searchAndCount(ctx, client, "go", "title")

	// === Test 2 ===
	fmt.Println("\n=== Test 2: Search by Author ===")
	fmt.Println(`Searching for "Martin"...`)
	searchAndCount(ctx, client, "Martin", "author")

	// === Test 3 ===
	fmt.Println("\n=== Test 3: Filter by Price ===")
	fmt.Println("Books between $20 and $45:")
	filterAndCount(ctx, client, 20, 45, 0, 0)

	// === Test 4 ===
	fmt.Println("\n=== Test 4: Filter by Year ===")
	fmt.Println("Books published after 2010:")
	filterAndCount(ctx, client, 0, 0, 2010, 0)

	// === Test 5 ===
	fmt.Println("\n=== Test 5: Get Statistics ===")
	doStats(ctx, client)

	// === Test 6 ===
	fmt.Println("\n=== Test 6: Error Cases ===")
	fmt.Println("Empty search query:")
	searchError(ctx, client, "", "title")

	fmt.Println("Invalid price range:")
	filterError(ctx, client, 50, 20, 0, 0)
}

func searchAndCount(ctx context.Context, client pb.BookCatalogClient, q, field string) {
	resp, err := client.SearchBooks(ctx, &pb.SearchBooksRequest{Query: q, Field: field})
	if err != nil {
		printGrpcError(err)
		return
	}

	fmt.Printf("Found %d books:\n", resp.Count)
	for _, b := range resp.Books {
		fmt.Printf("- %s\n", b.Title)
	}
}

func searchError(ctx context.Context, client pb.BookCatalogClient, q, field string) {
	_, err := client.SearchBooks(ctx, &pb.SearchBooksRequest{Query: q, Field: field})
	printGrpcError(err)
}

func filterAndCount(ctx context.Context, client pb.BookCatalogClient,
	minPrice, maxPrice float32, minYear, maxYear int32) {

	resp, err := client.FilterBooks(ctx, &pb.FilterBooksRequest{
		MinPrice: minPrice, MaxPrice: maxPrice, MinYear: minYear, MaxYear: maxYear,
	})

	if err != nil {
		printGrpcError(err)
		return
	}

	fmt.Printf("Found %d books\n", resp.Count)
}

func filterError(ctx context.Context, client pb.BookCatalogClient,
	minPrice, maxPrice float32, minYear, maxYear int32) {

	_, err := client.FilterBooks(ctx, &pb.FilterBooksRequest{
		MinPrice: minPrice, MaxPrice: maxPrice, MinYear: minYear, MaxYear: maxYear,
	})

	printGrpcError(err)
}

func doStats(ctx context.Context, client pb.BookCatalogClient) {
	resp, err := client.GetStats(ctx, &pb.GetStatsRequest{})
	if err != nil {
		printGrpcError(err)
		return
	}

	fmt.Printf("Total books: %d\n", resp.TotalBooks)
	fmt.Printf("Average price: $%.2f\n", resp.AveragePrice)
	fmt.Printf("Total stock: %d\n", resp.TotalStock)
	fmt.Printf("Year range: %d - %d\n", resp.EarliestYear, resp.LatestYear)
}

// Helper: clean gRPC error to match required output
func printGrpcError(err error) {
	if err == nil {
		return
	}
	st, ok := status.FromError(err)
	if ok {
		fmt.Printf("Error: %s\n", st.Message())
	} else {
		fmt.Printf("Error: %v\n", err)
	}
}