package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"

	pb "book-catalog-grpc/proto/proto"
	_ "github.com/mattn/go-sqlite3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type bookCatalogServer struct {
	pb.UnimplementedBookCatalogServer
	db *sql.DB
}

// ======================== GetBook ============================

func (s *bookCatalogServer) GetBook(ctx context.Context, req *pb.GetBookRequest) (*pb.GetBookResponse, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year FROM books WHERE id = ?",
		req.Id,
	)

	var book pb.Book
	err := row.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn,
		&book.Price, &book.Stock, &book.PublishedYear)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "book not found: id=%d", req.Id)
	}
	if err != nil {
		return nil, err
	}

	return &pb.GetBookResponse{Book: &book}, nil
}

// ======================== CreateBook ============================

func (s *bookCatalogServer) CreateBook(ctx context.Context, req *pb.CreateBookRequest) (*pb.CreateBookResponse, error) {
	res, err := s.db.ExecContext(ctx,
		"INSERT INTO books (title, author, isbn, price, stock, published_year) VALUES (?, ?, ?, ?, ?, ?)",
		req.Title, req.Author, req.Isbn, req.Price, req.Stock, req.PublishedYear)

	if err != nil {
		return nil, err
	}

	id, _ := res.LastInsertId()

	return &pb.CreateBookResponse{
		Book: &pb.Book{
			Id:            int32(id),
			Title:         req.Title,
			Author:        req.Author,
			Isbn:          req.Isbn,
			Price:         req.Price,
			Stock:         req.Stock,
			PublishedYear: req.PublishedYear,
		},
	}, nil
}

// ======================== UpdateBook ============================

func (s *bookCatalogServer) UpdateBook(ctx context.Context, req *pb.UpdateBookRequest) (*pb.UpdateBookResponse, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE books SET title=?, author=?, isbn=?, price=?, stock=?, published_year=? WHERE id=?`,
		req.Title, req.Author, req.Isbn, req.Price, req.Stock, req.PublishedYear, req.Id)

	if err != nil {
		return nil, err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil, status.Errorf(codes.NotFound, "book not found: id=%d", req.Id)
	}

	return &pb.UpdateBookResponse{
		Book: &pb.Book{
			Id:            req.Id,
			Title:         req.Title,
			Author:        req.Author,
			Isbn:          req.Isbn,
			Price:         req.Price,
			Stock:         req.Stock,
			PublishedYear: req.PublishedYear,
		},
	}, nil
}

// ======================== DeleteBook ============================

func (s *bookCatalogServer) DeleteBook(ctx context.Context, req *pb.DeleteBookRequest) (*pb.DeleteBookResponse, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM books WHERE id=?", req.Id)
	if err != nil {
		return nil, err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return &pb.DeleteBookResponse{
			Success: false,
			Message: "book not found",
		}, nil
	}

	return &pb.DeleteBookResponse{
		Success: true,
		Message: "book deleted successfully",
	}, nil
}

// ======================== ListBooks (Pagination) ============================

func (s *bookCatalogServer) ListBooks(ctx context.Context, req *pb.ListBooksRequest) (*pb.ListBooksResponse, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 5
	}

	offset := (req.Page - 1) * req.PageSize

	// Total count
	var total int32
	s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&total)

	// Query with pagination
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year FROM books LIMIT ? OFFSET ?",
		req.PageSize, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := []*pb.Book{}
	for rows.Next() {
		var b pb.Book
		rows.Scan(&b.Id, &b.Title, &b.Author, &b.Isbn, &b.Price, &b.Stock, &b.PublishedYear)
		books = append(books, &b)
	}

	return &pb.ListBooksResponse{
		Books:    books,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ===================== DB Initialization =======================

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./books.db")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS books (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			author TEXT,
			isbn TEXT,
			price REAL,
			stock INTEGER,
			published_year INTEGER
		);
	`)
	if err != nil {
		return nil, err
	}

	// Seed only when empty
	var count int
	db.QueryRow("SELECT COUNT(*) FROM books").Scan(&count)
	if count == 0 {
		fmt.Println("Seeding sample books...")
		db.Exec(`
			INSERT INTO books (title, author, isbn, price, stock, published_year) VALUES
			('The Go Programming Language','Alan Donovan','9780134190440',39.99,10,2015),
			('Clean Code','Robert Martin','9780132350884',42.50,15,2008),
			('Design Patterns','Erich Gamma','9780201633610',55.00,7,1994),
			('Concurrency in Go','Katherine Cox','9781491941195',33.99,12,2017),
			('Deep Work','Cal Newport','9781455586691',29.99,20,2016);
		`)
	}

	return db, nil
}

// ============================ main =============================

func main() {
	db, err := initDB()
	if err != nil {
		log.Fatal("DB init error:", err)
	}

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterBookCatalogServer(s, &bookCatalogServer{db: db})

	fmt.Println("📚 Book Catalog gRPC server running on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
