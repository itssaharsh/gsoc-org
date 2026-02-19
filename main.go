package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// --- Configuration ---
const (
	APIBaseURL = "https://api.gsocorganizations.dev/"
)

// --- Data Structures ---

// Org represents a single GSoC organization from the API
type Org struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Year        int    `json:"year"`
}

// PageData is passed to the HTML template
type PageData struct {
	Orgs      []Org
	SyncTime  string
	ErrorMsg  string
}

// --- Global Database Handle ---
var db *sql.DB

func main() {
	// 1. Database Connection with Retry Logic
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?parseTime=true", dbUser, dbPass, dbHost, dbName)
	
	var err error
	// Retry connection for 30 seconds (waiting for MySQL container to start)
	for i := 0; i < 10; i++ {
		db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
			if err == nil {
				break
			}
		}
		log.Println("Waiting for database...", err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatal("Could not connect to database after retries: ", err)
	}
	defer db.Close()

	// 2. Initialize Schema
	initDB()

	// 3. Setup Routes
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/sync", syncHandler)
	http.HandleFunc("/health", healthHandler) // For Smoke Tests

	port := ":8080"
	log.Printf("Server starting on %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}

// initDB creates the table if it doesn't exist
func initDB() {
	query := `
	CREATE TABLE IF NOT EXISTS organizations (
		id INT AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		description TEXT,
		url VARCHAR(255),
		year INT NOT NULL,
		UNIQUE KEY unique_org_year (name, year)
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}
}

// homeHandler displays the data
func homeHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT name, description, url, year FROM organizations ORDER BY year DESC, name ASC")
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var orgs []Org
	for rows.Next() {
		var o Org
		if err := rows.Scan(&o.Name, &o.Description, &o.URL, &o.Year); err != nil {
			continue
		}
		orgs = append(orgs, o)
	}

	tmpl, err := template.ParseFiles("templates/index.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	
	data := PageData{
		Orgs:     orgs,
		SyncTime: time.Now().Format(time.RFC1123),
	}
	tmpl.Execute(w, data)
}

// syncHandler fetches data from the external API for the last 4 years
func syncHandler(w http.ResponseWriter, r *http.Request) {
	currentYear := time.Now().Year()
	startYear := currentYear - 3 // e.g., 2022, 2023, 2024, 2025

	for year := startYear; year <= currentYear; year++ {
		// Note: The API structure varies. Assuming endpoint /year exists or generic fetch.
		// Since api.gsocorganizations.dev docs aren't public, we simulate the fetch logic 
		// assuming a standard endpoint like /<year>.json or similar.
		// If the real API returns all years in one go, remove the loop.
		
		url := fmt.Sprintf("%s%d.json", APIBaseURL, year) 
		// If the API is just the root, use that.
		
		log.Printf("Fetching data for year %d from %s", year, url)
		
		// 1. Fetch
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Failed to fetch %d: %v", year, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			log.Printf("API returned %d for year %d", resp.StatusCode, year)
			continue
		}

		body, _ := io.ReadAll(resp.Body)

		// 2. Parse (Adjust this struct to match exact API response)
		var orgs []Org
		if err := json.Unmarshal(body, &orgs); err != nil {
			log.Printf("JSON Parse error: %v", err)
			continue
		}

		// 3. Insert into DB
		stmt, _ := db.Prepare("INSERT IGNORE INTO organizations (name, description, url, year) VALUES (?, ?, ?, ?)")
		for _, o := range orgs {
			o.Year = year // Ensure year is set
			stmt.Exec(o.Name, o.Description, o.URL, o.Year)
		}
		stmt.Close()
	}

	// Redirect back to home
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// healthHandler is for Kubernetes liveness/readiness probes and smoke tests
func healthHandler(w http.ResponseWriter, r *http.Request) {
	if db == nil {
		http.Error(w, "DB not initialized", http.StatusServiceUnavailable)
		return
	}
	err := db.Ping()
	if err != nil {
		http.Error(w, "DB connection failed", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}