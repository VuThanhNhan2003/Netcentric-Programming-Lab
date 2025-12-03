package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	
	authorpb "book-catalog-grpc/proto/proto"
	bookpb "book-catalog-grpc/proto/proto"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authorCatalogServer struct {
	authorpb.UnimplementedAuthorCatalogServer
	db         *sql.DB
	bookClient bookpb.BookCatalogClient  // Client to Book service
}

func newServer(db *sql.DB, bookClient bookpb.BookCatalogClient) *authorCatalogServer {
	return &authorCatalogServer{
		db:         db,
		bookClient: bookClient,
	}
}

func (s *authorCatalogServer) GetAuthor(ctx context.Context, req *authorpb.GetAuthorRequest) (*authorpb.GetAuthorResponse, error) {
	// TODO: Implement GetAuthor
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *authorCatalogServer) CreateAuthor(ctx context.Context, req *authorpb.CreateAuthorRequest) (*authorpb.CreateAuthorResponse, error) {
	// TODO: Implement CreateAuthor
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *authorCatalogServer) ListAuthors(ctx context.Context, req *authorpb.ListAuthorsRequest) (*authorpb.ListAuthorsResponse, error) {
	// TODO: Implement ListAuthors with pagination
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (s *authorCatalogServer) GetAuthorBooks(ctx context.Context, req *authorpb.GetAuthorBooksRequest) (*authorpb.GetAuthorBooksResponse, error) {
	log.Printf("GetAuthorBooks: author_id=%d", req.AuthorId)
	
	// TODO: Get author from database
	var author authorpb.Author
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, bio, birth_year, country FROM authors WHERE id = ?",
		req.AuthorId,
	).Scan(&author.Id, &author.Name, &author.Bio, &author.BirthYear, &author.Country)
	
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "author not found: id=%d", req.AuthorId)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	
	// TODO: Call Book service to get books by this author
	// This demonstrates service-to-service communication!
	bookResp, err := s.bookClient.GetBooksByAuthor(ctx, &bookpb.GetBooksByAuthorRequest{
		AuthorId: req.AuthorId,
	})
	if err != nil {
		log.Printf("Failed to get books: %v", err)
		// Continue even if book service fails
		return &authorpb.GetAuthorBooksResponse{
			Author:    &author,
			Books:     nil,
			BookCount: 0,
		}, nil
	}
	
	// TODO: Convert books to BookSummary
	var bookSummaries []*authorpb.BookSummary
	for _, book := range bookResp.Books {
		bookSummaries = append(bookSummaries, &authorpb.BookSummary{
			Id:            book.Id,
			Title:         book.Title,
			Price:         book.Price,
			PublishedYear: book.PublishedYear,
		})
	}
	
	return &authorpb.GetAuthorBooksResponse{
		Author:    &author,
		Books:     bookSummaries,
		BookCount: int32(len(bookSummaries)),
	}, nil
}

func connectToBookService() (bookpb.BookCatalogClient, error) {
	// TODO: Connect to Book service
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	
	return bookpb.NewBookCatalogClient(conn), nil
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./authors.db")
	if err != nil {
		return nil, err
	}
	
	// TODO: Create authors table
	// TODO: Seed sample authors
	
	return db, nil
}

func main() {
	// TODO: Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()
	
	// TODO: Connect to Book service
	bookClient, err := connectToBookService()
	if err != nil {
		log.Fatalf("Failed to connect to Book service: %v", err)
	}
	
	// TODO: Create listener on different port (50052)
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	// TODO: Create gRPC server
	grpcServer := grpc.NewServer()
	
	// TODO: Register service
	authorpb.RegisterAuthorCatalogServer(grpcServer, newServer(db, bookClient))
	
	log.Println("🚀 Author Catalog gRPC server listening on :50052")
	log.Println("📚 Connected to Book Catalog service on :50051")
	
	// TODO: Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
