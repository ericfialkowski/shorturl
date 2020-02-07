package dao

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

/*
Still TODO:
	- parameterize user/password/host
	- retries for operations
	- get a client in each call?
	- parameterize timeouts
*/

type MongoDB struct {
	client mongo.Client
}

const dbName = "shorturl"
const collectionName = "urls"
const urlFieldName = "url"
const abvFieldName = "abv"

var once sync.Once

func CreateMongoDB(uri string) ShortUrlDao {
	client, err := mongo.NewClient(options.Client().
		ApplyURI(uri).
		SetAppName("shorturl"))

	if err != nil {
		log.Fatalf("Couldn't create client: %v", err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatalf("Couldn't connect: %v", err)
	}

	//go once.Do(func() {
	//	mod := mongo.IndexModel{
	//		Keys: bson.M{
	//			urlFieldName: 1, // index in ascending order
	//		}, Options: options.Index().SetUnique(true).SetName("uniqueness_ndx"),
	//	}
	//	collection := client.Database(dbName).Collection(collectionName)
	//	collection.Indexes().CreateOne(ctx, mod)
	//})

	//defer client.Disconnect(ctx)
	return &MongoDB{client: *client}
}

func (d *MongoDB) Cleanup() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	d.client.Disconnect(ctx)
}

func (d *MongoDB) IsLikelyOk() bool {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err := d.client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Printf("Ping failed: %v", err)
	}
	return err == nil
}

func (d *MongoDB) Save(abv string, url string) error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	collection := d.client.Database(dbName).Collection(collectionName)
	_, err := collection.InsertOne(ctx, bson.D{
		{abvFieldName, abv},
		{urlFieldName, url},
	})

	if err != nil {
		if !strings.Contains(err.Error(), "E11000 duplicate") {
			return fmt.Errorf("couldn't store (%s, %s): %v", abv, url, err)
		}
	}
	return nil
}

func (d *MongoDB) DeleteAbv(abv string) error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{abvFieldName: abv}
	_, err := collection.DeleteOne(ctx, m)
	if err != nil {
		return fmt.Errorf("couldn't delete abv %s: %v", abv, err)
	}

	return nil
}

func (d *MongoDB) DeleteUrl(url string) error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{urlFieldName: url}
	_, err := collection.DeleteOne(ctx, m)
	if err != nil {
		return fmt.Errorf("couldn't delete url %s: %v", url, err)
	}

	return nil
}

func (d *MongoDB) GetUrl(abv string) (string, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{abvFieldName: abv}
	result := collection.FindOne(ctx, m)

	if result.Err() != nil {
		//return false, fmt.Errorf("error looking up %s: %v", abv, result.Err())
		return "", nil
	}

	var data bson.M
	if err := result.Decode(&data); err != nil {
		return "", fmt.Errorf("error decoding return %s: %v", abv, result.Err())
	}

	return fmt.Sprintf("%v", data[urlFieldName]), nil
}

func (d *MongoDB) GetAbv(url string) (string, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{urlFieldName: url}
	result := collection.FindOne(ctx, m)

	if result.Err() != nil {
		//return false, fmt.Errorf("error looking up %s: %v", url, result.Err())
		return "", nil
	}

	var data bson.M
	if err := result.Decode(&data); err != nil {
		return "", fmt.Errorf("error decoding return %s: %v", url, result.Err())
	}

	return fmt.Sprintf("%v", data[abvFieldName]), nil
}
