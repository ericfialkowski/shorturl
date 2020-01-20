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
const fieldName = "name"

var once sync.Once

func CreateMongoDB(uri string) MongoDB {
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

	go once.Do(func() {
		mod := mongo.IndexModel{
			Keys: bson.M{
				fieldName: 1, // index in ascending order
			}, Options: options.Index().SetUnique(true).SetName("uniqueness_ndx"),
		}
		poisonCollection := client.Database(dbName).Collection(collectionName)
		poisonCollection.Indexes().CreateOne(ctx, mod)
	})

	//defer client.Disconnect(ctx)
	return MongoDB{client: *client}
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

func (d *MongoDB) Save(app string) error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	poisonCollection := d.client.Database(dbName).Collection(collectionName)
	_, err := poisonCollection.InsertOne(ctx, bson.D{
		{fieldName, app},
	})

	if err != nil {
		if !strings.Contains(err.Error(), "E11000 duplicate") {
			return fmt.Errorf("couldn't poison %s: %v", app, err)
		}
	}
	return nil
}

func (d *MongoDB) Delete(app string) error {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	poisonCollection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{fieldName: app}
	_, err := poisonCollection.DeleteOne(ctx, m)
	if err != nil {
		return fmt.Errorf("couldn't unpoison %s: %v", app, err)
	}

	return nil
}

func (d *MongoDB) Exists(app string) (bool, error) {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	poisonCollection := d.client.Database(dbName).Collection(collectionName)
	m := bson.M{fieldName: app}
	result := poisonCollection.FindOne(ctx, m)

	if result.Err() != nil {
		//return false, fmt.Errorf("error looking up %s: %v", app, result.Err())
		return false, nil
	}

	var data bson.M
	if err := result.Decode(&data); err != nil {
		return false, fmt.Errorf("error decoding return %s: %v", app, result.Err())
	}

	return true, nil
}

func (d *MongoDB) List() ([]string, error) {
	rtn := make([]string, 0)
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	poisonCollection := d.client.Database(dbName).Collection(collectionName)
	cursor, err := poisonCollection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("couldn't list applications: %v", err)
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var app bson.M
		if err = cursor.Decode(&app); err != nil {
			return rtn, fmt.Errorf("error decoding applications: %v", err)
		}
		rtn = append(rtn, fmt.Sprint(app[fieldName])) // TODO: see if there is a better way to do this
	}

	return rtn, nil
}
