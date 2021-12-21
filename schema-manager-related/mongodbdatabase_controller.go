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
	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
	apiv1_util "kmodules.xyz/client-go/api/v1"
	clientutil "kmodules.xyz/client-go/client"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	appcat "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	offshoot_v1 "kmodules.xyz/offshoot-api/api/v1"
	kd_catalog "kubedb.dev/apimachinery/apis/catalog/v1alpha1"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	dbClient "kubedb.dev/db-client-go/mongodb"
	schemav1alpha1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	kvm_engine "kubevault.dev/apimachinery/apis/engine/v1alpha1"
	kvm_server "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	repository "stash.appscode.dev/apimachinery/apis/stash/v1alpha1"
	stash "stash.appscode.dev/apimachinery/apis/stash/v1beta1"
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
	log := r.Log.WithValues("mongodbdatabase", req.NamespacedName)

	// First Get the actual CRD object of type MongoDBDatabase
	var obj = &schemav1alpha1.MongoDBDatabase{}
	if err := r.Client.Get(ctx, req.NamespacedName, obj); err != nil {
		log.Error(nil, "unable to fetch mongodb Database", req.Name, req.Namespace)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Getting The Mongo Server object
	var mongo kdm.MongoDB
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.DatabaseRef.Namespace,
		Name:      obj.Spec.DatabaseRef.Name,
	}, &mongo)
	if err != nil {
		log.Error(err, "Unable to get the Mongo Object")
		return ctrl.Result{}, err
	}

	// Getting The Vault Server object
	var vault kvm_server.VaultServer
	err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.VaultRef.Namespace,
		Name:      obj.Spec.VaultRef.Name,
	}, &vault)
	if err != nil {
		log.Error(err, "Unable to get the Vault server Object")
		return ctrl.Result{}, err
	}
	log.Info("Mongo Object and VaultServer object both have been found.")

	// If any of MongoDB & VaultServer is not Ready, wait until Reconcile call
	if mongo.Status.Phase != kdm.DatabasePhaseReady || vault.Status.Phase != kvm_server.VaultServerPhaseReady {
		return ctrl.Result{}, nil
	}

	if goForward, err := r.ensureOrDeleteFinalizer(ctx, obj, mongo, log); !goForward {
		// goForward == false means, some error has been occurred or Finalizer has been removed. Error can be nil or !nil , doesn't matter
		return ctrl.Result{}, err
	}

	// runMongoSchema is the actual worker function. If any error occurs in this func, it immediately returns.
	// This style gives us the chance to update Status for MongoDBDatabase only once, not after each 'ensure' functions
	err = r.runMongoSchema(ctx, obj, mongo, log)

	phase := GetPhaseFromCondition(obj.Status.Conditions)
	if obj.Status.Phase != phase {
		obj.Status.Phase = phase
	}
	fmt.Println("============ In Reconcile ===============> ", obj.Status.Conditions)

	if _, _, err2 := PatchStatus(r.Client, &schemav1alpha1.MongoDBDatabase{
		ObjectMeta: obj.ObjectMeta,
	}, func(object client.Object, createOp bool) client.Object {
		db := object.(*schemav1alpha1.MongoDBDatabase)
		db = obj
		return db
	}); err2 != nil {
		log.Error(err2, "Error occurred when Patching the MongoDBDatabase's status")
		return ctrl.Result{}, err2
	}

	return ctrl.Result{}, err
}

