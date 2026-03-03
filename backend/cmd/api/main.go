package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
func getPort() string {
	return getEnv("PORT", "8081")
}

func initDB() error {
	host := getEnv("POSTGRES_HOST", "postgres")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "")
	dbname := getEnv("POSTGRES_DB", "nightwatch")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	log.Printf("Connection to database at %s:%s/%s...", host, port, dbname)
	var err error
	for i := 0; i < 10; i++ {
		db, err = sql.Open("postgres", connStr)
		if err == nil {
			err = db.Ping()
			if err == nil {
				log.Printf("Database connection established successfully.")
				return nil
			}
		}
		log.Printf("Database connection attempt %d/10 failed: %v", i+1, err)
		time.Sleep(time.Duration(2*(i+1)) * time.Second)
	}

	return fmt.Errorf("failed to connect to database after 10 attempts: %v", err)
}

func healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	log.Printf("[%s] %s %s - Remote: %s", r.Method, r.URL.Path, r.Proto, r.RemoteAddr)
	fmt.Fprintf(w, `{"status": "ok"}`)
}

func main() {
	if err := initDB(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	port := getPort()
	http.HandleFunc("/status", healthcheckHandler)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
