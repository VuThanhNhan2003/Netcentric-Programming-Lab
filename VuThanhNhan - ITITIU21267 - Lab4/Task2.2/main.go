package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ============ TMDB Client Code (from Task 2.1) ============

const TMDBBaseURL = "https://api.themoviedb.org/3"

type TMDBSearchResponse struct {
	Page    int `json:"page"`
	Results []struct {
		ID          int     `json:"id"`
		Title       string  `json:"title"`
		Overview    string  `json:"overview"`
		ReleaseDate string  `json:"release_date"`
		VoteAverage float64 `json:"vote_average"`
		GenreIDs    []int   `json:"genre_ids"`
		PosterPath  string  `json:"poster_path"`
	} `json:"results"`
	TotalResults int `json:"total_results"`
}

type Movie struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Overview    string   `json:"overview"`
	ReleaseDate string   `json:"release_date"`
	Rating      float64  `json:"rating"`
	Genres      []string `json:"genres"`
	PosterURL   string   `json:"poster_url"`
}

type TMDBClient struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
	GenreMap   map[int]string
}

func NewTMDBClient(apiKey string) *TMDBClient {
	return &TMDBClient{
		APIKey:  apiKey,
		BaseURL: TMDBBaseURL,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		GenreMap: make(map[int]string),
	}
}

func (c *TMDBClient) loadGenres() error {
	endpoint := fmt.Sprintf("%s/genre/movie/list?api_key=%s", c.BaseURL, c.APIKey)
	resp, err := c.HTTPClient.Get(endpoint)
	if err != nil {
		return fmt.Errorf("failed to fetch genres: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("TMDB error: %s", body)
	}

	var data struct {
		Genres []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fmt.Errorf("failed to decode genres: %w", err)
	}

	for _, g := range data.Genres {
		c.GenreMap[g.ID] = g.Name
	}

	return nil
}

func (c *TMDBClient) searchMovies(query string) ([]Movie, error) {
	escaped := url.QueryEscape(query)
	endpoint := fmt.Sprintf("%s/search/movie?api_key=%s&query=%s", c.BaseURL, c.APIKey, escaped)

	resp, err := c.HTTPClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to search movies: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB error: %s", body)
	}

	var tmdbResp TMDBSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&tmdbResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	var movies []Movie
	for _, r := range tmdbResp.Results {
		m := Movie{
			ID:          r.ID,
			Title:       r.Title,
			Overview:    r.Overview,
			ReleaseDate: r.ReleaseDate,
			Rating:      r.VoteAverage,
			PosterURL:   "https://image.tmdb.org/t/p/w500" + r.PosterPath,
		}
		for _, gid := range r.GenreIDs {
			if name, ok := c.GenreMap[gid]; ok {
				m.Genres = append(m.Genres, name)
			}
		}
		movies = append(movies, m)
	}
	return movies, nil
}

// ============ Aggregator Code ============

type MovieInfo struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Director    string   `json:"director,omitempty"`
	Year        int      `json:"year"`
	Description string   `json:"description"`
	Genres      []string `json:"genres"`
	Rating      float64  `json:"rating"`
	Duration    int      `json:"duration_minutes,omitempty"`
	Source      string   `json:"source"`
	LastUpdated string   `json:"last_updated"`
}

type MovieSource interface {
	GetMovies(query string, limit int) ([]MovieInfo, error)
	GetName() string
}

// TMDBSource - uses TMDB API
type TMDBSource struct {
	client *TMDBClient
}

func NewTMDBSource(apiKey string) *TMDBSource {
	return &TMDBSource{
		client: NewTMDBClient(apiKey),
	}
}

func (t *TMDBSource) GetMovies(query string, limit int) ([]MovieInfo, error) {
	// Load genres if not already loaded
	if len(t.client.GenreMap) == 0 {
		if err := t.client.loadGenres(); err != nil {
			return nil, fmt.Errorf("failed to load genres: %w", err)
		}
	}

	// Search movies using TMDB client
	movies, err := t.client.searchMovies(query)
	if err != nil {
		return nil, err
	}

	// Convert Movie structs to MovieInfo
	var movieInfos []MovieInfo
	for i, movie := range movies {
		if i >= limit {
			break
		}

		// Extract year from release date
		year := 0
		if len(movie.ReleaseDate) >= 4 {
			if y, err := strconv.Atoi(movie.ReleaseDate[:4]); err == nil {
				year = y
			}
		}

		movieInfo := MovieInfo{
			ID:          fmt.Sprintf("tmdb-%d", movie.ID),
			Title:       movie.Title,
			Year:        year,
			Description: movie.Overview,
			Genres:      movie.Genres,
			Rating:      movie.Rating,
			Source:      "TMDB",
			LastUpdated: time.Now().Format(time.RFC3339),
		}
		movieInfos = append(movieInfos, movieInfo)
	}

	fmt.Printf("Querying %s... Found %d results\n", t.GetName(), len(movieInfos))
	return movieInfos, nil
}