func (r *MongoDBDatabaseReconciler) runMongoSchema(ctx context.Context, obj *schemav1alpha1.MongoDBDatabase, mongo kdm.MongoDB, log logr.Logger) error {
	// Ensuring the required resources one-by-one
	var fetchedSecretEngine *kvm_engine.SecretEngine
	var err error
	if fetchedSecretEngine, err = r.ensureSecretEngine(obj, log); err != nil {
		return err
	}
	var fetchedRole *kvm_engine.MongoDBRole
	if fetchedRole, err = r.ensureMongoDBRole(obj, log); err != nil {
		return err
	}
	if fetchedSecretEngine.Status.Phase != SecretEnginePhaseSuccess || fetchedRole.Status.Phase != kvm_engine.RolePhaseSuccess {
		return nil
	}
	/*if _, err = r.ensureServiceAccount(&obj, log); err != nil {
		return err
	}*/
	var fetchedAccessRequest *kvm_engine.SecretAccessRequest
	if fetchedAccessRequest, err = r.ensureSecretAccessRequest(obj, log); err != nil {
		return err
	}

	var singleCred v1.Secret
	if fetchedAccessRequest.Status.Phase != kvm_engine.RequestStatusPhaseApproved ||
		fetchedAccessRequest.Status.Secret == nil {
		return nil
	}
	err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: fetchedAccessRequest.Status.Secret.Namespace,
		Name:      fetchedAccessRequest.Status.Secret.Name,
	}, &singleCred)
	if err != nil {
		log.Error(err, "Unable to Get the required secret")
		return err
	}
	// if user say for restoring
	if obj.Spec.Restore != nil {
		if err := r.copyRepositoryAndSecret(ctx, obj, log); err != nil {
			return err
		}
		if err := r.ensureAppbinding(ctx, obj, log); err != nil {
			return err
		}
		if _, err := r.ensureRestoreSession(ctx, obj, mongo, log); err != nil {
			log.Error(err, "Error occurred when ensuring the restore session")
			return err
		}
		return nil
	}
	if obj.Spec.Init.Initialized {
		return nil
	}
	// We are here means , user didn't say for restoring. And initialized is still false
	var job batchv1.Job
	err = r.Get(ctx, types.NamespacedName{
		Namespace: obj.Namespace,
		Name:      getMongoInitJobName(obj.GetName()),
	}, &job)
	if errors.IsNotFound(err) {
		// so job is not create yet, create it
		_, err = r.makeTheJob(ctx, obj, mongo, singleCred, log)
		if err != nil {
			return err
		}
	} else if err != nil {
		// some serious error occurred
		return err
	} else { // no error
		if job.Status.CompletionTime != nil {
			obj.Spec.Init.Initialized = true
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schemav1alpha1.MongoDBDatabase{}).
		Owns(&kvm_engine.SecretAccessRequest{}).
		Owns(&kvm_engine.SecretEngine{}).
		Owns(&batchv1.Job{}).
		Watches(&source.Kind{Type: &kdm.MongoDB{}}, handler.EnqueueRequestsFromMapFunc(r.getHandlerFuncForMongoDB())).
		Watches(&source.Kind{Type: &kvm_server.VaultServer{}}, handler.EnqueueRequestsFromMapFunc(r.getHandlerFuncForVaultServer())).
		Complete(r)
}

func (r *MongoDBDatabaseReconciler) getHandlerFuncForMongoDB() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		obj := object.(*kdm.MongoDB)
		var schemas schemav1alpha1.MongoDBDatabaseList
		var reqs []reconcile.Request

		// Listing from all namespaces
		err := r.Client.List(context.TODO(), &schemas, &client.ListOptions{})
		if err != nil {
			return reqs
		}
		for _, schema := range schemas.Items {
			if schema.Spec.DatabaseRef.Name == obj.Name && schema.Spec.DatabaseRef.Namespace == obj.Namespace {
				// Yes , We have got the required MongoDBDatabase object
				reqs = append(reqs, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      schema.Name,
						Namespace: schema.Namespace,
					},
				})
			}
		}
		return reqs
	}
}

func (r *MongoDBDatabaseReconciler) getHandlerFuncForVaultServer() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		obj := object.(*kvm_server.VaultServer)
		var schemas schemav1alpha1.MongoDBDatabaseList
		var arr []reconcile.Request

		// Listing from all namespaces
		err := r.Client.List(context.TODO(), &schemas, &client.ListOptions{})
		if err != nil {
			return arr
		}
		for _, schema := range schemas.Items {
			if schema.Spec.VaultRef.Name == obj.Name && schema.Spec.VaultRef.Namespace == obj.Namespace {
				// Yes , We have got the required MongoDBDatabase object
				arr = append(arr, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      schema.Name,
						Namespace: schema.Namespace,
					},
				})
			}
		}
		return arr
	}
}

func (r *MongoDBDatabaseReconciler) doExternalThingsBeforeDelete(ctx context.Context, obj *schemav1alpha1.MongoDBDatabase, mongo *kdm.MongoDB, log logr.Logger) error {
	// delete any external resources associated with the obj Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object. our finalizer is present, so lets handle any external dependency
	mongoClient, err := dbClient.NewKubeDBClientBuilder(r.Client, mongo).WithContext(ctx).GetMongoClient()
	defer mongoClient.Close()
	if err != nil {
		log.Error(err, "Unable to run GetMongoClient() function")
		return err
	}
	cursor := mongoClient.Database(MongoDatabaseNameForEntry).Collection(MongoCollectionNameForEntry).FindOne(ctx, bson.D{
		{Key: obj.Spec.DatabaseSchema.Name, Value: true},
	})
	if cursor.Err() == nil {
		// entry found, so drop the database & Delete entry from kube-system database
		err = mongoClient.Database(obj.Spec.DatabaseSchema.Name).Drop(ctx)
		if err != nil {
			log.Error(err, "Can't drop the database")
			return err
		}
		_, err = mongoClient.Database(MongoDatabaseNameForEntry).Collection(MongoCollectionNameForEntry).DeleteOne(ctx, bson.D{
			{Key: obj.Spec.DatabaseSchema.Name, Value: true},
		})
		if err != nil {
			log.Error(err, "Error occurred when delete the entry form kube-system database")
			return err
		}
	}
	return nil
}

