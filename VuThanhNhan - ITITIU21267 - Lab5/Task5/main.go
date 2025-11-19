package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Book struct {
	ID            int     `json:"id"`
	Title         string  `json:"title" binding:"required,min=3"`
	Author        string  `json:"author"`
	AuthorID      *int    `json:"author_id"`
	ISBN          string  `json:"isbn" binding:"required"`
	Price         float64 `json:"price" binding:"required,min=0.01,max=1000"`
	Stock         int     `json:"stock" binding:"gte=0"`
	PublishedYear int     `json:"published_year"`
	Description   string  `json:"description"`
	CreatedAt     string  `json:"created_at"`
}

type Author struct {
	ID        int    `json:"id"`
	Name      string `json:"name" binding:"required"`
	Bio       string `json:"bio"`
	BirthYear int    `json:"birth_year"`
	Country   string `json:"country"`
	CreatedAt string `json:"created_at"`
}

type BookWithAuthor struct {
	ID            int     `json:"id"`
	Title         string  `json:"title"`
	AuthorID      *int    `json:"author_id"`
	AuthorName    string  `json:"author_name"`
	ISBN          string  `json:"isbn"`
	Price         float64 `json:"price"`
	Stock         int     `json:"stock"`
	PublishedYear int     `json:"published_year"`
	Description   string  `json:"description"`
	CreatedAt     string  `json:"created_at"`
}

type PaginationMeta struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

type PaginatedBooksResponse struct {
	Books      []BookWithAuthor `json:"books"`
	Pagination PaginationMeta   `json:"pagination"`
}

type Statistics struct {
	TotalBooks    int                `json:"total_books"`
	TotalAuthors  int                `json:"total_authors"`
	TotalValue    float64            `json:"total_value"`
	LowStock      int                `json:"low_stock"`
	OutOfStock    int                `json:"out_of_stock"`
	MostExpensive *BookWithAuthor    `json:"most_expensive"`
	Cheapest      *BookWithAuthor    `json:"cheapest"`
	MostStocked   *BookWithAuthor    `json:"most_stocked"`
	BooksByYear   map[int]int        `json:"books_by_year"`
	AveragePrice  float64            `json:"average_price"`
}

type RestockRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}

type SellRequest struct {
	Quantity int `json:"quantity" binding:"required,gt=0"`
}

type BulkCreateRequest struct {
	Books []Book `json:"books" binding:"required,min=1,dive"`
}

type BulkCreateResponse struct {
	Success      int      `json:"success"`
	Failed       int      `json:"failed"`
	CreatedBooks []Book   `json:"created_books"`
	Errors       []string `json:"errors,omitempty"`
}

var db *sql.DB

func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./bookstore.db")
	if err != nil {
		return err
	}

	// Create authors table first
	createAuthorsSQL := `
	CREATE TABLE IF NOT EXISTS authors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		bio TEXT,
		birth_year INTEGER,
		country TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(createAuthorsSQL)
	if err != nil {
		return err
	}

	// Create books table with author_id
	createBooksSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT,
		author_id INTEGER,
		isbn TEXT UNIQUE,
		price REAL NOT NULL CHECK(price > 0),
		stock INTEGER DEFAULT 0,
		published_year INTEGER,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (author_id) REFERENCES authors(id)
	);`

	_, err = db.Exec(createBooksSQL)
	return err
}

func seedAuthors() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&count)
	if count > 0 {
		return
	}

	authors := []Author{
		{Name: "Robert C. Martin", Bio: "Software engineer and influential author on software craftsmanship", BirthYear: 1952, Country: "USA"},
		{Name: "Erich Gamma", Bio: "One of the Gang of Four, design patterns pioneer", BirthYear: 1961, Country: "Switzerland"},
		{Name: "Alan Donovan", Bio: "Go team member at Google, co-author of The Go Programming Language", BirthYear: 0, Country: "USA"},
		{Name: "Andrew Hunt", Bio: "Co-author of The Pragmatic Programmer, software consultant", BirthYear: 0, Country: "USA"},
		{Name: "Steve McConnell", Bio: "Software engineering author and consultant", BirthYear: 1962, Country: "USA"},
	}

	for _, a := range authors {
		_, err := db.Exec(`INSERT INTO authors (name, bio, birth_year, country) VALUES (?, ?, ?, ?)`,
			a.Name, a.Bio, a.BirthYear, a.Country)
		if err != nil {
			log.Println("Failed to seed author:", a.Name, err)
		}
	}
}

