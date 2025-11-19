package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

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

// Create new client
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

// Load genres (id → name)
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

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil { //decode JSON
		return fmt.Errorf("failed to decode genres: %w", err)
	}

	for _, g := range data.Genres { // save to GenreMap
		c.GenreMap[g.ID] = g.Name
	}

	fmt.Printf("Loaded %d genres\n\n", len(c.GenreMap))
	return nil
}

// Search movies by keyword
func (c *TMDBClient) searchMovies(query string) ([]Movie, error) {
	escaped := url.QueryEscape(query) // escape query for URL
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

	var tmdbResp TMDBSearchResponse // parse search response
	if err := json.NewDecoder(resp.Body).Decode(&tmdbResp); err != nil { //decode JSON
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

// Get full movie details
func (c *TMDBClient) getMovieDetails(id int) (*Movie, error) {
	endpoint := fmt.Sprintf("%s/movie/%d?api_key=%s", c.BaseURL, id, c.APIKey)
	resp, err := c.HTTPClient.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch movie details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TMDB error: %s", body)
	}

	var data struct {
		ID          int     `json:"id"`
		Title       string  `json:"title"`
		Overview    string  `json:"overview"`
		ReleaseDate string  `json:"release_date"`
		VoteAverage float64 `json:"vote_average"`
		Genres      []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
		PosterPath string `json:"poster_path"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode details: %w", err)
	}

	var genreNames []string
	for _, g := range data.Genres {
		genreNames = append(genreNames, g.Name)
	}

	movie := &Movie{
		ID:          data.ID,
		Title:       data.Title,
		Overview:    data.Overview,
		ReleaseDate: data.ReleaseDate,
		Rating:      data.VoteAverage,
		Genres:      genreNames,
		PosterURL:   "https://image.tmdb.org/t/p/w500" + data.PosterPath,
	}
	return movie, nil
}

// Save movie list to JSON file
func saveMoviesToJSON(movies []Movie, filename string) error {
	data, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return os.WriteFile(filename, data, 0644)
}

func main() {
	apiKey := "33be097c32c7ec8df2864b26e113d643" 
	client := NewTMDBClient(apiKey)

	fmt.Println("Loading movie genres...")
	if err := client.loadGenres(); err != nil {
		fmt.Printf("Error loading genres: %v\n", err)
		return
	}

	query := "inception"
	fmt.Printf("Searching for: %s\n", query)
	movies, err := client.searchMovies(query)
	if err != nil {
		fmt.Printf("Search error: %v\n", err)
		return
	}

	fmt.Printf("Found %d movies\n\n", len(movies))

	if len(movies) > 0 {
		fmt.Println("Movie 1:")
		m := movies[0]
		fmt.Printf("  ID: %d\n  Title: %s\n  Release Date: %s\n  Rating: %.1f/10\n  
		Genres: %v\n  Overview: %.80s...\n\n",
			m.ID, m.Title, m.ReleaseDate, m.Rating, m.Genres, m.Overview)
	}

	if err := saveMoviesToJSON(movies, "tmdb_results.json"); err != nil {
		fmt.Printf("Error saving JSON: %v\n", err)
		return
	}

	fmt.Printf("Saved %d movies to tmdb_results.json\n", len(movies))
}
