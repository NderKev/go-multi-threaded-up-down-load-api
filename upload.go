package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"

	"github.com/jackc/pgx/v4"
	_ "github.com/lib/pq"
)

const (
	db_host     = "localhost"
	db_port     = 5432
	db_user     = "postgres"
	db_password = "BigBrain_70"
	db_name     = "upload"
	segmentSize = 1024 * 1024
)

// FileSegment represents a segment of a file
type FileSegment struct {
	SegmentID   int
	SegmentName string
	FileData    []byte
}

// uploadFileHandler handles file upload requests
func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(100 << 20) // 100 MB
	if err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fileName := handler.Filename
	fileID, err := upload(fileName)
	if err != nil {
		http.Error(w, "Error uploading file", http.StatusBadRequest)
		return
	}
	// Return FileID in JSON format
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fileID)

}

// getFileDataHandler retrieves metadata for a file
func getFileDataHandler(w http.ResponseWriter, r *http.Request) {
	fileId := r.URL.Query().Get("fileID")
	if fileId == "" {
		http.Error(w, "Missing fileID", http.StatusBadRequest)
		return
	}

	fileID, err := strconv.Atoi(fileId)
	if err != nil {
		fmt.Println("Error converting fileid to int:", err)
		return
	}

	response, err := retrieve(fileID)
	if err != nil {
		fmt.Println("Error retrieving files segments by File ID:", err)
		return
	}
	// Return metadata in JSON format
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// downloadFileHandler handles file download requests
func downloadFileHandler(w http.ResponseWriter, r *http.Request) {
	fileId := r.URL.Query().Get("fileID")
	if fileId == "" {
		http.Error(w, "Missing fileID", http.StatusBadRequest)
		return
	}

	fileID, err := strconv.Atoi(fileId)
	if err != nil {
		fmt.Println("Error converting fileid to int:", err)
		return
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		db_host, db_port, db_user, db_password, db_name)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Retrieve file segments
	segments, err := fetchFileSegments(db, fileID)
	if err != nil {
		http.Error(w, "Error retrieving file segments", http.StatusInternalServerError)
		return
	}

	// Sort segments by part_id to ensure correct order
	sort.Slice(segments, func(i, j int) bool {
		return segments[i].SegmentID < segments[j].SegmentID
	})

	// Merge file segments
	fileData, err := mergeFileSegments(segments)
	if err != nil {
		http.Error(w, "Error merging file segments", http.StatusInternalServerError)
		return
	}

	// Serve the file as a download
	w.Header().Set("Content-Disposition", "attachment; filename=\"merged_file\"")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(fileData)))
	w.Write(fileData)
}

// mergeFileSegments merges file segments into a single byte slice
func mergeFileSegments(segments []FileSegment) ([]byte, error) {
	var wg sync.WaitGroup
	dataChan := make(chan []byte, len(segments))
	errorChan := make(chan error, 1)

	for _, segment := range segments {
		wg.Add(1)
		go func(segment FileSegment) {
			defer wg.Done()
			dataChan <- segment.FileData
		}(segment)
	}

	go func() {
		wg.Wait()
		close(dataChan)
		close(errorChan)
	}()

	var fileData []byte
	for partData := range dataChan {
		fileData = append(fileData, partData...)
	}

	return fileData, nil
}

func upload(fileName string) (int, error) {
	file, err := os.Open(fileName)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return 0, err
	}
	defer file.Close()

	var wg sync.WaitGroup
	ch := make(chan string)

	// Create a new file record in the database
	fileID, err := createFileRecord(file.Name())
	if err != nil {
		fmt.Println("Error creating file record:", err)
		return 0, err
	}

	// Start workers for concurrent uploads
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go uploadWorker(&wg, ch, fileID)
	}

	err = splitFile(file, ch)
	if err != nil {
		fmt.Println("Error splitting file:", err)
		return 0, err
	}

	close(ch)
	wg.Wait()
	fmt.Printf("All file segments uploaded successfully. File ID: %d\n", fileID)
	return fileID, nil
}

func splitFile(file *os.File, ch chan<- string) error {
	partNum := 1
	buf := make([]byte, segmentSize)

	for {
		bytesRead, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if bytesRead == 0 {
			break
		}

		fileSegmentName := fmt.Sprintf("part_%d.mp4", partNum)
		fileSegment, err := os.Create(fileSegmentName)
		if err != nil {
			return err
		}

		_, err = fileSegment.Write(buf[:bytesRead])
		if err != nil {
			fileSegment.Close()
			return err
		}
		fileSegment.Close()

		ch <- fileSegmentName
		partNum++
	}

	return nil
}

func uploadWorker(wg *sync.WaitGroup, ch <-chan string, fileID int) {
	defer wg.Done()

	conn, err := connectDB()
	if err != nil {
		fmt.Println("Error connecting to DB:", err)
		return
	}
	defer conn.Close(context.Background())

	for fileSegmentName := range ch {
		fileID, err := uploadFileSegment(conn, fileSegmentName, fileID)
		if err != nil {
			fmt.Println("Error uploading part", fileSegmentName, ":", err)
		} else {
			fmt.Printf("Uploaded part: %s with file ID: %d\n", fileSegmentName, fileID)
		}
	}
}

func connectDB() (*pgx.Conn, error) {
	connStr := fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
		db_user, db_password, db_host, db_port, db_name)
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// createFileRecord creates a new file record and returns the file ID.
func createFileRecord(fileName string) (int, error) {
	conn, err := connectDB()
	if err != nil {
		return 0, err
	}
	defer conn.Close(context.Background())

	var fileID int
	err = conn.QueryRow(context.Background(),
		"INSERT INTO files (file_name) VALUES ($1) RETURNING file_id",
		fileName).Scan(&fileID)
	if err != nil {
		return 0, err
	}

	return fileID, nil
}

// uploadFileSegment uploads a file segment and links it to the given file ID.
func uploadFileSegment(conn *pgx.Conn, fileSegmentName string, fileID int) (int, error) {
	fileSegment, err := os.Open(fileSegmentName)
	if err != nil {
		return 0, err
	}
	defer fileSegment.Close()

	fileContent, err := io.ReadAll(fileSegment)
	if err != nil {
		return 0, err
	}

	var fileIndex int
	// Insert the file part and return the file ID of the associated file record
	err = conn.QueryRow(context.Background(),
		"INSERT INTO file_segments (file_id, file_name, file_data) VALUES ($1, $2, $3) RETURNING file_id", //(SELECT file_id FROM files WHERE file_name = $1 LIMIT 1)
		fileID, fileSegmentName, fileContent).Scan(&fileIndex)
	if err != nil {
		return 0, err
	}

	return fileIndex, nil
}

/**
func test() (error){

	connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database: ", err)
		return err
	}
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Connected to the database successfully!")
	})

	fmt.Println("Server started at :8989")
	log.Fatal(http.ListenAndServe(":8989", nil))

} **/