func (r *MongoDBDatabaseReconciler) ensureEntryIntoDatabase(ctx context.Context, mongo kdm.MongoDB, name string, log logr.Logger) error {
	mongoClient, err := dbClient.NewKubeDBClientBuilder(r.Client, &mongo).WithContext(ctx).GetMongoClient()
	if err != nil {
		klog.Fatalf("Running MongoDBClient failed. %s", err.Error())
		return err
	}
	defer mongoClient.Close()

	// For checking, if the Database named 'obj.Spec.DatabaseSchema.Name' already exist or not
	_, err = mongoClient.Database(name).ListCollectionNames(ctx, bson.D{}) // collectionList
	if err != nil {
		log.Error(err, "Error occurred when listing theh Collection names of %v database", MongoDatabaseNameForEntry)
		return err
	}
	// We are here means, Database doesn't already exist
	valTrue := true
	_, err = mongoClient.Database(MongoDatabaseNameForEntry).Collection(MongoCollectionNameForEntry).UpdateOne(
		ctx,
		bson.M{name: true},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: name, Value: true},
			}},
		},
		&options.UpdateOptions{
			Upsert: &valTrue,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to write on database with error: %s", err.Error())
	}
	return nil
}

func (r *MongoDBDatabaseReconciler) ensureOrDeleteFinalizer(ctx context.Context, obj *schemav1alpha1.MongoDBDatabase, mongo kdm.MongoDB, log logr.Logger) (bool, error) {
	finalizerName := schemav1alpha1.GroupVersion.Group
	// examine DeletionTimestamp to determine if object is under deletion
	if obj.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so Add entry to our Database & register our finalizer
		if !containsString(obj.GetFinalizers(), finalizerName) {
			err := r.ensureEntryIntoDatabase(ctx, mongo, obj.Spec.DatabaseSchema.Name, log)
			if err != nil {
				log.Error(err, "Error occurred when updating into kube-system database using mongoClient")
				return false, err
			}
			controllerutil.AddFinalizer(obj, finalizerName)
			if err := r.Update(ctx, obj); err != nil {
				log.Error(err, "Cant update MongoDBDatabase object when adding finalizer")
				return false, err
			}
		}
	} else if obj.Spec.DeletionPolicy == schemav1alpha1.DeletionPolicyDelete {
		// The object is assigned for Deletion
		if containsString(obj.GetFinalizers(), finalizerName) {
			if err := r.doExternalThingsBeforeDelete(ctx, obj, &mongo, log); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return false, err
			}
			log.Info("Finalizers will be removed now.")
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(obj, finalizerName)
			if err := r.Update(ctx, obj); err != nil {
				log.Error(err, "Cant update MongoDBDatabase object when removing finalizer")
				return false, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return false, nil
	}
	return true, nil
}

