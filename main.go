package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
	}
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Connected to the database successfully!")
	})
	http.HandleFunc("/upload", uploadFileHandler)
	http.HandleFunc("/getdata", getFileDataHandler)
	http.HandleFunc("/download", downloadFileHandler)

	fmt.Println("Server started at :8989")
	log.Fatal(http.ListenAndServe(":8989", nil))
}
