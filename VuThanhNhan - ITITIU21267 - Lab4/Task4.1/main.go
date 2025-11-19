package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

// ============================================================================
// Data Structures (from Task 2.1 and 2.2)
// ============================================================================

type MovieInfo struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Year        int      `json:"year"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	Rating      float64  `json:"rating"`
	Director    string   `json:"director,omitempty"`
	Source      string   `json:"source"`
	LastUpdated string   `json:"last_updated"`
}

type TMDBClient struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	Genres     map[int]string
}

type TMDBMovie struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Overview    string  `json:"overview"`
	ReleaseDate string  `json:"release_date"`
	Rating      float64 `json:"vote_average"`
	GenreIDs    []int   `json:"genre_ids"`
	Genres      []string
}

type TMDBSearchResponse struct {
	Results []TMDBMovie `json:"results"`
}

type TMDBGenreResponse struct {
	Genres []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
}

// ============================================================================
// TMDB Client
// ============================================================================

func NewTMDBClient(apiKey string) *TMDBClient {
	return &TMDBClient{
		APIKey:     apiKey,
		BaseURL:    "https://api.themoviedb.org/3",
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Genres:     make(map[int]string),
	}
}

func (c *TMDBClient) loadGenres() error {
	endpoint := fmt.Sprintf("%s/genre/movie/list?api_key=%s", c.BaseURL, c.APIKey)
	resp, err := c.HTTPClient.Get(endpoint)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var genreResp TMDBGenreResponse
	if err := json.NewDecoder(resp.Body).Decode(&genreResp); err != nil {
		return err
	}

	for _, genre := range genreResp.Genres {
		c.Genres[genre.ID] = genre.Name
	}
	return nil
}

func (c *TMDBClient) searchMovies(query string) ([]TMDBMovie, error) {
	endpoint := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s",
		c.BaseURL, c.APIKey, url.QueryEscape(query))

	resp, err := c.HTTPClient.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %d - %s", resp.StatusCode, string(body))
	}

	var searchResp TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, err
	}

	// Map genre IDs to names
	for i := range searchResp.Results {
		for _, genreID := range searchResp.Results[i].GenreIDs {
			if genreName, exists := c.Genres[genreID]; exists {
				searchResp.Results[i].Genres = append(searchResp.Results[i].Genres, genreName)
			}
		}
	}

	return searchResp.Results, nil
}

// ============================================================================
// Movie Database
// ============================================================================

type MovieDatabase struct {
	Movies      map[string]MovieInfo `json:"movies"`
	Genres      map[string][]string  `json:"genres"`
	Directors   map[string][]string  `json:"directors"`
	Years       map[int][]string     `json:"years"`
	LastUpdated time.Time            `json:"last_updated"`
	TotalCount  int                  `json:"total_count"`
}

func NewMovieDatabase() *MovieDatabase {
	return &MovieDatabase{
		Movies:    make(map[string]MovieInfo),
		Genres:    make(map[string][]string),
		Directors: make(map[string][]string),
		Years:     make(map[int][]string),
	}
}

func (db *MovieDatabase) Add(movie MovieInfo) error {
	movieID := movie.ID

	// Check if already exists
	if _, exists := db.Movies[movieID]; exists {
		return nil
	}

	// Add movie to Movies map
	db.Movies[movieID] = movie

	// Update genre index
	for _, genre := range movie.Genres {
		db.Genres[genre] = append(db.Genres[genre], movieID)
	}

	// Update director index
	if movie.Director != "" {
		db.Directors[movie.Director] = append(db.Directors[movie.Director], movieID)
	}

	// Update year index
	if movie.Year > 0 {
		db.Years[movie.Year] = append(db.Years[movie.Year], movieID)
	}

	// Update count
	db.TotalCount++

	return nil
}

func (db *MovieDatabase) Get(id string) (*MovieInfo, error) {
	movie, exists := db.Movies[id]
	if !exists {
		return nil, fmt.Errorf("movie not found: %s", id)
	}
	return &movie, nil
}

func (db *MovieDatabase) Search(query string) ([]MovieInfo, error) {
	var results []MovieInfo
	query = strings.ToLower(query)

	for _, movie := range db.Movies {
		if strings.Contains(strings.ToLower(movie.Title), query) {
			results = append(results, movie)
		}
	}

	return results, nil
}

func (db *MovieDatabase) GetByGenre(genre string) ([]MovieInfo, error) {
	var results []MovieInfo

	movieIDs, exists := db.Genres[genre]
	if !exists {
		return results, nil
	}

	for _, id := range movieIDs {
		if movie, err := db.Get(id); err == nil {
			results = append(results, *movie)
		}
	}

	return results, nil
}

func (db *MovieDatabase) GetByYear(year int) ([]MovieInfo, error) {
	var results []MovieInfo

	movieIDs, exists := db.Years[year]
	if !exists {
		return results, nil
	}

	for _, id := range movieIDs {
		if movie, err := db.Get(id); err == nil {
			results = append(results, *movie)
		}
	}

	return results, nil
}

func (db *MovieDatabase) GetByDirector(director string) ([]MovieInfo, error) {
	var results []MovieInfo

	movieIDs, exists := db.Directors[director]
	if !exists {
		return results, nil
	}

	for _, id := range movieIDs {
		if movie, err := db.Get(id); err == nil {
			results = append(results, *movie)
		}
	}

	return results, nil
}

func (db *MovieDatabase) Update(movie MovieInfo) error {
	if _, exists := db.Movies[movie.ID]; !exists {
		return fmt.Errorf("movie not found: %s", movie.ID)
	}
	db.Movies[movie.ID] = movie
	return nil
}

func (db *MovieDatabase) Delete(id string) error {
	if _, exists := db.Movies[id]; !exists {
		return fmt.Errorf("movie not found: %s", id)
	}
	delete(db.Movies, id)
	db.TotalCount--
	return nil
}

func (db *MovieDatabase) Save(filename string) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (db *MovieDatabase) Load(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, db)
}

func (db *MovieDatabase) PrintStatistics() {
	fmt.Println("\n=== Movie Database Statistics ===")
	fmt.Printf("Total Movies: %d\n", db.TotalCount)
	fmt.Printf("Total Genres: %d\n", len(db.Genres))
	fmt.Printf("Total Directors: %d\n", len(db.Directors))
	fmt.Printf("Year Range: %d entries\n\n", len(db.Years))

	// Genre statistics
	fmt.Println("Movies by Genre:")
	type genreCount struct {
		name  string
		count int
	}
	var genreCounts []genreCount
	for genre, movieIDs := range db.Genres {
		if len(movieIDs) > 0 {
			genreCounts = append(genreCounts, genreCount{genre, len(movieIDs)})
		}
	}
	sort.Slice(genreCounts, func(i, j int) bool {
		return genreCounts[i].count > genreCounts[j].count
	})
	for _, gc := range genreCounts {
		fmt.Printf("  - %s: %d movies\n", gc.name, gc.count)
	}
}

// ============================================================================
// Collection Pipeline
// ============================================================================

func buildMovieDatabase(apiKey string) (*MovieDatabase, error) {
	db := NewMovieDatabase()
	client := NewTMDBClient(apiKey)

	// Load genres first
	fmt.Println("Loading movie genres from TMDB...")
	err := client.loadGenres()
	if err != nil {
		return nil, fmt.Errorf("failed to load genres: %w", err)
	}
	fmt.Printf("Loaded %d genres\n", len(client.Genres))

	// Define search queries
	searchQueries := []string{
		"marvel", "star wars", "harry potter", "batman",
		"comedy", "horror", "romance", "action",
		"2023", "2022", "2021", "classic",
	}

	fmt.Println("\nBuilding movie database...")

	for i, query := range searchQueries {
		fmt.Printf("[%d/%d] Searching for: %s\n", i+1, len(searchQueries), query)

		// Search movies using TMDB client
		movies, err := client.searchMovies(query)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}

		// Add to database
		added := 0
		for _, movie := range movies {
			movieInfo := MovieInfo{
				ID:          fmt.Sprintf("%d", movie.ID),
				Title:       movie.Title,
				Year:        extractYear(movie.ReleaseDate),
				Description: movie.Overview,
				Genres:      movie.Genres,
				Rating:      movie.Rating,
				Source:      "TMDB",
				LastUpdated: time.Now().Format(time.RFC3339),
			}

			// Only add if not duplicate
			if _, exists := db.Movies[movieInfo.ID]; !exists {
				db.Add(movieInfo)
				added++
			}
		}

		fmt.Printf("  Added %d new movies (found %d total)\n", added, len(movies))

		// Rate limiting: 1 request per second
		time.Sleep(1 * time.Second)
	}

	db.LastUpdated = time.Now()
	fmt.Printf("\nDatabase building complete!\n")

	return db, nil
}

func extractYear(dateStr string) int {
	if len(dateStr) >= 4 {
		var year int
		fmt.Sscanf(dateStr[:4], "%d", &year)
		return year
	}
	return 0
}

// ============================================================================
// Main
// ============================================================================

func main() {
	fmt.Println("=== Movie Database Builder ===\n")

	apiKey := "33be097c32c7ec8df2864b26e113d643" // Replace with your key

	// Build database
	fmt.Println("Starting database collection...")
	db, err := buildMovieDatabase(apiKey)
	if err != nil {
		fmt.Printf("Error building database: %v\n", err)
		return
	}

	// Print statistics
	db.PrintStatistics()

	// Test search
	fmt.Println("\n=== Testing Search Functions ===")

	// Search by title
	fmt.Println("\nSearching for 'spider':")
	results, _ := db.Search("spider")
	for i, movie := range results {
		if i < 3 {
			fmt.Printf("  %d. %s (%d) - Rating: %.1f\n",
				i+1, movie.Title, movie.Year, movie.Rating)
		}
	}

	// Search by genre
	if len(db.Genres) > 0 {
		var firstGenre string
		for genre := range db.Genres {
			firstGenre = genre
			break
		}

		fmt.Printf("\nMovies in genre '%s':\n", firstGenre)
		genreMovies, _ := db.GetByGenre(firstGenre)
		for i, movie := range genreMovies {
			if i < 3 {
				fmt.Printf("  %d. %s (%d)\n", i+1, movie.Title, movie.Year)
			}
		}
	}

	// Save to file
	filename := "movie_database.json"
	err = db.Save(filename)
	if err != nil {
		fmt.Printf("Error saving database: %v\n", err)
		return
	}

	// Get file size
	fileInfo, _ := os.Stat(filename)
	sizeKB := fileInfo.Size() / 1024

	fmt.Printf("\n✓ Database saved successfully!\n")
	fmt.Printf("  File: %s\n", filename)
	fmt.Printf("  Size: %d KB\n", sizeKB)
	fmt.Printf("  Movies: %d\n", db.TotalCount)
}