// Create or Patch the Secret Engine
func (r *MongoDBDatabaseReconciler) ensureSecretEngine(obj *schemav1alpha1.MongoDBDatabase, log logr.Logger) (*kvm_engine.SecretEngine, error) {
	fetchedSecretEngine, _, err := clientutil.CreateOrPatch(r.Client, &kvm_engine.SecretEngine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getMongoSecretEngineName(obj.Name),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		se := object.(*kvm_engine.SecretEngine)

		se.Spec.VaultRef.Name = obj.Spec.VaultRef.Name
		se.Spec.VaultRef.Namespace = obj.Spec.VaultRef.Namespace

		se.Spec.SecretEngineConfiguration.MongoDB = &kvm_engine.MongoDBConfiguration{
			DatabaseRef: v1alpha1.AppReference{
				Namespace: obj.Spec.DatabaseRef.Namespace,
				Name:      obj.Spec.DatabaseRef.Name,
			},
		}
		se.Spec.SecretEngineConfiguration.MongoDB.PluginName = kvm_engine.DefaultMongoDBDatabasePlugin

		if createOp {
			core_util.EnsureOwnerReference(&se.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return se
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required Secret Engine")
		return nil, err
	}
	r.updateStatusForSecretEngine(obj, fetchedSecretEngine.(*kvm_engine.SecretEngine))
	return fetchedSecretEngine.(*kvm_engine.SecretEngine), nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForSecretEngine(obj *schemav1alpha1.MongoDBDatabase, se *kvm_engine.SecretEngine) {
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: se.Namespace,
		Name:      se.Name,
	}, se)
	if err != nil {
		SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionSecretEngineReady, v1.ConditionFalse, schemav1alpha1.SchemaDatabaseReason(err.Error()), "Secret Engine is not created yet")
		return
	}
	if !CheckSecretEngineConditions(se) {
		SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionSecretEngineReady, v1.ConditionFalse, schemav1alpha1.SchemaDatabaseReasonSecretEngineReady, "Secret Engine is provisioning")
		return
	}
	SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionSecretEngineReady, v1.ConditionTrue, schemav1alpha1.SchemaDatabaseReasonSecretEngineReady, "Secret Engine is ready")
}

// Create or Patch the MongoDBRole
func (r *MongoDBDatabaseReconciler) ensureMongoDBRole(obj *schemav1alpha1.MongoDBDatabase, log logr.Logger) (*kvm_engine.MongoDBRole, error) {
	fetchedMongoDbRole, _, err := clientutil.CreateOrPatch(r.Client, &kvm_engine.MongoDBRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getMongoAdminRoleName(obj.GetName()),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		mr := object.(*kvm_engine.MongoDBRole)
		mr.Spec.SecretEngineRef.Name = getMongoSecretEngineName(obj.GetName())
		generatedString := fmt.Sprintf("{ \"db\": \"%s\", \"roles\": [{ \"role\": \"dbAdmin\" }] }", obj.Spec.DatabaseSchema.Name)
		mr.Spec.CreationStatements = []string{generatedString}
		mr.Spec.RevocationStatements = []string{generatedString}
		if createOp {
			core_util.EnsureOwnerReference(&mr.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return mr
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required MongoDbRole")
		return nil, err
	}
	r.updateStatusForMongoDbRole(obj, fetchedMongoDbRole.(*kvm_engine.MongoDBRole))
	return fetchedMongoDbRole.(*kvm_engine.MongoDBRole), nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForMongoDbRole(obj *schemav1alpha1.MongoDBDatabase, mr *kvm_engine.MongoDBRole) {
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: mr.Namespace,
		Name:      mr.Name,
	}, mr)
	if err != nil {
		SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionMongoDBRoleReady, v1.ConditionFalse, schemav1alpha1.SchemaDatabaseReason(err.Error()), "MongoDB Role is not created yet")
		return
	}
	if !CheckMongoDBRoleConditions(mr) {
		SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionMongoDBRoleReady, v1.ConditionFalse, schemav1alpha1.SchemaDatabaseReasonMongoDBRoleReady, "MongoDB Role is being created")
		return
	}
	SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionMongoDBRoleReady, v1.ConditionTrue, schemav1alpha1.SchemaDatabaseReasonMongoDBRoleReady, "MongoDB Role is ready")
}

// Create or Patch the service account
/*
func (r *MongoDBDatabaseReconciler) ensureServiceAccount(obj *schemav1alpha1.MongoDBDatabase, log logr.Logger) (*v1.ServiceAccount, error) {
	fetchedServiceAccount, _, err := clientutil.CreateOrPatch(r.Client, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getMongoAdminServiceAccountName(obj.GetName()),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		sa := object.(*v1.ServiceAccount)
		if createOp {
			core_util.EnsureOwnerReference(&sa.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return sa
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the Service Account.")
		return nil, err
	}
	return fetchedServiceAccount.(*v1.ServiceAccount), nil
}*/

