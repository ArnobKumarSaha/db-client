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
	b64 "encoding/base64"
	"fmt"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientutil "kmodules.xyz/client-go/client"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
	"kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
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
	fmt.Println("req = ", req.NamespacedName)
	var obj schemav1alpha1.MongoDBDatabase
	if err := r.Client.Get(ctx, req.NamespacedName, &obj); err != nil {
		log.Error(err, "unable to fetch mongodb Database")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
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
	if mongo.Status.Phase != kdm.DatabasePhaseReady || vault.Status.Phase != kvm_server.VaultServerPhaseReady{
		return ctrl.Result{}, nil
	}

	// Finalizer related things
	myFinalizerName := "v1alpha1.kubedb.dev/finalizer"
	// examine DeletionTimestamp to determine if object is under deletion
	if obj.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so register our finalizer.
		if !containsString(obj.GetFinalizers(), myFinalizerName) {
			controllerutil.AddFinalizer(&obj, myFinalizerName)
			if err := r.Update(ctx, &obj); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else if obj.Spec.DeletionPolicy == schemav1alpha1.DeletionPolicyDelete {
		// The object is being deleted
		if containsString(obj.GetFinalizers(), myFinalizerName) {
			if err := r.doExternalThingsBeforeDelete(ctx, &obj, &mongo, log); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return ctrl.Result{}, err
			}
			fmt.Println("Finalizer need to be removed now.")
			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(&obj, myFinalizerName)
			if err := r.Update(ctx, &obj); err != nil {
				log.Error(err, "Cant update MongoDBDatabase object %v when removing finalizers %v", obj.GetName())
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	if _, err = r.ensureSecretEngine(obj, log); err != nil {
		return ctrl.Result{}, err
	}
	if _, err = r.ensureMongoDBRole(obj, log); err != nil {
		return ctrl.Result{}, err
	}
	if _, err = r.ensureServiceAccount(obj, log); err != nil {
		return ctrl.Result{}, err
	}
	var fetchedAccessRequest *kvm_engine.SecretAccessRequest
	if fetchedAccessRequest, err = r.ensureSecretAccessRequest(obj, log); err != nil {
		return ctrl.Result{}, err
	}

	// Here we need to automate the vault approve command here
	if fetchedAccessRequest.Status.Phase == kvm_engine.RequestStatusPhaseWaitingForApproval {
		return ctrl.Result{}, nil
	}
	var singleCred v1.Secret
	if fetchedAccessRequest.Status.Phase == kvm_engine.RequestStatusPhaseApproved &&
		fetchedAccessRequest.Status.Secret != nil{
		err = r.Client.Get(ctx, types.NamespacedName{
			Namespace: fetchedAccessRequest.Status.Secret.Namespace,
			Name:      fetchedAccessRequest.Status.Secret.Name,
		}, &singleCred)
		if err != nil {
			log.Error(err, "Unable to Get the required secret")
			return ctrl.Result{}, err
		}
		for _, v := range singleCred.Data {
			str := b64.StdEncoding.EncodeToString(v)
			_, err2 := b64.StdEncoding.DecodeString(str)
			if err2 != nil {
				log.Error(err, "Error occurred when base64 decoding")
				return ctrl.Result{}, err
			}
		}

		_, err = r.makeTheJob(ctx, log, obj, mongo, singleCred)
		if err != nil {
			log.Error(err, "Can't create the job")
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schemav1alpha1.MongoDBDatabase{}).
		Owns(&kvm_engine.SecretAccessRequest{}).
		Watches(&source.Kind{Type: &kdm.MongoDB{}}, handler.EnqueueRequestsFromMapFunc(r.getHandlerFuncForMongoDB())).
		Watches(&source.Kind{Type: &kvm_server.VaultServer{}}, handler.EnqueueRequestsFromMapFunc(r.getHandlerFuncForVaultServer())).
		Complete(r)
}

func (r *MongoDBDatabaseReconciler) getHandlerFuncForMongoDB() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		obj := object.(*kdm.MongoDB)
		var schemas schemav1alpha1.MongoDBDatabaseList
		var arr []reconcile.Request

		// Listing from all namespaces
		err := r.Client.List(context.TODO(), &schemas, &client.ListOptions{})
		if err != nil{
			return arr
		}
		for _, schema := range schemas.Items {
			if schema.Spec.DatabaseRef.Name == obj.Name && schema.Spec.DatabaseRef.Namespace == obj.Namespace{
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
	//core_util.IsOwnedBy()
}

func (r *MongoDBDatabaseReconciler) getHandlerFuncForVaultServer() handler.MapFunc {
	return func(object client.Object) []reconcile.Request {
		obj := object.(*kvm_server.VaultServer)
		var schemas schemav1alpha1.MongoDBDatabaseList
		var arr []reconcile.Request

		// Listing from all namespaces
		err := r.Client.List(context.TODO(), &schemas, &client.ListOptions{})
		if err != nil{
			return arr
		}
		for _, schema := range schemas.Items {
			if schema.Spec.VaultRef.Name == obj.Name && schema.Spec.VaultRef.Namespace == obj.Namespace{
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

func setDefaultOptionsForJobContainer(container *v1.Container, givenPodSpec offshoot_v1.PodSpec) {
	container.Resources = givenPodSpec.Resources
	container.LivenessProbe = givenPodSpec.LivenessProbe
	container.ReadinessProbe = givenPodSpec.ReadinessProbe
	container.Lifecycle = givenPodSpec.Lifecycle
	container.SecurityContext = givenPodSpec.ContainerSecurityContext
}

func (r *MongoDBDatabaseReconciler) doExternalThingsBeforeDelete(ctx context.Context, obj *schemav1alpha1.MongoDBDatabase, mongo *kdm.MongoDB, log logr.Logger) error {
	// delete any external resources associated with the obj Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object. our finalizer is present, so lets handle any external dependency
	mongoClient, err := dbClient.NewKubeDBClientBuilder(r.Client, mongo).WithContext(ctx).GetMongoClient()
	if err != nil {
		log.Error(err, "Unable to run GetMongoClient() function")
		return err
	}
	err = mongoClient.Database(obj.Spec.DatabaseSchema.Name).Drop(ctx)
	if err != nil {
		log.Error(err, "Can't drop the database")
		return err
	}
	return nil
}

// Create or Patch the Secret Engine
func (r *MongoDBDatabaseReconciler) ensureSecretEngine(obj schemav1alpha1.MongoDBDatabase, log logr.Logger) (*kvm_engine.SecretEngine, error) {
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
			core_util.EnsureOwnerReference(&se.ObjectMeta, metav1.NewControllerRef(&obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return se
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required Secret Engine")
		return nil, err
	}
	return fetchedSecretEngine.(*kvm_engine.SecretEngine), nil
}

// Create or Patch the MongoDBRole
func (r *MongoDBDatabaseReconciler) ensureMongoDBRole(obj schemav1alpha1.MongoDBDatabase, log logr.Logger) (*kvm_engine.MongoDBRole, error){
	fetchedMongoDbRole, _, err := clientutil.CreateOrPatch(r.Client, &kvm_engine.MongoDBRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getMongoAdminRoleName(obj.GetName()),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		mr := object.(*kvm_engine.MongoDBRole)
		mr.Spec.SecretEngineRef.Name = getMongoSecretEngineName(obj.GetName())
		generatedString := fmt.Sprintf("{ \"db\": \"%s\", \"roles\": [{ \"role\": \"dbAdmin\" }, { \"role\": \"readWrite\" }] }", obj.Spec.DatabaseSchema.Name)
		mr.Spec.CreationStatements = append(mr.Spec.CreationStatements, generatedString)

		if createOp {
			core_util.EnsureOwnerReference(&mr.ObjectMeta, metav1.NewControllerRef(&obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return mr
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required MongoDbRole")
		return nil, err
	}
	return fetchedMongoDbRole.(*kvm_engine.MongoDBRole), nil
}

// Create or Patch the service account
func (r *MongoDBDatabaseReconciler) ensureServiceAccount(obj schemav1alpha1.MongoDBDatabase, log logr.Logger) (*v1.ServiceAccount, error){
	fetchedServiceAccount, _, err := clientutil.CreateOrPatch(r.Client, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      getMongoAdminServiceAccountName(obj.GetName()),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		sa := object.(*v1.ServiceAccount)
		if createOp {
			core_util.EnsureOwnerReference(&sa.ObjectMeta, metav1.NewControllerRef(&obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return sa
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the Service Account.")
		return nil, err
	}
	return fetchedServiceAccount.(*v1.ServiceAccount), nil
}

// Create or Patch the secret Access Request
func (r *MongoDBDatabaseReconciler) ensureSecretAccessRequest(obj schemav1alpha1.MongoDBDatabase, log logr.Logger) (*kvm_engine.SecretAccessRequest, error){
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
			core_util.EnsureOwnerReference(&sar.ObjectMeta, metav1.NewControllerRef(&obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return sar
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the Secret Access request.")
		return nil, err
	}
	return fetchedAccessRequest.(*kvm_engine.SecretAccessRequest), nil
}

// Now Make the Job
func (r *MongoDBDatabaseReconciler) makeTheJob(ctx context.Context, log logr.Logger, obj schemav1alpha1.MongoDBDatabase, mongo kdm.MongoDB, singleCred v1.Secret) (*batchv1.Job, error) {
	// if the user doesn't provide init script, then no need to create the job
	if obj.Spec.Init == nil || obj.Spec.Init.Script == nil {
		return nil, nil
	}
	// If the user doesn't provide podTemplate, we'll fill it with default value
	isPodTemplateGiven := obj.Spec.Init.PodTemplate != nil
	var givenPodSpec offshoot_v1.PodSpec
	givenScript := obj.Spec.Init.Script

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
		fmt.Sprintf(`mongo --host %v.%v.svc.cluster.local --authenticationDatabase $MONGODB_DATABASE_NAME -u "$MONGODB_USERNAME" -p "$MONGODB_PASSWORD" < %v/%v;`, mongo.ServiceName(), mongo.Namespace, MongoInitScriptPath, InitScriptName),
	}
	if isPodTemplateGiven {
		givenPodSpec = obj.Spec.Init.PodTemplate.Spec
		envList = core_util.UpsertEnvVars(envList, givenPodSpec.Env...)
		argList = meta_util.UpsertArgumentList(givenPodSpec.Args, argList)
	}

	var volumeMounts []v1.VolumeMount
	volumeMounts = core_util.UpsertVolumeMount(volumeMounts, []v1.VolumeMount{
		{
			Name:      VolumeNameForPod,
			MountPath: MongoInitScriptPath,
		},
	}...)

	var volumes []v1.Volume
	volumes = core_util.UpsertVolume(volumes, v1.Volume{
		Name:         VolumeNameForPod,
		VolumeSource: givenScript.VolumeSource,
	})

	// now actually create it or patch
	createdJob, _, err := clientutil.CreateOrPatch(r.Client, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JobName,
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
				Name:            MongoImage,
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
			core_util.EnsureOwnerReference(&job.ObjectMeta, metav1.NewControllerRef(&obj, schemav1alpha1.GroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return job
	})
	return createdJob.(*batchv1.Job), err
}

/*
return []string{
"bash",
"-c",
fmt.Sprintf(`set -x; if [[ $(mongo admin --host=localhost %v --username=$MONGO_INITDB_ROOT_USERNAME --password=$MONGO_INITDB_ROOT_PASSWORD --authenticationDatabase=admin
--quiet --eval "db.adminCommand('ping').ok" ) -eq "1" ]]; then
          exit 0
        fi
        exit 1`, sslArgs),
}*/

/*
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout ./ca.key -out ./ca.crt -subj "/CN=mongo/O=kubedb"
kubectl create secret tls mongo-ca --cert=ca.crt --key=ca.key --namespace=demo

apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: mongo-ca-issuer
  namespace: demo
spec:
  ca:
    secretName: mongo-ca


apiVersion: kubedb.com/v1alpha2
kind: MongoDB
metadata:
  name: mgo-rs-tls
  namespace: demo
spec:
  version: "4.1.13-v1"
  sslMode: requireSSL
  tls:
    issuerRef:
      apiGroup: "cert-manager.io"
      kind: Issuer
      name: mongo-ca-issuer
  clusterAuthMode: x509
  replicas: 4
  replicaSet:
    name: rs0
  storage:
    storageClassName: "standard"
    accessModes:
      - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi


new secret will be generated automatically
kubectl exec -it mgo-rs-tls-0 -n demo bash   =>   ls /var/run/mongodb/tls
openssl x509 -in /var/run/mongodb/tls/client.pem -inform PEM -subject -nameopt RFC2253 -noout
mongo --tls --tlsCAFile /var/run/mongodb/tls/ca.crt --tlsCertificateKeyFile /var/run/mongodb/tls/client.pem admin --host localhost
				--authenticationMechanism MONGODB-X509 --authenticationDatabase='$external' -u "CN=root,O=kubedb" --quiet
db.adminCommand({ getParameter:1, sslMode:1 })
use $external     =>   show users


kubectl delete mongodb -n demo mgo-rs-tls
kubectl delete issuer -n demo mongo-ca-issuer
kubectl delete ns demo
 */