func seedData() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM books").Scan(&count)
	if count > 0 {
		return
	}

	// Get author IDs
	authorIDs := make(map[string]int)
	rows, _ := db.Query("SELECT id, name FROM authors")
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		authorIDs[name] = id
	}
	rows.Close()

	sampleBooks := []struct {
		Book
		AuthorName string
	}{
		{Book{Title: "The Go Programming Language", ISBN: "978-0134190440", Price: 39.99, Stock: 15, PublishedYear: 2015, Description: "Complete guide to Go programming"}, "Alan Donovan"},
		{Book{Title: "Clean Code", ISBN: "978-0132350884", Price: 29.99, Stock: 20, PublishedYear: 2008, Description: "A Handbook of Agile Software Craftsmanship"}, "Robert C. Martin"},
		{Book{Title: "Design Patterns", ISBN: "978-0201633610", Price: 49.99, Stock: 10, PublishedYear: 1994, Description: "Elements of Reusable Object-Oriented Software"}, "Erich Gamma"},
		{Book{Title: "The Pragmatic Programmer", ISBN: "978-0201616224", Price: 35.99, Stock: 12, PublishedYear: 1999, Description: "From Journeyman to Master"}, "Andrew Hunt"},
		{Book{Title: "Code Complete", ISBN: "978-0735619678", Price: 38.99, Stock: 8, PublishedYear: 2004, Description: "A Practical Handbook of Software Construction"}, "Steve McConnell"},
	}

	for _, b := range sampleBooks {
		authorID := authorIDs[b.AuthorName]
		_, err := db.Exec(`INSERT INTO books 
		(title, author_id, isbn, price, stock, published_year, description) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			b.Title, authorID, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description)
		if err != nil {
			log.Println("Failed to seed book:", b.Title, err)
		}
	}
}

// Validation functions
func validateISBN(isbn string) error {
	// Remove hyphens
	isbn = regexp.MustCompile(`-`).ReplaceAllString(isbn, "")

	// Check if it's 10 or 13 digits
	if len(isbn) != 10 && len(isbn) != 13 {
		return fmt.Errorf("ISBN must be 10 or 13 digits")
	}

	// Check if all characters are digits
	if !regexp.MustCompile(`^\d+$`).MatchString(isbn) {
		return fmt.Errorf("ISBN must contain only digits")
	}

	return nil
}

func validatePublishedYear(year int) error {
	currentYear := time.Now().Year()

	if year < 1800 || year > currentYear {
		return fmt.Errorf("published year must be between 1800 and %d", currentYear)
	}

	return nil
}

// Helper function to parse integer query parameters
func parseIntQuery(c *gin.Context, key string, defaultValue int) int {
	valueStr := c.Query(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

// Author Endpoints

// GET /authors
func getAuthors(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, bio, birth_year, country, created_at FROM authors ORDER BY name")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	authors := []Author{}
	for rows.Next() {
		var a Author
		err := rows.Scan(&a.ID, &a.Name, &a.Bio, &a.BirthYear, &a.Country, &a.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		authors = append(authors, a)
	}

	c.JSON(http.StatusOK, gin.H{
		"authors": authors,
		"count":   len(authors),
	})
}

// GET /authors/:id
func getAuthor(c *gin.Context) {
	id := c.Param("id")
	var a Author
	err := db.QueryRow(`SELECT id, name, bio, birth_year, country, created_at 
	FROM authors WHERE id = ?`, id).Scan(
		&a.ID, &a.Name, &a.Bio, &a.BirthYear, &a.Country, &a.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Author not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, a)
}

// POST /authors
func createAuthor(c *gin.Context) {
	var a Author
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	result, err := db.Exec(`INSERT INTO authors (name, bio, birth_year, country) VALUES (?, ?, ?, ?)`,
		a.Name, a.Bio, a.BirthYear, a.Country)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	a.ID = int(id)
	err = db.QueryRow(`SELECT created_at FROM authors WHERE id = ?`, a.ID).Scan(&a.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, a)
}

// PUT /authors/:id
func updateAuthor(c *gin.Context) {
	id := c.Param("id")
	var a Author
	if err := c.ShouldBindJSON(&a); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	res, err := db.Exec(`UPDATE authors SET name=?, bio=?, birth_year=?, country=? WHERE id=?`,
		a.Name, a.Bio, a.BirthYear, a.Country, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Author not found"})
		return
	}

	a.ID = atoi(id)
	c.JSON(http.StatusOK, a)
}

// DELETE /authors/:id
func deleteAuthor(c *gin.Context) {
	id := c.Param("id")

	// Check if author has books
	var bookCount int
	err := db.QueryRow("SELECT COUNT(*) FROM books WHERE author_id = ?", id).Scan(&bookCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if bookCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":      "Cannot delete author with existing books",
			"book_count": bookCount,
		})
		return
	}

	res, err := db.Exec("DELETE FROM authors WHERE id=?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Author not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Author deleted successfully"})
}

// GET /authors/:id/books
func getAuthorBooks(c *gin.Context) {
	authorID := c.Param("id")

	// Get author details
	var author Author
	err := db.QueryRow(`SELECT id, name, bio, birth_year, country, created_at 
	FROM authors WHERE id = ?`, authorID).Scan(
		&author.ID, &author.Name, &author.Bio, &author.BirthYear, &author.Country, &author.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Author not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	// Get author's books
	rows, err := db.Query(`SELECT id, title, author_id, isbn, price, stock, published_year, description, created_at 
	FROM books WHERE author_id = ? ORDER BY published_year DESC`, authorID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	books := []Book{}
	for rows.Next() {
		var b Book
		err := rows.Scan(&b.ID, &b.Title, &b.AuthorID, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"author": author,
		"books":  books,
		"count":  len(books),
	})
}

// Modified Book Endpoints

// GET /books - with pagination and author information
func getBooks(c *gin.Context) {
	// Parse pagination parameters
	page := parseIntQuery(c, "page", 1)
	limit := parseIntQuery(c, "limit", 20)

	// Validate parameters
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Get total count
	var total int
	err := db.QueryRow("SELECT COUNT(*) FROM books").Scan(&total)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to count books",
		})
		return
	}

	// Query books with LIMIT and OFFSET
	query := `
	SELECT b.id, b.title, b.author_id, a.name as author_name,
	       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
	FROM books b
	LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.id
	LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	books := []BookWithAuthor{}
	for rows.Next() {
		var b BookWithAuthor
		var authorName sql.NullString
		err := rows.Scan(&b.ID, &b.Title, &b.AuthorID, &authorName, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if authorName.Valid {
			b.AuthorName = authorName.String
		}
		books = append(books, b)
	}

	// Calculate pagination metadata
	totalPages := (total + limit - 1) / limit

	pagination := PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	// Return response with pagination
	c.JSON(http.StatusOK, PaginatedBooksResponse{
		Books:      books,
		Pagination: pagination,
	})
}

// GET /books/:id - with author information
func getBook(c *gin.Context) {
	id := c.Param("id")
	var b BookWithAuthor
	var authorName sql.NullString

	err := db.QueryRow(`SELECT b.id, b.title, b.author_id, a.name as author_name,
	b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
	FROM books b
	LEFT JOIN authors a ON b.author_id = a.id
	WHERE b.id = ?`, id).Scan(
		&b.ID, &b.Title, &b.AuthorID, &authorName, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	if authorName.Valid {
		b.AuthorName = authorName.String
	}
	c.JSON(http.StatusOK, b)
}

// POST /books - with enhanced validation
func createBook(c *gin.Context) {
	var b Book
	
	// Bind JSON with standard validation
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Custom validations
	if err := validateISBN(b.ISBN); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ISBN",
			"details": err.Error(),
		})
		return
	}

	if err := validatePublishedYear(b.PublishedYear); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid published year",
			"details": err.Error(),
		})
		return
	}

	// Validate author_id if provided
	if b.AuthorID != nil {
		var authorExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM authors WHERE id = ?)", *b.AuthorID).Scan(&authorExists)
		if err != nil || !authorExists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Author not found",
				"details": fmt.Sprintf("Author with ID %d does not exist", *b.AuthorID),
			})
			return
		}
	}

	// Insert book into database
	result, err := db.Exec(`INSERT INTO books 
	(title, author_id, isbn, price, stock, published_year, description) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.AuthorID, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	id, err := result.LastInsertId()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	b.ID = int(id)
	err = db.QueryRow(`SELECT created_at FROM books WHERE id = ?`, b.ID).Scan(&b.CreatedAt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, b)
}

// PUT /books/:id - with enhanced validation
func updateBook(c *gin.Context) {
	id := c.Param("id")
	var b Book
	
	// Bind JSON with standard validation
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	// Custom validations
	if err := validateISBN(b.ISBN); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid ISBN",
			"details": err.Error(),
		})
		return
	}

	if err := validatePublishedYear(b.PublishedYear); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid published year",
			"details": err.Error(),
		})
		return
	}

	// Validate author_id if provided
	if b.AuthorID != nil {
		var authorExists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM authors WHERE id = ?)", *b.AuthorID).Scan(&authorExists)
		if err != nil || !authorExists {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Author not found",
				"details": fmt.Sprintf("Author with ID %d does not exist", *b.AuthorID),
			})
			return
		}
	}

	res, err := db.Exec(`UPDATE books SET title=?, author_id=?, isbn=?, price=?, stock=?, published_year=?, description=? WHERE id=?`,
		b.Title, b.AuthorID, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	b.ID = atoi(id)
	c.JSON(http.StatusOK, b)
}

// DELETE /books/:id
func deleteBook(c *gin.Context) {
	id := c.Param("id")
	res, err := db.Exec("DELETE FROM books WHERE id=?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Book deleted successfully"})
}

// Statistics Endpoints

// GET /stats
func getStatistics(c *gin.Context) {
	var stats Statistics
	stats.BooksByYear = make(map[int]int)

	// Count total books
	db.QueryRow("SELECT COUNT(*) FROM books").Scan(&stats.TotalBooks)

	// Count total authors
	db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&stats.TotalAuthors)

	// Calculate total inventory value
	var totalValue sql.NullFloat64
	db.QueryRow("SELECT SUM(price * stock) FROM books").Scan(&totalValue)
	if totalValue.Valid {
		stats.TotalValue = totalValue.Float64
	}

	// Count low stock books (stock < 10 and > 0)
	db.QueryRow("SELECT COUNT(*) FROM books WHERE stock < 10 AND stock > 0").Scan(&stats.LowStock)

	// Count out of stock books
	db.QueryRow("SELECT COUNT(*) FROM books WHERE stock = 0").Scan(&stats.OutOfStock)

	// Get most expensive book
	var mostExpensive BookWithAuthor
	var authorName sql.NullString
	err := db.QueryRow(`
		SELECT b.id, b.title, b.author_id, a.name as author_name,
		       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
		FROM books b
		LEFT JOIN authors a ON b.author_id = a.id
		ORDER BY b.price DESC
		LIMIT 1
	`).Scan(&mostExpensive.ID, &mostExpensive.Title, &mostExpensive.AuthorID, &authorName,
		&mostExpensive.ISBN, &mostExpensive.Price, &mostExpensive.Stock,
		&mostExpensive.PublishedYear, &mostExpensive.Description, &mostExpensive.CreatedAt)
	if err == nil {
		if authorName.Valid {
			mostExpensive.AuthorName = authorName.String
		}
		stats.MostExpensive = &mostExpensive
	}

	// Get cheapest book
	var cheapest BookWithAuthor
	err = db.QueryRow(`
		SELECT b.id, b.title, b.author_id, a.name as author_name,
		       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
		FROM books b
		LEFT JOIN authors a ON b.author_id = a.id
		ORDER BY b.price ASC
		LIMIT 1
	`).Scan(&cheapest.ID, &cheapest.Title, &cheapest.AuthorID, &authorName,
		&cheapest.ISBN, &cheapest.Price, &cheapest.Stock,
		&cheapest.PublishedYear, &cheapest.Description, &cheapest.CreatedAt)
	if err == nil {
		if authorName.Valid {
			cheapest.AuthorName = authorName.String
		}
		stats.Cheapest = &cheapest
	}

	// Get most stocked book
	var mostStocked BookWithAuthor
	err = db.QueryRow(`
		SELECT b.id, b.title, b.author_id, a.name as author_name,
		       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
		FROM books b
		LEFT JOIN authors a ON b.author_id = a.id
		ORDER BY b.stock DESC
		LIMIT 1
	`).Scan(&mostStocked.ID, &mostStocked.Title, &mostStocked.AuthorID, &authorName,
		&mostStocked.ISBN, &mostStocked.Price, &mostStocked.Stock,
		&mostStocked.PublishedYear, &mostStocked.Description, &mostStocked.CreatedAt)
	if err == nil {
		if authorName.Valid {
			mostStocked.AuthorName = authorName.String
		}
		stats.MostStocked = &mostStocked
	}

	// Get books by year distribution
	rows, err := db.Query("SELECT published_year, COUNT(*) FROM books GROUP BY published_year")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var year, count int
			if err := rows.Scan(&year, &count); err == nil {
				stats.BooksByYear[year] = count
			}
		}
	}

	// Calculate average price
	var avgPrice sql.NullFloat64
	db.QueryRow("SELECT AVG(price) FROM books").Scan(&avgPrice)
	if avgPrice.Valid {
		stats.AveragePrice = avgPrice.Float64
	}

	c.JSON(http.StatusOK, stats)
}

// Top Books Endpoints

// GET /books/top/expensive?limit=5
func getTopExpensive(c *gin.Context) {
	limit := parseIntQuery(c, "limit", 5)
	if limit < 1 || limit > 100 {
		limit = 5
	}

	query := `
	SELECT b.id, b.title, b.author_id, a.name as author_name,
	       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
	FROM books b
	LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.price DESC
	LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	books := []BookWithAuthor{}
	for rows.Next() {
		var b BookWithAuthor
		var authorName sql.NullString
		err := rows.Scan(&b.ID, &b.Title, &b.AuthorID, &authorName, &b.ISBN,
			&b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if authorName.Valid {
			b.AuthorName = authorName.String
		}
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"books": books,
		"count": len(books),
	})
}