// Create or Patch the secret Access Request
func (r *MongoDBDatabaseReconciler) ensureSecretAccessRequest(obj *schemav1alpha1.MongoDBDatabase, log logr.Logger) (*kvm_engine.SecretAccessRequest, error) {
	fetchedAccessRequest, _, err := clientutil.CreateOrPatch(r.Client, &kvm_engine.SecretAccessRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getMongoAdminSecretAccessRequestName(obj.GetName()),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		sar := object.(*kvm_engine.SecretAccessRequest)

		sar.Spec.RoleRef.Kind = kvm_engine.ResourceKindMongoDBRole
		sar.Spec.RoleRef.Name = getMongoAdminRoleName(obj.GetName())

		var accessRequestFound = false
		for i := 0; i < len(sar.Spec.Subjects); i++ {
			sub := sar.Spec.Subjects[i]
			if sub.Name == getMongoAdminServiceAccountName(obj.GetName()) {
				accessRequestFound = true
			}
		}
		if !accessRequestFound {
			sar.Spec.Subjects = append(sar.Spec.Subjects, rbac.Subject{
				Kind:      rbac.ServiceAccountKind,
				Name:      getMongoAdminServiceAccountName(obj.GetName()),
				Namespace: obj.Namespace,
			})
		}
		if createOp {
			core_util.EnsureOwnerReference(&sar.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return sar
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the Secret Access request.")
		return nil, err
	}
	if !obj.Spec.AutoApproval {
		return fetchedAccessRequest.(*kvm_engine.SecretAccessRequest), nil
	}
	// user sets auto approval to
	sarForReturn := fetchedAccessRequest.(*kvm_engine.SecretAccessRequest)
	sarForReturn.Status.Conditions = apiv1_util.SetCondition(sarForReturn.Status.Conditions, apiv1_util.Condition{
		Type:    apiv1_util.ConditionRequestApproved,
		Status:  v1.ConditionTrue,
		Reason:  "ApprovedBySchemaManager",
		Message: "This was approved by: KubeDb SchemaManager",
	})
	_, _, err = PatchStatus(r.Client, &kvm_engine.SecretAccessRequest{
		ObjectMeta: sarForReturn.ObjectMeta,
	}, func(object client.Object, createOp bool) client.Object {
		insideSar := object.(*kvm_engine.SecretAccessRequest)
		insideSar = sarForReturn
		return insideSar
	})
	if err != nil {
		log.Error(err, "Unable to Patch the Secret Access request status.")
		return nil, err
	}
	r.updateStatusForSecretAccessRequest(obj, sarForReturn)
	return sarForReturn, nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForSecretAccessRequest(obj *schemav1alpha1.MongoDBDatabase, sar *kvm_engine.SecretAccessRequest) {
	fmt.Println("//////////////// before get //////////////////////", obj.Status.Conditions, "------------------------------", sar.Status.Conditions)
	// Todo:
	//  - if SchemaDatabaseConditionSecretAccessRequestReady: -> let's say not needeed
	//    - return
	//  - if SAR -> Approved & Available:
	//     - Set SchemaDatabaseConditionSecretAccessRequestReady = true
	//  - Processing -> Set relevant condition

	var newSar kvm_engine.SecretAccessRequest
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: sar.Namespace,
		Name:      sar.Name,
	}, &newSar)
	if err != nil {
		SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionSecretAccessRequestReady, v1.ConditionFalse, schemav1alpha1.SchemaDatabaseReason(err.Error()), "Secret access request is not created yet")
		return
	}
	fmt.Println("?????????????? after ?????????????????", obj.Status.Conditions, "------------------------------", newSar.Status.Conditions)
	if !CheckSecretAccessRequestConditions(&newSar) {
		fmt.Println("******************** something is not ready ********************************************")
		fmt.Println(newSar.Status.Conditions)
		SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionSecretAccessRequestReady, v1.ConditionFalse, schemav1alpha1.SchemaDatabaseReasonSecretAccessRequestReady, "Secret access request is being created")
		return
	}
	SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionSecretAccessRequestReady, v1.ConditionTrue, schemav1alpha1.SchemaDatabaseReasonSecretAccessRequestReady, "Secret access request is succeeded")
}

