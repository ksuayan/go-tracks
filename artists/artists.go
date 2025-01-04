package artists

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/ksuayan/go-tracks/mongodb"
	"github.com/ksuayan/go-tracks/musicbrainz"
	"github.com/ksuayan/go-tracks/utils"
)

// Update Artist in the database and return the artist ID
func UpdateArtists(db *mongo.Database, track map[string] interface {}, mbEnabled bool) (string, error) {

	artist := track["artist"].(string)

	tags, ok := utils.SafeGet(track, "ffprobe", "format", "tags")
	if !ok {
		fmt.Printf("No tags found for %s\n", artist)
	}

	artistsCollection := db.Collection("artists")
	artistFilter := bson.M{"name": artist}
	var artistUpdate bson.M

	if (mbEnabled) {
		mbArtistID, ok := tags.(map[string]interface{})["MusicBrainz Artist Id"].(string)
		if !ok {
			fmt.Printf("Error: MusicBrainz Artist Id not found or not a string")
		}

		mbArtistData, err := musicbrainz.FetchMusicBrainz("artist", mbArtistID)
		if err != nil {
			fmt.Printf("Error fetching MusicBrainz Artist Data for %s: %v\n", artist, err)
		}

		artistUpdate = bson.M{"$set": bson.M{
			"name": artist, 
			"musicbrainz": mbArtistData,
		}}
	} else {
		artistUpdate = bson.M{"$set": bson.M{
			"name": artist, 
		}}
	}

	artistOpts := options.Update().SetUpsert(true)
	artistRes, err := artistsCollection.UpdateOne(context.Background(), artistFilter, artistUpdate, artistOpts)
	if err != nil {
		return "", err
	}

	var artistID string
	if artistRes.UpsertedID != nil {
		artistID = mongodb.ToHex(artistRes.UpsertedID.(primitive.ObjectID))
		fmt.Printf("Upserted Artist ID: %s\n", artistID)
	} else {
		var existingArtist bson.M
		err := artistsCollection.FindOne(context.Background(), artistFilter).Decode(&existingArtist)
		if err != nil {
			return "", err
		}
		artistID = mongodb.ToHex(existingArtist["_id"].(primitive.ObjectID))
		fmt.Printf("Existing Artist ID: %s\n", artistID)
	}

	return artistID, nil
}
