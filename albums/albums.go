package albums

import (
	"context"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ksuayan/go-tracks/mongodb"
)

// Update Album in the database and return the album ID
func UpdateAlbums(db *mongo.Database, track map[string] interface {} ) (string, error) {

	album := track["album"].(string)
	artistID := track["artistID"].(string)
	albumArtist := track["albumArtist"].(string)
	coverArtHash := track["coverArtHash"].(string)

	if albumArtist == "" {
		albumArtist = artistID
	}

	albumsCollection := db.Collection("albums")
	albumFilter := bson.M{"name": album, "albumArtist": albumArtist}
	albumUpdate := bson.M{
		"$set": bson.M{
			"name":        album,
			"albumArtist":    albumArtist,
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
		albumID = mongodb.ToHex(albumRes.UpsertedID.(primitive.ObjectID))
		log.Printf("Upserted Album: %s\n", album)
	} else {
		var existingAlbum bson.M
		err := albumsCollection.FindOne(context.Background(), albumFilter).Decode(&existingAlbum)
		if err != nil {
			return "", err
		}
		albumID = mongodb.ToHex(existingAlbum["_id"].(primitive.ObjectID))
		// fmt.Printf("Existing Album: %s\n", album)
	}

	return albumID, nil
}