// GET /books/top/stocked?limit=5
func getTopStocked(c *gin.Context) {
	limit := parseIntQuery(c, "limit", 5)
	if limit < 1 || limit > 100 {
		limit = 5
	}

	query := `
	SELECT b.id, b.title, b.author_id, a.name as author_name,
	       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
	FROM books b
	LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.stock DESC
	LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	books := []BookWithAuthor{}
	for rows.Next() {
		var b BookWithAuthor
		var authorName sql.NullString
		err := rows.Scan(&b.ID, &b.Title, &b.AuthorID, &authorName, &b.ISBN,
			&b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if authorName.Valid {
			b.AuthorName = authorName.String
		}
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"books": books,
		"count": len(books),
	})
}

// GET /books/top/recent?limit=10
func getRecentBooks(c *gin.Context) {
	limit := parseIntQuery(c, "limit", 10)
	if limit < 1 || limit > 100 {
		limit = 10
	}

	query := `
	SELECT b.id, b.title, b.author_id, a.name as author_name,
	       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
	FROM books b
	LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.created_at DESC
	LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	books := []BookWithAuthor{}
	for rows.Next() {
		var b BookWithAuthor
		var authorName sql.NullString
		err := rows.Scan(&b.ID, &b.Title, &b.AuthorID, &authorName, &b.ISBN,
			&b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if authorName.Valid {
			b.AuthorName = authorName.String
		}
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"books": books,
		"count": len(books),
	})
}

