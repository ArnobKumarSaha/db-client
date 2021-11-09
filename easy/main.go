package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"log"
)

func main()  {
	ctx := context.TODO()
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	// /?replicaSet=replset  <- for replicaset
	// ,localhost:27018  <- for sharded

	mongoClient, err := mongo.Connect(ctx, clientOptions)
	defer func() {
		if err = mongoClient.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// checking connection
	err = mongoClient.Ping(ctx, readpref.Primary())

	if err != nil {
		fmt.Println("Error occured when pinging")
	}else{
		fmt.Println("Ping successful")
	}


	// To insert a document
	xy , _ := mongoClient.ListDatabaseNames(ctx, &options.ListDatabasesOptions{})
	fmt.Println("databases : ", xy)
	collection := mongoClient.Database("testing").Collection("numbers")

	//ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	//defer cancel()
	res, err := collection.InsertOne(ctx, bson.D{{"name", "pi"}, {"value", 3.14159}})
	id := res.InsertedID
	fmt.Println("insert id = ", id)

	cur, err := collection.Find(ctx, bson.D{})
	if err != nil { log.Fatal(err) }
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var result bson.D
		err := cur.Decode(&result)
		if err != nil { log.Fatal(err) }
		fmt.Println(result)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

	// single Instance
	var result struct {
		Value float64
	}
	filter := bson.D{{"name", "pi"}}

	err = collection.FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		// Do something when no record was found
		fmt.Println("record does not exist")
	} else if err != nil {
		log.Fatal(err)
	}else{
		fmt.Println("Single instance retrieved from collection -> ", result)
	}
}

