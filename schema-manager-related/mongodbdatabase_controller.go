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
		se.Spec.SecretEngineConfiguration.MongoDB.PluginName = kvm_engine.DefaultMongoDBDatabasePlugin

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
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		mr := object.(*kvm_engine.MongoDBRole)
		mr.Spec.SecretEngineRef.Name = makeSecretEngineName(obj.Name)
		generatedString := fmt.Sprintf("{ \"db\": \"%s\", \"roles\": [{ \"role\": \"readWrite\" }] }", obj.Spec.DatabaseSchema.Name)
		mr.Spec.CreationStatements = append(mr.Spec.CreationStatements, generatedString)

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
			Namespace: obj.Namespace,
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
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		sar := object.(*kvm_engine.SecretAccessRequest)

		sar.Spec.RoleRef.Kind = kvm_engine.ResourceKindMongoDBRole
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
				Namespace: obj.Namespace,
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
	// To do this, Firstly I have listed the secrets from obj.Namespace, then if that is our required secret -> captured it & decoded it.
	var credentials v1.SecretList
	var singleCred v1.Secret
	err = r.Client.List(ctx, &credentials, &client.ListOptions{Namespace: obj.Namespace})
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
			Namespace: obj.Namespace,
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

	// Get the configMap
	/*var userCm v1.ConfigMap
	err = r.Client.Get(ctx, types.NamespacedName{
		Namespace: "",
		Name:      "",
	}, &userCm)
	if err != nil {
		log.Error(err, "You have not created the configMap yet")
		return ctrl.Result{}, err
	}*/

	// Job related things
	createdJob, vt, err := clientutil.CreateOrPatch(r.Client, &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      JobName,
			Namespace: obj.Namespace,
		},
	}, func(object client.Object, createOp bool) client.Object {
		job := object.(*batchv1.Job)

		if len(job.Spec.Template.Spec.Volumes) == 0 {
			job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, v1.Volume{
				Name:         VolumeNameForPod,
				VolumeSource: obj.Spec.Init.Script.VolumeSource,
				/*
					ConfigMap: &v1.ConfigMapVolumeSource{
						/*LocalObjectReference: v1.LocalObjectReference{
							Name: obj.Spec.Init.Script.ConfigMap.Name,
						},
						Items: []v1.KeyToPath{
							{
								Key:  KeyNameForVolume,
								Path: KeyPathForVolume,
							},
						},
					},*/
			})
		}

		if len(job.Spec.Template.Spec.Containers) == 0 {
			job.Spec.Template.Spec.Containers = append(job.Spec.Template.Spec.Containers, v1.Container{
				Name:  MongoImage,
				Image: MongoImage,
				Env: []v1.EnvVar{
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
					/*
						{ // Checking spec.init.script.configMap
							Name: "HELLO",
							ValueFrom: &v1.EnvVarSource{
								ConfigMapKeyRef: &v1.ConfigMapKeySelector{
									LocalObjectReference: v1.LocalObjectReference{
										Name: obj.Spec.Init.Script.ConfigMap.Name,
									},
									Key: "hello",
								},
							},
						},*/
					{ // checking spec.init.pod_template.spec.env
						Name:  obj.Spec.Init.PodTemplate.Spec.Env[0].Name,
						Value: obj.Spec.Init.PodTemplate.Spec.Env[0].Value,
					},
				},
				Command: []string{"/bin/sh", "-c"},
				/*Args: []string{
					"sleep 30; touch a.txt; sleep 300;",
				},*/
				Args: []string{
					fmt.Sprintf("mongo --host mongodb.db.svc.cluster.local --authenticationDatabase $MONGODB_DATABASE_NAME -u $MONGODB_USERNAME -p $MONGODB_PASSWORD < %v/%v;", obj.Spec.Init.Script.ScriptPath, InitScriptName) +
						"touch a.txt;" +
						"sleep 300;",
				},
				ImagePullPolicy: v1.PullAlways,
				VolumeMounts: []v1.VolumeMount{
					{
						Name:      VolumeNameForPod,
						MountPath: obj.Spec.Init.Script.ScriptPath,
					},
				},
			})
		}

		job.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyOnFailure
		job.Spec.BackoffLimit = func(i int32) *int32 { return &i }(5)

		if createOp {
			core_util.EnsureOwnerReference(&job.ObjectMeta, metav1.NewControllerRef(&obj, kdm.SchemeGroupVersion.WithKind(ResourceKindMongoDBDatabase)))
		}
		return job
	})
	if err != nil {
		log.Error(err, "Can't create the job")
		return ctrl.Result{}, err
	}

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