// Now Make the Job
func (r *MongoDBDatabaseReconciler) makeTheJob(ctx context.Context, obj *schemav1alpha1.MongoDBDatabase, mongo kdm.MongoDB, singleCred v1.Secret, log logr.Logger) (*batchv1.Job, error) {
	// if the user doesn't provide init script, then no need to create the job
	if obj.Spec.Init == nil || obj.Spec.Init.Script == nil {
		return nil, nil
	}
	// If the user doesn't provide podTemplate, we'll fill it with default value
	isPodTemplateGiven := obj.Spec.Init.PodTemplate != nil
	var givenPodSpec offshoot_v1.PodSpec
	givenScript := obj.Spec.Init.Script

	// TLS related part
	var sslArgs string
	if mongo.Spec.SSLMode != kdm.SSLModeDisabled {
		var err error
		sslArgs, err = r.getSSLArgsForJob(ctx, mongo, sslArgs, log)
		if err != nil {
			return nil, err
		}
	}

	envList := []v1.EnvVar{
		{
			Name: "MONGODB_USERNAME",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: singleCred.Name,
					},
					Key: "username",
				},
			},
		},
		{
			Name: "MONGODB_PASSWORD",
			ValueFrom: &v1.EnvVarSource{
				SecretKeyRef: &v1.SecretKeySelector{
					LocalObjectReference: v1.LocalObjectReference{
						Name: singleCred.Name,
					},
					Key: "password",
				},
			},
		},
		{
			Name:  "MONGODB_DATABASE_NAME",
			Value: obj.Spec.DatabaseSchema.Name,
		},
	}
	argList := []string{
		fmt.Sprintf(`mongo --host=%v.%v.svc.cluster.local %v --authenticationDatabase=$MONGODB_DATABASE_NAME --username=$MONGODB_USERNAME --password=$MONGODB_PASSWORD < %v/%v; `,
			mongo.ServiceName(), mongo.Namespace, sslArgs, MongoInitScriptPath, InitScriptName),
	}

	if isPodTemplateGiven {
		givenPodSpec = obj.Spec.Init.PodTemplate.Spec
		envList = core_util.UpsertEnvVars(envList, givenPodSpec.Env...)
		argList = meta_util.UpsertArgumentList(givenPodSpec.Args, argList)
	}

	var volumeMounts []v1.VolumeMount
	volumeMounts = core_util.UpsertVolumeMount(volumeMounts, []v1.VolumeMount{
		{
			Name:      getMongoInitVolumeNameForPod(obj.GetName()),
			MountPath: MongoInitScriptPath,
		},
	}...)

	var volumes []v1.Volume
	volumes = core_util.UpsertVolume(volumes, v1.Volume{
		Name:         getMongoInitVolumeNameForPod(obj.GetName()),
		VolumeSource: givenScript.VolumeSource,
	})

	// now actually create it or patch
	createdJob, vt, err := clientutil.CreateOrPatch(r.Client, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getMongoInitJobName(obj.GetName()),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		job := object.(*batchv1.Job)

		job.Spec.Template.Spec.Volumes = core_util.UpsertVolume(job.Spec.Template.Spec.Volumes, volumes...)

		versionImage, err := func(version string) (string, error) {
			var ver kd_catalog.MongoDBVersion
			err := r.Client.Get(ctx, types.NamespacedName{
				Name: version,
			}, &ver)
			if err != nil {
				log.Error(err, "Error when getting the MongoVersion object")
				return "", err
			}
			return ver.Spec.DB.Image, nil
		}(mongo.Spec.Version)
		if err != nil {
			return nil
		}
		containers := []v1.Container{
			{
				Name:            getMongoInitContainerNameForPod(obj.GetName()),
				Image:           versionImage,
				Env:             envList,
				Command:         []string{"/bin/sh", "-c"},
				Args:            argList,
				ImagePullPolicy: v1.PullAlways,
				VolumeMounts:    volumeMounts,
			},
		}
		if isPodTemplateGiven {
			setDefaultOptionsForJobContainer(&containers[0], givenPodSpec)
			job.Spec.Template.Spec.ServiceAccountName = givenPodSpec.ServiceAccountName
			job.Spec.Template.Spec.NodeSelector = givenPodSpec.NodeSelector
			job.Spec.Template.Spec.Affinity = givenPodSpec.Affinity
			job.Spec.Template.Spec.SchedulerName = givenPodSpec.SchedulerName
			job.Spec.Template.Spec.Tolerations = givenPodSpec.Tolerations
			job.Spec.Template.Spec.ImagePullSecrets = givenPodSpec.ImagePullSecrets
			job.Spec.Template.Spec.PriorityClassName = givenPodSpec.PriorityClassName
			job.Spec.Template.Spec.Priority = givenPodSpec.Priority
			job.Spec.Template.Spec.HostNetwork = givenPodSpec.HostNetwork
			job.Spec.Template.Spec.HostPID = givenPodSpec.HostPID
			job.Spec.Template.Spec.HostIPC = givenPodSpec.HostIPC
			job.Spec.Template.Spec.ShareProcessNamespace = givenPodSpec.ShareProcessNamespace
			job.Spec.Template.Spec.SecurityContext = givenPodSpec.SecurityContext
			job.Spec.Template.Spec.DNSPolicy = givenPodSpec.DNSPolicy
			job.Spec.Template.Spec.DNSConfig = givenPodSpec.DNSConfig
		}
		job.Spec.Template.Spec.Containers = core_util.UpsertContainers(job.Spec.Template.Spec.Containers, containers)

		// InitContainers doesn't make any sense.
		job.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		job.Spec.BackoffLimit = func(i int32) *int32 { return &i }(5)

		if createOp {
			core_util.EnsureOwnerReference(&job.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return job
	})
	if vt == kutil.VerbCreated {
		log.Info("Job has been successfully created")
	}
	if err != nil {
		log.Error(err, "Can't create the job")
		return nil, err
	}
	r.updateStatusForInitJob(obj, createdJob.(*batchv1.Job))
	return createdJob.(*batchv1.Job), nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForInitJob(obj *schemav1alpha1.MongoDBDatabase, jb *batchv1.Job) {
	err := r.Get(context.TODO(), types.NamespacedName{
		Namespace: jb.Namespace,
		Name:      jb.Name,
	}, jb)
	if err != nil {
		SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionJobCompleted, v1.ConditionFalse, schemav1alpha1.SchemaDatabaseReason(err.Error()), "Job is not created yet")
		return
	}
	if !CheckJobConditions(jb) {
		SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionJobCompleted, v1.ConditionFalse, schemav1alpha1.SchemaDatabaseReasonJobCompleted, "Job is being created")
		return
	}
	SetCondition(obj, schemav1alpha1.SchemaDatabaseConditionJobCompleted, v1.ConditionTrue, schemav1alpha1.SchemaDatabaseReasonJobCompleted, "Job is completed")
}

