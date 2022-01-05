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
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"
	kutil "kmodules.xyz/client-go"
	apiv1util "kmodules.xyz/client-go/api/v1"
	clientutil "kmodules.xyz/client-go/client"
	coreutil "kmodules.xyz/client-go/core/v1"
	metautil "kmodules.xyz/client-go/meta"
	appcatutil "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	offshootv1util "kmodules.xyz/offshoot-api/api/v1"
	kd_catalog "kubedb.dev/apimachinery/apis/catalog/v1alpha1"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	dbClient "kubedb.dev/db-client-go/mongodb"
	smv1a1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
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
	// Get the actual CRD object of type MongoDBDatabase
	var obj smv1a1.MongoDBDatabase
	if err := r.Client.Get(ctx, req.NamespacedName, &obj); err != nil {
		log.Error(nil, "unable to fetch mongodb Database", req.Name, req.Namespace)
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	mongo, _, err, goForward := r.getMongoDBAndVaultServer(ctx, &obj, log)
	if err != nil || !goForward {
		return ctrl.Result{}, err
	}
	if goForward, err := r.ensureOrDeleteFinalizer(ctx, &obj, mongo, log); !goForward {
		// goForward == false means, some error has been occurred or Finalizer has been removed. Error can be nil or !nil , doesn't matter
		return ctrl.Result{}, err
	}

	// runMongoSchema is the actual worker function. If any error occurs in this func, it immediately returns.
	// This style gives us the chance to update Status for MongoDBDatabase only once, not after each 'ensure' functions
	err = r.runMongoSchema(ctx, &obj, mongo, log)

	if err2 := r.updateSchemaMongoDBDatabaseStatus(ctx, &obj, log, ""); err2 != nil {
		return ctrl.Result{}, err2
	}
	return ctrl.Result{}, err
}

func (r *MongoDBDatabaseReconciler) getMongoDBAndVaultServer(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger) (kdm.MongoDB, kvm_server.VaultServer, error, bool) {
	if obj.Status.Phase == "" {
		// updateMongoDBDatabaseStatus is called here to get "initializing" phase in MongoDBDatabase
		if err2 := r.updateSchemaMongoDBDatabaseStatus(ctx, obj, log, smv1a1.SchemaDatabasePhaseWaiting); err2 != nil {
			log.Error(err2, "Error occurred when updating the MongoDBDatabase status")
			return kdm.MongoDB{}, kvm_server.VaultServer{}, err2, false
		}
	}
	var mongo kdm.MongoDB
	var vault kvm_server.VaultServer
	// Getting The Mongo Server object
	err := r.Client.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.DatabaseRef.Namespace,
		Name:      obj.Spec.DatabaseRef.Name,
	}, &mongo)
	r.updateStatusForMongoDB(ctx, obj, &mongo)
	if err != nil {
		log.Error(err, "Unable to get the Mongo Object")
		return kdm.MongoDB{}, kvm_server.VaultServer{}, err, false
	}

	// Getting The Vault Server object
	err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.VaultRef.Namespace,
		Name:      obj.Spec.VaultRef.Name,
	}, &vault)
	r.updateStatusForVaultServer(ctx, obj, &vault)
	if err != nil {
		log.Error(err, "Unable to get the Vault server Object")
		return mongo, kvm_server.VaultServer{}, err, false
	}
	log.Info("Mongo Object and VaultServer object both have been found.")

	// If any of MongoDB & VaultServer is not Ready, wait until Reconcile call
	if mongo.Status.Phase != kdm.DatabasePhaseReady || vault.Status.Phase != kvm_server.VaultServerPhaseReady {
		return mongo, vault, nil, false
	}
	// They are ready, so reflect it in the MongoDBDatabase status
	if err2 := r.updateSchemaMongoDBDatabaseStatus(ctx, obj, log, ""); err2 != nil {
		log.Error(err2, "Error occurred when updating the MongoDBDatabase status")
		return mongo, vault, err2, false
	}
	return mongo, vault, err, true
}

func (r *MongoDBDatabaseReconciler) updateStatusForMongoDB(ctx context.Context, obj *smv1a1.MongoDBDatabase, mongo *kdm.MongoDB) {
	err := r.Get(ctx, types.NamespacedName{
		Namespace: mongo.Namespace,
		Name:      mongo.Name,
	}, mongo)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionDBReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "MongoDB is not created yet")
		return
	}
	if !CheckMongoDBConditions(mongo) {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionDBReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReasonDBReady, "MongoDB is provisioning")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionDBReady, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonDBReady, "MongoDB is ready")
}