func (t *TMDBSource) GetName() string {
	return "TMDB"
}

// MockScraperSource - simulates a scraped source
type MockScraperSource struct {
	name string
}

func NewMockScraperSource(name string) *MockScraperSource {
	return &MockScraperSource{name: name}
}

func (m *MockScraperSource) GetMovies(query string, limit int) ([]MovieInfo, error) {
	// Return mock data simulating scraped results
	mockMovies := []MovieInfo{
		{
			ID:          "scraper-1",
			Title:       strings.Title(query) + ": The Beginning",
			Year:        2019,
			Description: "An epic tale about " + query + " that captivated audiences worldwide.",
			Genres:      []string{"Drama", "Action"},
			Rating:      7.5,
			Duration:    132,
			Director:    "John Director",
			Source:      m.name,
			LastUpdated: time.Now().Format(time.RFC3339),
		},
		{
			ID:          "scraper-2",
			Title:       "The Amazing " + strings.Title(query),
			Year:        2021,
			Description: "A fresh take on the " + query + " story with stunning visuals.",
			Genres:      []string{"Adventure", "Fantasy"},
			Rating:      8.2,
			Duration:    145,
			Director:    "Jane Director",
			Source:      m.name,
			LastUpdated: time.Now().Format(time.RFC3339),
		},
	}

	// Apply limit
	result := mockMovies
	if limit < len(mockMovies) {
		result = mockMovies[:limit]
	}

	fmt.Printf("Querying %s... Found %d results\n", m.GetName(), len(result))
	return result, nil
}

func (m *MockScraperSource) GetName() string {
	return m.name
}

// MovieAggregator - combines multiple sources
type MovieAggregator struct {
	Sources []MovieSource
}

func NewMovieAggregator(sources ...MovieSource) *MovieAggregator {
	return &MovieAggregator{Sources: sources}
}

func (a *MovieAggregator) Search(query string, limitPerSource int) ([]MovieInfo, error) {
	var allMovies []MovieInfo
	var mu sync.Mutex
	var wg sync.WaitGroup

	// TODO: Query all sources concurrently using goroutines
	// TODO: Combine results
	// TODO: Remove duplicates
	// TODO: Merge duplicate movie data

	for _, source := range a.Sources {
		wg.Add(1)
		go func(src MovieSource) {
			defer wg.Done()

			// Query source
			movies, err := src.GetMovies(query, limitPerSource)
			if err != nil {
				fmt.Printf("Error from %s: %v\n", src.GetName(), err)
				return
			}

			// Add to results
			mu.Lock()
			allMovies = append(allMovies, movies...)
			mu.Unlock()
		}(source)
	}

	wg.Wait()

	// Deduplicate and merge
	deduplicated := deduplicateMovies(allMovies)

	return deduplicated, nil
}

func deduplicateMovies(movies []MovieInfo) []MovieInfo {
	if len(movies) == 0 {
		return movies
	}

	// Find duplicate movies by title similarity
	var unique []MovieInfo
	used := make([]bool, len(movies))

	for i := 0; i < len(movies); i++ {
		if used[i] {
			continue
		}

		// This movie is the "master" record
		master := movies[i]
		genreSet := make(map[string]bool)
		for _, g := range master.Genres {
			genreSet[g] = true
		}

		// Find all duplicates
		for j := i + 1; j < len(movies); j++ {
			if used[j] {
				continue
			}

			similarity := calculateSimilarity(movies[i].Title, movies[j].Title)
			if similarity >= 0.8 { // 80% similarity threshold
				used[j] = true

				// Merge data: keep highest rating
				if movies[j].Rating > master.Rating {
					master.Rating = movies[j].Rating
				}

				// Combine genres
				for _, g := range movies[j].Genres {
					genreSet[g] = true
				}

				// Keep director if master doesn't have one
				if master.Director == "" && movies[j].Director != "" {
					master.Director = movies[j].Director
				}

				// Keep duration if master doesn't have one
				if master.Duration == 0 && movies[j].Duration > 0 {
					master.Duration = movies[j].Duration
				}
			}
		}

		// Update genres from set
		master.Genres = make([]string, 0, len(genreSet))
		for g := range genreSet {
			master.Genres = append(master.Genres, g)
		}
		sort.Strings(master.Genres)

		unique = append(unique, master)
	}

	// Sort by rating (descending)
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Rating > unique[j].Rating
	})

	return unique
}

