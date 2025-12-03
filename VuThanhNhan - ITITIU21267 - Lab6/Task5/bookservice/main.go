package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"strings"

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
		"SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE id = ?",
		req.Id,
	)

	var book pb.Book
	err := row.Scan(&book.Id, &book.Title, &book.Author, &book.Isbn,
		&book.Price, &book.Stock, &book.PublishedYear, &book.AuthorId)

	if err == sql.ErrNoRows {
		return nil, status.Errorf(codes.NotFound, "book not found: id=%d", req.Id)
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}

	return &pb.GetBookResponse{Book: &book}, nil
}

// ======================== GetBooksByAuthor ============================
func (s *bookCatalogServer) GetBooksByAuthor(ctx context.Context, req *pb.GetBooksByAuthorRequest) (*pb.GetBooksByAuthorResponse, error) {
	log.Printf("GetBooksByAuthor: author_id=%d", req.AuthorId)
	
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year, author_id FROM books WHERE author_id = ?",
		req.AuthorId,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db error: %v", err)
	}
	defer rows.Close()

	books := []*pb.Book{}
	for rows.Next() {
		var b pb.Book
		if err := rows.Scan(&b.Id, &b.Title, &b.Author, &b.Isbn,
			&b.Price, &b.Stock, &b.PublishedYear, &b.AuthorId); err != nil {
			return nil, status.Errorf(codes.Internal, "scan error: %v", err)
		}
		books = append(books, &b)
	}

	return &pb.GetBooksByAuthorResponse{
		Books: books,
		Count: int32(len(books)),
	}, nil
}

// ======================== CreateBook ============================
func (s *bookCatalogServer) CreateBook(ctx context.Context, req *pb.CreateBookRequest) (*pb.CreateBookResponse, error) {
	log.Printf("CreateBook: title=%s, author_id=%d", req.Title, req.AuthorId)
	
	if strings.TrimSpace(req.Title) == "" || strings.TrimSpace(req.Author) == "" {
		return nil, status.Error(codes.InvalidArgument, "title and author are required")
	}

	res, err := s.db.ExecContext(ctx,
		"INSERT INTO books (title, author, isbn, price, stock, published_year, author_id) VALUES (?, ?, ?, ?, ?, ?, ?)",
		req.Title, req.Author, req.Isbn, req.Price, req.Stock, req.PublishedYear, req.AuthorId)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create book: %v", err)
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
			AuthorId:      req.AuthorId,
		},
	}, nil
}

// ======================== UpdateBook ============================
func (s *bookCatalogServer) UpdateBook(ctx context.Context, req *pb.UpdateBookRequest) (*pb.UpdateBookResponse, error) {
	res, err := s.db.ExecContext(ctx,
		`UPDATE books SET title=?, author=?, isbn=?, price=?, stock=?, published_year=?, author_id=? WHERE id=?`,
		req.Title, req.Author, req.Isbn, req.Price, req.Stock, req.PublishedYear, req.AuthorId, req.Id)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update book: %v", err)
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
			AuthorId:      req.AuthorId,
		},
	}, nil
}

// ======================== DeleteBook ============================
func (s *bookCatalogServer) DeleteBook(ctx context.Context, req *pb.DeleteBookRequest) (*pb.DeleteBookResponse, error) {
	res, err := s.db.ExecContext(ctx, "DELETE FROM books WHERE id=?", req.Id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete book: %v", err)
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
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&total); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count books: %v", err)
	}

	// Query with pagination
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, author, isbn, price, stock, published_year, COALESCE(author_id, 0) FROM books LIMIT ? OFFSET ?",
		req.PageSize, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "query failed: %v", err)
	}
	defer rows.Close()

	books := []*pb.Book{}
	for rows.Next() {
		var b pb.Book
		if err := rows.Scan(&b.Id, &b.Title, &b.Author, &b.Isbn, &b.Price, &b.Stock, &b.PublishedYear, &b.AuthorId); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		books = append(books, &b)
	}

	return &pb.ListBooksResponse{
		Books:    books,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ======================== SearchBooks ============================
func (s *bookCatalogServer) SearchBooks(ctx context.Context, req *pb.SearchBooksRequest) (*pb.SearchBooksResponse, error) {
	logPrefix := "SearchBooks"
	logf := func(msg string, args ...interface{}) {
		log.Printf("%s: %s", logPrefix, fmt.Sprintf(msg, args...))
	}

	logf("query=%q, field=%q", req.Query, req.Field)

	if strings.TrimSpace(req.Query) == "" {
		return nil, status.Error(codes.InvalidArgument, "search query required")
	}

	field := strings.ToLower(strings.TrimSpace(req.Field))

	var sqlQuery string
	var args []interface{}
	searchPattern := "%" + req.Query + "%"

	switch field {
	case "title":
		sqlQuery = "SELECT id, title, author, isbn, price, stock, published_year, COALESCE(author_id, 0) FROM books WHERE title LIKE ?"
		args = []interface{}{searchPattern}
	case "author":
		sqlQuery = "SELECT id, title, author, isbn, price, stock, published_year, COALESCE(author_id, 0) FROM books WHERE author LIKE ?"
		args = []interface{}{searchPattern}
	case "isbn":
		sqlQuery = "SELECT id, title, author, isbn, price, stock, published_year, COALESCE(author_id, 0) FROM books WHERE isbn = ?"
		args = []interface{}{req.Query}
	case "all", "":
		sqlQuery = `SELECT id, title, author, isbn, price, stock, published_year, COALESCE(author_id, 0)
		            FROM books 
		            WHERE title LIKE ? OR author LIKE ? OR isbn LIKE ?`
		args = []interface{}{searchPattern, searchPattern, searchPattern}
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid field: %s", req.Field)
	}

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db query failed: %v", err)
	}
	defer rows.Close()

	books := []*pb.Book{}
	for rows.Next() {
		var b pb.Book
		if err := rows.Scan(&b.Id, &b.Title, &b.Author, &b.Isbn, &b.Price, &b.Stock, &b.PublishedYear, &b.AuthorId); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		books = append(books, &b)
	}

	return &pb.SearchBooksResponse{
		Books: books,
		Count: int32(len(books)),
		Query: req.Query,
	}, nil
}

