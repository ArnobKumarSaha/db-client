package main

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const(
	SampleMfilxDatabase string = "sample_mflix"
	MoviesCollection string = "movies"
)
/*
You should use bson.D() if order matters.
general approach ::
Use bson.D() when creating the filter Option,
use bson.M() when to declare the variable 'result'
 */

func main() {
	uri := "mongodb+srv://arnobkumarsaha:sustcse16@cluster0.nj6lk.mongodb.net/testdb?retryWrites=true&w=majority"
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	//findOne(client)
	//find(client)

	//insertOne(client)
	//insertMany(client)
	//update(client)

	//bulk(client)
	//watch(client)

	// client.Database("").CreateCollection()
	// What about DatabaseOptions, CreateCollectionOptions from https://github.com/mongodb/mongo-go-driver/tree/master/mongo/options ??
}


func watch(client *mongo.Client)  {
	// Call insertOne from another terminal to do work with it .
	coll := client.Database("insertDB").Collection("haikus")
	pipeline := mongo.Pipeline{bson.D{{"$match", bson.D{{"operationType", "insert"}}}}}
	cs, err := coll.Watch(context.TODO(), pipeline)
	if err != nil {
		panic(err)
	}
	defer cs.Close(context.TODO())
	fmt.Println("Waiting For Change Events. Insert something in MongoDB!")
	for cs.Next(context.TODO()) {
		var event bson.M
		if err := cs.Decode(&event); err != nil {
			panic(err)
		}
		output, err := json.MarshalIndent(event["fullDocument"], "", "    ")
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", output)
	}
	if err := cs.Err(); err != nil {
		panic(err)
	}
}

func bulk(client *mongo.Client)  {
	// Doing multiple thing in one command.
	// begin bulk
	coll := client.Database("insertDB").Collection("haikus")
	models := []mongo.WriteModel{
		mongo.NewReplaceOneModel().SetFilter(bson.D{{"title", "Record of a Shriveled Datum"}}).
			SetReplacement(bson.D{{"title", "Dodging Greys"}, {"text", "When there're no matches, no longer need to panic. You can use upsert"}}),
		mongo.NewUpdateOneModel().SetFilter(bson.D{{"title", "Dodging Greys"}}).
			SetUpdate(bson.D{{"$set", bson.D{{"title", "Dodge The Greys"}}}}),
	}
	opts := options.BulkWrite().SetOrdered(true)

	results, err := coll.BulkWrite(context.TODO(), models, opts)
	// end bulk

	if err != nil {
		panic(err)
	}

	// When you run this file for the first time, it should print:
	// Number of documents replaced or modified: 2
	fmt.Printf("Number of documents replaced or modified: %d", results.ModifiedCount)
}

func update(client *mongo.Client)  {
	// begin updateone
	coll := client.Database("sample_restaurants").Collection("restaurants")
	id, _ := primitive.ObjectIDFromHex("5eb3d668b31de5d588f42a7a")
	filter := bson.D{{"_id", id}}
	update := bson.D{{"$set", bson.D{{"avg_rating", 4.2}}}}
	// if avg_rating exists, the value of it will be modified
	// otherwise , will be created.
	// if this field is exactly as same as you specified ,  this will not be updated.

	result, err := coll.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		panic(err)
	}
	// end updateone

	// When you run this file for the first time, it should print:
	// Number of documents replaced: 1
	fmt.Printf("Documents updated: %v\n", result.ModifiedCount)
}

func insertMany(client *mongo.Client)  {
	coll := client.Database("insertDB").Collection("haikus")
	docs := []interface{}{
		bson.D{{"title", "Record of a Shriveled Datum"}, {"text", "No bytes, no problem. Just insert a document, in MongoDB"}},
		bson.D{{"title", "Showcasing a Blossoming Binary"}, {"text", "Binary data, safely stored with GridFS. Bucket the data"}},
	}
	result, err := coll.InsertMany(context.TODO(), docs)
	if err != nil {
		panic(err)
	}
	for _, id := range result.InsertedIDs {
		fmt.Printf("\t%s\n", id)
	}
}

func insertOne(client *mongo.Client)  {
	coll := client.Database("insertDB").Collection("haikus")
	doc := bson.D{{"title", "Record of a Shriveled Datum"}, {"text", "No bytes, no problem. Just insert a document, in MongoDB"}}
	result, err := coll.InsertOne(context.TODO(), doc)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Document inserted with ID: %s\n", result.InsertedID)
}

func find(client *mongo.Client)  {
	// begin find
	coll := client.Database("sample_training").Collection("zips")
	filter := bson.D{{"pop", bson.D{{"$lte", 500}}}}

	cursor, err := coll.Find(context.TODO(), filter)
	if err != nil {
		panic(err)
	}
	// end find

	var results []bson.M
	// Look, cursor.All() used, It iterates the cursor and decodes each document into results.
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}
	for _, result := range results {
		output, err := json.MarshalIndent(result, "", "    ")
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s\n", output)
	}
}

func findOne(client *mongo.Client)  {
	coll := client.Database(SampleMfilxDatabase).Collection(MoviesCollection)
	var result bson.M

	// find one
	err := coll.FindOne(context.TODO(), bson.D{{"title", "The Room"}}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// This error means your query did not match any documents.
			return
		}
		panic(err)
	}
	// end findOne

	output, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", output)
}

