/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Free Trial License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Free-Trial-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package schema

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	"kubedb.dev/apimachinery/client/clientset/versioned"
	dbclient "kubedb.dev/db-client-go/mongodb"
	schemav1alpha1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MongoDBDatabaseReconciler reconciles a MongoDBDatabase object
type MongoDBDatabaseReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=schema.kubedb.com,resources=mongodbdatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=schema.kubedb.com,resources=mongodbdatabases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=schema.kubedb.com,resources=mongodbdatabases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MongoDBDatabase object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *MongoDBDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	//_ = log.FromContext(ctx)
	log := r.Log.WithValues("mongodbdatabases", req.NamespacedName)

	// TODO(user): your logic here
	fmt.Println("Yeeh")
	kubeClient, versionedClient, err := GetTheClients()
	if err != nil {
		log.Error(err, "Unable to make the Clients")
		return ctrl.Result{}, err
	}
	log.Info("Clinets created successfully.")
	item, err := GetMongoObject(versionedClient)
	if err != nil {
		log.Error(err, "Unable to get the Mongo Object")
		return ctrl.Result{}, err
	}
	mongoCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	mongoClient, err := GetMongoClient(mongoCtx, kubeClient, item)
	if err != nil {
		log.Error(err, "Cant create MongoClient")
		return ctrl.Result{}, err
	}
	defer mongoClient.Close()

	var db schemav1alpha1.MongoDBDatabase
	if err := r.Get(ctx, req.NamespacedName, &db); err != nil{
		log.Error(err, "")
		return ctrl.Result{}, err
	}
	crudCtx := context.TODO()
	exists := checkIfExists(crudCtx, mongoClient, db.Spec.DatabaseName, db.Spec.CollectionName)

	if !exists {
		InsertIntoDatabase(crudCtx, mongoClient)
	}
	FindInDatabase(crudCtx, mongoClient)
	FindSingleInstance(crudCtx, mongoClient)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schemav1alpha1.MongoDBDatabase{}).
		Complete(r)
}

const (
	masterURL  string = ""
	kubeconfig string = "" //"/home/arnob/.kube/config"
)
const (
	mongoDBName      string = "mgo-rs"
	mongoDBNamespace string = "demo"
	mongoDBVersion   string = "4.2.3"
)
var (
	DatabaseName   string //= "testing"
	CollectionName string //= "numbers"
)

func checkIfExists(ctx context.Context, mongoClient *dbclient.Client, dbName, collName string ) bool {
	DatabaseName = dbName
	CollectionName = collName
	lst ,err:= ListDatabases(ctx, mongoClient)
	if err != nil{
		fmt.Println(err)
		return false
	}
	for i:= 0; i < len(lst); i+=1 {
		if lst[i] == dbName{
			collections, err := mongoClient.Database(dbName).ListCollectionNames(ctx, bson.D{})
			if err != nil{
				fmt.Println(err)
				return false
			}
			for j:=0; j<len(collections); j+=1{
				if collections[j] == CollectionName{
					return true
				}
			}
			return false
		}
	}
	return false
}

func FindSingleInstance(ctx context.Context, mongoClient *dbclient.Client) {
	collection := mongoClient.Database(DatabaseName).Collection(CollectionName)
	var result struct {
		Value float64
	}
	filter := bson.D{primitive.E{Key: "name", Value: "pi"}}
	err := collection.FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		// Do something when no record was found
		fmt.Println("record does not exist")
	} else if err != nil {
		klog.Fatalf("Error when decoding %s", err.Error())
	} else {
		fmt.Println("Single instance retrieved from collection -> ", result)
	}
}

func FindInDatabase(ctx context.Context, mongoClient *dbclient.Client) {
	collection := mongoClient.Database(DatabaseName).Collection(CollectionName)
	cur, err := collection.Find(ctx, bson.D{})
	if err != nil {
		klog.Fatalf("Error occurred when finding in the collection %s,  %s", CollectionName, err.Error())
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
		klog.Fatalf("Error on the cursor %s", err.Error())
		return
	}
}

func InsertIntoDatabase(ctx context.Context, mongoClient *dbclient.Client) {
	collection := mongoClient.Database(DatabaseName).Collection(CollectionName)
	res, err := collection.InsertOne(ctx, bson.D{primitive.E{Key: "name", Value: "pi"}, primitive.E{Key: "value", Value: 3.14159}})
	if err != nil {
		klog.Fatalf("Can't insert data to the collection %s, %s", CollectionName, err.Error())
		return
	}
	id := res.InsertedID
	fmt.Println("inserted id = ", id)
}

func ListDatabases(ctx context.Context, mongoClient *dbclient.Client) ([]string, error){
	dbList, err := mongoClient.ListDatabaseNames(ctx, &options.ListDatabasesOptions{})
	if err != nil {
		klog.Fatalf("Can't list the Databases. %s", err.Error())
		return nil, err
	}
	fmt.Println("Listed databases are :", dbList)
	return dbList, nil
}

func GetMongoClient(mongoCtx context.Context, kubeClient client.Client, item *v1alpha2.MongoDB) (*dbclient.Client, error) {

	mongoClient, err := dbclient.NewKubeDBClientBuilder(kubeClient, item).WithContext(mongoCtx).GetMongoClient()
	if err != nil {
		klog.Fatalf("Running MongoDBClient failed. %s", err.Error())
		return nil, err
	}
	fmt.Println("GetMongoClient ran successfully")
	return mongoClient, nil
}

func GetMongoObject(versionedClient *versioned.Clientset) (*v1alpha2.MongoDB, error) {
	item, err := versionedClient.KubedbV1alpha2().MongoDBs(mongoDBNamespace).Get(context.Background(), mongoDBName, metav1.GetOptions{})
	if err != nil {
		klog.Fatalf("Error when getting the mongoDB object. %s", err.Error())
		return nil, err
	}
	fmt.Println("MongoDB object found")
	return item, nil
}

func GetTheClients() (client.Client, *versioned.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
		return nil, nil, err
	}
	//kubeClient, err := kubernetes.NewForConfig(cfg)
	kubeClient, err := client.New(cfg, client.Options{})
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
		return nil, nil, err
	}
	versionedClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building versioned clientset: %s", err.Error())
		return nil, nil, err
	}
	fmt.Println("All clients has been caught successfully.")
	return kubeClient, versionedClient, nil
}
