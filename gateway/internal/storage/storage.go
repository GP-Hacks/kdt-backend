package storage

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var client *mongo.Client
var collection *mongo.Collection

func Connect(uri, dbName, collectionName string) error {
	clientOptions := options.Client().ApplyURI(uri)
	var err error
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return err
	}
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return err
	}
	collection = client.Database(dbName).Collection(collectionName)
	return nil
}

func AddUserToken(userID, token string) error {
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"user_id": userID},
		bson.M{"$addToSet": bson.M{"tokens": token}},
		options.Update().SetUpsert(true),
	)
	return err
}
