package tracks

import (
	"context"
	"fmt"

	"github.com/ksuayan/go-tracks/coverart"
	"github.com/ksuayan/go-tracks/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// updateTracks updates the track metadata in the database with artist and album IDs and cover art hash
func UpdateTracks(db *mongo.Database, track map[string] interface{}) error {

	albumID := track["albumID"].(string)
	artistID := track["artistID"].(string)
	coverArtHash := track["coverArtHash"].(string)

	artistObjectID := mongodb.SafeObjectIDFromHex(artistID)
	albumObjectID := mongodb.SafeObjectIDFromHex(albumID)

	coverArt, err := coverart.GetCoverArtPathFromHash("", coverArtHash)
	if err != nil {
		return fmt.Errorf("error getting cover art path for trackID %v: %v", track["_id"], err)
	}
	_, err = db.Collection("tracks").UpdateOne(
		context.Background(),
		bson.M{"_id": track["_id"]},
		bson.M{
			"$set": bson.M{
				"coverArtHash": coverArtHash,
				"coverArt":     coverArt, 
				"artistID":     artistObjectID,
				"albumID":      albumObjectID,
				"status":       "cover",
			},
		},
	)
	if err != nil {
		return fmt.Errorf("error updating database for track with ID %v: %v", track["_id"], err)
	}
	return nil
}
