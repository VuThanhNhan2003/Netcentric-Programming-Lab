package main

import (
	"fmt"
	"log"
	
	// Import the generated protobuf code
	pb "book-catalog-grpc/proto"
	"google.golang.org/protobuf/proto"
)

func main() {
	// Create a Book instance
	book := &pb.Book{
		Id:           1,
		Title:        "The Go Programming Language",
		Author:       "Alan Donovan",
		Isbn:         "978-0134190440",
		Price:        39.99,
		Stock:        15,
		PublishedYear: 2015,
	}
	
	fmt.Printf("Book: %v\n", book)
	
	// Create DetailedBook with category and tags
	detailedBook := &pb.DetailedBook{
		Book: &pb.Book{
			Id:           1,
			Title:        "The Go Programming Language",
			Author:       "Alan Donovan",
			Isbn:         "978-0134190440",
			Price:        39.99,
			Stock:        15,
			PublishedYear: 2015,
		},
		Category:   pb.BookCategory_NONFICTION,
		Description: "A comprehensive introduction to Go programming.",
		Tags:       []string{"programming", "go", "technical"},
		Rating:     4.5,
	}
	
	fmt.Printf("\nDetailed Book: %v\n", detailedBook)
	fmt.Printf("Category: %s\n", detailedBook.Category)
	fmt.Printf("Tags: %v\n", detailedBook.Tags)
	
	// Serialize to bytes
	data, err := proto.Marshal(book)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("\nSerialized size: %d bytes\n", len(data))
	
	// Deserialize from bytes
	newBook := &pb.Book{}
	err = proto.Unmarshal(data, newBook)
	if err != nil {
		log.Fatal(err)
	}
	
	fmt.Printf("Deserialized book: %v\n", newBook)
	
	// Create Author with multiple books
	author := &pb.Author{
		Id:    1,
		Name:  "Robert C. Martin",
		Bio:   "A renowned author in software development.",
		BirthYear: 1952,
		Books: []*pb.Book{
			{
				Id:           1,
				Title:        "Clean Code",
				Author:       "Robert C. Martin",
				Isbn:         "978-0132350884",
				Price:        29.99,
				Stock:        50,
				PublishedYear: 2008,
			},
			{
				Id:           2,
				Title:        "Clean Architecture",
				Author:       "Robert C. Martin",
				Isbn:         "978-0134494166",
				Price:        34.99,
				Stock:        30,
				PublishedYear: 2017,
			},
		},
	}
	
	fmt.Printf("\nAuthor: %s\n", author.Name)
	fmt.Printf("Books written: %d\n", len(author.Books))
	for i, b := range author.Books {
		fmt.Printf("  %d. %s\n", i+1, b.Title)
	}
}
