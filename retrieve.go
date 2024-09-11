package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func retrieve(fileID int) ([]FileSegment, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		db_host, db_port, db_user, db_password, db_name)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	defer db.Close()

	//gofileID := 4 // Example file ID to retrieve

	fileName, err := fetchFileName(db, fileID)
	if err != nil {
		log.Fatal("Error retrieving file name:", err)
	}

	fmt.Printf("File Metadata:\nFile ID: %d\nFile Name: %s\n", fileID, fileName)

	segments, err := fetchFileSegments(db, fileID)
	if err != nil {
		log.Fatal("Error retrieving file segments:", err)
	}

	fmt.Printf("File Parts:\n")
	for _, segment := range segments {
		fmt.Printf("Segment ID: %d\nSegment Name: %s\nSegment Length: %d bytes\n",
			segment.SegmentID, segment.SegmentName, len(segment.FileData))
	}

	return segments, nil
}

// fetchFileName retrieves the file name for a given file ID
func fetchFileName(db *sql.DB, fileID int) (string, error) {
	var fileName string
	err := db.QueryRow("SELECT file_name FROM files WHERE file_id = $1", fileID).Scan(&fileName)
	if err != nil {
		return "", err
	}
	return fileName, nil
}

// fetchFileSegments retrieves all segments associated with a given file ID
func fetchFileSegments(db *sql.DB, fileID int) ([]FileSegment, error) {
	rows, err := db.Query("SELECT segment_id, file_name, file_data FROM file_segments WHERE file_id = $1", fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var segments []FileSegment
	for rows.Next() {
		var segment FileSegment
		if err := rows.Scan(&segment.SegmentID, &segment.SegmentName, &segment.FileData); err != nil {
			return nil, err
		}
		segments = append(segments, segment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return segments, nil
}
