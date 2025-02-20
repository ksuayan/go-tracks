package worker

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/ksuayan/go-tracks/albums"
	"github.com/ksuayan/go-tracks/artists"
	"github.com/ksuayan/go-tracks/coverart"
	"github.com/ksuayan/go-tracks/tracks"
)

const (
	mbEnabled = false
)

// Worker function for processing tracks
func Worker(tasks <-chan map[string]interface{}, db *mongo.Database, outputDir string, wg *sync.WaitGroup) {
	defer wg.Done()
	
	for track := range tasks {
		// Validate required fields

		rootDir := track["rootDir"].(string)
		subDir := track["subDir"].(string)
		fileName := track["fileName"].(string)
		filePath := fmt.Sprintf("%s/%s/%s", rootDir, subDir, fileName)

		// fmt.Printf("Processing file: %s\n", filePath)

		// Extract Cover Art
		coverArtHash, coverArtPath, err := coverart.ExtractCoverArt(db, track, outputDir)
		if err != nil {
			log.Printf("Error extracting cover art for %s: %v\n", coverArtPath, err)
			continue
		}
		track["coverArtHash"] = coverArtHash

		// Update Artist
		artistID, err := artists.UpdateArtists(db, track, mbEnabled)
		if err != nil {
			log.Printf("Error updating artist for %s: %v\n", filePath, err)
			continue
		}
		track["artistID"] = artistID 

		// Update Album
		albumID, err := albums.UpdateAlbums(db, track)
		if err != nil {
			log.Printf("Error updating album for %s: %v\n", filePath, err)
			continue
		}
		track["albumID"] = albumID

		// Update Track Metadata
		err = tracks.UpdateTracks(db, track)
		if err != nil {
			log.Printf("Error updating track metadata for %s: %v\n", filePath, err)
		}
	}
}

// Enqueue tasks for worker pool
func EnqueueTasks(db *mongo.Database, tasks chan<- map[string]interface{}, wg *sync.WaitGroup) error {
	defer wg.Done()
	cursor, err := db.Collection("tracks").Find(context.Background(), bson.M{"status": bson.M{"$in": []string{"new", "updated"}}})
	if err != nil {
			return err
	}
	defer cursor.Close(context.Background())

	count := 0
	for cursor.Next(context.Background()) {
		var track map[string]interface{}
		if err := cursor.Decode(&track); err != nil {
			return err
		}
		tasks <- track
		count++
	}
	log.Printf("Total tasks enqueued: %d\n", count) // Log total tasks enqueued
	return nil
}