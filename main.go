package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Println("Usage: go run main.go <input_dir> <output_dir> <num_workers>")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputDir := os.Args[2]
	numWorkers := parseNumWorkers(os.Args[3])

	// Initialize MongoDB client
	mongoURI := "mongodb://localhost:27017"
	client, db, err := connectToMongoDB(mongoURI)
	if err != nil {
		fmt.Printf("Error connecting to MongoDB: %v\n", err)
		os.Exit(1)
	}
	defer client.Disconnect(context.Background())

	// Create temporary directory
	tempDir := filepath.Join(outputDir, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("Failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Step 1: Scan directory and update tracks
	fmt.Println("Scanning directory and updating tracks...")
	err = scanDirectoryAndUpdateDB(inputDir, db)
	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	// Step 2: Start worker pool
	fmt.Println("Processing cover art and updating metadata...")
	var wg sync.WaitGroup
	tasks := make(chan map[string]interface{}, numWorkers) // Task channel

	// Launch workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go worker(tasks, db, outputDir, &wg)
	}

	// Enqueue tasks
	err = enqueueTasks(db, tasks)
	close(tasks) // Close tasks channel after enqueueing
	if err != nil {
		fmt.Printf("Error enqueueing tasks: %v\n", err)
		os.Exit(1)
	}

	// Wait for all workers to finish
	wg.Wait()
	fmt.Println("All tasks completed successfully!")

}
