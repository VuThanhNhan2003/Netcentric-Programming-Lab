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
	Author        string  `json:"author"`
	AuthorID      *int    `json:"author_id"`
	ISBN          string  `json:"isbn"`
	Price         float64 `json:"price" binding:"required,gt=0"`
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
	db.QueryRow("SELECT COUNT(*) FROM authors").Scan(&count) // check table empty?
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

// GET /books - with author information
func getBooks(c *gin.Context) {
	query := `
	SELECT b.id, b.title, b.author_id, a.name as author_name, 
	       b.isbn, b.price, b.stock, b.published_year, b.description, b.created_at
	FROM books b
	LEFT JOIN authors a ON b.author_id = a.id
	ORDER BY b.id`

	rows, err := db.Query(query)
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

	c.JSON(http.StatusOK, gin.H{
		"books": books,
		"count": len(books),
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

// POST /books - with author_id support
func createBook(c *gin.Context) {
	var b Book
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Validate author_id if provided
	if b.AuthorID != nil {
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM authors WHERE id = ?", *b.AuthorID).Scan(&exists)
		if err != nil || exists == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid author_id"})
			return
		}
	}

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

// PUT /books/:id
func updateBook(c *gin.Context) {
	id := c.Param("id")
	var b Book
	if err := c.ShouldBindJSON(&b); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Validate author_id if provided
	if b.AuthorID != nil {
		var exists int
		err := db.QueryRow("SELECT COUNT(*) FROM authors WHERE id = ?", *b.AuthorID).Scan(&exists)
		if err != nil || exists == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid author_id"})
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

	// Author routes
	router.GET("/authors", getAuthors)
	router.GET("/authors/:id", getAuthor)
	router.POST("/authors", createAuthor)
	router.PUT("/authors/:id", updateAuthor)
	router.DELETE("/authors/:id", deleteAuthor)
	router.GET("/authors/:id/books", getAuthorBooks)

	// Book routes
	router.GET("/books", getBooks)
	router.GET("/books/:id", getBook)
	router.POST("/books", createBook)
	router.PUT("/books/:id", updateBook)
	router.DELETE("/books/:id", deleteBook)
	
	router.Run(":8080")
}