func (r *MongoDBDatabaseReconciler) updateStatusForVaultServer(ctx context.Context, obj *smv1a1.MongoDBDatabase, vault *kvm_server.VaultServer) {
	err := r.Get(ctx, types.NamespacedName{
		Namespace: vault.Namespace,
		Name:      vault.Name,
	}, vault)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionVaultReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "VaultServer is not created yet")
		return
	}
	if !CheckVaultConditions(vault) {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionVaultReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReasonVaultReady, "VaultServer is provisioning")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionVaultReady, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonVaultReady, "VaultServer is ready")
}

/*
Ensuring the required resources one-by-one :  SecretEngine, MongoDbRole, ServiceAccount, SecretAccessRequest
It also handles the Init-Restore case, by calling corresponding methods
*/
func (r *MongoDBDatabaseReconciler) runMongoSchema(ctx context.Context, obj *smv1a1.MongoDBDatabase, mongo kdm.MongoDB, log logr.Logger) error {
	var fetchedSecretEngine *kvm_engine.SecretEngine
	var err error
	if fetchedSecretEngine, err = r.ensureSecretEngine(ctx, obj, log); err != nil {
		return err
	}
	var fetchedRole *kvm_engine.MongoDBRole
	if fetchedRole, err = r.ensureMongoDBRole(ctx, obj, log); err != nil {
		return err
	}
	if fetchedSecretEngine.Status.Phase != smv1a1.SecretEnginePhaseSuccess || fetchedRole.Status.Phase != kvm_engine.RolePhaseSuccess {
		return nil
	}
	if _, err = r.ensureServiceAccount(ctx, obj, log); err != nil {
		return err
	}
	var fetchedAccessRequest *kvm_engine.SecretAccessRequest
	if fetchedAccessRequest, err = r.ensureSecretAccessRequest(ctx, obj, log); err != nil {
		return err
	}

	var singleCred corev1.Secret
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
		res, err := r.ensureRestoreSession(ctx, obj, mongo, log)
		if err != nil {
			log.Error(err, "Error occurred when ensuring the restore session")
			return err
		}
		if (obj.Spec.Init == nil || !obj.Spec.Init.Initialized) && CheckRestoreSessionPhaseSucceed(res) {
			obj.Spec.Init = &smv1a1.InitSpec{
				Initialized: true,
			}
		}
		return nil
	}

	// as makeTheInitJob is only called once (hence updateStatusForInitJob also) if job is not created yet, We need to get the updated status for job
	r.updateStatusForInitJob(ctx, obj, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetMongoInitJobName(),
			Namespace: obj.Namespace,
		},
	})
	if obj.Spec.Init.Initialized {
		return nil
	}
	// We are here means , user didn't say for restoring. And initialized is still false
	var job batchv1.Job
	err = r.Get(ctx, types.NamespacedName{
		Namespace: obj.Namespace,
		Name:      obj.GetMongoInitJobName(),
	}, &job)
	if kerrors.IsNotFound(err) {
		// so job is not create yet, create it
		_, err = r.makeTheInitJob(ctx, obj, mongo, singleCred, log)
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

func (r *MongoDBDatabaseReconciler) updateSchemaMongoDBDatabaseStatus(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger, typ smv1a1.SchemaDatabasePhase) error {
	var phase smv1a1.SchemaDatabasePhase
	if typ == smv1a1.SchemaDatabasePhaseIgnored {
		phase = typ
	} else {
		phase = GetPhaseFromCondition(obj.Status.Conditions, obj.Spec.Restore != nil)
	}
	if obj.Status.Phase != phase {
		obj.Status.Phase = phase
	}

	if _, _, err2 := clientutil.PatchStatus(ctx, r.Client, &smv1a1.MongoDBDatabase{
		ObjectMeta: obj.ObjectMeta,
	}, func(object client.Object, createOp bool) client.Object {
		db := object.(*smv1a1.MongoDBDatabase)
		db.Status = obj.Status
		return db
	}); err2 != nil {
		log.Error(err2, "Error occurred when Patching the MongoDBDatabase's status")
		return err2
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&smv1a1.MongoDBDatabase{}).
		Owns(&kvm_engine.SecretAccessRequest{}).
		Owns(&kvm_engine.SecretEngine{}).
		Owns(&kvm_engine.MongoDBRole{}).
		Owns(&batchv1.Job{}).
		Owns(&stash.RestoreSession{}).
		Watches(&source.Kind{Type: &kdm.MongoDB{}}, handler.EnqueueRequestsFromMapFunc(r.getHandlerFuncForMongoDB())).
		Watches(&source.Kind{Type: &kvm_server.VaultServer{}}, handler.EnqueueRequestsFromMapFunc(r.getHandlerFuncForVaultServer())).
		Complete(r)
}

func (r *MongoDBDatabaseReconciler) getHandlerFuncForMongoDB() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		obj := object.(*kdm.MongoDB)
		var schemas smv1a1.MongoDBDatabaseList
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
		var schemas smv1a1.MongoDBDatabaseList
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

/*
delete any external resources associated with the obj. Ensure that delete implementation is idempotent and safe to invoke
multiple times for same object. our finalizer is present, so lets handle any external dependency

If we find the {UID: dbName} entry in the 'kubedb-system' database
	Drop the 'dbName' database, & delete that corresponding entry
*/
func (r *MongoDBDatabaseReconciler) doExternalThingsBeforeDelete(ctx context.Context, obj *smv1a1.MongoDBDatabase, mongo *kdm.MongoDB, log logr.Logger) error {
	mongoClient, err := dbClient.NewKubeDBClientBuilder(r.Client, mongo).WithContext(ctx).GetMongoClient()
	defer mongoClient.Close()
	if err != nil {
		log.Error(err, "Unable to run GetMongoClient() function")
		return err
	}
	name := obj.Spec.DatabaseSchema.Name
	cursor := mongoClient.Database(smv1a1.MongoDatabaseNameForEntry).Collection(smv1a1.MongoCollectionNameForEntry).FindOne(ctx, bson.D{
		{Key: string(obj.UID), Value: name},
	})
	if cursor.Err() == nil {
		// entry found, so drop the database & Delete entry from kube-system database
		err = mongoClient.Database(name).Drop(ctx)
		if err != nil {
			log.Error(err, "Can't drop the database")
			return err
		}
		_, err = mongoClient.Database(smv1a1.MongoDatabaseNameForEntry).Collection(smv1a1.MongoCollectionNameForEntry).DeleteOne(ctx, bson.D{
			{Key: string(obj.UID), Value: name},
		})
		if err != nil {
			log.Error(err, "Error occurred when delete the entry form kube-system database")
			return err
		}
	}
	return nil
}

/*
If the requested database already exists
	return error
Otherwise update-or-insert the corresponding entry accordingly
*/
func (r *MongoDBDatabaseReconciler) ensureEntryIntoDatabase(ctx context.Context, obj *smv1a1.MongoDBDatabase, mongo kdm.MongoDB, log logr.Logger) error {
	mongoClient, err := dbClient.NewKubeDBClientBuilder(r.Client, &mongo).WithContext(ctx).GetMongoClient()
	if err != nil {
		klog.Fatalf("Running MongoDBClient failed. %s", err.Error())
		return err
	}
	defer mongoClient.Close()
	name := obj.Spec.DatabaseSchema.Name

	// For checking, if the Database named 'obj.Spec.DatabaseSchema.Name' already exist or not
	res, err := mongoClient.Database(name).ListCollectionNames(ctx, bson.D{}) // collectionList
	if err != nil {
		log.Error(err, "Error occurred when listing the Collection names of  database")
		return err
	}
	if len(res) > 0 {
		if err2 := r.updateSchemaMongoDBDatabaseStatus(ctx, obj, log, smv1a1.SchemaDatabasePhaseIgnored); err2 != nil {
			return err2
		}
		return errors.New("database already exist")
	}
	// We are here means, Database doesn't already exist
	valTrue := true
	_, err = mongoClient.Database(smv1a1.MongoDatabaseNameForEntry).Collection(smv1a1.MongoCollectionNameForEntry).UpdateOne(
		ctx,
		bson.M{string(obj.UID): name},
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: string(obj.UID), Value: name},
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

/*
If obj is not requested for deletion & finalizerName is still not added:
	ensureEntryIntroDatabase() + addFinalizer()
else if obj is requested for deletion & finalizerName is found:
	doExternalThingsBeforeDelete() + deleteFinalizer()
In both case, Update the obj
*/
func (r *MongoDBDatabaseReconciler) ensureOrDeleteFinalizer(ctx context.Context, obj *smv1a1.MongoDBDatabase, mongo kdm.MongoDB, log logr.Logger) (bool, error) {
	finalizerName := smv1a1.GroupVersion.Group
	// examine DeletionTimestamp to determine if object is under deletion
	if obj.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so Add entry to our Database & register our finalizer
		if !ContainsString(obj.GetFinalizers(), finalizerName) {
			err := r.ensureEntryIntoDatabase(ctx, obj, mongo, log)
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
	} else if obj.Spec.DeletionPolicy == smv1a1.DeletionPolicyDelete {
		// The object is assigned for Deletion
		if ContainsString(obj.GetFinalizers(), finalizerName) {
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
func (r *MongoDBDatabaseReconciler) ensureSecretEngine(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger) (*kvm_engine.SecretEngine, error) {
	fetchedSecretEngine, _, err := clientutil.CreateOrPatch(ctx, r.Client, &kvm_engine.SecretEngine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetMongoSecretEngineName(),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		se := object.(*kvm_engine.SecretEngine)

		se.Spec.VaultRef.Name = obj.Spec.VaultRef.Name
		se.Spec.VaultRef.Namespace = obj.Spec.VaultRef.Namespace

		se.Spec.SecretEngineConfiguration.MongoDB = &kvm_engine.MongoDBConfiguration{
			DatabaseRef: appcatutil.AppReference{
				Namespace: obj.Spec.DatabaseRef.Namespace,
				Name:      obj.Spec.DatabaseRef.Name,
			},
		}
		se.Spec.SecretEngineConfiguration.MongoDB.PluginName = kvm_engine.DefaultMongoDBDatabasePlugin

		if createOp {
			coreutil.EnsureOwnerReference(&se.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return se
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required Secret Engine")
		return nil, err
	}
	r.updateStatusForSecretEngine(ctx, obj, fetchedSecretEngine.(*kvm_engine.SecretEngine))
	return fetchedSecretEngine.(*kvm_engine.SecretEngine), nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForSecretEngine(ctx context.Context, obj *smv1a1.MongoDBDatabase, se *kvm_engine.SecretEngine) {
	err := r.Get(ctx, types.NamespacedName{
		Namespace: se.Namespace,
		Name:      se.Name,
	}, se)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionSecretEngineReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "Secret Engine is not created yet")
		return
	}
	if !CheckSecretEngineConditions(se) {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionSecretEngineReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReasonSecretEngineReady, "Secret Engine is provisioning")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionSecretEngineReady, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonSecretEngineReady, "Secret Engine is ready")
}

// Create or Patch the MongoDBRole
func (r *MongoDBDatabaseReconciler) ensureMongoDBRole(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger) (*kvm_engine.MongoDBRole, error) {
	fetchedMongoDbRole, _, err := clientutil.CreateOrPatch(ctx, r.Client, &kvm_engine.MongoDBRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetMongoAdminRoleName(),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		mr := object.(*kvm_engine.MongoDBRole)
		mr.Spec.SecretEngineRef.Name = obj.GetMongoSecretEngineName()
		generatedString := fmt.Sprintf("{ \"db\": \"%s\", \"roles\": [{ \"role\": \"dbOwner\" }] }", obj.Spec.DatabaseSchema.Name)
		mr.Spec.CreationStatements = []string{generatedString}
		mr.Spec.RevocationStatements = []string{generatedString}
		if createOp {
			coreutil.EnsureOwnerReference(&mr.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return mr
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required MongoDbRole")
		return nil, err
	}
	r.updateStatusForMongoDbRole(ctx, obj, fetchedMongoDbRole.(*kvm_engine.MongoDBRole))
	return fetchedMongoDbRole.(*kvm_engine.MongoDBRole), nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForMongoDbRole(ctx context.Context, obj *smv1a1.MongoDBDatabase, mr *kvm_engine.MongoDBRole) {
	err := r.Get(ctx, types.NamespacedName{
		Namespace: mr.Namespace,
		Name:      mr.Name,
	}, mr)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionMongoDBRoleReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "MongoDB Role is not created yet")
		return
	}
	if !CheckMongoDBRoleConditions(mr) {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionMongoDBRoleReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReasonMongoDBRoleReady, "MongoDB Role is being created")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionMongoDBRoleReady, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonMongoDBRoleReady, "MongoDB Role is ready")
}

// Create or Patch the service account
func (r *MongoDBDatabaseReconciler) ensureServiceAccount(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger) (*corev1.ServiceAccount, error) {
	fetchedServiceAccount, _, err := clientutil.CreateOrPatch(ctx, r.Client, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetMongoAdminServiceAccountName(),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		sa := object.(*corev1.ServiceAccount)
		if createOp {
			coreutil.EnsureOwnerReference(&sa.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return sa
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the Service Account.")
		return nil, err
	}
	return fetchedServiceAccount.(*corev1.ServiceAccount), nil
}

/*
Create or Patch the secret Access Request
assigning RoleRef & Subjects in the secretAccessRequest spec, where subjects.name = our serviceAccount name
I auto Approval On : An approved condition will be set accordingly
*/
func (r *MongoDBDatabaseReconciler) ensureSecretAccessRequest(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger) (*kvm_engine.SecretAccessRequest, error) {

	fetchedAccessRequest, _, err := clientutil.CreateOrPatch(ctx, r.Client, &kvm_engine.SecretAccessRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetMongoAdminSecretAccessRequestName(),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		sar := object.(*kvm_engine.SecretAccessRequest)

		sar.Spec.RoleRef.Kind = kvm_engine.ResourceKindMongoDBRole
		sar.Spec.RoleRef.Name = obj.GetMongoAdminRoleName()

		var accessRequestFound = false
		for i := 0; i < len(sar.Spec.Subjects); i++ {
			sub := sar.Spec.Subjects[i]
			if sub.Name == obj.GetMongoAdminServiceAccountName() {
				accessRequestFound = true
			}
		}
		if !accessRequestFound {
			sar.Spec.Subjects = append(sar.Spec.Subjects, rbac.Subject{
				Kind:      rbac.ServiceAccountKind,
				Name:      obj.GetMongoAdminServiceAccountName(),
				Namespace: obj.Namespace,
			})
		}
		if createOp {
			coreutil.EnsureOwnerReference(&sar.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return sar
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the Secret Access request.")
		return nil, err
	}
	if !obj.Spec.AutoApproval {
		r.updateStatusForSecretAccessRequest(ctx, obj, fetchedAccessRequest.(*kvm_engine.SecretAccessRequest))
		return fetchedAccessRequest.(*kvm_engine.SecretAccessRequest), nil
	}

	// user sets auto approval ON
	sarForReturn := fetchedAccessRequest.(*kvm_engine.SecretAccessRequest)
	sarForReturn.Status.Conditions = apiv1util.SetCondition(sarForReturn.Status.Conditions, apiv1util.Condition{
		Type:    apiv1util.ConditionRequestApproved,
		Status:  corev1.ConditionTrue,
		Reason:  string(smv1a1.SchemaDatabaseAutoApprovalReason),
		Message: "This was approved by: KubeDb SchemaManager",
	})
	tmpSar, _, err := clientutil.PatchStatus(ctx, r.Client, &kvm_engine.SecretAccessRequest{
		ObjectMeta: sarForReturn.ObjectMeta,
	}, func(object client.Object, createOp bool) client.Object {
		insideSar := object.(*kvm_engine.SecretAccessRequest)
		insideSar.Status = sarForReturn.Status
		return insideSar
	})
	if err != nil {
		log.Error(err, "Unable to Patch the Secret Access request status.")
		return nil, err
	}
	sarForReturn = tmpSar.(*kvm_engine.SecretAccessRequest)
	r.updateStatusForSecretAccessRequest(ctx, obj, sarForReturn)
	return sarForReturn, nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForSecretAccessRequest(ctx context.Context, obj *smv1a1.MongoDBDatabase, sar *kvm_engine.SecretAccessRequest) {
	var newSar kvm_engine.SecretAccessRequest
	err := r.Get(ctx, types.NamespacedName{
		Namespace: sar.Namespace,
		Name:      sar.Name,
	}, &newSar)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionSecretAccessRequestReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "Secret access request is not created yet")
		return
	}
	if !CheckSecretAccessRequestConditions(&newSar) {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionSecretAccessRequestReady, corev1.ConditionFalse, smv1a1.SchemaDatabaseReasonSecretAccessRequestReady, "Secret access request is being created")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionSecretAccessRequestReady, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonSecretAccessRequestReady, "Secret access request is succeeded")
}

func (r *MongoDBDatabaseReconciler) setDefaultOptionsForJobContainer(container *corev1.Container, givenPodSpec offshootv1util.PodSpec) {
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

// It returns the initContainer spec by assigning Image, args, env & volumeMounts fields
func (r *MongoDBDatabaseReconciler) getTheInitContainerSpec(ctx context.Context, obj *smv1a1.MongoDBDatabase, mongo kdm.MongoDB, singleCred corev1.Secret, log logr.Logger) (corev1.Container, error) {
	// If the user doesn't provide podTemplate, we'll fill it with default value
	isPodTemplateGiven := obj.Spec.Init.PodTemplate != nil
	var givenPodSpec offshootv1util.PodSpec

	// TLS related part
	var sslArgs string
	if mongo.Spec.SSLMode != kdm.SSLModeDisabled {
		var err error
		sslArgs, err = r.getSSLArgsForJob(ctx, mongo, sslArgs, log)
		if err != nil {
			return corev1.Container{}, err
		}
	}

	envList := []corev1.EnvVar{
		{
			Name: "MONGODB_USERNAME",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: singleCred.Name,
					},
					Key: "username",
				},
			},
		},
		{
			Name: "MONGODB_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
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
			mongo.ServiceName(), mongo.Namespace, sslArgs, smv1a1.MongoInitScriptPath, smv1a1.InitScriptName),
	}
	if isPodTemplateGiven {
		givenPodSpec = obj.Spec.Init.PodTemplate.Spec
		envList = coreutil.UpsertEnvVars(envList, givenPodSpec.Env...)
		argList = metautil.UpsertArgumentList(givenPodSpec.Args, argList)
	}

	var volumeMounts []corev1.VolumeMount
	volumeMounts = coreutil.UpsertVolumeMount(volumeMounts, []corev1.VolumeMount{
		{
			Name:      obj.GetMongoInitVolumeNameForPod(),
			MountPath: smv1a1.MongoInitScriptPath,
		},
	}...)

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
		return corev1.Container{}, err
	}
	return corev1.Container{
		Name:            obj.GetMongoInitContainerNameForPod(),
		Image:           versionImage,
		Env:             envList,
		Command:         []string{"/bin/sh", "-c"},
		Args:            argList,
		ImagePullPolicy: corev1.PullAlways,
		VolumeMounts:    volumeMounts,
	}, nil
}

// Now Make the Job
func (r *MongoDBDatabaseReconciler) makeTheInitJob(ctx context.Context, obj *smv1a1.MongoDBDatabase, mongo kdm.MongoDB, singleCred corev1.Secret, log logr.Logger) (*batchv1.Job, error) {
	// if the user doesn't provide init script, then no need to create the job
	if obj.Spec.Init == nil || obj.Spec.Init.Script == nil {
		return nil, nil
	}
	isPodTemplateGiven := obj.Spec.Init.PodTemplate != nil
	var givenPodSpec offshootv1util.PodSpec
	givenScript := obj.Spec.Init.Script
	if isPodTemplateGiven {
		givenPodSpec = obj.Spec.Init.PodTemplate.Spec
	}

	var volumes []corev1.Volume
	volumes = coreutil.UpsertVolume(volumes, corev1.Volume{
		Name:         obj.GetMongoInitVolumeNameForPod(),
		VolumeSource: givenScript.VolumeSource,
	})
	cont, err := r.getTheInitContainerSpec(ctx, obj, mongo, singleCred, log)
	if err != nil {
		log.Error(err, "Error occurred when getting the Init container spec")
		return nil, err
	}

	// now actually create it or patch
	createdJob, vt, err := clientutil.CreateOrPatch(ctx, r.Client, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetMongoInitJobName(),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		job := object.(*batchv1.Job)

		job.Spec.Template.Spec.Volumes = coreutil.UpsertVolume(job.Spec.Template.Spec.Volumes, volumes...)
		containers := []corev1.Container{
			cont,
		}
		if isPodTemplateGiven {
			r.setDefaultOptionsForJobContainer(&containers[0], givenPodSpec)
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
		job.Spec.Template.Spec.Containers = coreutil.UpsertContainers(job.Spec.Template.Spec.Containers, containers)

		// InitContainers doesn't make any sense.
		job.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyOnFailure
		job.Spec.BackoffLimit = func(i int32) *int32 { return &i }(5)

		if createOp {
			coreutil.EnsureOwnerReference(&job.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
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
	r.updateStatusForInitJob(ctx, obj, createdJob.(*batchv1.Job))
	return createdJob.(*batchv1.Job), nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForInitJob(ctx context.Context, obj *smv1a1.MongoDBDatabase, jb *batchv1.Job) {
	var newJob batchv1.Job
	err := r.Get(ctx, types.NamespacedName{
		Namespace: jb.Namespace,
		Name:      jb.Name,
	}, &newJob)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionJobCompleted, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "Job is not created yet")
		return
	}
	if !CheckJobConditions(&newJob) {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionJobCompleted, corev1.ConditionFalse, smv1a1.SchemaDatabaseReasonJobCompleted, "Job is being created")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionJobCompleted, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonJobCompleted, "Job is completed")
}

func (r *MongoDBDatabaseReconciler) copyRepositoryAndSecret(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger) error {
	var repo repository.Repository
	err := r.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.Restore.Repository.Namespace,
		Name:      obj.Spec.Restore.Repository.Name,
	}, &repo)
	if err != nil {
		log.Error(err, "Can't get the Repository")
		return err
	}
	fetcherdRepository, _, err := clientutil.CreateOrPatch(ctx, r.Client, &repository.Repository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      repo.Name,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		rep := object.(*repository.Repository)
		rep.Spec = repo.Spec
		if createOp {
			coreutil.EnsureOwnerReference(&rep.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return rep
	})
	if err != nil {
		log.Error(err, "Error occurred when createOrPatch called for the Repository")
		return err
	}

	var secret corev1.Secret
	err = r.Get(ctx, types.NamespacedName{
		Namespace: repo.Namespace,
		Name:      repo.Spec.Backend.StorageSecretName,
	}, &secret)
	if err != nil {
		log.Error(err, "Can't get the secret")
		return err
	}
	_, _, err = clientutil.CreateOrPatch(ctx, r.Client, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.Name,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		s := object.(*corev1.Secret)
		s.Data = secret.Data
		if createOp {
			coreutil.EnsureOwnerReference(&s.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return s
	})
	if err != nil {
		log.Error(err, "Error occurred when createOrPatch called for the secret")
		return err
	}
	r.updateStatusForRepository(ctx, obj, fetcherdRepository.(*repository.Repository))
	return nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForRepository(ctx context.Context, obj *smv1a1.MongoDBDatabase, rp *repository.Repository) {
	var repo repository.Repository
	err := r.Get(ctx, types.NamespacedName{
		Namespace: rp.Namespace,
		Name:      rp.Name,
	}, &repo)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionRepositoryFound, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "Repository is not created yet")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionRepositoryFound, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonRepositoryFound, "Repository is copied successfully")
}

/*
this appbinding is same as the appbinding created in db namespace; Just .Spec.ClientConfig.Service.Name has been changed
to access the db-service from different namespace
*/
func (r *MongoDBDatabaseReconciler) ensureAppbinding(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger) error {
	/// Getting the root credential from db namespace to schema-manager namespace
	var authSecret corev1.Secret
	err := r.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.DatabaseRef.Namespace,
		Name:      obj.GetAuthSecretName(obj.Spec.DatabaseRef.Name),
	}, &authSecret)
	if err != nil {
		log.Error(err, "Cant get the auth secret")
		return err
	}
	_, _, err = clientutil.CreateOrPatch(ctx, r.Client, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      authSecret.Name,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		s := object.(*corev1.Secret)
		s.Data = authSecret.Data
		if createOp {
			coreutil.EnsureOwnerReference(&s.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return s
	})

	var appbinding appcatutil.AppBinding
	err = r.Get(ctx, types.NamespacedName{
		Namespace: obj.Spec.DatabaseRef.Namespace,
		Name:      obj.Spec.DatabaseRef.Name,
	}, &appbinding)
	if err != nil {
		log.Error(err, "Cant get the appbinding named")
		return err
	}

	fetchedAppbinding, _, err := clientutil.CreateOrPatch(ctx, r.Client, &appcatutil.AppBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appbinding.Name,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		ab := object.(*appcatutil.AppBinding)
		ab.Spec = appbinding.Spec
		ab.Spec.ClientConfig.Service.Name = func(obj *smv1a1.MongoDBDatabase) string {
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
			coreutil.EnsureOwnerReference(&ab.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return ab
	})
	if err != nil {
		log.Error(err, "Error occurred when createOrPatch called for AppBinding")
		return err
	}
	r.updateStatusForAppbinding(ctx, obj, fetchedAppbinding.(*appcatutil.AppBinding))
	return nil
}

/*
This is a utility function which will be used on ensureAppbinding's CreateOrPatch()
when vault-secret will be used instead of the auth-secret
*/
func (r *MongoDBDatabaseReconciler) getVaultCreatedSecretName(ctx context.Context, obj *smv1a1.MongoDBDatabase, log logr.Logger) (string, error) {
	var sar kvm_engine.SecretAccessRequest
	err := r.Get(ctx, types.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetMongoAdminSecretAccessRequestName(),
	}, &sar)
	if err != nil {
		log.Error(err, "error occurred when getting the secret which was created by Vault")
		return "", err
	}
	if sar.Status.Secret == nil {
		return "", errors.New("secret is not generated yet in the secretAccessRequest")
	}
	return sar.Status.Secret.Name, nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForAppbinding(ctx context.Context, obj *smv1a1.MongoDBDatabase, ab *appcatutil.AppBinding) {
	var bind appcatutil.AppBinding
	err := r.Get(ctx, types.NamespacedName{
		Namespace: ab.Namespace,
		Name:      ab.Name,
	}, &bind)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionAppbindingFound, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "Appbinding is not created yet")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionAppbindingFound, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonAppbindingFound, "Appbinding creation succeeded")
}

/*
Create or Patch the RestoreSession
Task, Target.ref, Target.Rules & Repository.Name fields will be set of the restoreSession Spec
The task Name has been taken by getting the MongoDBVersion object
*/
func (r *MongoDBDatabaseReconciler) ensureRestoreSession(ctx context.Context, obj *smv1a1.MongoDBDatabase, mongo kdm.MongoDB, log logr.Logger) (*stash.RestoreSession, error) {
	// Getting the  MongoDBVersion object
	var mongoVer kd_catalog.MongoDBVersion
	err := r.Get(ctx, types.NamespacedName{
		Name: mongo.Spec.Version,
	}, &mongoVer)
	if err != nil {
		log.Error(err, "Cant get the MongoDBVersion ")
		return nil, err
	}

	fetchedRestoreSession, _, err := clientutil.CreateOrPatch(ctx, r.Client, &stash.RestoreSession{
		ObjectMeta: metav1.ObjectMeta{
			Name:      obj.GetMongoRestoreSessionName(),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		rs := object.(*stash.RestoreSession)

		rs.Spec.Task = stash.TaskRef{
			Name: mongoVer.Spec.Stash.Addon.RestoreTask.Name,
		}
		rs.Spec.Target = &stash.RestoreTarget{
			Ref: stash.TargetRef{
				APIVersion: appcatutil.SchemeGroupVersion.String(),
				Kind:       appcatutil.ResourceKindApp,
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
			coreutil.EnsureOwnerReference(&rs.ObjectMeta, metav1.NewControllerRef(obj, smv1a1.GroupVersion.WithKind(smv1a1.ResourceKindMongoDBDatabase)))
		}
		return rs
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required Secret Engine")
		return nil, err
	}
	r.updateStatusForRestoreSession(ctx, obj, fetchedRestoreSession.(*stash.RestoreSession))
	return fetchedRestoreSession.(*stash.RestoreSession), nil
}

func (r *MongoDBDatabaseReconciler) updateStatusForRestoreSession(ctx context.Context, obj *smv1a1.MongoDBDatabase, rs *stash.RestoreSession) {
	var restore stash.RestoreSession
	err := r.Get(ctx, types.NamespacedName{
		Namespace: rs.Namespace,
		Name:      rs.Name,
	}, &restore)
	if err != nil {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionRestoreSession, corev1.ConditionFalse, smv1a1.SchemaDatabaseReason(err.Error()), "RestoreSession is not created yet")
		return
	}
	if CheckRestoreSessionPhaseFailed(&restore) {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionRestoreSession, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonRestoreSessionSFailed, "Restoring process is failed")
		return
	}
	if CheckRestoreSessionPhaseSucceed(&restore) {
		obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionRestoreSession, corev1.ConditionTrue, smv1a1.SchemaDatabaseReasonRestoreSessionSucceed, "Restoring process is succeeded")
		return
	}
	obj.Status.Conditions = obj.SetCondition(smv1a1.SchemaDatabaseConditionRestoreSession, corev1.ConditionFalse, smv1a1.SchemaDatabaseReasonRestoreSessionSucceed, "Restoring process is running")
}