// ======================== FilterBooks ============================
func (s *bookCatalogServer) FilterBooks(ctx context.Context, req *pb.FilterBooksRequest) (*pb.FilterBooksResponse, error) {
	log.Printf("FilterBooks: price[%.2f-%.2f], year[%d-%d]", req.MinPrice, req.MaxPrice, req.MinYear, req.MaxYear)

	if req.MinPrice < 0 || req.MaxPrice < 0 {
		return nil, status.Error(codes.InvalidArgument, "price cannot be negative")
	}
	if req.MaxPrice > 0 && req.MinPrice > req.MaxPrice {
		return nil, status.Error(codes.InvalidArgument, "min_price cannot be greater than max_price")
	}
	if req.MinYear != 0 && req.MaxYear != 0 && req.MinYear > req.MaxYear {
		return nil, status.Error(codes.InvalidArgument, "min_year cannot be greater than max_year")
	}

	query := "SELECT id, title, author, isbn, price, stock, published_year, COALESCE(author_id, 0) FROM books WHERE 1=1"
	var args []interface{}

	if req.MinPrice > 0 {
		query += " AND price >= ?"
		args = append(args, req.MinPrice)
	}
	if req.MaxPrice > 0 {
		query += " AND price <= ?"
		args = append(args, req.MaxPrice)
	}
	if req.MinYear != 0 {
		query += " AND published_year >= ?"
		args = append(args, req.MinYear)
	}
	if req.MaxYear != 0 {
		query += " AND published_year <= ?"
		args = append(args, req.MaxYear)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "db query failed: %v", err)
	}
	defer rows.Close()

	books := []*pb.Book{}
	for rows.Next() {
		var b pb.Book
		if err := rows.Scan(&b.Id, &b.Title, &b.Author, &b.Isbn, &b.Price, &b.Stock, &b.PublishedYear, &b.AuthorId); err != nil {
			return nil, status.Errorf(codes.Internal, "scan failed: %v", err)
		}
		books = append(books, &b)
	}

	return &pb.FilterBooksResponse{
		Books: books,
		Count: int32(len(books)),
	}, nil
}

// ======================== GetStats ============================
func (s *bookCatalogServer) GetStats(ctx context.Context, req *pb.GetStatsRequest) (*pb.GetStatsResponse, error) {
	log.Println("GetStats called")

	var totalBooks int32
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&totalBooks); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to count books: %v", err)
	}

	var avgPrice sql.NullFloat64
	if err := s.db.QueryRowContext(ctx, "SELECT AVG(price) FROM books").Scan(&avgPrice); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute average price: %v", err)
	}

	var totalStock sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT SUM(stock) FROM books").Scan(&totalStock); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute total stock: %v", err)
	}

	var earliest sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT MIN(published_year) FROM books").Scan(&earliest); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get earliest year: %v", err)
	}
	var latest sql.NullInt64
	if err := s.db.QueryRowContext(ctx, "SELECT MAX(published_year) FROM books").Scan(&latest); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get latest year: %v", err)
	}

	resp := &pb.GetStatsResponse{
		TotalBooks:   totalBooks,
		AveragePrice: float32(0),
		TotalStock:   0,
		EarliestYear: 0,
		LatestYear:   0,
	}
	if avgPrice.Valid {
		resp.AveragePrice = float32(avgPrice.Float64)
	}
	if totalStock.Valid {
		resp.TotalStock = int32(totalStock.Int64)
	}
	if earliest.Valid {
		resp.EarliestYear = int32(earliest.Int64)
	}
	if latest.Valid {
		resp.LatestYear = int32(latest.Int64)
	}

	return resp, nil
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
			published_year INTEGER,
			author_id INTEGER DEFAULT 0
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
			('Deep Work','Cal Newport','9781455586691',29.99,20,2016),
			('Learning Go','Jon Bodner','9781492077213',31.50,10,2021);
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

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterBookCatalogServer(s, &bookCatalogServer{db: db})

	fmt.Println("📚 Book Catalog gRPC server running on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}