package main

import (
	"context"
	"fmt"
	"log"
	"time"
	
	authorpb "book-catalog-grpc/proto/proto"
	bookpb "book-catalog-grpc/proto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to both services
	bookConn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer bookConn.Close()
	
	authorConn, err := grpc.Dial("localhost:50052",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer authorConn.Close()
	
	bookClient := bookpb.NewBookCatalogClient(bookConn)
	authorClient := authorpb.NewAuthorCatalogClient(authorConn)
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	fmt.Println("=== Microservice Demo ===\n")
	
	// 1. Create author
	fmt.Println("1. Creating author...")
	authorResp, err := authorClient.CreateAuthor(ctx, &authorpb.CreateAuthorRequest{
		Name:      "Martin Fowler",
		Bio:       "Software development expert",
		BirthYear: 1963,
		Country:   "UK",
	})
	if err != nil {
		log.Fatalf("Failed to create author: %v", err)
	}
	
	fmt.Printf("✓ Created author: %s (ID: %d)\n\n", 
		authorResp.Author.Name, authorResp.Author.Id)
	
	// 2. Create books for this author
	fmt.Println("2. Creating books for author...")
	book1, err := bookClient.CreateBook(ctx, &bookpb.CreateBookRequest{
		Title:         "Refactoring",
		Author:        authorResp.Author.Name,
		AuthorId:      authorResp.Author.Id,
		Isbn:          "978-0134757599",
		Price:         49.99,
		Stock:         15,
		PublishedYear: 2018,
	})
	if err != nil {
		log.Fatalf("Failed to create book 1: %v", err)
	}
	if book1 == nil || book1.Book == nil {
		log.Fatal("Book 1 response is nil")
	}
	fmt.Printf("✓ Created book: %s\n", book1.Book.Title)
	
	book2, err := bookClient.CreateBook(ctx, &bookpb.CreateBookRequest{
		Title:         "Patterns of Enterprise Application Architecture",
		Author:        authorResp.Author.Name,
		AuthorId:      authorResp.Author.Id,
		Isbn:          "978-0321127426",
		Price:         54.99,
		Stock:         8,
		PublishedYear: 2002,
	})
	if err != nil {
		log.Fatalf("Failed to create book 2: %v", err)
	}
	if book2 == nil || book2.Book == nil {
		log.Fatal("Book 2 response is nil")
	}
	fmt.Printf("✓ Created book: %s\n\n", book2.Book.Title)
	
	// 3. Get author's books (cross-service call)
	fmt.Println("3. Fetching author's books (cross-service call)...")
	booksResp, err := authorClient.GetAuthorBooks(ctx, &authorpb.GetAuthorBooksRequest{
		AuthorId: authorResp.Author.Id,
	})
	if err != nil {
		log.Fatalf("Failed to get author books: %v", err)
	}
	
	fmt.Printf("✓ Author: %s\n", booksResp.Author.Name)
	fmt.Printf("✓ Books written: %d\n", booksResp.BookCount)
	for i, book := range booksResp.Books {
		fmt.Printf("  %d. %s (%d) - $%.2f\n", 
			i+1, book.Title, book.PublishedYear, book.Price)
	}
	
	fmt.Println("\n✓ Microservice demo completed!")
}