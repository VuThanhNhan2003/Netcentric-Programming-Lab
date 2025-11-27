package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

type Book struct {
	ID            int     `json:"id"`
	Title         string  `json:"title" binding:"required"`
	Author        string  `json:"author" binding:"required"`
	ISBN          string  `json:"isbn"`
	Price         float64 `json:"price" binding:"required,gt=0"`
	Stock         int     `json:"stock" binding:"gte=0"`
	PublishedYear int     `json:"published_year"`
	Description   string  `json:"description"`
	CreatedAt     string  `json:"created_at"`
}

var db *sql.DB
// Init database and create books table if not exists
func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./bookstore.db")
	if err != nil {
		return err
	}
// Create books table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		isbn TEXT UNIQUE,
		price REAL NOT NULL CHECK(price > 0),
		stock INTEGER DEFAULT 0,
		published_year INTEGER,
		description TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableSQL)
	return err
}
// Seed database with sample data if empty
func seedData() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM books").Scan(&count) // check table empty?
	if count > 0 {
		return
	}

	sampleBooks := []Book{
		{Title: "The Go Programming Language", Author: "Alan Donovan", ISBN: "978-0134190440", Price: 39.99, Stock: 15, PublishedYear: 2015, Description: "Complete guide to Go programming"},
		{Title: "Clean Code", Author: "Robert C. Martin", ISBN: "978-0132350884", Price: 29.99, Stock: 20, PublishedYear: 2008, Description: "A Handbook of Agile Software Craftsmanship"},
		{Title: "Design Patterns", Author: "Erich Gamma", ISBN: "978-0201633610", Price: 49.99, Stock: 10, PublishedYear: 1994, Description: "Elements of Reusable Object-Oriented Software"},
		{Title: "The Pragmatic Programmer", Author: "Andrew Hunt", ISBN: "978-0201616224", Price: 35.99, Stock: 12, PublishedYear: 1999, Description: "From Journeyman to Master"},
		{Title: "Code Complete", Author: "Steve McConnell", ISBN: "978-0735619678", Price: 38.99, Stock: 8, PublishedYear: 2004, Description: "A Practical Handbook of Software Construction"},
	}
	// Insert sample books
	for _, b := range sampleBooks {
		_, err := db.Exec(`INSERT INTO books 
		(title, author, isbn, price, stock, published_year, description) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			b.Title, b.Author, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description)
		if err != nil {
			log.Println("Failed to seed book:", b.Title, err)
		}
	}
}

// GET /books
func getBooks(c *gin.Context) { // get all books
	rows, err := db.Query("SELECT id, title, author, isbn, price, stock, published_year, description, created_at FROM books")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close() // close rows after function ends

	books := []Book{}
	for rows.Next() { 
		var b Book
		// Scan row into Book struct
		err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		books = append(books, b) // add book to slice
	}

	c.JSON(http.StatusOK, gin.H{
		"books": books, 
		"count": len(books),
	})
}

// GET /books/:id
func getBook(c *gin.Context) {
	id := c.Param("id")
	var b Book
	// Query book by ID and scan into Book struct 
	err := db.QueryRow(`SELECT id, title, author, isbn, price, stock, published_year, description, created_at 
	FROM books WHERE id = ?`, id).Scan(
		&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, b)
}

// POST /books
func createBook(c *gin.Context) {
	var b Book
	if err := c.ShouldBindJSON(&b); err != nil { // bind JSON to Book struct
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	result, err := db.Exec(`INSERT INTO books 
	(title, author, isbn, price, stock, published_year, description) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.Title, b.Author, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Get the last inserted ID
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

// PUT /books/:id
func updateBook(c *gin.Context) {
	id := c.Param("id")
	var b Book
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	res, err := db.Exec(`UPDATE books SET title=?, author=?, isbn=?, price=?, stock=?, published_year=?, description=? WHERE id=?`,
		b.Title, b.Author, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// If no rows affected, book not found
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	b.ID = atoi(id) // convert id to int
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

// helper to convert string to int
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

	seedData()

	router := gin.Default() // create Gin router
	router.GET("/books", getBooks)
	router.GET("/books/:id", getBook)
	router.POST("/books", createBook)
	router.PUT("/books/:id", updateBook)
	router.DELETE("/books/:id", deleteBook)

	fmt.Println("🚀 Server starting on :8080")
	router.Run(":8080")
}
