package storage

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"net/http"
	"regexp"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoStorage struct {
	requestsCollection *mongo.Collection
}

func NewMongoStorage(dbUri string) *MongoStorage {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbUri))
	if err != nil {
		log.Fatal(err)
	}
	//defer client.Disconnect(ctx)  // TODO: Disconnect when application about to exit.

	dbName := regexp.MustCompile(`[^/]/(\w+)`).FindStringSubmatch(dbUri)[1]
	log.Printf("Opening database %q", dbName)
	requestsCollection := client.Database(dbName).Collection("requests")

	indexName := "request_expiry_index"
	_, err = requestsCollection.Indexes().DropOne(ctx, indexName)
	if err != nil {
		log.Printf("Error deleting index %v", err)
	}

	expireAfterSeconds := int32(600)
	_, err = requestsCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{"pushedAt": 1},
		Options: &options.IndexOptions{
			Name:               &indexName,
			ExpireAfterSeconds: &expireAfterSeconds,
		},
	})
	if err != nil {
		log.Printf("Error creating expiry index %v", err)
	}

	return &MongoStorage{
		requestsCollection: requestsCollection,
	}
}

func (st *MongoStorage) PushRequestToInbox(name string, request http.Request) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	_, err := st.requestsCollection.InsertOne(
		ctx,
		Entry{
			Inbox:    name,
			Protocol: request.Proto,
			Scheme:   request.URL.Scheme,
			Host:     request.Host,
			Path:     request.URL.Path,
			Method:   request.Method,
			Params:   request.URL.Query(),
			Headers:  request.Header,
			Fragment: request.URL.Fragment,
			PushedAt: time.Now().UTC(),
		},
	)
	if err != nil {
		log.Printf("Error inserting request to MongoDB: %v", err)
	}
}

func (st *MongoStorage) GetFromInbox(name string) []Entry {
	ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFunc()

	limit := int64(10)
	cursor, err := st.requestsCollection.Find(
		ctx,
		bson.M{
			"inbox": name,
		},
		&options.FindOptions{
			Limit: &limit,
			Sort: bson.M{
				"pushedAt": 1,
			},
		},
	)
	if err != nil {
		log.Printf("Error finding requests from MongoDB: %v", err)
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Printf("Error closing MongoDB cursor: %v", err)
		}
	}(cursor, ctx)

	var requests []Entry
	err = cursor.All(ctx, &requests)
	if err != nil {
		log.Printf("Error loading requests from MongoDB cursor: %v", err)
	}

	return requests
}