func setDefaultOptionsForJobContainer(container *v1.Container, givenPodSpec offshoot_v1.PodSpec) {
	container.Resources = givenPodSpec.Resources
	container.LivenessProbe = givenPodSpec.LivenessProbe
	container.ReadinessProbe = givenPodSpec.ReadinessProbe
	container.Lifecycle = givenPodSpec.Lifecycle
	container.SecurityContext = givenPodSpec.ContainerSecurityContext
}

func (r *MongoDBDatabaseReconciler) getSSLArgsForJob(ctx context.Context, mongo kdm.MongoDB, sslArgs string, log logr.Logger) (string, error) {
	if mongo.Spec.TLS == nil {
		return "", fmt.Errorf("SSLMode in mongoDB object in enabled, but issuerRef is not given")
	}
	sslArgs = fmt.Sprintf("--tls --tlsCAFile %v/%v --tlsCertificateKeyFile %v/%v",
		kdm.MongoCertDirectory, kdm.TLSCACertFileName, kdm.MongoCertDirectory, kdm.MongoClientFileName)
	breakingVer, _ := semver.NewVersion("4.1")
	exceptionVer, _ := semver.NewVersion("4.1.4")
	var ver kd_catalog.MongoDBVersion
	err := r.Client.Get(ctx, types.NamespacedName{
		Name: mongo.Spec.Version,
	}, &ver)
	if err != nil {
		log.Error(err, "Error when getting the MongoVersion object")
		return "", err
	}
	currentVer, err := semver.NewVersion(ver.Spec.Version)
	if err != nil {
		log.Error(err, "Unable to parse MongoDB Version")
		return "", err
	}
	if currentVer.Equal(exceptionVer) {
		sslArgs = fmt.Sprintf("--tls --tlsCAFile=%v/%v --tlsPEMKeyFile=%v/%v", kdm.MongoCertDirectory, kdm.TLSCACertFileName, kdm.MongoCertDirectory, kdm.MongoClientFileName)
	} else if currentVer.LessThan(breakingVer) {
		sslArgs = fmt.Sprintf("--ssl --sslCAFile=%v/%v --sslPEMKeyFile=%v/%v", kdm.MongoCertDirectory, kdm.TLSCACertFileName, kdm.MongoCertDirectory, kdm.MongoClientFileName)
	}
	return sslArgs, nil
}

