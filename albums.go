package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Update Album in the database and return the album ID
func updateAlbums(db *mongo.Database, track map[string] interface {} ) (string, error) {
	album := track["album"].(string)
	artistID := track["artistID"].(string)
	coverArtHash := track["coverArtHash"].(string)

	albumsCollection := db.Collection("albums")
	albumFilter := bson.M{"name": album, "artistID": artistID}
	albumUpdate := bson.M{
		"$set": bson.M{
			"name":        album,
			"artistID":    artistID,
			"coverArtHash": coverArtHash,
		},
	}
	albumOpts := options.Update().SetUpsert(true)
	albumRes, err := albumsCollection.UpdateOne(context.Background(), albumFilter, albumUpdate, albumOpts)
	if err != nil {
		return "", err
	}

	var albumID string
	if albumRes.UpsertedID != nil {
		albumID = toHex(albumRes.UpsertedID.(primitive.ObjectID))
		fmt.Printf("Upserted Album ID: %s\n", albumID)
	} else {
		var existingAlbum bson.M
		err := albumsCollection.FindOne(context.Background(), albumFilter).Decode(&existingAlbum)
		if err != nil {
			return "", err
		}
		albumID = toHex(existingAlbum["_id"].(primitive.ObjectID))
		fmt.Printf("Existing Album ID: %s\n", albumID)
	}

	return albumID, nil
}
