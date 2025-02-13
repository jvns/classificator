package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Comment struct {
	ID       int    `json:"id"`
	Comment  string `json:"comment"`
	Category string `json:"category"`
}

type Server struct {
	db *sql.DB
}

func (s *Server) getComments(w http.ResponseWriter, r *http.Request) {
	datasetID, err := strconv.Atoi(r.PathValue("dataset_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rows, err := s.db.Query(`
        SELECT rowid, comment, category
        FROM comments
        WHERE comment != ''
        AND dataset_id = ?
        ORDER BY lower(category) DESC,
        lower(comment) ASC
    `, datasetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.Comment, &c.Category); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		comments = append(comments, c)
	}

	json.NewEncoder(w).Encode(comments)
}

func (s *Server) getCategories(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := s.db.Query("SELECT DISTINCT category FROM comments ORDER BY category")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		if err := rows.Scan(&category); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		categories = append(categories, category)
	}

	json.NewEncoder(w).Encode(categories)
}

func (s *Server) updateComment(w http.ResponseWriter, r *http.Request) {
	var c Comment
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := s.db.Exec("UPDATE comments SET comment = ?, category = ? WHERE rowid = ?",
		c.Comment, c.Category, c.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) splitComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var c Comment
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = tx.Exec("DELETE FROM comments WHERE rowid = ?", c.ID)
	if err != nil {
		tx.Rollback()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, comment := range strings.Split(c.Comment, "\n") {
		if strings.TrimSpace(comment) == "" {
			continue
		}
		_, err = tx.Exec("INSERT INTO comments (comment, category) VALUES (?, ?)",
			strings.TrimSpace(comment), c.Category)
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func readJSONValues(jsonContents io.Reader) ([]string, error) {
	var values []string
	if err := json.NewDecoder(jsonContents).Decode(&values); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}
	return values, nil
}

func readCSVValues(csvContents io.Reader) ([]string, error) {
	r := csv.NewReader(csvContents)
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %v", err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	return records[0], nil // Return first row
}

func (s *Server) createDataset(w http.ResponseWriter, r *http.Request) {
	datasetName := r.FormValue("name")
	if datasetName == "" {
		http.Error(w, "Dataset name required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	res, err := s.db.Exec("INSERT INTO datasets (name) VALUES (?)", datasetName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	datasetID, err := res.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	isJSON := strings.HasSuffix(strings.ToLower(header.Filename), ".json")
	isCSV := strings.HasSuffix(strings.ToLower(header.Filename), ".csv")

	var comments []string

	if isJSON {
		comments, err = readJSONValues(bufio.NewReader(file))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else if isCSV {
		comments, err = readCSVValues(bufio.NewReader(file))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Unsupported file format", http.StatusBadRequest)
		return
	}

	// Insert comments into the database

	tx, err := s.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, comment := range comments {
		_, err = tx.Exec("INSERT INTO comments (dataset_id, comment, category) VALUES (?, ?, ?)", datasetID, comment, "")
		if err != nil {
			tx.Rollback()
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/?id=%d", datasetID), http.StatusFound)
}

func (s *Server) getDatasets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT id, name FROM datasets WHERE NOT deleted ORDER BY id DESC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var datasets []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	for rows.Next() {
		var d struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := rows.Scan(&d.ID, &d.Name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		datasets = append(datasets, d)
	}

	json.NewEncoder(w).Encode(datasets)
}

func (s *Server) deleteDataset(w http.ResponseWriter, r *http.Request) {
	datasetID, err := strconv.Atoi(r.PathValue("dataset_id"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err = s.db.Exec("UPDATE datasets SET deleted = TRUE WHERE id = ?", datasetID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) exportComments(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query("SELECT comment, category FROM comments")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Set headers so the browser knows it's a CSV file
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=comments_export.csv")
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header row
	writer.Write([]string{"comment", "category"})

	// Write each record from the database
	for rows.Next() {
		var comment, category string
		if err := rows.Scan(&comment, &category); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writer.Write([]string{comment, category})
	}
}

func main() {
	db, err := sql.Open("sqlite3", "comments.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := &Server{db: db}

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("GET /api/comments/{dataset_id}", server.getComments)
	http.HandleFunc("/api/categories", server.getCategories)
	http.HandleFunc("PUT /api/comments/", server.updateComment)
	http.HandleFunc("POST /api/dataset", server.createDataset)
	http.HandleFunc("/api/split/", server.splitComment)
	http.HandleFunc("/api/datasets", server.getDatasets)
	http.HandleFunc("/api/export", server.exportComments)
	http.HandleFunc("DELETE /api/datasets/{dataset_id}", server.deleteDataset)

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
