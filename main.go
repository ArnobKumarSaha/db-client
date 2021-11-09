package main

import (
	"context"
	"fmt"
	"github.com/Arnobkumarsaha/mongotry/dbclient"
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
	kubeconfig string = "/home/arnob/.kube/config"
)



func main()  {
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		fmt.Println("Error building kubernetes clientset")
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}
	versionedClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		fmt.Println("Error building versioned clientset")
		klog.Fatalf("Error building versioned clientset: %s", err.Error())
	}

	/*
	// sample controller approach
	factory := commonInformer.NewSharedInformerFactory(versionedClient, time.Second *3)
	mongoInformer := factory.Kubedb().V1alpha2().MongoDBs()
	mongoLister := mongoInformer.Lister()

	item, errr := mongoLister.MongoDBs(mongoDBNamespace).Get(mongoDBName)
	fmt.Println("item = ", item)
	if errr != nil{
		fmt.Println("Error in lister.Get()")
	}else{
		fmt.Println("Whooo Got the item", item)
	}*/

	item , err := versionedClient.KubedbV1alpha2().MongoDBs(mongoDBNamespace).Get(context.Background(), mongoDBName, metav1.GetOptions{})
	if err != nil {
		fmt.Println("Error in direct appraoch")
	}else{
		fmt.Println("item found")
	}
	mongoCtx, cancel := context.WithTimeout(context.Background(), 15 * time.Second)
	defer cancel()


	mongoClient, err := dbclient.NewKubeDBClientBuilder( item, kubeClient).WithContext(mongoCtx).WithPod("mgo-rs-0").GetMongoClient()
	if err != nil{
		fmt.Println(err)
	}else{
		fmt.Println("MongoClient is being printed.")
	}

	defer mongoClient.Close()






	// *******************************************************************


	// To insert a document
	//ctx:= context.TODO()
	//xy , _ := mongoClient.ListDatabaseNames(ctx, &options.ListDatabasesOptions{})
	//fmt.Println("databases : ", xy)
	//collection := mongoClient.Database("testing").Collection("numbers")
	//
	////ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	////defer cancel()
	//res, err := collection.InsertOne(ctx, bson.D{{"name", "pi"}, {"value", 3.14159}})
	//id := res.InsertedID
	//fmt.Println("insert id = ", id)
	//
	//
	//cur, err := collection.Find(ctx, bson.D{})
	//if err != nil { log.Fatal(err) }
	//defer cur.Close(ctx)
	//for cur.Next(ctx) {
	//	var result bson.D
	//	err := cur.Decode(&result)
	//	if err != nil { log.Fatal(err) }
	//	fmt.Println(result)
	//}
	//if err := cur.Err(); err != nil {
	//	log.Fatal(err)
	//}
	//
	//// single Instance
	//var result struct {
	//	Value float64
	//}
	//filter := bson.D{{"name", "pi"}}
	////ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	////defer cancel()
	//err = collection.FindOne(ctx, filter).Decode(&result)
	//if err == mongo.ErrNoDocuments {
	//	// Do something when no record was found
	//	fmt.Println("record does not exist")
	//} else if err != nil {
	//	log.Fatal(err)
	//}else{
	//	fmt.Println("Single instance retrieved from collection -> ", result)
	//}
}