// Inventory Management

// POST /books/:id/restock
func restockBook(c *gin.Context) {
	id := c.Param("id")

	var req RestockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Update stock
	result, err := db.Exec("UPDATE books SET stock = stock + ? WHERE id = ?", req.Quantity, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to restock",
		})
		return
	}

	// Check if book exists
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Book not found",
		})
		return
	}

	// Get updated book
	var book BookWithAuthor
	var authorName sql.NullString
	err = db.QueryRow(`
		SELECT b.id, b.title, b.author_id, a.name as author_name,
		       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
		FROM books b
		LEFT JOIN authors a ON b.author_id = a.id
		WHERE b.id = ?`, id).Scan(
		&book.ID, &book.Title, &book.AuthorID, &authorName, &book.ISBN,
		&book.Price, &book.Stock, &book.PublishedYear, &book.Description, &book.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if authorName.Valid {
		book.AuthorName = authorName.String
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Book restocked successfully",
		"book":    book,
	})
}

// POST /books/:id/sell
func sellBook(c *gin.Context) {
	id := c.Param("id")

	var req SellRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Check current stock first
	var currentStock int
	err := db.QueryRow("SELECT stock FROM books WHERE id = ?", id).Scan(&currentStock)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Book not found",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Check if sufficient stock
	if currentStock < req.Quantity {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "Insufficient stock",
			"available": currentStock,
			"requested": req.Quantity,
		})
		return
	}

	// Update stock
	_, err = db.Exec("UPDATE books SET stock = stock - ? WHERE id = ?", req.Quantity, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to sell book",
		})
		return
	}

	// Get updated book
	var book BookWithAuthor
	var authorName sql.NullString
	err = db.QueryRow(`
		SELECT b.id, b.title, b.author_id, a.name as author_name,
		       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
		FROM books b
		LEFT JOIN authors a ON b.author_id = a.id
		WHERE b.id = ?`, id).Scan(
		&book.ID, &book.Title, &book.AuthorID, &authorName, &book.ISBN,
		&book.Price, &book.Stock, &book.PublishedYear, &book.Description, &book.CreatedAt,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if authorName.Valid {
		book.AuthorName = authorName.String
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Book sold successfully",
		"book":    book,
	})
}