func calculateSimilarity(title1, title2 string) float64 {
	// Normalize titles
	t1 := strings.ToLower(strings.TrimSpace(title1))
	t2 := strings.ToLower(strings.TrimSpace(title2))

	// Exact match
	if t1 == t2 {
		return 1.0
	}

	// Remove common words and punctuation for better comparison
	t1Clean := strings.Map(func(r rune) rune {
		if r == ':' || r == '-' || r == ',' {
			return ' '
		}
		return r
	}, t1)
	t2Clean := strings.Map(func(r rune) rune {
		if r == ':' || r == '-' || r == ',' {
			return ' '
		}
		return r
	}, t2)

	words1 := strings.Fields(t1Clean)
	words2 := strings.Fields(t2Clean)

	// Check word overlap
	matchCount := 0
	totalWords := len(words1)
	if len(words2) > totalWords {
		totalWords = len(words2)
	}

	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 && len(w1) > 2 { // Ignore short words
				matchCount++
				break
			}
		}
	}

	if totalWords == 0 {
		return 0.0
	}

	wordSimilarity := float64(matchCount) / float64(totalWords)

	// Check if one contains the other
	if strings.Contains(t1, t2) || strings.Contains(t2, t1) {
		return 0.9
	}

	// Use Levenshtein-inspired simple distance
	if wordSimilarity > 0.5 {
		return 0.7 + wordSimilarity*0.3
	}

	return wordSimilarity * 0.6
}

func generateReport(movies []MovieInfo) {
	fmt.Println("\n=== Movie Aggregation Report ===")
	fmt.Printf("Total movies found: %d\n\n", len(movies))

	// Count by source
	sourceCount := make(map[string]int)
	for _, movie := range movies {
		sourceCount[movie.Source]++
	}

	fmt.Println("Movies by Source:")
	for source, count := range sourceCount {
		fmt.Printf("  - %s: %d movies\n", source, count)
	}

	// Count by genre
	genreMap := make(map[string]int)
	for _, movie := range movies {
		for _, genre := range movie.Genres {
			genreMap[genre]++
		}
	}
	
	fmt.Println("\nTop Genres:")
	
	// Sort genres by count
	type genreCount struct {
		name  string
		count int
	}
	var genreList []genreCount
	for genre, count := range genreMap {
		genreList = append(genreList, genreCount{name: genre, count: count})
	}
	sort.Slice(genreList, func(i, j int) bool {
		return genreList[i].count > genreList[j].count
	})
	
	// Display top 10 genres
	displayCount := 10
	if len(genreList) < displayCount {
		displayCount = len(genreList)
	}
	for i := 0; i < displayCount; i++ {
		fmt.Printf("  - %s: %d\n", genreList[i].name, genreList[i].count)
	}	// Calculate average rating
	var totalRating float64
	ratedCount := 0
	for _, movie := range movies {
		if movie.Rating > 0 {
			totalRating += movie.Rating
			ratedCount++
		}
	}

	if ratedCount > 0 {
		avgRating := totalRating / float64(ratedCount)
		fmt.Printf("\nAverage Rating: %.2f/10\n", avgRating)
	}
}

func saveToJSON(movies []MovieInfo, filename string) error {
	// Marshal to JSON with pretty printing
	data, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// Write to file
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("\nSaved %d movies to %s\n", len(movies), filename)
	return nil
}

func main() {
	apiKey := "33be097c32c7ec8df2864b26e113d643" // Replace with your TMDB API key

	// Create aggregator with multiple sources
	aggregator := NewMovieAggregator(
		NewTMDBSource(apiKey),
		NewMockScraperSource("MovieScraper"),
	)

	query := "spider-man"
	fmt.Printf("=== Multi-Source Movie Aggregator ===\n")
	fmt.Printf("Searching for: %s\n\n", query)

	// Search all sources concurrently
	movies, err := aggregator.Search(query, 10) //10 movies per source
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Display results
	fmt.Printf("\nFound %d movies after deduplication\n", len(movies))

	// Show first 5 movies with details
	displayCount := 5
	if len(movies) < displayCount {
		displayCount = len(movies)
	}

	for i := 0; i < displayCount; i++ {
		movie := movies[i]
		fmt.Printf("\n%d. %s (%d)\n", i+1, movie.Title, movie.Year)
		fmt.Printf("   Source: %s\n", movie.Source)
		fmt.Printf("   Rating: %.1f/10\n", movie.Rating)
		fmt.Printf("   Genres: %v\n", movie.Genres)
		if movie.Director != "" {
			fmt.Printf("   Director: %s\n", movie.Director)
		}
		if movie.Duration > 0 {
			fmt.Printf("   Duration: %d minutes\n", movie.Duration)
		}
	}

	// Generate report
	generateReport(movies)

	// Save to JSON
	err = saveToJSON(movies, "aggregated_movies.json")
	if err != nil {
		fmt.Printf("Error saving to JSON: %v\n", err)
	}
}
