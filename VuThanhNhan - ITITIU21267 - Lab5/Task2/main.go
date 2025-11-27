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

func initDB() error {
	var err error
	db, err = sql.Open("sqlite3", "./bookstore.db")
	if err != nil {
		return err
	}

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

func seedData() {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM books").Scan(&count)
	if count > 0 {
		return
	}

	sampleBooks := []Book{
		{Title: "The Go Programming Language", Author: "Alan Donovan", ISBN: "978-0134190440", Price: 39.99, Stock: 15, PublishedYear: 2015, Description: "Complete guide to Go"},
		{Title: "Clean Code", Author: "Robert C. Martin", ISBN: "978-0132350884", Price: 42.50, Stock: 10, PublishedYear: 2008, Description: "A Handbook of Agile Software Craftsmanship"},
		{Title: "Design Patterns", Author: "Erich Gamma", ISBN: "978-0201633610", Price: 49.99, Stock: 5, PublishedYear: 1994, Description: "Elements of Reusable Object-Oriented Software"},
		{Title: "The Pragmatic Programmer", Author: "Andrew Hunt", ISBN: "978-0201616224", Price: 37.99, Stock: 8, PublishedYear: 1999, Description: "Your Journey to Mastery"},
		{Title: "Code Complete", Author: "Steve McConnell", ISBN: "978-0735619678", Price: 45.00, Stock: 12, PublishedYear: 2004, Description: "A Practical Handbook of Software Construction"},
	}

	for _, b := range sampleBooks {
		_, err := db.Exec(`INSERT INTO books(title, author, isbn, price, stock, published_year, description) VALUES(?, ?, ?, ?, ?, ?, ?)`,
			b.Title, b.Author, b.ISBN, b.Price, b.Stock, b.PublishedYear, b.Description)
		if err != nil {
			log.Println("Seed error:", err)
		}
	}
}

// ------------------ CRUD ------------------

// GET /books with optional filters and sorting
// Example: /books?min_price=20&max_price=50&author=Martin&year=2008&sort=price_desc
func getBooks(c *gin.Context) {
	sortBy := c.DefaultQuery("sort", "id")
	validSorts := map[string]string{
		"price_asc":  "price ASC",
		"price_desc": "price DESC",
		"title":      "title ASC",
		"year_desc":  "published_year DESC",
		"id":         "id ASC",
	}

	orderBy, ok := validSorts[sortBy]
	if !ok {
		orderBy = "id ASC"
	}
	//  where 1=1 to simplify appending conditions
	query := "SELECT id, title, author, isbn, price, stock, published_year, description, created_at FROM books WHERE 1=1"
	var args []interface{}

	if minPrice := c.Query("min_price"); minPrice != "" {
		query += " AND price >= ?"
		args = append(args, minPrice)
	}
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		query += " AND price <= ?"
		args = append(args, maxPrice)
	}
	if author := c.Query("author"); author != "" {
		query += " AND LOWER(author) LIKE LOWER(?)"
		args = append(args, "%"+author+"%")
	}
	if year := c.Query("year"); year != "" {
		query += " AND published_year = ?"
		args = append(args, year)
	}

	query += " ORDER BY " + orderBy

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Price, 
			&b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{"books": books, "count": len(books)})
}

// GET /books/:id
func getBook(c *gin.Context) {
	id := c.Param("id")
	var b Book
	err := db.QueryRow("SELECT id, title, author, isbn, price, stock, published_year, description, created_at FROM books WHERE id = ?", id).
		Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, b)
}

// POST /books
func createBook(c *gin.Context) {
	var b Book
	if err := c.ShouldBindJSON(&b); err != nil {
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
	var book Book
	if err := c.ShouldBindJSON(&book); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := db.Exec(`UPDATE books SET title=?, author=?, isbn=?, price=?, stock=?, published_year=?, description=? WHERE id=?`,
		book.Title, book.Author, book.ISBN, book.Price, book.Stock, book.PublishedYear, book.Description, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}

	book.ID = atoi(id)
	c.JSON(http.StatusOK, book)
}

// DELETE /books/:id
func deleteBook(c *gin.Context) {
	id := c.Param("id")
	res, err := db.Exec("DELETE FROM books WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Book not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Book deleted"})
}

// ------------------ Advanced Queries ------------------

// GET /books/search?q=query
func searchBooks(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query required"})
		return
	}

	searchPattern := "%" + query + "%"
	rows, err := db.Query(`
		SELECT id, title, author, isbn, price, stock, published_year, description, created_at
		FROM books
		WHERE LOWER(title) LIKE LOWER(?) OR LOWER(author) LIKE LOWER(?)`,
		searchPattern, searchPattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, 
			&b.ISBN, &b.Price, &b.Stock, &b.PublishedYear, &b.Description, &b.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"books": books,
		"count": len(books),
		"query": query,
	})
}

// GET /books/filter?min_price=X&max_price=Y&author=Z&year=YYYY
func filterBooks(c *gin.Context) {
	query := "SELECT id, title, author, isbn, price, stock, published_year, description, created_at FROM books WHERE 1=1"
	var args []interface{}

	if minPrice := c.Query("min_price"); minPrice != "" {
		query += " AND price >= ?"
		args = append(args, minPrice)
	}
	if maxPrice := c.Query("max_price"); maxPrice != "" {
		query += " AND price <= ?"
		args = append(args, maxPrice)
	}
	if author := c.Query("author"); author != "" {
		query += " AND LOWER(author) LIKE LOWER(?)"
		args = append(args, "%"+author+"%")
	}
	if year := c.Query("year"); year != "" {
		query += " AND published_year = ?"
		args = append(args, year)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.ISBN, &b.Price, &b.Stock, 
			&b.PublishedYear, &b.Description, &b.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		books = append(books, b)
	}

	c.JSON(http.StatusOK, gin.H{
		"books":   books,
		"count":   len(books),
		"filters": c.Request.URL.Query(),
	})
}

// Helper: string to int
func atoi(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// ------------------ Main ------------------

func main() {
	if err := initDB(); err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	seedData()

	router := gin.Default()

	// CRUD routes
	router.GET("/books", getBooks)
	router.GET("/books/:id", getBook)
	router.POST("/books", createBook)
	router.PUT("/books/:id", updateBook)
	router.DELETE("/books/:id", deleteBook)

	// Advanced queries
	router.GET("/books/search", searchBooks)
	router.GET("/books/filter", filterBooks)

	fmt.Println("🚀 Server running on :8080")
	router.Run(":8080")
}