// Bulk Operations

// POST /books/bulk
func createBulkBooks(c *gin.Context) {
	var req BulkCreateRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	var response BulkCreateResponse

	// Loop through books and create each one
	for _, book := range req.Books {
		// Custom validations
		if err := validateISBN(book.ISBN); err != nil {
			response.Failed++
			response.Errors = append(response.Errors,
				fmt.Sprintf("Book '%s': %v", book.Title, err))
			continue
		}

		if err := validatePublishedYear(book.PublishedYear); err != nil {
			response.Failed++
			response.Errors = append(response.Errors,
				fmt.Sprintf("Book '%s': %v", book.Title, err))
			continue
		}

		// Validate author_id if provided
		if book.AuthorID != nil {
			var authorExists bool
			err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM authors WHERE id = ?)", *book.AuthorID).Scan(&authorExists)
			if err != nil || !authorExists {
				response.Failed++
				response.Errors = append(response.Errors,
					fmt.Sprintf("Book '%s': Author with ID %d does not exist", book.Title, *book.AuthorID))
				continue
			}
		}

		// Insert book
		result, err := db.Exec(
			"INSERT INTO books (title, author_id, isbn, price, stock, published_year, description) VALUES (?, ?, ?, ?, ?, ?, ?)",
			book.Title, book.AuthorID, book.ISBN, book.Price, book.Stock, book.PublishedYear, book.Description,
		)

		if err != nil {
			response.Failed++
			response.Errors = append(response.Errors,
				fmt.Sprintf("Book '%s': %v", book.Title, err))
			continue
		}

		// Get ID and add to success list
		id, _ := result.LastInsertId()
		book.ID = int(id)
		db.QueryRow("SELECT created_at FROM books WHERE id = ?", book.ID).Scan(&book.CreatedAt)
		response.CreatedBooks = append(response.CreatedBooks, book)
		response.Success++
	}

	c.JSON(http.StatusCreated, response)
}

