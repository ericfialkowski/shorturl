package dao

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ericfialkowski/shorturl/env"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type MongoDB struct {
	client *mongo.Client
}

const (
	dbName              = "shorturl"
	collectionName      = "urls"
	urlFieldName        = "url"
	abvFieldName        = "abv"
	hitsFieldName       = "hits"
	lastAccessFieldName = "last_access"
	dailyHitsFieldName  = "daily_hits"
)

var once sync.Once

func newContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), env.DurationOrDefault("mongo_timeout", 10*time.Second))
}

func CreateMongoDB(uri string) ShortUrlDao {
	client, err := mongo.Connect(options.Client().
		ApplyURI(uri).
		SetAppName("shorturl"))

	if err != nil {
		log.Fatalf("Couldn't create client: %v", err)
	}
	ctx, cancel := newContext()
	defer cancel()

	once.Do(func() {
		mod := mongo.IndexModel{
			Keys: bson.M{
				abvFieldName: 1, // index in ascending order
			}, Options: options.Index().SetUnique(true).SetName("abv_uniqueness_ndx"),
		}
		collection := client.Database(dbName).Collection(collectionName)
		if _, err = collection.Indexes().CreateOne(ctx, mod); err != nil {
			log.Printf("Error creating index %v", err)
		}

		mod = mongo.IndexModel{
			Keys: bson.M{
				urlFieldName: 1, // index in ascending order
			}, Options: options.Index().SetUnique(true).SetName("url_uniqueness_ndx"),
		}
		if _, err = collection.Indexes().CreateOne(ctx, mod); err != nil {
			log.Printf("Error creating index %v", err)
		}
	})

	return &MongoDB{client: client}
}

func (d *MongoDB) Cleanup() {
	ctx, cancel := newContext()
	defer cancel()
	_ = d.client.Disconnect(ctx)
}

func (d *MongoDB) IsLikelyOk() bool {
	ctx, cancel := newContext()
	defer cancel()
	if err := d.client.Ping(ctx, readpref.Primary()); err != nil {
		log.Printf("Ping failed: %v", err)
		return false
	}
	return true
}

func (d *MongoDB) Save(abv string, url string) error {
	ctx, cancel := newContext()
	defer cancel()
	collection := d.client.Database(dbName).Collection(collectionName)
	data := ShortUrl{Abbreviation: abv, Url: url, Hits: 0}
	if _, err := collection.InsertOne(ctx, data); err != nil {
		if !strings.Contains(err.Error(), "E11000 duplicate") {
			return fmt.Errorf("couldn't store (%s, %s): %v", abv, url, err)
		}
	}
	return nil
}

func (d *MongoDB) DeleteAbv(abv string) error {
	ctx, cancel := newContext()
	defer cancel()
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{abvFieldName: abv}
	if _, err := collection.DeleteOne(ctx, m); err != nil {
		return fmt.Errorf("couldn't delete Abbreviation %s: %v", abv, err)
	}

	return nil
}

func (d *MongoDB) DeleteUrl(url string) error {
	ctx, cancel := newContext()
	defer cancel()
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{urlFieldName: url}
	if _, err := collection.DeleteOne(ctx, m); err != nil {
		return fmt.Errorf("couldn't delete Url %s: %v", url, err)
	}

	return nil
}

func (d *MongoDB) GetUrl(abv string) (string, error) {
	ctx, cancel := newContext()
	defer cancel()
	collection := d.client.Database(dbName).Collection(collectionName)
	abvKey := bson.M{abvFieldName: abv}
	result := collection.FindOne(ctx, abvKey)

	if result.Err() != nil {
		return "", nil
	}

	var data ShortUrl
	if err := result.Decode(&data); err != nil {
		return "", fmt.Errorf("error decoding return %s: %v", abv, result.Err())
	}

	go func() {
		ctx, cancel := newContext()
		defer cancel()
		update := bson.D{{Key: "$inc", Value: bson.D{{Key: hitsFieldName, Value: 1}}},
			{Key: "$currentDate", Value: bson.D{{Key: lastAccessFieldName, Value: true}}},
			{Key: "$inc", Value: bson.D{{Key: dailyHitsFieldName + "." + Date(), Value: 1}}},
		}
		if _, err := collection.UpdateOne(ctx, abvKey, update); err != nil {
			log.Printf("Error updating doc %v", err)
		}
	}()
	return data.Url, nil
}

func (d *MongoDB) GetStats(abv string) (ShortUrl, error) {
	ctx, cancel := newContext()
	defer cancel()
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{abvFieldName: abv}
	result := collection.FindOne(ctx, m)

	if result.Err() != nil {
		log.Printf("error getting stats %v", result.Err())
		return ShortUrl{}, nil
	}

	var data ShortUrl
	if err := result.Decode(&data); err != nil {
		return ShortUrl{}, fmt.Errorf("error decoding return %s: %v", abv, result.Err())
	}

	return data, nil
}

func (d *MongoDB) GetAbv(url string) (string, error) {
	ctx, cancel := newContext()
	defer cancel()
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{urlFieldName: url}
	result := collection.FindOne(ctx, m)

	if result.Err() != nil {
		log.Printf("error getting abreviation %v", result.Err())
		return "", nil
	}

	var data ShortUrl
	if err := result.Decode(&data); err != nil {
		return "", fmt.Errorf("error decoding return %s: %v", url, result.Err())
	}

	return data.Abbreviation, nil
}
