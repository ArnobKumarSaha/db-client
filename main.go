package main

import (
	"context"
	"fmt"
	"github.com/Arnobkumarsaha/mongotry/dbclient"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"kubedb.dev/apimachinery/client/clientset/versioned"
	"time"
)

const(
	mongoDBName string = "mgo-rs"
	mongoDBNamespace string = "demo"
	mongoDBVersion string = "4.2.3"
)
const(
	masterURL string = ""
	kubeconfig string = "" //"/home/arnob/.kube/config"
)
const(
	DatabaseName string = "testing"
	CollectionName string = "numbers"
)


func main()  {
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
		return
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
		return
	}
	versionedClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building versioned clientset: %s", err.Error())
		return
	}
	fmt.Println("All clients has been caught successfully.")

	item , err := versionedClient.KubedbV1alpha2().MongoDBs(mongoDBNamespace).Get(context.Background(), mongoDBName, metav1.GetOptions{})
	if err != nil {
		klog.Fatalf("Error when getting the mongoDB object. %s", err.Error())
		return
	}
	fmt.Println("MongoDB object found")


	mongoCtx, cancel := context.WithTimeout(context.Background(), 15 * time.Second)
	defer cancel()

	mongoClient, err := dbclient.NewKubeDBClientBuilder( item, kubeClient).WithContext(mongoCtx).GetMongoClient()
	// WithPod("mgo-rs-0") deleted
	// if we dont specify the pod our dbClient will automatically find the master, otherwise not.
	if err != nil{
		klog.Fatalf("Running MongoDBClient failed. %s", err.Error())
		return
	}
	fmt.Println("GetMongoClient ran successfully")
	defer mongoClient.Close()






	// *************************** Doing some query now ****************************************


	//To insert a document
	ctx:= context.TODO()
	dbList , err := mongoClient.ListDatabaseNames(ctx, &options.ListDatabasesOptions{})
	if err != nil{
		klog.Fatalf("Can't list the Databases. %s", err.Error())
		return
	}
	fmt.Println("Listed databases are :", dbList)
	collection := mongoClient.Database(DatabaseName).Collection(CollectionName)
	res, err := collection.InsertOne(ctx, bson.D{{"name", "pi"}, {"value", 3.14159}})
	if err != nil{
		klog.Fatalf("Can't insert data to the collection %s, %s", CollectionName, err.Error())
		return
	}
	id := res.InsertedID
	fmt.Println("inserted id = ", id)


	// Finding a document
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		klog.Fatalf( "Error occurred when finding in the collection %s,  %s", CollectionName, err.Error())
		return
	}
	defer cur.Close(ctx)
	// Looping through the cursor
	for cur.Next(ctx) {
		var result bson.D
		err := cur.Decode(&result)
		if err != nil {
			klog.Fatalf("Can't decode the object. %s", err.Error())
			return
		}
		fmt.Println(result)
	}
	if err := cur.Err(); err != nil {
		klog.Fatalf("Error on the cursor %s",err.Error())
		return
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
		klog.Fatalf("Error when decoding %s", err.Error())
	}else{
		fmt.Println("Single instance retrieved from collection -> ", result)
	}
}