// API Documentation

// GET / - API Documentation
func getAPIDocumentation(c *gin.Context) {
	docs := gin.H{
		"name":    "Bookstore API",
		"version": "1.0.0",
		"endpoints": gin.H{
			"books": []string{
				"GET /books - List all books (with pagination)",
				"GET /books/:id - Get book by ID",
				"POST /books - Create new book",
				"PUT /books/:id - Update book",
				"DELETE /books/:id - Delete book",
				"GET /books/top/expensive - Most expensive books",
				"GET /books/top/stocked - Most stocked books",
				"GET /books/top/recent - Recently added books",
				"POST /books/:id/restock - Restock book",
				"POST /books/:id/sell - Sell book",
				"POST /books/bulk - Create multiple books",
			},
			"authors": []string{
				"GET /authors - List all authors",
				"GET /authors/:id - Get author by ID",
				"POST /authors - Create new author",
				"PUT /authors/:id - Update author",
				"DELETE /authors/:id - Delete author",
				"GET /authors/:id/books - Get author's books",
			},
			"statistics": []string{
				"GET /stats - Get bookstore statistics",
			},
		},
		"query_parameters": gin.H{
			"pagination": "?page=1&limit=20",
			"limit":      "?limit=5 (for top endpoints)",
		},
	}

	c.JSON(http.StatusOK, docs)
}

