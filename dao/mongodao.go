package dao

import (
	"context"
	"fmt"
	"log"
	"shorturl/environment"
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
	- retries for operations
*/

type MongoDB struct {
	client mongo.Client
}

const dbName = "shorturl"
const collectionName = "urls"
const urlFieldName = "url"
const abvFieldName = "abv"
const hitsFieldName = "hits"
const lastAccessFieldName = "last_access"
const dailyHitsFieldName = "daily_hits"

var once sync.Once

func ctx() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), environment.GetEnvDurationOrDefault("timeout", 10*time.Second))
	return ctx
}

func CreateMongoDB(uri string) ShortUrlDao {
	client, err := mongo.NewClient(options.Client().
		ApplyURI(uri).
		SetAppName("shorturl"))

	if err != nil {
		log.Fatalf("Couldn't create client: %v", err)
	}
	ctx := ctx()
	if err = client.Connect(ctx); err != nil {
		log.Fatalf("Couldn't connect: %v", err)
	}

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
	})

	return &MongoDB{client: *client}
}

func (d *MongoDB) Cleanup() {
	ctx := ctx()
	_ = d.client.Disconnect(ctx)
}

func (d *MongoDB) IsLikelyOk() bool {
	ctx := ctx()
	if err := d.client.Ping(ctx, readpref.Primary()); err != nil {
		log.Printf("Ping failed: %v", err)
		return false
	}
	return true
}

func (d *MongoDB) Save(abv string, url string) error {
	ctx := ctx()
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
	ctx := ctx()
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{abvFieldName: abv}
	if _, err := collection.DeleteOne(ctx, m); err != nil {
		return fmt.Errorf("couldn't delete Abbreviation %s: %v", abv, err)
	}

	return nil
}

func (d *MongoDB) DeleteUrl(url string) error {
	ctx := ctx()
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{urlFieldName: url}
	if _, err := collection.DeleteOne(ctx, m); err != nil {
		return fmt.Errorf("couldn't delete Url %s: %v", url, err)
	}

	return nil
}

func (d *MongoDB) GetUrl(abv string) (string, error) {
	ctx := ctx()
	collection := d.client.Database(dbName).Collection(collectionName)
	abvKey := bson.M{abvFieldName: abv}
	result := collection.FindOne(ctx, abvKey)

	if result.Err() != nil {
		//return false, fmt.Errorf("error looking up %s: %v", Abbreviation, result.Err())
		return "", nil
	}

	var data ShortUrl
	if err := result.Decode(&data); err != nil {
		return "", fmt.Errorf("error decoding return %s: %v", abv, result.Err())
	}

	go func() {
		update := bson.D{{"$inc", bson.D{{hitsFieldName, 1}}},
			{"$currentDate", bson.D{{lastAccessFieldName, true}}},
			{"$inc", bson.D{{dailyHitsFieldName + "." + Date(), 1}}},
		}
		if _, err := collection.UpdateOne(ctx, abvKey, update); err != nil {
			log.Printf("Error updating doc %v", err)
		}
	}()
	return data.Url, nil
}

func (d *MongoDB) GetStats(abv string) (ShortUrl, error) {
	ctx := ctx()
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{abvFieldName: abv}
	result := collection.FindOne(ctx, m)

	if result.Err() != nil {
		//return false, fmt.Errorf("error looking up %s: %v", Abbreviation, result.Err())
		return ShortUrl{}, nil
	}

	var data ShortUrl
	if err := result.Decode(&data); err != nil {
		return ShortUrl{}, fmt.Errorf("error decoding return %s: %v", abv, result.Err())
	}

	return data, nil
}

func (d *MongoDB) GetAbv(url string) (string, error) {
	ctx := ctx()
	collection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{urlFieldName: url}
	result := collection.FindOne(ctx, m)

	if result.Err() != nil {
		//return false, fmt.Errorf("error looking up %s: %v", Url, result.Err())
		return "", nil
	}

	var data ShortUrl
	if err := result.Decode(&data); err != nil {
		return "", fmt.Errorf("error decoding return %s: %v", url, result.Err())
	}

	return data.Abbreviation, nil
}
