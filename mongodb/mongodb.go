package mongodb

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect to MongoDB
func ConnectToMongoDB(uri string) (*mongo.Client, *mongo.Database, error) {
	client, err := mongo.NewClient(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		return nil, nil, err
	}

	db := client.Database("musicdb")
	return client, db, nil
}

func ToHex(id primitive.ObjectID) string {
	return id.Hex()
}

// SafeObjectIDFromHex attempts to convert a string to an ObjectID.
// If the input is invalid, it returns a zero-value ObjectID.
func SafeObjectIDFromHex(id string) primitive.ObjectID {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return primitive.NilObjectID
	}
	return objectID
}