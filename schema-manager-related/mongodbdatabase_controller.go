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
	"kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
	kdm "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	schemav1alpha1 "kubedb.dev/schema-manager/apis/schema/v1alpha1"
	kvm_engine "kubevault.dev/apimachinery/apis/engine/v1alpha1"
	kvm_server "kubevault.dev/apimachinery/apis/kubevault/v1alpha1"
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
	log := r.Log.WithValues("mongodbdatabase", req.NamespacedName)

	// TODO(user): your logic here
	/*
		reqName, reqNamespace := req.Name, req.Namespace
		var obj schemav1alpha1.MongoDBDatabase
		if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
			log.Error(err, "unable to fetch mongodb Database")
			// we'll ignore not-found errors, since they can't be fixed by an immediate
			// requeue (we'll need to wait for a new notification), and we can get them
			// on deleted requests.
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		fmt.Println("Got the object successfully !", reqName, reqNamespace)

		kubeClient, versionedClient, vaultServerClient, err := GetTheClients()
		if err != nil {
			log.Error(err, "Unable to make the Clients")
			return ctrl.Result{}, err
		}
		log.Info("Clinets created successfully.")
		mongoItem, err := GetMongoObject(&obj, versionedClient)
		if err != nil {
			log.Error(err, "Unable to get the Mongo Object")
			return ctrl.Result{}, err
		}
		vaultServerItem, err := GetVaultServerObject(&obj, vaultServerClient)
		if err != nil {
			log.Error(err, "Unable to get the Vault server Object")
			return ctrl.Result{}, err
		}
		log.Info("Mongo Object and VaultServer object has been found.")


			//Now Create the SecretEngine & MongoDBRole. And approve the request.

		fetchedEngine, err := vaultServerClient.EngineV1alpha1().SecretEngines(obj.Namespace).Get(ctx, makeSecretEngineName(obj.Name), metav1.GetOptions{})
		if err != nil {
			fmt.Printf("secret Engine of name %s has not been found. Trying to create it", makeSecretEngineName(obj.Name))
			fetchedEngine, err = vaultServerClient.EngineV1alpha1().SecretEngines(obj.Namespace).Create(ctx, secretEngineSpecification(&obj), metav1.CreateOptions{})
			if err != nil {
				fmt.Println("Cant create the secret engine for mongodb")
				return ctrl.Result{}, err
			}
		}

		fetchedRole, err := vaultServerClient.EngineV1alpha1().MongoDBRoles(obj.Namespace).Get(ctx, MongoReaderWriterRole, metav1.GetOptions{})
		if err != nil {
			fmt.Printf("MongoDbRole of name %s doesn't exist. trying to create one", MongoReaderWriterRole)
			vaultServerClient.EngineV1alpha1().MongoDBRoles(obj.Namespace).Create(ctx, ReaderWriterRoleSpecification(&obj), metav1.CreateOptions{})
			if err != nil {
				fmt.Println("Cant create the MongoDBRole")
				return ctrl.Result{}, err
			}
		}

		// *******************************************************
		fetchedRequest, err := vaultServerClient.EngineV1alpha1().SecretAccessRequests(obj.Namespace).Get(ctx, MongoDBReadWriteSecretAccessRequest, metav1.GetOptions{})
		if err != nil{
			fmt.Printf("SecretAccessRequest of name %s has not been found. Trying to create one.", MongoDBReadWriteSecretAccessRequest)
			fetchedRequest,err = vaultServerClient.EngineV1alpha1().SecretAccessRequests(obj.Namespace).Create(ctx, readWriteAccessRequestSpecification(&obj), metav1.CreateOptions{})
			if err != nil {
				fmt.Println("Cant create the Secret access request")
				return ctrl.Result{}, err
			}
		}
		// Role , secretEngine checked,
		// But accessRequest not checked.
		fmt.Println("from print")
		log.Info("from log.Info")

		fmt.Println(mongoItem.Name, vaultServerItem.Name, kubeClient, fetchedEngine.Name, fetchedRole.Name, fetchedRequest.Name)
	*/

	// First Get the actual CRD object of type MongoDBDatabase
	reqName, reqNamespace := req.Name, req.Namespace
	var obj schemav1alpha1.MongoDBDatabase
	if err := r.Client.Get(ctx, req.NamespacedName, &obj); err != nil {
		log.Error(err, "unable to fetch mongodb Database")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	fmt.Println("Got the object successfully !", reqName, reqNamespace)

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

	// Create or Patch the Secret Engine
	fetchedSecretEngine, vt, err := clientutil.CreateOrPatch(r.Client, &kvm_engine.SecretEngine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeSecretEngineName(obj.Name),
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		se := object.(*kvm_engine.SecretEngine)

		se.Spec.VaultRef.Name = obj.Spec.VaultRef.Name
		se.Spec.VaultRef.Namespace = obj.Spec.VaultRef.Namespace

		//se.Spec.SecretEngineConfiguration.MongoDB.DatabaseRef.Name = obj.Spec.DatabaseRef.Name
		//se.Spec.SecretEngineConfiguration.MongoDB.DatabaseRef.Namespace = obj.Spec.DatabaseRef.Namespace

		se.Spec.SecretEngineConfiguration.MongoDB = &kvm_engine.MongoDBConfiguration{
			DatabaseRef: v1alpha1.AppReference{
				Namespace: obj.Spec.DatabaseRef.Namespace,
				Name:      obj.Spec.DatabaseRef.Name,
			},
		}
		se.Spec.SecretEngineConfiguration.MongoDB.PluginName = MongoDBDatabasePlugin

		if createOp {
			core_util.EnsureOwnerReference(&se.ObjectMeta, metav1.NewControllerRef(&obj, kdm.SchemeGroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return se
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the required Secret Engine")
		return ctrl.Result{}, err
	}

	// CreateOrPatch the MongoDBRole
	fetchedMongoDbRole, vt, err := clientutil.CreateOrPatch(r.Client, &kvm_engine.MongoDBRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MongoReaderWriterRole,
			Namespace: UserNamespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		mr := object.(*kvm_engine.MongoDBRole)
		mr.Spec.SecretEngineRef.Name = makeSecretEngineName(obj.Name)
		mr.Spec.CreationStatements = append(mr.Spec.CreationStatements, "{ \"db\": \"mydb\", \"roles\": [{ \"role\": \"readWrite\" }] }")

		if createOp {
			core_util.EnsureOwnerReference(&mr.ObjectMeta, metav1.NewControllerRef(&obj, kdm.SchemeGroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return mr
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the MongoDbRole.")
		return ctrl.Result{}, err
	}

	// a service account
	fetchedServiceAccount, vt, err := clientutil.CreateOrPatch(r.Client, &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MongoDbReadWriteServiceAccount,
			Namespace: UserNamespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		sa := object.(*v1.ServiceAccount)
		if createOp {
			core_util.EnsureOwnerReference(&sa.ObjectMeta, metav1.NewControllerRef(&obj, kdm.SchemeGroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return sa
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the Service Account.")
		return ctrl.Result{}, err
	}

	// Now make secret Access Request
	fetchedAccessRequest, vt, err := clientutil.CreateOrPatch(r.Client, &kvm_engine.SecretAccessRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MongoDBReadWriteSecretAccessRequest,
			Namespace: UserNamespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		sar := object.(*kvm_engine.SecretAccessRequest)

		sar.Spec.RoleRef.Kind = ResourceKindMongoDBRole
		sar.Spec.RoleRef.Name = MongoReaderWriterRole

		var accessRequestFound = false
		for i := 0; i < len(sar.Spec.Subjects); i++ {
			sub := sar.Spec.Subjects[i]
			if sub.Name == MongoDbReadWriteServiceAccount {
				accessRequestFound = true
			}
		}
		if !accessRequestFound {
			sar.Spec.Subjects = append(sar.Spec.Subjects, rbac.Subject{
				Kind:      rbac.ServiceAccountKind,
				Name:      MongoDbReadWriteServiceAccount,
				Namespace: UserNamespace,
			})
		}
		if createOp {
			core_util.EnsureOwnerReference(&sar.ObjectMeta, metav1.NewControllerRef(&obj, kdm.SchemeGroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}

		return sar
	})
	if err != nil {
		log.Error(err, "Unable to createOrPatch the Secret Access request.")
		return ctrl.Result{}, err
	}
	// Here we need to automate the vault approve command
	////

	// Get the username-password secret
	// To do this, Firstly I have listed the secrets from userNamespace, then if that is our required secret -> captured it & decoded it.
	var credentials v1.SecretList
	var singleCred v1.Secret
	err = r.Client.List(ctx, &credentials, &client.ListOptions{Namespace: UserNamespace})
	if err != nil {
		log.Error(err, "Can't get the secret which has been created after vault approve command")
		return ctrl.Result{}, err
	}
	for _, cr := range credentials.Items {
		fmt.Println(cr.Name)
		if !checkPrefixMatch(cr.Name, MongoDBReadWriteSecretAccessRequest) {
			continue
		}
		err = r.Client.Get(ctx, types.NamespacedName{
			Namespace: UserNamespace,
			Name:      cr.Name,
		}, &singleCred)
		if err != nil {
			log.Error(err, "Unable to Get the required secret")
			return ctrl.Result{}, err
		}
		fmt.Println("Listing the data from secret")
		for k, v := range singleCred.Data {
			str := b64.StdEncoding.EncodeToString(v)
			val, err2 := b64.StdEncoding.DecodeString(str)
			if err2 != nil {
				log.Error(err, "Error occured when base64 decoding")
				return ctrl.Result{}, err
			}
			fmt.Println(k, string(val))
		}
	}


	// Job related things
	createdJob, vt, err := clientutil.CreateOrPatch(r.Client, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{},
	}, func(object client.Object, createOp bool) client.Object {
		job := object.(*batchv1.Job)

		//if len(job.Spec.Template.Spec.Containers) >
		return job
	})

	fmt.Println(fetchedSecretEngine.GetName(), vt, fetchedMongoDbRole.GetName(), fetchedAccessRequest.GetName(), fetchedServiceAccount.GetName(), createdJob.GetName())
	return ctrl.Result{}, nil
}

/*
VaultServer -> MongoDBServer -> SecretEngine -> MongoDBRole -> SecretAccessRequest ->
*/

// SetupWithManager sets up the controller with the Manager.
func (r *MongoDBDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&schemav1alpha1.MongoDBDatabase{}).
		Complete(r)
}

const (
	MongoDBDatabasePlugin       string = "mongodb-database-plugin"
	ResourceKindMongoDBDatabase string = "MongoDBDatabase"
	ResourceKindMongoDBRole     string = "MongoDBRole"
	UserNamespace               string = "dev"
)

func checkPrefixMatch(secretName, accessRequestName string) bool {
	if len(accessRequestName) > len(secretName) {
		return false
	}
	for i := 0; i < len(accessRequestName); i++ {
		if accessRequestName[i] != secretName[i] {
			return false
		}
	}
	return true
}

func makeSecretEngineName(name string) string {
	return name + "-secret-engine"
}

const (
	// Roles
	MongoReaderWriterRole string = "mongodb-reader-writer-role"
	MongoReaderRole       string = "mongodb-reader-role"
	MongoDBAdminRole      string = "mongodb-db-admin"

	// Secret Access requests
	MongoDBReadWriteSecretAccessRequest string = "mongodb-read-write-access-req"

	// Service Accounts
	MongoDbReadWriteServiceAccount string = "mongodb-read-write-sa"
)

/*
func (r *MongoDBDatabaseReconciler) GetMongoDBObject(obj *schemav1alpha1.MongoDBDatabase)  {
	//r.Client.Get(c)
}

// This block is for MongoDB Role , SecretAccessRequest & ServiceAccounts

const (
	// Roles
	MongoReaderWriterRole string = "mongodb-reader-writer-role"
	MongoReaderRole       string = "mongodb-reader-role"
	MongoDBAdminRole      string = "mongodb-db-admin"

	// Secret Access requests
	MongoDBReadWriteSecretAccessRequest string = "mongodb-read-write-access-req"

	// Service Accounts
	MongoDbReadWriteServiceAccount string = "mongodb-read-write-sa"
)

// This block is for some kind and Plugin related constants

const (
	MongoDBDatabasePlugin string = "mongodb-database-plugin"

	KindMongoDBRole string = "MongoDBRole"
	KindServiceAccount string = "ServiceAccount"
)
func readWriteAccessRequestSpecification(obj *schemav1alpha1.MongoDBDatabase) *engineV1alpha1.SecretAccessRequest {
	return &engineV1alpha1.SecretAccessRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name: MongoDBReadWriteSecretAccessRequest,
			Namespace: obj.Namespace,
		},
		Spec:       engineV1alpha1.SecretAccessRequestSpec{
			RoleRef:                          corev1.TypedLocalObjectReference{
				Kind:     KindMongoDBRole,
				Name:     MongoReaderWriterRole,
			},
			Subjects:  []rbacv1.Subject{
				{
					Kind:      KindServiceAccount,
					Name:      MongoDbReadWriteServiceAccount,
					Namespace: obj.Namespace,
				},
			},
		},
		Status:     engineV1alpha1.SecretAccessRequestStatus{},
	}
}

func ReaderWriterRoleSpecification(obj *schemav1alpha1.MongoDBDatabase) *engineV1alpha1.MongoDBRole {
	return &engineV1alpha1.MongoDBRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MongoReaderWriterRole,
			Namespace: obj.Namespace,
		},
		Spec: engineV1alpha1.MongoDBRoleSpec{
			SecretEngineRef: corev1.LocalObjectReference{
				Name: makeSecretEngineName(obj.Name),
			},
			CreationStatements: []string{
				"{ \"db\": \"mydb\", \"roles\": [{ \"role\": \"readWrite\" }] }",
			},
		},
	}
}

func makeSecretEngineName(name string) string {
	return name + "-secret-engine"
}

func secretEngineSpecification(obj *schemav1alpha1.MongoDBDatabase) *engineV1alpha1.SecretEngine {
	return &engineV1alpha1.SecretEngine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      makeSecretEngineName(obj.Name),
			Namespace: obj.Namespace,
		},
		Spec: engineV1alpha1.SecretEngineSpec{
			VaultRef: kmapi.ObjectReference{
				Name:      obj.Spec.VaultRef.Name,
				Namespace: obj.Spec.VaultRef.Namespace,
			},
			SecretEngineConfiguration: engineV1alpha1.SecretEngineConfiguration{
				MongoDB: &engineV1alpha1.MongoDBConfiguration{
					DatabaseRef: v1alpha1.AppReference{
						Name:      obj.Spec.DatabaseRef.Name,
						Namespace: obj.Spec.DatabaseRef.Namespace,
					},
					PluginName: MongoDBDatabasePlugin,
				},
			},
		},
	}
}

func GetVaultServerObject(obj *schemav1alpha1.MongoDBDatabase, vaultVersionedClient *vaultversioned.Clientset) (*vaultV1alpha1.VaultServer, error) {
	item, err := vaultVersionedClient.KubevaultV1alpha1().VaultServers(obj.Spec.VaultRef.Namespace).Get(context.Background(), obj.Spec.VaultRef.Name, metav1.GetOptions{})
	if err != nil {
		klog.Fatalf("Error when getting the vault server object. %s", err.Error())
		return nil, err
	}
	fmt.Println("Vault server object found")
	return item, nil
}

func GetMongoObject(obj *schemav1alpha1.MongoDBDatabase, versionedClient *versioned.Clientset) (*v1alpha2.MongoDB, error) {
	item, err := versionedClient.KubedbV1alpha2().MongoDBs(obj.Spec.DatabaseRef.Namespace).Get(context.Background(), obj.Spec.DatabaseRef.Name, metav1.GetOptions{})
	if err != nil {
		klog.Fatalf("Error when getting the mongoDB object. %s", err.Error())
		return nil, err
	}
	fmt.Println("MongoDB object found")
	return item, nil
}

const (
	masterURL  string = ""
	kubeconfig string = "/home/arnob/.kube/config"
)


//GetTheClients gets the KubernetesClient, kubeDBClient & kubeVaultClient respectively

func GetTheClients() (client.Client, *versioned.Clientset, *vaultversioned.Clientset, error) {
	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
		return nil, nil, nil, err
	}
	//kubeClient, err := kubernetes.NewForConfig(cfg)
	kubeClient, err := client.New(cfg, client.Options{})
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
		return nil, nil, nil, err
	}
	versionedClient, err := versioned.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building versioned clientset: %s", err.Error())
		return nil, nil, nil, err
	}
	vaultServerClient, err := vaultversioned.NewForConfig(cfg)
	if err != nil {
		klog.Fatalf("Error building versioned vault server clientset: %s", err.Error())
		return nil, nil, nil, err
	}
	fmt.Println("All clients has been caught successfully.")
	return kubeClient, versionedClient, vaultServerClient, nil
}
*/
