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
	bookClient bookpb.BookCatalogClient
}

func newServer(db *sql.DB, bookClient bookpb.BookCatalogClient) *authorCatalogServer {
	return &authorCatalogServer{
		db:         db,
		bookClient: bookClient,
	}
}

func (s *authorCatalogServer) GetAuthor(ctx context.Context, req *authorpb.GetAuthorRequest) (*authorpb.GetAuthorResponse, error) {
	log.Printf("GetAuthor: id=%d", req.Id)
	
	var author authorpb.Author
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, bio, birth_year, country FROM authors WHERE id = ?",
		req.Id,
	).Scan(&author.Id, &author.Name, &author.Bio, &author.BirthYear, &author.Country)
	
	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "author not found: id=%d", req.Id)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "database error: %v", err)
	}
	
	return &authorpb.GetAuthorResponse{
		Author: &author,
	}, nil
}

func (s *authorCatalogServer) CreateAuthor(ctx context.Context, req *authorpb.CreateAuthorRequest) (*authorpb.CreateAuthorResponse, error) {
	log.Printf("CreateAuthor: name=%s", req.Name)
	
	// Validate input
	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}
	
	// Insert author into database
	result, err := s.db.ExecContext(ctx,
		"INSERT INTO authors (name, bio, birth_year, country) VALUES (?, ?, ?, ?)",
		req.Name, req.Bio, req.BirthYear, req.Country,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to insert author: %v", err)
	}
	
	// Get the last inserted ID
	id, err := result.LastInsertId()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get author ID: %v", err)
	}
	
	return &authorpb.CreateAuthorResponse{
		Author: &authorpb.Author{
			Id:        int32(id),
			Name:      req.Name,
			Bio:       req.Bio,
			BirthYear: req.BirthYear,
			Country:   req.Country,
		},
	}, nil
}

func (s *authorCatalogServer) ListAuthors(ctx context.Context, req *authorpb.ListAuthorsRequest) (*authorpb.ListAuthorsResponse, error) {
	log.Printf("ListAuthors: page=%d, page_size=%d", req.Page, req.PageSize)
	
	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	
	// Get total count
	var total int32
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM authors").Scan(&total)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count authors: %v", err)
	}
	
	// Calculate offset
	offset := (req.Page - 1) * req.PageSize
	
	// Query authors with pagination
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, bio, birth_year, country FROM authors LIMIT ? OFFSET ?",
		req.PageSize, offset,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer rows.Close()
	
	var authors []*authorpb.Author
	for rows.Next() {
		var author authorpb.Author
		err := rows.Scan(&author.Id, &author.Name, &author.Bio, &author.BirthYear, &author.Country)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		authors = append(authors, &author)
	}
	
	return &authorpb.ListAuthorsResponse{
		Authors: authors,
		Total:   total,
	}, nil
}

func (s *authorCatalogServer) GetAuthorBooks(ctx context.Context, req *authorpb.GetAuthorBooksRequest) (*authorpb.GetAuthorBooksResponse, error) {
	log.Printf("GetAuthorBooks: author_id=%d", req.AuthorId)
	
	// Get author from database
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
	
	// Call Book service to get books by this author
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
	
	// Convert books to BookSummary
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
	
	// Create authors table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS authors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			bio TEXT,
			birth_year INTEGER,
			country TEXT
		);
	`)
	if err != nil {
		return nil, err
	}
	
	// Seed sample authors (only if table is empty)
	var count int
	db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&count)
	if count == 0 {
		log.Println("Seeding sample authors...")
		_, err = db.Exec(`
			INSERT INTO authors (name, bio, birth_year, country) VALUES
			('J.K. Rowling', 'British author, best known for Harry Potter series', 1965, 'UK'),
			('George R.R. Martin', 'American novelist, author of A Song of Ice and Fire', 1948, 'USA'),
			('Stephen King', 'American author of horror and suspense novels', 1947, 'USA'),
			('Agatha Christie', 'English writer known for detective novels', 1890, 'UK'),
			('Isaac Asimov', 'American writer and professor of biochemistry', 1920, 'USA');
		`)
		if err != nil {
			return nil, err
		}
	}
	
	return db, nil
}

func main() {
	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer db.Close()
	
	// Connect to Book service
	bookClient, err := connectToBookService()
	if err != nil {
		log.Fatalf("Failed to connect to Book service: %v", err)
	}
	
	// Create listener on port 50052
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	
	// Create gRPC server
	grpcServer := grpc.NewServer()
	
	// Register service
	authorpb.RegisterAuthorCatalogServer(grpcServer, newServer(db, bookClient))
	
	log.Println("🚀 Author Catalog gRPC server listening on :50052")
	log.Println("📚 Connected to Book Catalog service on :50051")
	
	// Start serving
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

