package main

import (
	"context"
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Worker function for processing tracks
func worker(tasks <-chan map[string]interface{}, db *mongo.Database, outputDir string, wg *sync.WaitGroup) {
	defer wg.Done()

	for track := range tasks {

		// Extract track metadata
		filePath := track["filePath"].(string)
		fmt.Printf("Processing %s\n", filePath)

		// Extract Cover Art
		coverArtHash, err := extractCoverArt(filePath, outputDir); 
		if err != nil {
			fmt.Printf("Error extracting coverArtHash for %s: %v\n", filePath, err)
			continue
		}
		track["coverArtHash"] = coverArtHash

		// Update Artist
		artistID, err := updateArtists(db, track)
		if err != nil {
			fmt.Printf("Error updating artist for %s: %v\n", filePath, err)
			continue
		}
		track["artistID"] = artistID 

		// Update Album
		albumID, err := updateAlbums(db, track)
		if err != nil {
			fmt.Printf("Error updating album for %s: %v\n", filePath, err)
			continue
		}
		track["albumID"] = albumID

		// Update Track Metadata
		err = updateTracks(db, track)
		if err != nil {
			fmt.Printf("Error updating track metadata for %s: %v\n", filePath, err)
		}
	}
}

// Enqueue tasks for worker pool
func enqueueTasks(db *mongo.Database, tasks chan<- map[string]interface{}) error {
	cursor, err := db.Collection("tracks").Find(context.Background(), bson.M{"status": bson.M{"$in": []string{"new", "updated"}}})
	if err != nil {
		return err
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var track map[string]interface{}
		if err := cursor.Decode(&track); err != nil {
			return err
		}
		tasks <- track
	}
	return nil
}