func (r *MongoDBDatabaseReconciler) copyRepositoryAndSecret(ctx context.Context, obj *schemav1alpha1.MongoDBDatabase, log logr.Logger) error {
	var repo repository.Repository
	err := r.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.Restore.Repository.Namespace,
		Name:      obj.Spec.Restore.Repository.Name,
	}, &repo)
	if err != nil {
		log.Error(err, "Can't get the Repository")
		return err
	}
	_, _, err = clientutil.CreateOrPatch(r.Client, &repository.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      repo.Name,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		rep := object.(*repository.Repository)
		rep.Spec = repo.Spec
		if createOp {
			core_util.EnsureOwnerReference(&rep.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return rep
	})
	if err != nil {
		log.Error(err, "Error occurred when createOrPatch called for the Repository")
		return err
	}

	var secret v1.Secret
	err = r.Get(ctx, types.NamespacedName{
		Namespace: repo.Namespace,
		Name:      repo.Spec.Backend.StorageSecretName,
	}, &secret)
	if err != nil {
		log.Error(err, "Can't get the secret")
		return err
	}
	_, _, err = clientutil.CreateOrPatch(r.Client, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		s := object.(*v1.Secret)
		s.Data = secret.Data
		if createOp {
			core_util.EnsureOwnerReference(&s.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return s
	})
	if err != nil {
		log.Error(err, "Error occurred when createOrPatch called for the secret")
		return err
	}
	return nil
}

func (r *MongoDBDatabaseReconciler) ensureAppbinding(ctx context.Context, obj *schemav1alpha1.MongoDBDatabase, log logr.Logger) error {
	/// Getting the root credential from db namespace to schema-manager namespace
	var authSecret v1.Secret
	err := r.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.DatabaseRef.Namespace,
		Name:      getAuthSecretName(obj.Spec.DatabaseRef.Name),
	}, &authSecret)
	if err != nil {
		log.Error(err, "Cant get the auth secret")
		return err
	}
	_, _, err = clientutil.CreateOrPatch(r.Client, &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authSecret.Name,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		s := object.(*v1.Secret)
		s.Data = authSecret.Data
		if createOp {
			core_util.EnsureOwnerReference(&s.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return s
	})

	var appbinding appcat.AppBinding
	err = r.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.DatabaseRef.Namespace,
		Name:      obj.Spec.DatabaseRef.Name,
	}, &appbinding)
	if err != nil {
		log.Error(err, "Cant get the appbinding named")
		return err
	}
	_, _, err = clientutil.CreateOrPatch(r.Client, &appcat.AppBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appbinding.Name,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		ab := object.(*appcat.AppBinding)
		ab.Spec = appbinding.Spec
		ab.Spec.ClientConfig.Service.Name = func(obj *schemav1alpha1.MongoDBDatabase) string {
			var mongo kdm.MongoDB
			err := r.Get(ctx, types.NamespacedName{
				Namespace: obj.Spec.DatabaseRef.Namespace,
				Name:      obj.Spec.DatabaseRef.Name,
			}, &mongo)
			if err != nil {
				log.Error(err, "Cant get the Mongo Object")
				return ""
			}
			svc := fmt.Sprintf("%v.%v.svc.cluster.local", mongo.Name, mongo.Namespace)
			return svc
		}(obj)
		if createOp {
			core_util.EnsureOwnerReference(&ab.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return ab
	})
	if err != nil {
		log.Error(err, "Error occurred when createOrPatch called for AppBinding")
		return err
	}
	return nil
}

// Create or Patch the RestoreSession
func (r *MongoDBDatabaseReconciler) ensureRestoreSession(ctx context.Context, obj *schemav1alpha1.MongoDBDatabase, mongo kdm.MongoDB, log logr.Logger) (*stash.RestoreSession, error) {
	// Getting the  MongoDBVersion object
	var mongoVer kd_catalog.MongoDBVersion
	err := r.Get(ctx, types.NamespacedName{
		Name: mongo.Spec.Version,
	}, &mongoVer)
	if err != nil {
		log.Error(err, "Cant get the MongoDBVersion ")
		return nil, err
	}

	fetchedRestoreSession, _, err := clientutil.CreateOrPatch(r.Client, &stash.RestoreSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getMongoRestoreSessionName(obj.GetName()),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		rs := object.(*stash.RestoreSession)

		rs.Spec.Task = stash.TaskRef{
			Name: mongoVer.Spec.Stash.Addon.RestoreTask.Name,
		}
		rs.Spec.Target = &stash.RestoreTarget{
			Ref: stash.TargetRef{
				APIVersion: appcat.SchemeGroupVersion.String(),
				Kind:       appcat.ResourceKindApp,
				Name:       mongo.Name,
			},
		}
		rs.Spec.Repository.Name = obj.Spec.Restore.Repository.Name
		rs.Spec.Target.Rules = []stash.Rule{
			{
				Snapshots: []string{
					obj.Spec.Restore.Snapshot,
				},
			},
		}

		if createOp {
			core_util.EnsureOwnerReference(&rs.ObjectMeta, metav1.NewControllerRef(obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return rs
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required Secret Engine")
		return nil, err
	}
	return fetchedRestoreSession.(*stash.RestoreSession), nil
}
