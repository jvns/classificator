package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
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
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := s.db.Query("SELECT rowid, comment, category FROM comments ORDER BY category DESC")
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
	if r.Method != "PUT" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

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

func main() {
	db, err := sql.Open("sqlite3", "comments.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	server := &Server{db: db}

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/api/comments", server.getComments)
	http.HandleFunc("/api/categories", server.getCategories)
	http.HandleFunc("/api/comments/", server.updateComment)
	http.HandleFunc("/api/split/", server.splitComment)

	log.Fatal(http.ListenAndServe(":8080", nil))
}
