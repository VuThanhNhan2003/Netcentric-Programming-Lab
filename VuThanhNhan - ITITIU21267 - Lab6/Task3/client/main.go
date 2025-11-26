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

	// --- List Books ---
	fmt.Println("=== Test 1: List All Books ===")
	list, _ := client.ListBooks(ctx, &pb.ListBooksRequest{Page: 1, PageSize: 10})
	fmt.Println("Total books:", list.Total)
	for i, b := range list.Books {
		fmt.Printf("%d. %s by %s - $%.2f\n", i+1, b.Title, b.Author, b.Price)
	}

	// --- Get Book ---
	fmt.Println("\n=== Test 2: Get Book ===")
	book, _ := client.GetBook(ctx, &pb.GetBookRequest{Id: 1})
	fmt.Printf("Book ID: %d\nTitle: %s\nAuthor: %s\nPrice: %.2f\n",
		book.Book.Id, book.Book.Title, book.Book.Author, book.Book.Price)

	// --- Create Book ---
	fmt.Println("\n=== Test 3: Create Book ===")
	created, _ := client.CreateBook(ctx, &pb.CreateBookRequest{
		Title:         "Learning Go",
		Author:        "Jon Bodner",
		Isbn:          "9781492077213",
		Price:         31.50,
		Stock:         10,
		PublishedYear: 2021,
	})
	fmt.Println("Created book ID:", created.Book.Id)

	// --- Update Book ---
	fmt.Println("\n=== Test 4: Update Book ===")
	updated, _ := client.UpdateBook(ctx, &pb.UpdateBookRequest{
		Id:            1,
		Title:         "The Go Programming Language (2nd Edition)",
		Author:        "Alan Donovan",
		Isbn:          "9780134190440",
		Price:         35.99,
		Stock:         8,
		PublishedYear: 2024,
	})
	fmt.Printf("Updated book: %s\nNew price: %.2f\n", updated.Book.Title, updated.Book.Price)

	// --- Delete Book ---
	fmt.Println("\n=== Test 5: Delete Book ===")
	del, _ := client.DeleteBook(ctx, &pb.DeleteBookRequest{Id: 6})
	fmt.Println(del.Message)

	// --- Pagination ---
	fmt.Println("\n=== Test 6: Pagination ===")
	p1, _ := client.ListBooks(ctx, &pb.ListBooksRequest{Page: 1, PageSize: 3})
	fmt.Println("Page 1:", len(p1.Books), "books")

	p2, _ := client.ListBooks(ctx, &pb.ListBooksRequest{Page: 2, PageSize: 3})
	fmt.Println("Page 2:", len(p2.Books), "books")
}
