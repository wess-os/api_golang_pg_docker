package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func main() {
	// connect to database
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// create the table if it doesn't exist
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, name VARCHAR(255), email VARCHAR(255))")
	if err != nil {
		log.Fatal(err)
	}

	// create router
	router := mux.NewRouter()
	router.HandleFunc("/users", getUsers(db)).Methods("GET")
	router.HandleFunc("/users/{id}", getUser(db)).Methods("GET")
	router.HandleFunc("/users", createUser(db)).Methods("POST")
	router.HandleFunc("/users/{id}", updateUser(db)).Methods("PUT")
	router.HandleFunc("/users/{id}", deleteUser(db)).Methods("DELETE")

	// start server
	log.Fatal(http.ListenAndServe(":8000", jsonContentTypeMiddleware(router)))
}

func jsonContentTypeMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// get all users
func getUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT * FROM users")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		users := []User{}
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				log.Fatal(err)
			}
			users = append(users, u)
		}

		if err := rows.Err(); err != nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(users)
	}
}

// get user by id
func getUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		var u User
		err := db.QueryRow("SELECT * FROM users WHERE id = $1", id).Scan(&u.ID, &u.Name, &u.Email)
		if err != nil {
			// TODO: fix error handling
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(u)
	}
}

// create user
func createUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// validations on the request
		if u.Name == "" || !isValidEmail(u.Email) {
			http.Error(w, "Invalid input: Name is required and Email must be valid", http.StatusBadRequest)
			return
		}

		// verify if the user already exists
		var existingUser User
		err := db.QueryRow("SELECT id, name, email FROM users WHERE name = $1 OR email = $2", u.Name, u.Email).Scan(&existingUser.ID, &existingUser.Name, &existingUser.Email)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Error checking for existing user", http.StatusInternalServerError)
			log.Println(err)
			return
		}
		if existingUser.ID != 0 {
			http.Error(w, "User with the same name or email already exists", http.StatusConflict)
			return
		}

		// insert the user in the database
		err = db.QueryRow("INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id", u.Name, u.Email).Scan(&u.ID)
		if err != nil {
			http.Error(w, "Error creating user", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		// Resposta de sucesso
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(u)
	}
}

// update user
func updateUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		// verify if the fields are not empty
		if u.Name == "" || u.Email == "" {
			http.Error(w, "Name and Email are required", http.StatusBadRequest)
			return
		}

		vars := mux.Vars(r)
		id := vars["id"]

		// verify if the email is not already in use
		var existingUser User
		err := db.QueryRow("SELECT id, name, email FROM users WHERE email = $1 AND id != $2", u.Email, id).Scan(&existingUser.ID, &existingUser.Name, &existingUser.Email)
		if err != nil && err != sql.ErrNoRows {
			log.Fatal(err)
		}
		if existingUser.ID != 0 {
			http.Error(w, "Email already in use by another user", http.StatusConflict)
			return
		}

		// verify if the data is the same as the current user
		var currentUser User
		err = db.QueryRow("SELECT name, email FROM users WHERE id = $1", id).Scan(&currentUser.Name, &currentUser.Email)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		if currentUser.Name == u.Name && currentUser.Email == u.Email {
			http.Error(w, "No changes detected", http.StatusBadRequest)
			return
		}

		// update the user in the database
		_, err = db.Exec("UPDATE users SET name = $1, email = $2 WHERE id = $3", u.Name, u.Email, id)
		if err != nil {
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode(u)
	}
}

// delete user
func deleteUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id := vars["id"]

		_, err := db.Exec("DELETE FROM users WHERE id = $1", id)
		if err != nil {
			// TODO: fix error handling
			log.Fatal(err)
		}

		json.NewEncoder(w).Encode("User deleted")
	}
}

// isValidEmail verify if the email is valid
func isValidEmail(email string) bool {
	// simple regex to validate email
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}
