package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"

	authorpb "book-catalog-grpc/proto/proto"
	bookpb "book-catalog-grpc/proto/proto"

	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

//
// ======================= DB INIT ==========================
//
func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./authors.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS authors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			bio TEXT,
			birth_year INTEGER,
			country TEXT
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create authors table: %v", err)
	}

	return db, nil
}

//
// ======================= SERVER STRUCT ====================
//
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

//
// ======================= RPC: GetAuthor =====================
//
func (s *authorCatalogServer) GetAuthor(ctx context.Context, req *authorpb.GetAuthorRequest) (*authorpb.GetAuthorResponse, error) {
	var a authorpb.Author
	err := s.db.QueryRowContext(ctx,
		"SELECT id, name, bio, birth_year, country FROM authors WHERE id = ?",
		req.Id,
	).Scan(&a.Id, &a.Name, &a.Bio, &a.BirthYear, &a.Country)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "author not found")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}

	return &authorpb.GetAuthorResponse{Author: &a}, nil
}

//
// ======================= RPC: CreateAuthor ==================
//
func (s *authorCatalogServer) CreateAuthor(ctx context.Context, req *authorpb.CreateAuthorRequest) (*authorpb.CreateAuthorResponse, error) {

	if req.Name == "" {
		return nil, status.Error(codes.InvalidArgument, "name is required")
	}

	res, err := s.db.ExecContext(ctx,
		"INSERT INTO authors(name, bio, birth_year, country) VALUES(?, ?, ?, ?)",
		req.Name, req.Bio, req.BirthYear, req.Country)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "insert failed: %v", err)
	}

	id, _ := res.LastInsertId()

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

//
// ======================= RPC: ListAuthors ===================
//
func (s *authorCatalogServer) ListAuthors(ctx context.Context, req *authorpb.ListAuthorsRequest) (*authorpb.ListAuthorsResponse, error) {

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 5
	}

	offset := (req.Page - 1) * req.PageSize

	rows, err := s.db.QueryContext(ctx,
		"SELECT id, name, bio, birth_year, country FROM authors LIMIT ? OFFSET ?",
		req.PageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db query failed: %v", err)
	}
	defer rows.Close()

	list := []*authorpb.Author{}
	for rows.Next() {
		var a authorpb.Author
		rows.Scan(&a.Id, &a.Name, &a.Bio, &a.BirthYear, &a.Country)
		list = append(list, &a)
	}

	return &authorpb.ListAuthorsResponse{
		Authors: list,
		Total:   int32(len(list)),
	}, nil
}

//
// ======================= RPC: GetAuthorBooks (cross-service) ======
//
func (s *authorCatalogServer) GetAuthorBooks(ctx context.Context, req *authorpb.GetAuthorBooksRequest) (*authorpb.GetAuthorBooksResponse, error) {
	log.Printf("GetAuthorBooks: author_id=%d", req.AuthorId)

	// === 1) Lấy author từ DB ===
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

	// === 2) Gọi BookService: GetBooksByAuthor() ===
	bookResp, err := s.bookClient.GetBooksByAuthor(ctx, &bookpb.GetBooksByAuthorRequest{
		AuthorId: req.AuthorId,
	})
	if err != nil {
		log.Printf("Book service failed: %v", err)
		return &authorpb.GetAuthorBooksResponse{
			Author:    &author,
			Books:     nil,
			BookCount: 0,
		}, nil
	}

	// === 3) Convert Book → BookSummary ===
	summary := []*authorpb.BookSummary{}
	for _, b := range bookResp.Books {
		summary = append(summary, &authorpb.BookSummary{
			Id:            b.Id,
			Title:         b.Title,
			Price:         b.Price,
			PublishedYear: b.PublishedYear,
		})
	}

	// === 4) Return response ===
	return &authorpb.GetAuthorBooksResponse{
		Author:    &author,
		Books:     summary,
		BookCount: int32(len(summary)),
	}, nil
}

//
// ======================= Connect to Book Service ==================
func connectToBookService() (bookpb.BookCatalogClient, error) {
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return bookpb.NewBookCatalogClient(conn), nil
}

//
// ======================= MAIN =============================
//
func main() {
	// Init DB
	db, err := initDB()
	if err != nil {
		log.Fatalf("DB init failed: %v", err)
	}
	defer db.Close()

	// Connect Book service
	bookClient, err := connectToBookService()
	if err != nil {
		log.Fatalf("Cannot connect to Book service: %v", err)
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Listen failed: %v", err)
	}

	grpcServer := grpc.NewServer()
	authorpb.RegisterAuthorCatalogServer(grpcServer, newServer(db, bookClient))

	log.Println("🚀 Author Catalog gRPC server running on :50052")
	log.Println("📚 Connected to Book Catalog service on :50051")

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Serve failed: %v", err)
	}
}