// helper
func atoi(s string) int {
	var i int
	fmt.Sscan(s, &i)
	return i
}

func main() {
	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	seedAuthors()
	seedData()

	router := gin.Default()

	// Documentation
	router.GET("/", getAPIDocumentation)

	// Author routes
	router.GET("/authors", getAuthors)
	router.GET("/authors/:id", getAuthor)
	router.POST("/authors", createAuthor)
	router.PUT("/authors/:id", updateAuthor)
	router.DELETE("/authors/:id", deleteAuthor)
	router.GET("/authors/:id/books", getAuthorBooks)

	// Book routes (with pagination and enhanced validation)
	router.GET("/books", getBooks)
	router.GET("/books/:id", getBook)
	router.POST("/books", createBook)
	router.PUT("/books/:id", updateBook)
	router.DELETE("/books/:id", deleteBook)

	// Statistics
	router.GET("/stats", getStatistics)

	// Top books
	router.GET("/books/top/expensive", getTopExpensive)
	router.GET("/books/top/stocked", getTopStocked)
	router.GET("/books/top/recent", getRecentBooks)

	// Inventory management
	router.POST("/books/:id/restock", restockBook)
	router.POST("/books/:id/sell", sellBook)

	// Bulk operations
	router.POST("/books/bulk", createBulkBooks)

	fmt.Println("🚀 Complete Bookstore API started on :8080")
	fmt.Println("📚 Visit http://localhost:8080/ for API documentation")
	router.Run(":8080")
}